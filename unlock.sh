#!/usr/bin/env bash
# Unlock the worksheet's next question.
#
#   ./unlock.sh N <answer>   N is the question you answered; prints the next one
set -eu

usage() {
  echo "usage: $0 N <answer>   (N is the question you answered)" >&2
  exit 2
}

[ "$#" -ge 2 ] || usage

n=$1
shift
answer="$*"

cd "$(dirname "$0")"
exec docker compose exec -T orders /labctl unlock "$n" "$answer"
