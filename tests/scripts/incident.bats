#!/usr/bin/env bats
# Tests for incident.sh with a stubbed docker that records its arguments
# and answers like labctl.

bats_require_minimum_version 1.5.0

setup() {
  REPO="$(cd "$BATS_TEST_DIRNAME/../.." && pwd)"
  STUB="$BATS_TEST_TMPDIR/bin"
  mkdir -p "$STUB"
  export DOCKER_ARGS="$BATS_TEST_TMPDIR/docker-args"
  cat > "$STUB/docker" <<'STUBEOF'
#!/usr/bin/env bash
printf '%s\n' "$@" > "$DOCKER_ARGS"
last="${*: -1}"
case "$last" in
  clear) echo "cleared" ;;
  status) echo "none" ;;
  *) echo "active: $last (3m0s left)" ;;
esac
STUBEOF
  chmod +x "$STUB/docker"
  export PATH="$STUB:$PATH"
}

# T4.1-4
@test "invalid letter exits non-zero with usage on stderr" {
  run --separate-stderr "$REPO/incident.sh" X
  [ "$status" -ne 0 ]
  [[ "$stderr" == *usage* ]]
  [ -z "$output" ]
  [ ! -e "$DOCKER_ARGS" ]
}

@test "no argument exits non-zero with usage" {
  run --separate-stderr "$REPO/incident.sh"
  [ "$status" -ne 0 ]
  [[ "$stderr" == *usage* ]]
}

# T4.1-5
@test "output for A, B and C differs only by the letter and reveals nothing" {
  declare -a normalized
  for letter in A B C; do
    run "$REPO/incident.sh" "$letter"
    [ "$status" -eq 0 ]
    for word in SKU latency slow token auth deploy 1.4.3 incident; do
      if [[ "$(printf '%s' "$output" | tr '[:upper:]' '[:lower:]')" == *"$(printf '%s' "$word" | tr '[:upper:]' '[:lower:]')"* ]]; then
        echo "output for $letter contains '$word': $output"
        return 1
      fi
    done
    normalized+=("${output//$letter/?}")

    # docker received exactly an exec of labctl with only the letter.
    expected=$(printf '%s\n' compose exec -T orders /labctl "$letter")
    [ "$(cat "$DOCKER_ARGS")" = "$expected" ]
  done
  [ "${normalized[0]}" = "${normalized[1]}" ]
  [ "${normalized[0]}" = "${normalized[2]}" ]
}

@test "lowercase letters are passed upper-cased" {
  run "$REPO/incident.sh" b
  [ "$status" -eq 0 ]
  [ "$(tail -n 1 "$DOCKER_ARGS")" = "B" ]
}

@test "clear and status are passed through" {
  for cmd in clear status; do
    run "$REPO/incident.sh" "$cmd"
    [ "$status" -eq 0 ]
    [ "$(tail -n 1 "$DOCKER_ARGS")" = "$cmd" ]
  done
}
