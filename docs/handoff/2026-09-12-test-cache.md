# 2026-09-12 — feat/test-cache: two debts paid with a flag and a log

Answering "do the debts have cheap solutions?" by paying the two cheapest ones
instead of describing them.

**Cold Go test cache in worktrees** (ADR-0105's line, still listed as debt):
measured `go test ./internal/store/` → `(cached)` in the same tree, 10.4 s in
another worktree with identical content. Cause: the build id carries the
absolute path. Fix: `-trimpath` on the four `go test` invocations in
`scripts/go-test.sh`; after it, the second tree reports `(cached)`. Full suite
green with the flag (`GOFLAGS=-trimpath ./scripts/go-test.sh ./...`, 58
packages). Nothing else changed: no cache daemon, no per-worktree cache
directory, no tooling.

**The unreproduced `make ci` failure**: the run's output was gone with the
terminal. `scripts/ci.sh` is now the body of `make ci` — same parallel targets,
output tee'd to `var/ci-last.log` (git-ignored), and a failure names the file.
Instrumentation, not retries; the cause is still unknown and the debt says so.

Also removed from `docs/handoff/open/process.md`: the cold-cache line (fixed).

Verified: `make test` green with `-trimpath`; the wrapper's non-zero path
proven with `(exit 7) | tee` → `PIPESTATUS[0]=7`; `make close`.

visual-review: n/a (no UI change).

Merge: fast-forward ready.
