#!/usr/bin/env bash
# Switch the lab between modes.
#
#   ./incident.sh A|B|C   start a mode (it ends by itself after a few minutes)
#   ./incident.sh clear   end it now
#   ./incident.sh status  show what is running
set -eu

usage() {
  echo "usage: $0 A|B|C|clear|status" >&2
  exit 2
}

[ "$#" -eq 1 ] || usage

case "$1" in
  A | a | B | b | C | c) arg=$(printf '%s' "$1" | tr '[:lower:]' '[:upper:]') ;;
  clear | status) arg=$1 ;;
  *) usage ;;
esac

cd "$(dirname "$0")"
exec docker compose exec -T orders /labctl "$arg"
