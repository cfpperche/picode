# 2026-09-09 — tmux orphan leak (`feat/tmux-cleanup-leak`)

**Shipped.** `killTmuxOnCleanup` in `internal/server/server_test.go` kills a
test's tmux session with `context.Background()`; `filetree_test.go` and
`term_state_test.go` use it instead of `t.Context()`, which the testing
package cancels *before* Cleanup runs, so their kill never spawned.
`TestKillTmuxOnCleanupOutlivesTheTestContext` pins both halves. `qa-scratch.sh
stop` now deletes the instance's terminals through its own API (exact ids)
before killing the daemon, and says so when the daemon is already gone.

**Measured.** The dev machine held 260 tmux sessions; 8 were claimed by the
six live instances. 202 came from `TestTerminalBrowse`, one per suite run
since the cleanup was written; ~48 from QA scratches whose worktrees were
later removed. 252 orphans were killed by exact name after asking every live
daemon what it owns (`/api/terminals` answers 200 on loopback). A full
`make ci-scoped` afterwards left **zero** new sessions.

**Verified.** `ci-scoped: PASS (full)`, 57 Go packages. No UI change, so no
visual review and no changelog fragment.

**Debts.** `picode-sh-shell-6d4d44` (idle bash in `~/picode`, unclaimed) was
left alone deliberately. A scratch whose daemon died before `stop` still
strands its shells; the script names the recovery instead of sweeping.

**Merge.** `git merge --ff-only feat/tmux-cleanup-leak && make ci`.
