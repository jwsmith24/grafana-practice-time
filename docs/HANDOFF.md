# Handoff: start building the Grafana Troubleshooting Lab

You're picking up a planned but unbuilt project. The repo has no code yet: just `README.md`, `docs/PLAN.md`, and this file. Your job is to implement the GitHub issues, slice by slice, in the order below.

## Read first

1. `docs/PLAN.md`: the issue index, critical path, test conventions, the acceptance-criteria → test traceability, and known spec risks.
2. The GitHub issues in `jwsmith24/grafana-practice-time`:
   - Epics #1, #6, #11, #17, #22, #31, #36 (one per milestone, M1–M7).
   - Each epic's sub-issues are the work items. Every slice issue has Scope plus an **Acceptance criteria → tests** table. Those named tests (`T2.2-6`, etc.) are the definition of done.
3. The spec itself is **not in the repo** (see "Public repo" below). The issues carry everything needed from it. If you need the full spec, ask the user.

## How to work

- **One slice = one branch = one PR.** Put `Closes #N` in the PR body. Don't bundle slices.
- **Implement the slice's named tests along with the code.** The `[U]`, `[S]`, `[I]`, and `[E]` tests are part of the slice. For `[M]` (manual) tests, leave a checklist comment on the issue for the human; don't fake them.
- **Keep CI green.** S1.1 creates CI, and every later slice extends it.
- **Ask the human** instead of guessing on anything listed under "Decisions pending".
- **Stay inside the spec's repo layout:**
  - `services/` (one Go module; `cmd/{orders,inventory,loadgen}`; `internal/`)
  - `alloy/`, `loki/`, `prometheus/`, `grafana/provisioning/{datasources,dashboards}`
  - `materials/`
  - root `docker-compose.yml`, `.env.example`, `incident.sh`, `check.sh`

  The approved additions are `tests/`, `.github/workflows/`, `docker-compose.test.yml`, and `docs/`.

## Start here: work order

| Order | Slice | Issue | Notes |
|---|---|---|---|
| 1 | S1.1 Walking skeleton: Grafana + Loki + test harness + CI | #2 | Sets up `tests/` (Go, `integration` build tag), `docker-compose.test.yml`, and `.github/workflows/ci.yml`. Everything later builds on it. |
| 2 | S1.2 Dummy logs → Alloy → Loki | #3 | |
| 3 | S2.1 Inventory | #7 | Check #5 (label decision) first. If it's still open, proceed with `service`/`level`/`env` as the spec says. |
| 4 | S2.2 Orders + request ID | #8 | M2 exit test T2.2-6 |
| 5 | S2.3 Loadgen | #9 | Removes `dummy` |
| 6 | S2.4 Red herring | #10 | |
| 7 | S3.1 → S3.5 | #12–#16 | M3 exit test T3.4-2 (Playwright) |
| 8 | S4.1 → S4.2 (B) → S4.3 (A) → S4.4 (C) | #18–#21 | S4.4 has pending decisions. Build S4.1–S4.3 first. |
| 9 | S5.1, S5.2, S5.6, then S5.3, S5.4, S5.5 | #23–#28 | S5.1/S5.2/S5.6 can start once S2.3 + S3.1 land |
| 10 | S6.1, S6.2, S6.4 | #32, #33, #35 | S6.3 (#34, answer key) stays **outside** this repo |

Not for you: #4 (S1.3, manual pull test on an ASWF Mac), #29 (S5.7, timed Mac setup), #37 (dry run), and #30 (GHCR mirror, only if #4 fails).

**Deadline:** the spec wants M1–M6 done by Oct 22, 2026, with the session on Oct 29. Favor the critical path over polish.

## Fixed conventions (from the spec and the plan; don't change)

- **Ports:** Grafana is on host port `${GRAFANA_PORT:-3300}`. The attendee compose file publishes nothing else; test-only ports go in `docker-compose.test.yml`.
- **Fixed UIDs:**
  - datasource `loki` (the default)
  - datasource `prometheus`
  - dashboard `lab-overview` (the home dashboard)
- **Grafana auth:** anonymous access with the `Editor` role; admin login kept.
- **Loki labels:** exactly `service`, `level`, and `env="lab"`.
  - Drop Alloy's `filename` label.
  - Disable Loki's automatic `service_name` (`discover_service_name: []`).
  - Every other field stays in the line and is parsed with `| json`.
- **Logs:**
  - `log/slog` JSON, written to stdout and to `/var/log/lab/<service>.log` on the `lab-logs` volume.
  - The file is truncated on service start.
  - One line per entry; stack traces stay inside `error`.
  - Field names follow the spec schema in #7.
- **Alloy:** mounts `lab-logs` read-only, uses `local.file_match`, and never touches the Docker socket.
- **Metrics:**
  - `http_requests_total{service,endpoint,status}`
  - `http_request_duration_seconds{service,endpoint}`
  - `inventory_upstream_errors_total{reason}` (orders only)

  `endpoint` is the route template, e.g. `GET /inventory/{item_id}`. Prometheus scrapes every 5 s.
- **Request IDs:** loadgen sends a UUID v4 as `X-Request-ID`; orders logs it and forwards it to inventory.
- **Orders → inventory:**
  - 2 s timeout → orders returns 504 (`context deadline exceeded`)
  - inventory 5xx → orders returns 502
  - orders always sends a service token, and inventory validates it (incident C needs this)
- **Loadgen:** about 5 req/s over ~50 item IDs in `SKU-NNNN` form, including `SKU-1042`.
- **Red herring:** inventory logs WARN `config key ttl_seconds is deprecated, use cache_ttl` every 2 s.
- **Deploy events:** orders logs `event="deploy"` `version="1.4.2"` at startup. Incident C logs `1.4.3`.
- **Incidents:**
  - `./incident.sh A|B|C|clear|status` passes **only the letter** through `docker compose exec`.
  - Each incident auto-clears after `INCIDENT_DURATION`, which defaults to `3m`. The override exists for fast CI.
  - **No log line or script output may say "incident" or hint at a cause.**
- **Images:** pinned to exact versions, multi-arch (amd64 + arm64), and prefixed with `${IMAGE_REGISTRY}`. Exactly 7 containers in the final stack.
- **Scripts:** must run on macOS's default bash 3.2 and pass shellcheck.

## Decisions pending (ask the human; don't decide alone)

1. **#5 / S1.4:** which labels ASWF's real Grafana uses, and whether it uses Loki. This may rename the labels and add a cheat-sheet translation section.
2. **#21 / S4.4: AC6 conflict.** Incident C's `auth token rejected` WARN would outnumber the red herring about 10×. The recommendation is to rate-limit that WARN in inventory.
3. **#21 / S4.4:** on clear, should orders log a rollback deploy event (`1.4.2`)? Recommended: yes.
4. **#20 / S4.3:** whether to weight loadgen toward `SKU-1042` so incident A shows up on the dashboard within 30 s.
5. **Truncated spec text** (incident A's discovery path, the cheat-sheet contents). The issues use the best reading; confirm before writing the materials.

## Public repo: don't leak the answers

The repo is public, and attendees clone it. Keep the answer key, the spec, and anything that narrates incident causes **out of the repo**. That covers attendee-facing files like README, `materials/`, and script output. Code necessarily contains the faults, so keep it unremarkable: no comments like `// incident A: break SKU-1042`. `docs/PLAN.md` and the issues do describe causes; the human knows and will decide whether to make them private before the session.

## Environment notes

- Docker Compose v2 (`docker compose`), with `--wait` for health-gated startup. Give every service a healthcheck where the image allows it.
- In Claude Code cloud sessions, Chromium is preinstalled for Playwright (`PLAYWRIGHT_BROWSERS_PATH=/opt/pw-browsers`). Don't run `playwright install`.
- The integration tests in `tests/integration` assume the stack is up via `docker compose -f docker-compose.yml -f docker-compose.test.yml up -d --wait`.
