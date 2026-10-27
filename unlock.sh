#!/usr/bin/env bash
# Unlock the worksheet's next scenario.
#
#   ./unlock.sh N <answer>   N is the scenario you solved; prints the next one
set -eu

usage() {
  echo "usage: $0 N <answer>   (N is the scenario you solved)" >&2
  exit 2
}

[ "$#" -ge 2 ] || usage

n=$1
shift
answer="$*"

cd "$(dirname "$0")"
exec docker compose exec -T orders /labctl unlock "$n" "$answer"
