# Milestone verification

Each milestone PR copies the matching section into its description, and the reviewer runs it before merging. Every section has two parts:

- **Automated:** the slice tests that cover the milestone. CI runs them, and you can run them locally.
- **By hand:** the milestone's "done when" from the spec, checked against a running stack.

Start the stack the way CI does. The test override publishes orders on 8080, Loki on 3100, and Prometheus on 9090.

```sh
export INCIDENT_DURATION=15s   # only matters from M4 on
docker compose -f docker-compose.yml -f docker-compose.test.yml up -d --build --wait
G=http://localhost:3300
L=$G/api/datasources/proxy/uid/loki/loki/api/v1
```

Run the whole automated suite:

```sh
(cd services && go test -race ./...)                       # [U]
(cd tests && go test ./...)                                # [S]
bats tests/scripts && shellcheck incident.sh               # [S], from M4
(cd tests && go test -tags integration -count=1 ./integration/...)   # [I], ~2 min
(cd tests/e2e && npm ci && npx playwright test)            # [E]
(cd tests && LIFECYCLE_OFFLINE=1 go test -tags lifecycle -count=1 -timeout 60m ./lifecycle/...)  # [I], from M5, ~15 min
```

The lifecycle tests start, stop and reset the stack themselves, so don't run them while another suite is using it. `LIFECYCLE_OFFLINE=1` lets them block outbound traffic with iptables (root or passwordless sudo). `LAB_SOAK=1` adds the 14-minute memory soak, which the nightly run sets.

Run the integration tests against a fresh stack (`down -v` first). T2.4-4 asserts that the last 2 minutes contain no ERROR lines, and later tests deliberately create some.

---

## M1 Log foundation: lines appear in Explore with `service` and `level` labels

**Automated:** T1.1-1 to T1.1-5, and T1.2-1 to T1.2-5 (`tests/integration/grafana_test.go`, `pipeline_test.go`, `tests/static/compose_test.go`).

**By hand:**
```sh
curl -s $L/labels              # {"status":"success","data":["env","level","service"]}
curl -s $L/label/env/values    # {"status":"success","data":["lab"]}
```
1. Open http://localhost:3300/explore without logging in. Loki should be the selected datasource.
2. Run `{service="orders"}`. Lines appear, each stream shows only the labels `env`, `level`, and `service`, and `| json` parses every line.

## M2 Services: one `request_id` query returns lines from both services

**Automated:** T2.1-x to T2.4-x. The exit test is **T2.2-6** (`TestRequestIDAcrossServices`).

**By hand:**
```sh
ID=$(cat /proc/sys/kernel/random/uuid)
curl -s -o /dev/null -w '%{http_code}\n' -X POST localhost:8080/orders \
  -H 'Content-Type: application/json' -H "X-Request-ID: $ID" \
  -d '{"item_id":"SKU-1001","quantity":1}'                 # 201
sleep 5
curl -s -G $L/query_range \
  --data-urlencode "query={service=~\"orders|inventory\"} | json | request_id=\"$ID\"" \
  --data-urlencode "start=$(( $(date +%s) - 300 ))000000000" \
  | python3 -c 'import json,sys; print(sorted(s["stream"]["service"] for s in json.load(sys.stdin)["data"]["result"]))'
# ['inventory', 'orders']
```
Also check, in Explore:
- `{service="loadgen"}` has lines.
- `{level="WARN"}` is only `config key ttl_seconds is deprecated, use cache_ttl`.
- `{level="ERROR"}` is empty on a healthy stack.

## M3 Metrics and dashboard: a panel's data link opens the matching logs in Explore

**Automated:** T3.1-x to T3.5-x. The exit test is **T3.4-2** (`tests/e2e/specs/data-links.spec.ts`).

**By hand:**
```sh
curl -s $G/api/dashboards/home | python3 -c 'import json,sys; d=json.load(sys.stdin)["dashboard"]; print(d["uid"], [p["title"] for p in d["panels"]])'
# lab-overview ['Request rate by service', 'Error rate (5xx) by endpoint', 'p95 latency by service', 'Error log volume by service', 'Recent errors']
```
1. Open http://localhost:3300 without logging in. The home page is **Lab Overview**, and request rate and p95 show data for orders and inventory.
2. On *Request rate by service*, click `orders` in the legend, click on the line, then choose **Logs for orders**. Explore opens on Loki with `{service="orders"}` and the dashboard's time range, and log rows render.
3. A yellow deploy marker sits at stack start. Hovering it shows `1.4.2`.

## M4 Incidents: each incident is findable by its expected discovery path

**Automated:** T4.1-x to T4.4-x. The exit tests are T4.2-4, T4.3-4, and T4.4-5.

Files: `services/internal/incident`, `services/cmd/labctl`, `tests/scripts/incident.bats`, and `tests/integration/incident_test.go`, `mode_a_test.go`, `mode_b_test.go`, `mode_c_test.go`, `session_test.go`. CI runs them with `INCIDENT_DURATION=15s`. The nightly run uses the 3m default, which makes `session_test.go` a full-session soak.

**By hand**, with the stack started as above (`INCIDENT_DURATION=15s`):
```sh
./incident.sh status            # none
./incident.sh A                 # active: A (15s left)
./incident.sh status            # active: A (…s left)
sleep 16; ./incident.sh status  # none
./incident.sh X; echo $?        # usage on stderr, then 2
./incident.sh b                 # active: B (15s left)   (lower case works)
./incident.sh clear             # cleared
```
For each of `A`, `B`, and `C`, on a stack that has been quiet for a minute:
1. Open http://localhost:3300 (Lab Overview, last 5 minutes, refresh every 5 s) and run `./incident.sh <letter>`.
2. Within 30 s, a dashboard panel shows the change.
3. Follow the discovery path given for that letter in the PR description, from that panel to the evidence in Explore.
4. The switch ends by itself after 15 s, and `./incident.sh status` prints `none`. New requests succeed at once. The panels use 1m windows, so they take about a minute to settle.

Then check that switching leaves no trace in the logs: in Explore, `{env="lab"} |~ "(?i)incident"` returns no lines.

## M5 Setup tooling: a fresh clone reaches a passing `check.sh` with no help

**Automated:** T5.1-x to T5.6-x.
- **Static:** `tests/static/env_test.go`, `readme_test.go`, `tests/hygiene`, and gitleaks in CI.
- **Lifecycle:** `tests/lifecycle`, a separate CI job.
- **Scripts:** shellcheck, plus `bash -n` under `bash:3.2.57`.

The exit test is **T5.3-1** (`TestReset`).

**By hand**, as an attendee would, starting from a fresh clone and no `.env`:
```sh
cp .env.example .env
docker compose up -d
./check.sh; echo $?             # ten PASS lines, "All checks passed", then 0
docker compose stop loki
./check.sh; echo $?             # FAIL  Loki is ready, a fix, the facilitator line, then 1
docker compose up -d --wait
docker compose down -v && docker compose up -d --wait && ./check.sh   # passes again
docker stats --no-stream        # the lab's containers add up to well under 2 GB
```
Then:
1. Follow `README.md` from top to bottom, and check that every command in it works as written.
2. Set `GRAFANA_PORT=3301` in `.env` and run `docker compose up -d`. Grafana moves to http://localhost:3301, and `./check.sh` follows it.

`[M]` on a real Mac: MT5.4-1 (Wi-Fi off) on #26, and MT5.5-1 (Docker Desktop memory) on #27, recorded during S5.7 (#29).


## M6 Materials: worksheet, demo script and cheat sheet

**Automated:** T6.1-x, T6.2-x and T6.4-x.
- **Static:** `tests/static/cheatsheet_test.go`, `worksheet_test.go` and `demo_test.go` (structure, techniques, LogQL only, screenshots exist, the demo runs only `./incident.sh B`). `tests/hygiene` keeps the evidence of every mode out of the cheat sheet and the worksheet, and the other modes' evidence out of the demo script.
- **Integration:**
  - `materials_baseline_test.go`: T6.2-3, the warm-ups and the stretch question at baseline.
  - `materials_demo_test.go`: T6.4-3, every demo query during mode B, in order.
  - `materials_test.go`: T6.1-3, every cheat-sheet query runs. T6.1-4 (lab AC5) chains cheat-sheet queries only, each filled in from the results before it, from a mode's first signal to its root cause, for each of A, B and C.

The exit tests are T6.1-4 and T6.4-3. S6.3, the answer key, is kept outside this repo.

**By hand**, with the stack started as above:
1. Print `materials/logql-cheat-sheet.md` to PDF (for example from GitHub's rendered view). It fits on one page (MT6.1-1, on #32).
2. Read `materials/worksheet.md` (question 1; since S6.5, questions 2 to 6 are printed by `./unlock.sh`), and none of them names a cause. Paste each cheat-sheet query into Explore with the placeholders filled in, and check that each one returns data.
3. Walk through `materials/demo-script.md` with `./incident.sh B`. Each step's query returns what the step describes, and its screenshot matches the current UI (MT6.4-2, on #35).
4. To regenerate the screenshots after a dashboard or Grafana change, start a fresh stack (`down -v`, then `up`), wait two minutes, and run:
   ```sh
   (cd tests/e2e && npx playwright test -c capture.config.ts)   # about 1.5 min; rewrites materials/demo/*.png
   ```

`[M]` for the facilitator: MT6.2-1 (solo worksheet run in 12 minutes or less; superseded by MT6.5-1) on #33, and MT6.4-1 (timed rehearsal in 15 minutes or less) on #35.

## S6.5 Gated worksheet: each answer unlocks the next question

**Automated:** T6.5-1 to T6.5-6, and T6.2-1 as updated.
- **Unit:** `services/cmd/labctl/unlock_test.go`. T6.5-1 runs `labctl unlock` against a made-up question table: lenient matching, the neutral wrong-answer line and exit 1, usage and exit 2, the closing message. `TestLabWorksheet` checks the real table: questions 1 to 6 in order, each mode question runs only its own letter, and every stored answer is a 64-hex SHA-256 digest.
- **Static:** `tests/static/worksheet_test.go` (T6.2-1, T6.5-3: the worksheet holds only question 1, the intro, the notes template, how to unlock and the escape hatch), `readme_test.go` (T6.5-6), and `tests/hygiene` (T6.5-5: the worksheet's cause-word list also covers `services/cmd/labctl/questions.go`, `unlock.go` and `unlock.sh`).
- **Scripts:** `tests/scripts/unlock.bats` (T6.5-2, with a stubbed `docker`), shellcheck, and `bash -n` under `bash:3.2.57`.
- **Integration:** `materials_unlock_test.go`, T6.5-4. It takes each answer from the running lab with the cheat-sheet queries (reusing the T6.1-4 chains) and feeds it to `./unlock.sh`, from question 1 to the closing message. It sorts after `materials_test.go` and before `session_test.go`.

The exit test is T6.5-4. The answers are not in the repo: they are on the facilitator's answer key (S6.3).

**By hand**, with the stack started as above:
```sh
./unlock.sh 1 wrong; echo $?      # "Not quite. Check your answer and try again.", then 1
./unlock.sh 1; echo $?            # usage on stderr, then 2
./unlock.sh 7 anything; echo $?   # usage on stderr, then 2
```
Then, with the facilitator's answer key, run `./unlock.sh N <answer>` for N = 1 to 6. Each prints the next question (the sixth, the closing message) and exits 0. Try one answer in capitals, with extra spaces and in quotes: it still unlocks. Read every printed question: none names a cause.

`[M]` for the facilitator: MT6.5-1 (solo gated run in 12 minutes or less; supersedes MT6.2-1) and MT6.5-2 (the six passwords are on the answer key), both on #54.
