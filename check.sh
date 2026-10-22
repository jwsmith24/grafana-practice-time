#!/usr/bin/env bash
# Check that the lab is up and ready.
#
#   ./check.sh   prints one line per check; exits 0 when all of them pass
set -u

cd "$(dirname "$0")" || exit 1

port=${GRAFANA_PORT:-}
if [ -z "$port" ] && [ -f .env ]; then
  port=$(sed -n 's/^GRAFANA_PORT=//p' .env | tail -n 1)
fi
port=${port:-3300}
grafana="http://localhost:$port"
services="grafana loki alloy prometheus inventory orders loadgen"
failed=0

pass() { printf 'PASS  %s\n' "$1"; }
fail() {
  printf 'FAIL  %s\n      %s\n' "$1" "$2"
  failed=1
}

# get prints the body of a successful GET, or nothing.
get() { curl -fsS -m 5 "$@" 2>/dev/null; }

# shellcheck disable=SC2317  # called through retry
# query runs a PromQL or LogQL instant query through Grafana's datasource
# proxy. $1 is the datasource uid, $2 the API path, $3 the query.
query() { get -G "$grafana/api/datasources/proxy/uid/$1$2" --data-urlencode "query=$3"; }

# retry runs "$@" every 2 s until it succeeds or 15 s pass, so a stack that
# has only just started has time to ship its first lines and scrapes.
retry() {
  deadline=$(($(date +%s) + 15))
  until "$@"; do
    [ "$(date +%s)" -ge "$deadline" ] && return 1
    sleep 2
  done
}

footer() {
  echo
  if [ "$failed" -eq 0 ]; then
    echo "All checks passed. Open $grafana to see the Lab Overview dashboard."
  else
    echo "Some checks failed. Try the fix under each FAIL line, then run ./check.sh again."
    echo "If it still fails, message the facilitator and include this output."
  fi
  exit "$failed"
}

if ! docker info >/dev/null 2>&1; then
  fail "Docker is running" "Docker isn't running. Start Docker Desktop, wait until it says it is running, then run ./check.sh again."
  footer
fi
pass "Docker is running"

running=$(docker compose ps --status running --services 2>/dev/null)
missing=""
for service in $services; do
  printf '%s\n' "$running" | grep -qx "$service" || missing="$missing $service"
done
if [ -z "$missing" ]; then
  pass "all 7 containers are running"
else
  fail "all 7 containers are running" "Not running: ${missing# }. Run: docker compose up -d"
fi

grafana_id=$(docker compose ps -aq grafana 2>/dev/null)
grafana_err=""
if [ -n "$grafana_id" ]; then
  grafana_err=$(docker inspect --format '{{.State.Error}}' "$grafana_id" 2>/dev/null)
fi
case "$grafana_err" in
  *"already in use"* | *"already allocated"*)
    fail "Grafana port $port is free" "Another program is using port $port. Set GRAFANA_PORT to a free port (for example GRAFANA_PORT=3301) in .env, then run: docker compose up -d"
    footer
    ;;
esac

if get "$grafana/api/health" | grep -q '"database": *"ok"'; then
  pass "Grafana answers on port $port"
else
  fail "Grafana answers on port $port" "Grafana isn't answering at $grafana. Run: docker compose up -d grafana, wait 30 seconds, then check: docker compose logs grafana"
  footer
fi

loki_ok=0
if get "$grafana/api/datasources/proxy/uid/loki/ready" | grep -q ready; then
  pass "Loki is ready"
  loki_ok=1
else
  fail "Loki is ready" "Loki isn't ready. Run: docker compose up -d loki, wait 30 seconds, then check: docker compose logs loki"
fi

prom_ok=0
if get "$grafana/api/datasources/proxy/uid/prometheus/-/ready" >/dev/null; then
  pass "Prometheus is ready"
  prom_ok=1
else
  fail "Prometheus is ready" "Prometheus isn't ready. Run: docker compose up -d prometheus, then check: docker compose logs prometheus"
fi

# shellcheck disable=SC2317  # called through retry
has_lines() {
  query loki /loki/api/v1/query 'sum by (service) (count_over_time({env="lab"}[1m]))' | grep -q "\"service\":\"$1\""
}
for service in orders inventory loadgen; do
  if [ "$loki_ok" -eq 0 ]; then
    echo "SKIP  Loki has lines from $service (needs Loki)"
  elif retry has_lines "$service"; then
    pass "Loki has lines from $service"
  else
    fail "Loki has lines from $service" "There are no lines from $service in the last minute. Run: docker compose up -d $service alloy, then check: docker compose logs $service"
  fi
done

# shellcheck disable=SC2317  # called through retry
scraped() {
  query prometheus /api/v1/query "up{instance=\"$1:8080\"} == 1" | grep -q "\"instance\":\"$1:8080\""
}
for service in orders inventory; do
  if [ "$prom_ok" -eq 0 ]; then
    echo "SKIP  Prometheus scrapes $service (needs Prometheus)"
  elif retry scraped "$service"; then
    pass "Prometheus scrapes $service"
  else
    fail "Prometheus scrapes $service" "Prometheus can't scrape $service. Run: docker compose up -d $service, then check: docker compose logs $service"
  fi
done

footer
