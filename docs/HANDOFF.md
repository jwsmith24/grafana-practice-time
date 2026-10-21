# Handoff: Grafana Troubleshooting Lab

You're continuing a partly built project. M1–M3 are implemented, tested, and waiting in one PR. M4 has started.

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

## Status (2026-10-21, 20:00 UTC)

| Milestone | State | Where |
|---|---|---|
| M1 Log foundation (S1.1 #2, S1.2 #3) | **Merged** | PR #50 |
| M2 Services (S2.1–S2.4: #7–#10) | **Merged** | PR #50 |
| M3 Metrics and dashboard (S3.1–S3.5: #12–#16) | **Merged** | PR #50 |
| M4 Incidents (S4.1–S4.4: #18–#21) | S4.1 done and verified on a cold stack; S4.2 next; no PR | branch `claude/practical-brown-l9t5pe` (continues `claude/s4.1-incident-switch`) |
| M5 Setup tooling (#23–#30) | Not started | |
| M6 Materials (#32, #33, #35) | Not started | |

- **PR #50 merged M1–M3 into `main`.** It merged just before two follow-up commits landed, so `main` is missing both:
  - **`e2e: switch off deploy annotations before following a data link`.** Without it, T3.4-3 flakes on `main`: the deploy-annotation line can be mistaken for the series line.
  - **The refreshed handoff** (this file).

  Both are on `claude/s4.1-incident-switch` and will reach `main` with the M4 PR. If `main` CI flakes on T3.4-3 before then, ask the human whether to land the fix sooner in a small PR of its own.
- **Playwright retries once in CI.** A line like `1 flaky` in the E2E output means a real problem to fix, not a pass.
- **Manual test still open:** MT1.1-1 (CI goes red on a failing test). The checklist is on #2.
- **Decision recorded on #5:** labels stay `service`/`level`/`env`, and ASWF uses Loki, so S6.1 needs no translation section. The human still has to update the spec's "Decided" list.
- **Not for an agent:** #4 (S1.3, Mac pull test), #29 (S5.7, timed Mac setup), #37 (dry run), #30 (GHCR fallback, only if #4 fails), #34 (S6.3, the answer key; it stays out of this repo).

## Next steps

1. ~~Finish S4.1~~: every test layer passed on a cold stack with `INCIDENT_DURATION=15s` (T4.1-6 cleared in 15.7 s; e2e 3/3, none flaky).
2. **S4.2 (#19, incident B)** has no pending decision. Build it next.
3. **Before S4.3 and S4.4,** ask the human about decisions 2–4 below.
4. **Open one M4 PR, based on `main`,** once S4.1–S4.4 are done and verified. Include the M4 verification section.
5. Then M5. S5.1/S5.2/S5.6 could have started after S2.3 + S3.1, but the rule above is one milestone at a time, then M6.

## Decisions pending (ask the human; don't decide alone)

1. ~~#5 label convention~~: resolved, see above.
2. **#21 / S4.4, AC6 conflict:** incident C's `auth token rejected` WARN would outnumber the red herring about 10×. The recommendation is to rate-limit that WARN in inventory.
3. **#21 / S4.4:** on clear, should orders log a rollback `event="deploy"` with `version="1.4.2"`? Recommended: yes.
4. **#20 / S4.3:** should loadgen weight its traffic toward `SKU-1042` so incident A shows on the dashboard within 30 s? Loadgen currently picks items uniformly: 50 items at about 5 req/s, so `SKU-1042` gets roughly 0.1 req/s.
5. **Truncated spec text** (incident A's discovery path, the cheat-sheet contents): confirm with the human before writing materials (M6).

## What exists (map of the code)

```
docker-compose.yml        loki, prometheus, alloy, inventory, orders, loadgen, grafana (7 containers)
docker-compose.test.yml   CI only: publishes loki 3100, prometheus 9090, orders 8080
loki/loki-config.yaml     filesystem/TSDB, 24h retention, discover_service_name: [], discover_log_levels: false
alloy/config.alloy        file_match /var/log/lab/*.log → json → timestamp(time) → labels service, level + env="lab", drop filename
prometheus/prometheus.yml scrape orders:8080, inventory:8080 every 5s
grafana/provisioning/     datasources (uid loki = default, uid prometheus); dashboards/lab-overview.json (home)
services/                 one Go module (client_golang is the only dependency)
  cmd/inventory           GET /inventory/{item_id}; token auth; config watcher (deprecated-key WARN every 2s)
  cmd/orders              POST /orders, GET /orders/{id}; inventory client (2s timeout, 504/502); deploy event 1.4.2
  cmd/loadgen             ~5 req/s to orders; summary line every 10s
  cmd/labctl              (S4.1 branch) set/clear/status of the lab mode
  internal/logging        slog JSON → stdout + /var/log/lab/<svc>.log (truncated on start); time = RFC 3339 ms UTC; levels INFO/WARN/ERROR
  internal/httpx          request log line + metrics observer; Annotate / SetLevel; -healthcheck flag
  internal/metrics        http_requests_total, http_request_duration_seconds
  internal/logschema      strict log-line validator used by the service tests
  internal/incident       (S4.1 branch) state file on lab-state volume; Status computes expiry (no timers)
  internal/{auth,catalog,requestid}
  Dockerfile              one cross-compiling build stage; scratch target per service
incident.sh               (S4.1 branch) bash 3.2; passes only the argument to `docker compose exec -T orders /labctl`
tests/static              Go: compose config, image is static + has no shell, prometheus.yml, dashboard JSON
tests/integration         Go, build tag `integration`; reaches Loki/Prometheus via Grafana's datasource proxy
tests/e2e                 Playwright 1.56.1 (data links, deploy annotation)
tests/scripts             bats tests for incident.sh with a stubbed docker (S4.1 branch)
.github/workflows/ci.yml  unit → static → (bats, shellcheck) → compose up --build --wait → integration → e2e
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
- **Deploy events:** orders logs `event="deploy"` `version="1.4.2"` right after its startup line. Incident C will log `1.4.3`.
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
- **Docker Hub sometimes returns 429.** Retry after a short wait.
- **Playwright:**
  - Chromium is preinstalled (`PLAYWRIGHT_BROWSERS_PATH=/opt/pw-browsers`); don't run `playwright install` locally. CI does run it.
  - The config pins `locale: 'en-US'`, because the sandbox's `en-US@posix` locale stops Grafana's frontend from booting.
  - Data links only appear for the series under the cursor, so `followDataLink` clicks on the line itself. It finds the line by scanning the panel canvas.
- **bats and shellcheck** aren't preinstalled: `apt-get install -y bats shellcheck`.
- **New Prometheus series.** On a fresh stack, the first 5xx creates a new `http_requests_total{status="502"}` series, and `rate()` reads 0 until the counter rises between two scrapes. Any test that forces errors must keep them coming across at least 2 scrapes (more than 10 s).
- **Test order matters.** Go runs test files in name order. `baseline_test.go` (no ERROR lines in the last 2 min) runs before tests that deliberately create errors. Incident tests will create a lot of errors, so keep baseline assertions first, or scope them by time.
- **Docker isn't running at session start, and the daemon can die mid-session.** Start it with `dockerd > <scratchpad>/dockerd.log 2>&1 &`, then wait for `docker info`. If you get `ERR_CONNECTION_REFUSED` or "Cannot connect to the Docker daemon" partway through, restart it and run `compose up -d --wait` again.
- **The CI log can be hard to read.** On failure, CI dumps compose logs (capped at 200 lines per service), and the failing step's output sits above that dump. Downloading the raw log from the blob URL is blocked by the sandbox proxy (403). Use the GitHub `get_job_logs` tool with a `tail_lines` large enough to reach the step.
- **Deploy annotations and canvas clicks.** The yellow deploy-annotation lines cross every timeseries panel. Any e2e test that clicks on a series line must switch the **Deploys** toggle off first (`followDataLink` does this).

## Public repo: don't leak the answers

The repo is public, and attendees clone it. Keep the answer key, the spec, and anything that narrates incident causes **out of the repo**. That includes attendee-facing files (README, `materials/`, script output) and `docs/VERIFY.md`. Fault code has to exist, but keep it unremarkable: no comments like `// incident A: break SKU-1042`. `docs/PLAN.md` and the issues do describe causes; the human knows and will decide whether to make them private.
