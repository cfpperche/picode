#!/usr/bin/env bash
# scripts/go-test.sh [packages...] — `go test` with the heavy package sharded
# across processes (ADR-0105).
#
# internal/server is 365 serial tests, 75–90 s on their own; they swap
# package-level probes, so t.Parallel would race them (ADR-0086). Separate
# processes do not share package globals, so the same tests split four ways
# finish in ~22 s with nothing changed in the tests. Every other package runs
# in one ordinary `go test` alongside.
#
# PICODE_TERM_ID is unset for the run: internal/install reads it, which makes
# it a test-cache input — with it set, every PiCode terminal had its own cold
# cache for the same tree.
#
# -trimpath keeps the build id independent of the tree's absolute path. Without it the same package in two worktrees is two different test
# binaries and the second run misses the shared cache; measured at 10.4 s per
# package, repeated for every package a session touched.
set -uo pipefail
cd "$(dirname "$0")/.."

SHARDS=${GO_TEST_SHARDS:-4}
HEAVY=${GO_TEST_HEAVY:-github.com/cfpperche/picode/internal/server}
[ $# -gt 0 ] || set -- ./...

pkgs=$(go list "$@") || exit 1
rest=$(printf '%s\n' $pkgs | grep -vx "$HEAVY" || true)
heavy=$(printf '%s\n' $pkgs | grep -x "$HEAVY" || true)

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
status=0

if [ -n "$rest" ]; then
  ( env -u PICODE_TERM_ID go test -trimpath $rest > "$tmp/rest.log" 2>&1; echo $? > "$tmp/rest.rc" ) &
fi

if [ -n "$heavy" ]; then
  names=$(env -u PICODE_TERM_ID go test -trimpath -list '.*' "$heavy" 2>/dev/null | grep -vE '^(ok|\?|FAIL)' || true)
  if [ -z "$names" ] || [ "$SHARDS" -le 1 ]; then
    SHARDS=1
    ( env -u PICODE_TERM_ID go test -trimpath "$heavy" > "$tmp/shard0.log" 2>&1; echo $? > "$tmp/shard0.rc" ) &
  else
    for i in $(seq 0 $((SHARDS - 1))); do
      rx="^($(printf '%s\n' $names | awk -v i="$i" -v n="$SHARDS" 'NR % n == i' | paste -sd'|'))$"
      ( env -u PICODE_TERM_ID go test -trimpath -run "$rx" "$heavy" > "$tmp/shard$i.log" 2>&1; echo $? > "$tmp/shard$i.rc" ) &
    done
  fi
fi
wait

report() { # <label> <log> <rc-file>
  if [ "$(cat "$3")" != "0" ]; then
    status=1
    echo "go-test: FAIL in $1"
    cat "$2"
  fi
}
if [ -n "$rest" ]; then
  report "packages" "$tmp/rest.log" "$tmp/rest.rc"
  if [ "$(cat "$tmp/rest.rc")" = "0" ]; then
    total=$(grep -cE '^ok' "$tmp/rest.log" || true)
    cached=$(grep -cE '^ok.*\(cached\)' "$tmp/rest.log" || true)
    echo "go-test: $total package(s) ok ($cached cached)"
  fi
fi
if [ -n "$heavy" ]; then
  longest=0
  for i in $(seq 0 $((SHARDS - 1))); do
    report "$HEAVY shard $i" "$tmp/shard$i.log" "$tmp/shard$i.rc"
    s=$(grep -oE '[0-9.]+s$' "$tmp/shard$i.log" | tail -1 | tr -d s)
    longest=$(awk -v a="$longest" -v b="${s:-0}" 'BEGIN { print (a > b) ? a : b }')
  done
  [ "$status" -eq 0 ] && echo "go-test: ${HEAVY#github.com/cfpperche/picode/} ok in $SHARDS shard(s), longest ${longest}s"
fi
exit $status
