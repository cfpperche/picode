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
# Since 2026-09-21 the WHOLE ambient PICODE_* family is unset, not just those
# two: the agent runtime exports several of them into every PiCode terminal,
# and any production code that reads one process-wide turns a gate run from
# inside PiCode into a run against the live instance. The credentials vault
# was exactly that (PICODE_DATA resolved before HOME; 30 test fixtures into
# the owner's real vault across 15 providers, found by the owner as "what are
# all these accounts?"). Tests that need a PICODE_* value set it themselves.
# The
# harnesses pin it too (internal/server/cleanup_test.go); this is the layer
# that covers packages nobody has thought about yet.
#
# -trimpath keeps the build id independent of the tree's absolute path. Without it the same package in two worktrees is two different test
# binaries and the second run misses the shared cache; measured at 10.4 s per
# package, repeated for every package a session touched.
set -uo pipefail
cd "$(dirname "$0")/.."

# The go tool links every test binary under GOTMPDIR, which defaults to
# TMPDIR — on the owner's machine a 16 GB tmpfs in memory that several
# sessions share. On 2026-09-25 it filled (session scratchpads, test temp
# dirs) and `make ci` on main failed three times at the link step with
# "no space left on device" while the disk had 772 GB free. Link outputs go
# to a cache directory on disk instead; a GOTMPDIR set by the caller wins.
# Test temp dirs (t.TempDir) still follow TMPDIR: tmux sockets live there,
# and their path length is measured (internal/tmuxtest).
if [ -z "${GOTMPDIR:-}" ]; then
  GOTMPDIR="${XDG_CACHE_HOME:-$HOME/.cache}/picode-gotmp"
  if mkdir -p "$GOTMPDIR" 2>/dev/null; then export GOTMPDIR; else unset GOTMPDIR; fi
fi

SHARDS=${GO_TEST_SHARDS:-4}
# Extra flags for every `go test` below. CI passes -race: the race detector
# makes internal/server take over ten minutes in one process, which is the
# default per-binary timeout, so `go test -race ./...` on a runner died with
# `panic: test timed out after 10m0s` every run. Sharded, each shard finishes
# well inside it (measured: 275-385 s per shard, 446 s wall clock).
FLAGS=${GO_TEST_FLAGS:-}
# More than one package is heavy now. internal/store joined the list when CI
# started passing -race: TestEveryMutationAppendsAnEvent alone runs 2m30s
# under the detector and the package went past `go test`'s 10-minute default
# in one process, exactly as internal/server had. Sharding one and not the
# other just moved which package killed the job.
HEAVY=${GO_TEST_HEAVY:-github.com/cfpperche/picode/internal/server github.com/cfpperche/picode/internal/store}
[ $# -gt 0 ] || set -- ./...

pkgs=$(go list "$@") || exit 1
# BSD paste answers `paste -sd'|'` with its usage line, which left this empty
# and made every downstream match a silent no-op — on macOS the heavy packages
# ran inside the job that had just excluded them (2026-09-24, `panic: test
# timed out after 25m0s` in `internal/server`). `tr` is POSIX; nothing here
# may depend on a GNU-only form.
heavy_re=$(printf '%s\n' $HEAVY | tr '\n' '|')
heavy_re=${heavy_re%|}
if [ -n "$HEAVY" ] && [ -z "$heavy_re" ]; then
  echo "go-test: GO_TEST_HEAVY is set but produced an empty pattern" >&2
  exit 1
fi
# CI runs the heavy packages in a job of their own (2026-09-24, owner
# approved): the job that covers everything else drops them from its list
# rather than sharding them, so each half of the suite keeps its own ceiling
# and its own verdict. `make ci` and local runs keep them here.
if [ "${GO_TEST_EXCLUDE_HEAVY:-}" = "1" ]; then
  pkgs=$(printf '%s\n' "$pkgs" | grep -vxE "$heavy_re" || true)
fi
rest=$(printf '%s\n' $pkgs | grep -vxE "$heavy_re" || true)
heavy=$(printf '%s\n' $pkgs | grep -xE "$heavy_re" || true)

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
status=0

if [ -n "$rest" ]; then
  ( env $(printenv | sed -n "s/^\(PICODE_[A-Z_0-9]*\)=.*/-u \1/p" | tr "\n" " ") go test $FLAGS -trimpath $rest > "$tmp/rest.log" 2>&1; echo $? > "$tmp/rest.rc" ) &
fi

h=0
for pkg in $heavy; do
  names=$(env $(printenv | sed -n "s/^\(PICODE_[A-Z_0-9]*\)=.*/-u \1/p" | tr "\n" " ") go test -trimpath -list '.*' "$pkg" 2>/dev/null | grep -vE '^(ok|\?|FAIL)' || true)
  echo "$pkg" > "$tmp/heavy$h.pkg"
  if [ -z "$names" ] || [ "$SHARDS" -le 1 ]; then
    echo 1 > "$tmp/heavy$h.n"
    ( env $(printenv | sed -n "s/^\(PICODE_[A-Z_0-9]*\)=.*/-u \1/p" | tr "\n" " ") go test $FLAGS -trimpath "$pkg" > "$tmp/heavy$h.shard0.log" 2>&1; echo $? > "$tmp/heavy$h.shard0.rc" ) &
  else
    echo "$SHARDS" > "$tmp/heavy$h.n"
    for i in $(seq 0 $((SHARDS - 1))); do
      rx="^($(printf '%s\n' $names | awk -v i="$i" -v n="$SHARDS" 'NR % n == i' | tr '\n' '|' | sed 's/|$//'))$"
      ( env $(printenv | sed -n "s/^\(PICODE_[A-Z_0-9]*\)=.*/-u \1/p" | tr "\n" " ") go test $FLAGS -trimpath -run "$rx" "$pkg" > "$tmp/heavy$h.shard$i.log" 2>&1; echo $? > "$tmp/heavy$h.shard$i.rc" ) &
    done
  fi
  h=$((h + 1))
done
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
j=0
while [ "$j" -lt "$h" ]; do
  pkg=$(cat "$tmp/heavy$j.pkg")
  n=$(cat "$tmp/heavy$j.n")
  longest=0
  for i in $(seq 0 $((n - 1))); do
    report "$pkg shard $i" "$tmp/heavy$j.shard$i.log" "$tmp/heavy$j.shard$i.rc"
    s=$(grep -oE '[0-9.]+s$' "$tmp/heavy$j.shard$i.log" | tail -1 | tr -d s)
    longest=$(awk -v a="$longest" -v b="${s:-0}" 'BEGIN { print (a > b) ? a : b }')
  done
  [ "$status" -eq 0 ] && echo "go-test: ${pkg#github.com/cfpperche/picode/} ok in $n shard(s), longest ${longest}s"
  j=$((j + 1))
done
exit $status
