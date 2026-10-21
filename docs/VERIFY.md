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
```

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

**By hand:** fill this in with the M4 PR. As a minimum:
```sh
./incident.sh status    # none
./incident.sh A         # active: A (15s left)
./incident.sh status    # active: A (...)
sleep 16; ./incident.sh status   # none   (with INCIDENT_DURATION=15s; default 3m)
./incident.sh X; echo $?        # usage on stderr, exit 2
```
Then, for each incident letter, describe the discovery path from the spec. Check that the first signal reaches the dashboard within 30 s and that the incident clears by itself. Keep those steps in the PR description only, not in this file: the repo is public, and attendees read it.
