# Terminals

## Next

- Tab strip: keyboard close of `.mtab-close`; live needs-you arrow; `scrollbar-width: thin` vs webkit.

## Debts

- Terminal menus (2026-09-09): the phone's rows are hand-built, not from `termRowMenu.js` (they gained **Send to terminal…** / **Run command…** on 2026-09-13), so desktop rows must be mirrored by hand; **Continue in…** waits on that merge.
- Scrollbars: the web terminal draws none (a tmux client has no scrollback — `term-scrollbar.test.mjs`); a draggable bar means taking the tmux client off the alternate screen, a decision.
- CLI pane-death (ADR-0085): an owned OpenCode QA stop left two processes after its pane closed; ADR-0084 pins nothing for terminals stopped before it.
- **`make ci` is red on `main` (2026-09-13): `TestPeerStopStubbornChildStaysPending`**
  (`peer_stop_linux_test.go:73`) fails "stubborn child reported stopped: <nil>" —
  deterministically, in isolation and in the shards, and already at `f3df855e`
  (before the `snip` starters merge). The test came in at 14:29 (`d9bc2dd9`);
  `TestPaneRootSurvivesSIGHUP` also failed in that run — measured again
  2026-09-13 22:5x: **it does not pass alone under load.** `-count=5` fails
  5/5 (`.worktrees/thinking-format`) and 4/5 (root, `main`), while a single
  run at low load passes. The cause is a race in the test, not the manager:
  `tmux.NewSession` returns when tmux accepts `new-session`, with no wait for
  the pane's command to start, and `PanePID` reads `#{pane_pid}` immediately,
  so the SIGHUP can arrive before the script's `trap '' HUP` / `exec sleep`
  runs. A standalone probe with the same script and a private socket
  (`tmux -L`) reproduced it: HUP right after `new-session` → pane root still
  the starting process, **died** 3/3; HUP 300 ms later → root `sleep 30`,
  **alive** 3/3. Fix is on the test side: wait for the pane root to reach the
  expected process (poll `PanePID`/cmdline) before signalling. Two more of
  the package's tests ("TestCLIAdapterPreviewMatchesExecution",
  "TestCLITerminalResumeDecisionTable") failed only inside a full-package run
  with a live scratch server and passed alone — contention flakiness, not
  reproduced on their own.
  `feat/tmux-isolation` / `feat/tmux-app` are in this code now — whoever lands
  first owns this line.
- tmux: never kill by prefix (a prefix sweep killed 29 sessions in 2026-09-06) — exact names from a fixture's API only; a scratch whose daemon dies before `qa-scratch stop` strands its shells.
