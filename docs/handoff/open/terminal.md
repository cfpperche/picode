# Terminals

## Next

- Tab strip: keyboard close of `.mtab-close`; live needs-you arrow; `scrollbar-width: thin` vs webkit.

## Debts

- Terminal menus (2026-09-09): the phone's rows are hand-built, not from `termRowMenu.js` (they gained **Send to terminal…** / **Run command…** on 2026-09-13), so desktop rows must be mirrored by hand; **Continue in…** waits on that merge.
- Scrollbars: the web terminal draws none (a tmux client has no scrollback — `term-scrollbar.test.mjs`); a draggable bar means taking the tmux client off the alternate screen, a decision.
- CLI pane-death (ADR-0085): an owned OpenCode QA stop left two processes after its pane closed; ADR-0084 pins nothing for terminals stopped before it.
- **`make ci` was red on `main` (2026-09-13): `TestPeerStopStubbornChildStaysPending`**
  (`peer_stop_linux_test.go:73`) fails "stubborn child reported stopped: <nil>" —
  deterministically, in isolation and in the shards, and already at `f3df855e`
  (before the `snip` starters merge). The test came in at 14:29 (`d9bc2dd9`).
  Owned and fixed by `feat/peer-stop-child-race`: same race as the SIGHUP test
  below (the pane root is the forked tmux client until it execs the shell and
  installs its traps), fixed there with a ready-marker handshake plus every
  tmux command without the ambient `TMUX` and `-t =name`, verified with 20
  consecutive runs. `feat/thinking-format` measures the same race but carries
  no duplicate of the fix.
- **`TestCLITerminalResumeDecisionTable`, the same family one layer up:** the
  resume assertion read back the *creation* launch because `waitCLIFile`
  returns as soon as the path exists — sharded runs failed 2/2 at load ~5
  (always shard 3) while a single process passed; the message was
  `resume argv=["--default"]`. **Fixed on `feat/thinking-format`** by removing
  the fixture file before the resume — the idiom the same test already uses
  for its plain start, so the assertion now reads the new launch (flipping the
  expectation to `--default` fails on `["--resume" "sess-1"]`, which proves
  it).
- `TestCLIAdapterPreviewMatchesExecution`: failed once in a full worktree run
  with a live scratch server, passed alone — not diagnosed.
- **`TestPaneRootSurvivesSIGHUP`** (same commit as the peer-stop test) failed
  in that run too — measured again 2026-09-13 22:5x: **it does not pass alone
  under load.** `-count=5` fails 5/5 (`.worktrees/thinking-format`) and 4/5
  (root, `main`), while a single run at low load passes. The cause is a race in
  the test, not the manager: `tmux.NewSession` returns when tmux accepts
  `new-session`, with no wait for the pane's command to start, and `PanePID`
  reads `#{pane_pid}` immediately, so the SIGHUP can arrive before the
  script's `trap '' HUP` / `exec sleep` runs. A standalone probe with the same
  script and a private socket (`tmux -L`) reproduced it: HUP right after
  `new-session` → pane root still the starting process, **died** 3/3; HUP
  300 ms later → root `sleep 30`, **alive** 3/3. **Fixed on
  `feat/thinking-format`** on the test side: the script touches a marker after
  its trap line and the test waits for it (5/5 pass under the same load), and
  the control still fails when the untrapped pane is given the trap.
  `feat/tmux-isolation` / `feat/tmux-app` are in this code now — whoever lands
  first owns this line.
- tmux: never kill by prefix (a prefix sweep killed 29 sessions in 2026-09-06) — exact names from a fixture's API only; a scratch whose daemon dies before `qa-scratch stop` strands its shells.
