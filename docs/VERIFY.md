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

