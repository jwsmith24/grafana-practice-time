# Grafana Troubleshooting Lab: Build Plan

This is the delivery plan for the *Grafana Troubleshooting Lab* spec (Oct 21, 2026). Each spec milestone is an **epic issue**. Under it, each sub-issue is a **thin vertical slice**: it ends with something runnable, and each of its acceptance criteria maps to one or more named tests.

Schedule from the spec: M1–M6 are done by **Oct 22, 2026**; the dry run (M7) runs between then and the session on **Oct 29, 2026**.

## Issue index

| Milestone | Epic | Slices |
|---|---|---|
| M1 Log foundation | [#1](https://github.com/jwsmith24/grafana-practice-time/issues/1) | S1.1 [#2](https://github.com/jwsmith24/grafana-practice-time/issues/2) walking skeleton + CI · S1.2 [#3](https://github.com/jwsmith24/grafana-practice-time/issues/3) Alloy → Loki pipeline · S1.3 [#4](https://github.com/jwsmith24/grafana-practice-time/issues/4) Docker Hub spike on ASWF Mac · S1.4 [#5](https://github.com/jwsmith24/grafana-practice-time/issues/5) label/Loki decision |
| M2 Services | [#6](https://github.com/jwsmith24/grafana-practice-time/issues/6) | S2.1 [#7](https://github.com/jwsmith24/grafana-practice-time/issues/7) inventory · S2.2 [#8](https://github.com/jwsmith24/grafana-practice-time/issues/8) orders + request ID · S2.3 [#9](https://github.com/jwsmith24/grafana-practice-time/issues/9) loadgen · S2.4 [#10](https://github.com/jwsmith24/grafana-practice-time/issues/10) red herring |
| M3 Metrics and dashboard | [#11](https://github.com/jwsmith24/grafana-practice-time/issues/11) | S3.1 [#12](https://github.com/jwsmith24/grafana-practice-time/issues/12) metrics + Prometheus · S3.2 [#13](https://github.com/jwsmith24/grafana-practice-time/issues/13) Prometheus panels + home · S3.3 [#14](https://github.com/jwsmith24/grafana-practice-time/issues/14) Loki panels · S3.4 [#15](https://github.com/jwsmith24/grafana-practice-time/issues/15) data links · S3.5 [#16](https://github.com/jwsmith24/grafana-practice-time/issues/16) deploy annotations |
| M4 Incidents | [#17](https://github.com/jwsmith24/grafana-practice-time/issues/17) | S4.1 [#18](https://github.com/jwsmith24/grafana-practice-time/issues/18) switch + `incident.sh` · S4.2 [#19](https://github.com/jwsmith24/grafana-practice-time/issues/19) incident A · S4.3 [#20](https://github.com/jwsmith24/grafana-practice-time/issues/20) incident C · S4.4 [#21](https://github.com/jwsmith24/grafana-practice-time/issues/21) incident B |
| M5 Setup tooling | [#22](https://github.com/jwsmith24/grafana-practice-time/issues/22) | S5.1 [#23](https://github.com/jwsmith24/grafana-practice-time/issues/23) `.env` + pinned images · S5.2 [#24](https://github.com/jwsmith24/grafana-practice-time/issues/24) `check.sh` · S5.3 [#25](https://github.com/jwsmith24/grafana-practice-time/issues/25) reset · S5.4 [#26](https://github.com/jwsmith24/grafana-practice-time/issues/26) offline · S5.5 [#27](https://github.com/jwsmith24/grafana-practice-time/issues/27) memory · S5.6 [#28](https://github.com/jwsmith24/grafana-practice-time/issues/28) README + hygiene · S5.7 [#29](https://github.com/jwsmith24/grafana-practice-time/issues/29) timed Mac setup · S5.8 [#30](https://github.com/jwsmith24/grafana-practice-time/issues/30) GHCR fallback (conditional) |
| M6 Materials | [#31](https://github.com/jwsmith24/grafana-practice-time/issues/31) | S6.1 [#32](https://github.com/jwsmith24/grafana-practice-time/issues/32) cheat sheet · S6.2 [#33](https://github.com/jwsmith24/grafana-practice-time/issues/33) worksheet · S6.3 [#34](https://github.com/jwsmith24/grafana-practice-time/issues/34) answer key (private) · S6.4 [#35](https://github.com/jwsmith24/grafana-practice-time/issues/35) demo script · S6.5 [#54](https://github.com/jwsmith24/grafana-practice-time/issues/54) gated worksheet (`./unlock.sh`) |
| M7 Dry run | [#36](https://github.com/jwsmith24/grafana-practice-time/issues/36) | S7.1 [#37](https://github.com/jwsmith24/grafana-practice-time/issues/37) run the dry run · S7.2 [#38](https://github.com/jwsmith24/grafana-practice-time/issues/38) fix findings |

**Mode letters.** The modes were relabelled so the worksheet meets them in order A, B, C. In this repo, mode A is the slow inventory (#19, titled "incident B"), mode B the bad release (#21, titled "incident C"), and mode C the one broken item (#20, titled "incident A"). The issues and test IDs keep their original letters.

## Critical path

```
S1.1 → S1.2 → S2.1 → S2.2 → S2.3 ─┬→ S3.1 → S3.2 → S3.3 → S3.4
          S1.4 ┘                  │            └→ S3.5 ─┐
                                  └→ S4.1 → S4.2 → S4.3 → S4.4 → S5.3
                                                   S6.1 → S6.2 → S6.3
S5.1 / S5.2 / S5.6 can run in parallel once S2.3 + S3.1 land.
S1.3 (Mac pull spike) should happen on day 1; it decides S5.8.
```

## Test conventions

Each acceptance criterion in an issue names its tests as `T<slice>-<n>` (automated) or `MT<slice>-<n>` (manual), tagged with a type:

| Tag | Type | Where | Runs |
|---|---|---|---|
| `[U]` | Go unit test | `services/**/_test.go` (`go test ./...`) | every push |
| `[S]` | Static check: compose config, dashboard JSON, Alloy/Prometheus config, Markdown structure, shellcheck, bats tests for scripts with a stubbed `docker` | `tests/static`, `tests/scripts` | every push |
| `[I]` | Integration test against the live compose stack | `tests/integration` (Go, `//go:build integration`); tests that start, stop, reset or break the stack themselves go in `tests/lifecycle` (`//go:build lifecycle`, its own CI job) | every push (fast); nightly for long runs |
| `[E]` | Browser e2e (Playwright) | `tests/e2e` | every push |
| `[M]` | Manual check, with the result recorded as a comment on its issue | — | per issue |

Harness decisions (introduced in S1.1):

- Integration tests reach Loki and Prometheus through Grafana's datasource proxy (`/api/datasources/proxy/uid/<uid>/…`). A CI-only `docker-compose.test.yml` override publishes extra ports (orders 8080, etc.). The attendee compose file publishes only Grafana.
- `INCIDENT_DURATION` (default `3m`) can be shortened in CI so incident tests are fast. A **nightly** job runs with the default to cover the spec's 3-minute auto-clear and the full-session soak tests (T4.1-6 default, T4.4-7, T5.5-x).
- Fixed datasource UIDs are `loki` and `prometheus`; the dashboard UID is `lab-overview`.
- Repo additions beyond the spec's layout: `tests/`, `.github/workflows/`, `docker-compose.test.yml`, `docs/`.

## Lab acceptance criteria → tests

| # | Spec acceptance criterion | Automated tests | Manual tests |
|---|---|---|---|
| AC1 | A clean clone reaches a passing `check.sh` in under 10 min on Apple Silicon and Intel Macs | T5.7-4 (CI cold-start proxy), T5.1-3 (multi-arch images), T5.2-1 | MT5.7-1, MT5.7-2, MT7.1-1 |
| AC2 | The stack uses under 2 GB of memory with loadgen running | T5.5-1, T5.5-2, T5.5-3 | MT5.5-1 |
| AC3 | After the first image pull, the lab starts and runs with networking disabled | T5.4-1, T5.4-2, T5.4-3, T5.1-5 | MT5.4-1 |
| AC4 | Each incident's first signal is on the dashboard within 30 s of triggering, and it clears after 3 min | T4.1-2, T4.1-6, T4.2-3, T4.2-5, T4.3-3, T4.3-5, T4.4-4, T4.4-6 | MT7.1-4 |
| AC5 | Each incident's root cause is findable using only the cheat-sheet queries | T6.1-3, T6.1-4 | — |
| AC6 | The red herring is the highest-volume WARN line in the lab | T2.4-3 (baseline), T4.4-7 (full session) | — |
| AC7 | `docker compose down -v` then `up -d` restores a clean lab with no manual steps | T5.3-1, T5.3-2, T5.3-3, T5.3-4 | — |
| AC8 | A dry-run pair completes the worksheet in 17 minutes or less | — | MT6.5-1 (facilitator pre-check; supersedes MT6.2-1), MT7.1-2 |

## Milestone "done when" → tests

| Milestone | Spec "done when" | Exit test |
|---|---|---|
| M1 | Lines appear in Explore with `service` and `level` labels | T1.2-1, T1.2-2 |
| M2 | One `request_id` query returns lines from both services | T2.2-6 |
| M3 | A panel's data link opens the matching logs in Explore | T3.4-2 |
| M4 | Each incident is findable by its expected discovery path | T4.2-4, T4.3-4, T4.4-5 |
| M5 | `check.sh`, `.env` port handling, pinned images, README | T5.1-x, T5.2-x, T5.6-x |
| M6 | Worksheet, answer key, demo script, cheat sheet | T6.1-x – T6.5-x |
| M7 | The dry-run pair completes setup and the worksheet unaided | MT7.1-1, MT7.1-2 |

## Spec gaps and risks found while planning

1. **AC6 conflicts with incident B.** Incident B's inventory WARN `auth token rejected` fires on every orders call, about 5/s. The red herring fires at 0.5/s. As written, the red herring stops being the loudest WARN once B runs. Recommendation: rate-limit the token WARN in inventory. Decide in S4.4 (#21).
2. **Truncated spec cells.**
   - Incident C's discovery path / root-cause evidence (read as "`| json` and `count_over_time` by `item_id` show one item").
   - The cheat-sheet contents after "regex,".
   Both are cut off identically in the PDF and the markdown export: the `|` in `| json` breaks the table cell. Please confirm both against the original doc.
3. **Deadline.** M1–M6 are due Oct 22, 2026, the day after this plan was written. If anything slips, prioritize the critical path above. The worksheet and cheat sheet (M6) block the dry run; the GHCR fallback (S5.8) is conditional.
4. **The incident C signal may be faint.** `SKU-1042` is ~2% of ~5 req/s (≈ 0.1 req/s). With a 5 s scrape and a `[1m]` rate window, the 30 s first-signal target is tight. S4.3 makes the SKU weight tunable and decides using T4.3-3.
5. **Loki adds labels on its own.** Recent Loki versions add `service_name`, and Alloy file sources add `filename`. Both are dropped or disabled so streams carry only `service`, `level`, and `env` (T1.2-2).
6. **The incident switch must be silent.** Services must not log "incident" or anything that names the cause. `incident.sh` passes only the letter. Covered by T4.1-5, T4.1-7, T4.1-8, T5.6-3, and T6.2-2.
7. **Incident B clear behavior is unspecified.** Proposal: on clear, orders logs a rollback `event="deploy"` with `version="1.4.2"`, so the annotations stay consistent. Decide in S4.4.
8. **Open questions in the spec** (ASWF label names; whether ASWF uses Loki). These affect the Alloy labels and the cheat sheet, so they're tracked as S1.4 (#5) and should be answered before M2.
