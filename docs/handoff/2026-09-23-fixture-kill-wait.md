# 2026-09-23 — feat/fixture-kill-wait: tmux fixtures end the pane's process group

A known gate flake turned main red after a landing earlier today: `TempDir RemoveAll cleanup: directory not empty` in `TestCLITerminalCreationAnnouncesItsIdentityOnTheFeed` and `TestCLIAdapterPreviewMatchesExecution`.
Cause (measured): the launch root ignores SIGHUP by design (ADR-0085), so the fixtures' tmux `kill-session` left the pane's process tree alive, still writing under the test's data dir while TempDir removed it. Two such pairs were found alive ~17 h after their test.

Shipped (b6731792, test-only):
- `killFixture` (`internal/server/cli_terminal_feed_test.go`) and `killTmuxOnCleanup` (`internal/server/server_test.go`) read the pane pid before the kill.
- `reapPaneGroup` (`internal/server/fixture_reap_unix_test.go`; no-op stub in `fixture_reap_windows_test.go`) sends SIGTERM to the pane's process group, then SIGKILL after 5 s. It acts only when that pid still leads its own group.
- Debts paid: `docs/handoff/open/agent-clis-native.md` (gate flake) and `docs/handoff/open/terminal.md` (the undiagnosed 2026-09-13 failure).

Verified: three parallel loops of both tests. Main failed 2 of 24 runs (both `directory not empty`); the fix failed 0 of 48. `make close` PASS (fmt, vet, hooks, go, living-docs). Blind spot: this was measured on Linux/WSL only. The Windows stub reaps nothing, and, by construction (not measured), a pane child that starts its own process group escapes the reap.
visual-review: n/a (no UI change). No changelog fragment: the change is test-only and nothing user-visible changed.
Not done / debts: none new.
Merge: fast-forward ready (`git merge --ff-only feat/fixture-kill-wait && make ci`).
