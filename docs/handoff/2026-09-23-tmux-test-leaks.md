# 2026-09-23 — feat/tmux-test-leaks: tmux test-harness cleanup can end its own servers

Root cause identified: when tmuxtest.Main sets TMUX_TMPDIR to a private namespace, tmux.KillIsolatedServer's guard read the process's current env and refused every kill, leaving ~490 servers orphaned (320 from TestHandoffPiTerminalLandingNative, 159 from TestOmpSigninUnknownProviderStaysTerminal). The pane's launch script ignores SIGHUP (ADR-0085), so each session stayed alive after the test exited. With the owner's OK the ones older than 30 min were ended by exact PID (473 servers, 545 pane groups).

Shipped (commit 1ca148dd):
- internal/tmux keeps the user's socket dir as found at process start (always refused in the kill guard)
- honours TMUX_TMPDIR only when the calling process calls tmux.MarkIsolated first
- tmuxtest.Main now calls MarkIsolated before setting TMUX_TMPDIR
- Guard tests assert user's real dir remains refused; new test asserts only marked dirs are permitted

Verified: measured on TestHandoffPiTerminalLandingNative and TestOmpSigninUnknownProviderStaysTerminal. Before: one leaked server per test run; after: zero, no orphaned pane processes. `make close` PASS (fmt, vet, hooks, go). Blind spot: measured on Linux/WSL only.
visual-review: n/a (test-harness only, no UI change).
Not done / debts: other sessions' worktrees keep leaking until they merge main.
Merge: fast-forward ready (`git merge --ff-only feat/tmux-test-leaks && make ci`).
