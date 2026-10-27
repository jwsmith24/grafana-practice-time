# Handoff: Grafana Troubleshooting Lab

You're continuing a nearly built project. M1–M6 are merged, including S6.5 (the gated worksheet, #54, PR #55), which was added on 2026-10-22 before the dry run. What's left is M7, after the human's dry run.

Read this file, then `docs/PLAN.md` for the issue index, the test conventions, and the spec risks, then `docs/VERIFY.md` for how to verify each milestone. The GitHub issues in `jwsmith24/grafana-practice-time` carry the scope and the named tests for every slice. Each slice issue has an **Acceptance criteria → tests** table, and those tests are the definition of done. The spec itself is not in the repo; ask the human if you need it.

## Working rules (from the project owner; follow them)

1. **One milestone at a time.** Finish, verify, and open the PR for a milestone before you start the next one.
2. **One PR per milestone.**
   - The slices are commits, or small groups of commits, inside it.
   - The PR body has a `Closes #N` line for every slice issue it completes.
   - The PR body has a **Milestone verification** section: the milestone's section from `docs/VERIFY.md`, filled in and run. For M4 and later, write that section as part of the milestone, and keep anything that names an incident cause out of the repo (see "Public repo" below).
3. **Implement each slice's named tests with its code.** `[U]`, `[S]`, `[I]`, and `[E]` tests live in the repo. For `[M]` tests, leave a checklist comment on the issue for the human; never fake them.
4. **Keep CI green.** Never skip, disable, or quarantine a test to get green.
5. **Stop and ask the human** about anything under "Decisions pending". Don't decide alone.
6. **Prove it before you push.** Run the suite against a *fresh* stack (`down -v`, then `up`). Several bugs in this project only showed up on a cold stack.

## Status (2026-10-22)

| Milestone | State | Where |
|---|---|---|
| M1 Log foundation (S1.1 #2, S1.2 #3) | **Merged** | PR #50 |
| M2 Services (S2.1–S2.4: #7–#10) | **Merged** | PR #50 |
| M3 Metrics and dashboard (S3.1–S3.5: #12–#16) | **Merged** | PR #50 |
| M4 Incidents (S4.1–S4.4: #18–#21) | **Merged** | PR #51 |
| M5 Setup tooling (S5.1–S5.6: #23–#28) | **Merged** | PR #52 |
| M6 Materials (S6.1 #32, S6.2 #33, S6.4 #35) | **Merged** | PR #53 |
| S6.5 Gated worksheet (#54) | **Merged** | PR #55 |
| M7 Dry run (#37 manual, #38) | Not started | |

- **Playwright retries once in CI.** A line like `1 flaky` in the E2E output means a real problem to fix, not a pass.
- **Manual tests still open:** MT1.1-1 (CI goes red on a failing test) on #2; MT5.4-1 (Wi-Fi off) on #26; MT5.5-1 (Docker Desktop memory) on #27; MT6.1-1 (cheat sheet prints on one page) on #32; MT6.5-1 (solo gated worksheet run ≤ 12 min; supersedes MT6.2-1 on #33) and MT6.5-2 (the six passwords are on the answer key) on #54; MT6.4-1 (timed demo ≤ 15 min) and MT6.4-2 (screenshots match the UI) on #35. Checklists are on the issues.
- **Decision recorded on #5:** labels stay `service`/`level`/`env`, and ASWF uses Loki, so S6.1 needs no translation section. The human still has to update the spec's "Decided" list.
- **Not for an agent:** #4 (S1.3, Mac pull test), #29 (S5.7, timed Mac setup), #37 (dry run), #30 (GHCR fallback, only if #4 fails), #34 (S6.3, the answer key; it stays out of this repo).
- **Disk prerequisite is tight.** The seven images take 2.34 GB, and the service build adds the Go image and build cache. The README keeps the spec's 3 GB; the human has been told it may need to be 4 GB.

## Next steps

1. **M7:** #38 fixes what the human's dry run (#37) finds. It still waits on the dry-run findings; don't start #38 before them.
   - If a finding changes the dashboard or Grafana, regenerate the demo screenshots (see "Materials" below) and ask the human to redo MT6.4-2.
   - If a finding changes a log line, a field, or a cheat-sheet query, T6.1-4 and T6.4-3 will tell you whether the materials still work.

## Decisions pending (ask the human; don't decide alone)

1. ~~#5 label convention~~: resolved, see above.
2. ~~#21 AC6 conflict~~: **decided** — inventory rate-limits the `auth token rejected` WARN (option (a) on the issue). Done in S4.4: at most one per 5 s per caller, carrying `rejected` (the count) and `window_s`. `msg` stays constant, so `topk by (msg)` still groups it.
3. ~~#21 on clear~~: **decided** — orders logs a rollback `event="deploy"` with `version="1.4.2"`.
4. ~~#20 loadgen weighting~~: **decided** — weight toward `SKU-1042`. Done in S4.3: three "bestsellers" (`SKU-1007`, `SKU-1023`, `SKU-1042`) share 20% of orders, so each gets ~8%. The weighting is always on, so it gives nothing away.
5. ~~Truncated spec text~~: **confirmed by the human** (recorded on #32 and #20). Incident C's discovery path is as read: error panel → Explore on orders ERROR lines → `| json` + `count_over_time` by `item_id` shows one item. The cheat-sheet techniques are as listed in #32's scope.

## What exists (map of the code)

```
docker-compose.yml        loki, prometheus, alloy, inventory, orders, loadgen, grafana (7 containers)
docker-compose.test.yml   CI only: publishes loki 3100, prometheus 9090, orders 8080
loki/loki-config.yaml     filesystem/TSDB, 24h retention, discover_service_name: [], discover_log_levels: false
alloy/config.alloy        file_match /var/log/lab/*.log → json → timestamp(time) → labels service, level + env="lab", drop filename
prometheus/prometheus.yml scrape orders:8080, inventory:8080 every 5s
grafana/provisioning/     datasources (uid loki = default, uid prometheus); dashboards/lab-overview.json (home)
services/                 one Go module (client_golang is the only dependency)
  cmd/inventory           GET /inventory/{item_id}; token auth + rate-limited reject WARN; config watcher (deprecated-key WARN every 2s); store reads the lab mode
  cmd/orders              POST /orders, GET /orders/{id}; inventory client (2s timeout, 504/502); deploy event 1.4.2; release watcher (deploy events on mode change)
  cmd/loadgen             ~5 req/s to orders; summary line every 10s
  cmd/labctl              set/clear/status of the lab mode; `unlock N <answer>` for the gated worksheet
                          questions.go: scenarios 2–6 and the closing message, answers as SHA-256 hashes only
                          unlock.go: matching (lowercase, trim, collapse spaces, strip quotes/backticks); 0 right, 1 wrong, 2 usage;
                          a right answer starts the next scenario's mode (3–5 → A, B, C) or clears the lab
  internal/logging        slog JSON → stdout + /var/log/lab/<svc>.log (truncated on start); time = RFC 3339 ms UTC; levels INFO/WARN/ERROR
  internal/httpx          request log line + metrics observer; Annotate / SetLevel; -healthcheck flag
  internal/metrics        http_requests_total, http_request_duration_seconds
  internal/logschema      strict log-line validator used by the service tests
  internal/incident       state file on lab-state volume; Status computes expiry (no timers)
  internal/{auth,catalog,requestid}
  Dockerfile              one cross-compiling build stage; scratch target per service
incident.sh               bash 3.2; passes only the argument to `docker compose exec -T orders /labctl`
unlock.sh                 bash 3.2; passes only `unlock N <answer words…>` (joined by spaces) to the same labctl;
                          during the worksheet it replaces incident.sh (which stays for the demo and the facilitator)
tests/static              Go: compose config, image is static + has no shell, prometheus.yml, dashboard JSON
tests/integration         Go, build tag `integration`; reaches Loki/Prometheus via Grafana's datasource proxy
                          incident_test.go: T4.1 + shared helpers (incidentSh, waitQuiet, orders5xx, postOrder)
                          mode_{a,b,c}_test.go: one TestMode<letter> per incident
                          materials_baseline_test.go (T6.2-3), materials_demo_test.go (T6.4-3), materials_test.go (T6.1-3, T6.1-4 chains)
                          materials_unlock_test.go (T6.5-4): whole gated chain via ./unlock.sh, answers taken from the lab
                          session_test.go: sorts last; T4.4-7 red herring over the whole run
materials/                logql-cheat-sheet.md (10 templates, <placeholder> names), worksheet.md (intro, tips, scenario 1; 2–6 come from ./unlock.sh), demo-script.md (mode A, 7 steps)
                          demo/step-N.png: the demo's screenshot fallbacks, made by tests/e2e/capture
tests/e2e                 Playwright 1.56.1 (data links, deploy annotation)
                          capture/ + capture.config.ts: regenerates materials/demo/*.png; not part of the suite
tests/scripts             bats tests for incident.sh and unlock.sh with a stubbed docker
tests/lifecycle           Go, build tag `lifecycle`: tests that start, stop, reset or break the stack themselves (check.sh
                          faults, port conflict, GRAFANA_PORT, reset, offline via iptables, memory soak, image sizes, multi-arch)
tests/hygiene             denylist scan of tracked files (denylist.txt); per-file cause words for materials/; gitleaks runs in CI
check.sh                  bash 3.2 + curl + docker; PASS/FAIL per check with a fix; reaches Loki/Prometheus via Grafana's proxy
.env.example              GRAFANA_PORT, IMAGE_REGISTRY, pinned tags; compose has the same defaults, so .env is optional
.github/workflows/ci.yml  job test: unit → static → gitleaks → (shellcheck, bash 3.2 -n, bats) → compose up --build --wait → integration → e2e
                          job lifecycle (parallel): compose build → go test -tags lifecycle (LIFECYCLE_OFFLINE=1; LAB_SOAK=1 nightly)
```

## Conventions (fixed by the spec and the plan; don't change)

- **Ports:** Grafana on `${GRAFANA_PORT:-3300}`. The attendee compose file publishes nothing else.
- **Fixed UIDs:**
  - datasource `loki` (the default)
  - datasource `prometheus`
  - dashboard `lab-overview` (the home dashboard)
- **Grafana auth:** anonymous `Editor`; the admin login is kept (`admin`/`admin` unless overridden).
- **Loki labels:** exactly `service`, `level`, and `env="lab"`. Everything else stays in the line and is parsed with `| json`.
- **Logs:**
  - One JSON line per entry. Stack traces stay inside `error`.
  - The file is truncated on start.
  - Level values are uppercase: the issues' LogQL uses `level="WARN"` and `level="ERROR"`.
  - 4xx request lines are `INFO`, so they never compete with the red herring. The exception is an unknown-item order, which is `ERROR` per T3.3-4.
- **Metrics:** `endpoint` is the route template. Orders adds `inventory_upstream_errors_total{reason=timeout|server_error|unauthorized}`, with all three reasons pre-registered at 0.
- **Request IDs:** loadgen sends a UUID v4 `X-Request-ID`, and orders forwards it to inventory.
- **Orders → inventory:** 2 s timeout → 504 with `error="context deadline exceeded"`; inventory 5xx or 401 → 502. A service token (`X-Service-Token`) is always sent and always checked.
- **Red herring:** inventory logs `WARN config key ttl_seconds is deprecated, use cache_ttl` every 2 s.
- **Deploy events:** orders logs `event="deploy"` `version="1.4.2"` right after its startup line. Incident B will log `1.4.3`.
- **Incidents:**
  - `./incident.sh A|B|C|clear|status` passes only the argument.
  - An incident auto-clears after `INCIDENT_DURATION` (default `3m`). CI uses `15s`, and a nightly cron run uses the 3m default.
  - **No log line or script output may say "incident" or hint at a cause.**
- **Images:** pinned exact versions; exactly 7 containers. The `${IMAGE_REGISTRY}` prefix and `.env.example` come in S5.1.
- **Scripts:** macOS bash 3.2, and they must pass shellcheck.

## Environment notes (gotchas that cost time)

- **Docker builds in Claude Code cloud sessions can't download Go modules.** The session's TLS-intercepting proxy isn't trusted inside build containers. Don't change the repo's Dockerfile for this, and don't vendor (13 MB). For local builds, use a scratch copy of the Dockerfile that trusts the session CA:
  ```sh
  S=<your scratchpad>; cp /root/.ccr/ca-bundle.crt $S/ca.crt
  sed 's|^WORKDIR /src|WORKDIR /src\nCOPY --from=ca ca.crt /etc/ssl/certs/sandbox-ca.crt\nENV SSL_CERT_FILE=/etc/ssl/certs/sandbox-ca.crt|' services/Dockerfile > $S/Dockerfile.local
  for s in inventory orders loadgen; do docker build -f $S/Dockerfile.local --build-context ca=$S --target $s -t grafana-lab/$s:dev services; done
  docker compose -f docker-compose.yml -f docker-compose.test.yml up -d --wait   # no --build
  ```
  - **Regenerate `Dockerfile.local` whenever `services/Dockerfile` changes.** A stale copy cost a debugging round once.
  - `go test -skip ServiceImages ./...` in `tests/` skips the one static test that builds images. CI builds normally.
- **Docker Hub sometimes returns 429**, and it can last a long time. BuildKit checks the registry for `golang:…-alpine` even when it's cached. To build offline from the cache, pin the digest in your scratch `Dockerfile.local`: get it with `docker buildx du --verbose | grep golang`, then write `FROM … golang:1.24.9-alpine@sha256:<digest> AS build`.
- **Playwright:**
  - Chromium is preinstalled (`PLAYWRIGHT_BROWSERS_PATH=/opt/pw-browsers`); don't run `playwright install` locally. CI does run it.
  - The config pins `locale: 'en-US'`, because the sandbox's `en-US@posix` locale stops Grafana's frontend from booting.
  - Data links only appear for the series under the cursor, so `followDataLink` clicks on the line itself. It finds the line by scanning the panel canvas.
- **bats and shellcheck** aren't preinstalled: `apt-get install -y bats shellcheck`.
- **New Prometheus series.** On a fresh stack, the first 5xx creates a new `http_requests_total{status="502"}` series, and `rate()` reads 0 until the counter rises between two scrapes. Any test that forces errors must keep them coming across at least 2 scrapes (more than 10 s).
- **Every incident test calls `waitQuiet` first.** All three letters are real faults now (T4.1-6 switches on A), so wait until orders' 5xx rate is back to 0 before triggering. Otherwise a "5xx > 0 within 30 s" check can pass on leftovers.
- **Zeroed 5xx series.** Orders initialises `http_requests_total` for `POST /orders` 502/504, and inventory for 500, at 0 (`metrics.HTTP.Init`). Without that, a short burst of errors between two scrapes creates a new series that `rate()` never sees. Incident C's signal failed half the time on a cold stack before this.
- **Test order matters.** Go runs test files in name order. `baseline_test.go` (no ERROR lines in the last 2 min) runs before tests that deliberately create errors. Incident tests will create a lot of errors, so keep baseline assertions first, or scope them by time.
- **Don't `pkill -f` a pattern that also appears in your own command line.** It kills the shell running it (exit 144).
- **The lifecycle tests own the stack.** Don't run them while integration or e2e tests are using it, and don't run two lifecycle tests at once.
- **Multi-arch builds in the sandbox** need a docker-container builder that trusts the session CA: write a `buildkitd.toml` with `[registry."docker.io"] ca=["<scratchpad>/ca.crt"]`, run `docker buildx create --name sandbox-multiarch --driver docker-container --config <that file>`, and build from a CA-patched Dockerfile copy. Remove the builder afterwards: its container shows up in `docker stats`.
- **Docker isn't running at session start, and the daemon can die mid-session.** Start it with `dockerd > <scratchpad>/dockerd.log 2>&1 &`, then wait for `docker info`. If you get `ERR_CONNECTION_REFUSED` or "Cannot connect to the Docker daemon" partway through, restart it and run `compose up -d --wait` again.
- **The CI log can be hard to read.** On failure, CI dumps compose logs (capped at 200 lines per service), and the failing step's output sits above that dump. Downloading the raw log from the blob URL is blocked by the sandbox proxy (403). Use the GitHub `get_job_logs` tool with a `tail_lines` large enough to reach the step.
- **Deploy annotations and canvas clicks.** The yellow deploy-annotation lines cross every timeseries panel. Any e2e test that clicks on a series line must switch the **Deploys** toggle off first (`followDataLink` does this).

## Materials (M6)

- **The tests read the materials.** T6.1-3, T6.1-4 and T6.4-3 parse the fenced query blocks out of the Markdown and fill the `<placeholders>`. If you edit a query, run the integration suite. The integration tests find a template by a marker it contains (`template(t, "request_id=")`), so keep each template's marker unique.
- **The placeholder names are fixed:** `service`, `level`, `text`, `other_text`, `pattern`, `field`, `value`, `ms`, `request_id`, `range`. T6.1-6 rejects any other name.
- **File order matters here too.** `materials_baseline_test.go` must run before `materials_test.go`, whose mode B chain leaves non-red-herring WARN lines that the stretch scenario's check would see.
- **Short mode tests must wait for their own errors to stop.** The M6 chains clear a mode seconds after starting it, often before Prometheus has scraped any 5xx, so the next test's `waitQuiet` reads 0 while 504s and 502s are still in flight. Those then land in the next mode's window (it broke T4.3-4 once). `triggerMode`'s cleanup waits until orders has logged no ERROR line for 10 s. Use `triggerMode` for any new test that switches a mode on.
- **T6.1-4 for C starts from the dashboard's Deploys annotation.** The chain reads the field and value to filter on from the annotation's expression, the way an attendee reads the marker.
- **Regenerating the screenshots:** start a fresh stack (`down -v`, `up`), wait about two minutes so the graphs have a healthy stretch, then run `(cd tests/e2e && npx playwright test -c capture.config.ts)`. A fresh stack matters: on a used one, earlier modes' deploy markers and WARN lines show up in the pictures. The spec keeps mode A running by re-running `./incident.sh A` every 10 s, so `INCIDENT_DURATION` doesn't matter. Look at every image before you commit.
- **The gated worksheet (S6.5).** Scenarios 2–6 live in `services/cmd/labctl/questions.go`; editing them means rebuilding the orders image. After the human's own run (2026-10-22) the questions were rewritten as step-by-step instructions that point at cheat-sheet query numbers and dashboard panels, the notes template was dropped, and unlocking starts and clears the modes itself, so pairs never run `incident.sh`. The human chose this over the original "don't narrow the discovery path" rule. The answers are stored only as `sha256("lab-q<N>:" + normalize(answer))`, and the plain answers are on the facilitator's answer key, never in the repo, an issue or a PR. T6.5-4 writes down only scenario 1's field and scenario 3's service, and takes the other answers from the lab. Any change to a mode's evidence that changes an answer needs a new hash and an answer-key update. Later the human asked for the questions to become small scenarios, each a problem someone at the shop brings the pair and each practising one part of Grafana (Explore, a field filter, a panel's tooltip and legend, annotations, a data link plus a metric query, the Table view), still ramping in difficulty. To make the ramp work, scenarios 3–5 were reordered (slowest service, then the marker's version, then the one failing item), so the hashes of answers 3–5 changed (the hash includes N) though the answers themselves didn't; the answer key's order needs the same change. The modes were relabelled to match, so scenarios 3–5 run A, B, C in order: **mode A is now the slow inventory (issue #19, titled B), mode B the bad release (#21, titled C), and mode C the one broken item (#20, titled A).** The issues, and test IDs such as T4.2 and T4.4, keep their original letters; everything in the repo uses the new ones.
- **Cause words.** `tests/hygiene` keeps every mode's evidence (`sku`, `token`, `deploy`, `1.4.3`, `deadline`, `timeout`, …) out of the cheat sheet, the worksheet, `labctl`'s unlock messages and `unlock.sh`, the answers out of `labctl`'s questions (which may name the latency panel and the Deploys toggle), and B's and C's out of the demo script. The words aren't matched in the PNGs, so check them by eye.

## Public repo: don't leak the answers

The repo is public, and attendees clone it. Keep the answer key, the spec, and anything that narrates incident causes **out of the repo**. That includes attendee-facing files (README, `materials/`, script output) and `docs/VERIFY.md`. Fault code has to exist, but keep it unremarkable: no comments like `// incident C: break SKU-1042`. `docs/PLAN.md` and the issues do describe causes; the human knows and will decide whether to make them private.
