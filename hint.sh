#!/usr/bin/env bash
# Print a hint for a worksheet scenario.
#
#   ./hint.sh N [LEVEL]   hint LEVEL (1 to 3, default 1) for scenario N
set -eu

usage() {
  echo "usage: $0 N [LEVEL]   (N is the scenario; LEVEL is 1 to 3)" >&2
  exit 2
}

case "$#" in
  1 | 2) ;;
  *) usage ;;
esac

cd "$(dirname "$0")"
exec docker compose exec -T orders /labctl hint "$@"
