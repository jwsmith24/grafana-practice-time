#!/usr/bin/env bats
# Tests for hint.sh with a stubbed docker that records its arguments.

bats_require_minimum_version 1.5.0

setup() {
  REPO="$(cd "$BATS_TEST_DIRNAME/../.." && pwd)"
  STUB="$BATS_TEST_TMPDIR/bin"
  mkdir -p "$STUB"
  export DOCKER_ARGS="$BATS_TEST_TMPDIR/docker-args"
  cat > "$STUB/docker" <<'STUBEOF'
#!/usr/bin/env bash
printf '%s\n' "$@" > "$DOCKER_ARGS"
echo "Scenario 4 isn't unlocked yet."
exit 1
STUBEOF
  chmod +x "$STUB/docker"
  export PATH="$STUB:$PATH"
}

@test "N alone is passed through" {
  run "$REPO/hint.sh" 3
  expected=$(printf '%s\n' compose exec -T orders /labctl hint 3)
  [ "$(cat "$DOCKER_ARGS")" = "$expected" ]
}

@test "N and the level are passed through" {
  run "$REPO/hint.sh" 3 2
  expected=$(printf '%s\n' compose exec -T orders /labctl hint 3 2)
  [ "$(cat "$DOCKER_ARGS")" = "$expected" ]
}

@test "labctl's output and exit code come back unchanged" {
  run "$REPO/hint.sh" 4
  [ "$status" -eq 1 ]
  [ "$output" = "Scenario 4 isn't unlocked yet." ]
}

@test "no arguments or too many print usage and exit 2" {
  for args in "" "1 2 3"; do
    # shellcheck disable=SC2086
    run --separate-stderr "$REPO/hint.sh" $args
    [ "$status" -eq 2 ]
    [[ "$stderr" == *usage* ]]
    [ -z "$output" ]
    [ ! -e "$DOCKER_ARGS" ]
  done
}
