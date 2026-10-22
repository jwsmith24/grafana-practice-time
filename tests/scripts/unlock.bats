#!/usr/bin/env bats
# Tests for unlock.sh with a stubbed docker that records its arguments.

bats_require_minimum_version 1.5.0

setup() {
  REPO="$(cd "$BATS_TEST_DIRNAME/../.." && pwd)"
  STUB="$BATS_TEST_TMPDIR/bin"
  mkdir -p "$STUB"
  export DOCKER_ARGS="$BATS_TEST_TMPDIR/docker-args"
  cat > "$STUB/docker" <<'STUBEOF'
#!/usr/bin/env bash
printf '%s\n' "$@" > "$DOCKER_ARGS"
echo "Not quite. Check your answer and try again."
exit 1
STUBEOF
  chmod +x "$STUB/docker"
  export PATH="$STUB:$PATH"
}

# T6.5-2
@test "N and the answer are passed through unchanged" {
  run "$REPO/unlock.sh" 3 Some-Value_1
  expected=$(printf '%s\n' compose exec -T orders /labctl unlock 3 Some-Value_1)
  [ "$(cat "$DOCKER_ARGS")" = "$expected" ]
}

@test "a multi-word answer reaches labctl as one argument joined by single spaces" {
  run "$REPO/unlock.sh" 2 two   words  here
  expected=$(printf '%s\n' compose exec -T orders /labctl unlock 2 "two words here")
  [ "$(cat "$DOCKER_ARGS")" = "$expected" ]
}

@test "labctl's output and exit code come back unchanged" {
  run "$REPO/unlock.sh" 1 nope
  [ "$status" -eq 1 ]
  [ "$output" = "Not quite. Check your answer and try again." ]
}

@test "a missing answer or no arguments print usage and exit 2" {
  for args in "" "1"; do
    # shellcheck disable=SC2086
    run --separate-stderr "$REPO/unlock.sh" $args
    [ "$status" -eq 2 ]
    [[ "$stderr" == *usage* ]]
    [ -z "$output" ]
    [ ! -e "$DOCKER_ARGS" ]
  done
}
