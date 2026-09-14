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
  `TestPaneRootSurvivesSIGHUP` also failed in that run but passes alone (load).
  `feat/tmux-isolation` / `feat/tmux-app` are in this code now — whoever lands
  first owns this line.
- tmux: never kill by prefix (a prefix sweep killed 29 sessions in 2026-09-06) — exact names from a fixture's API only; a scratch whose daemon dies before `qa-scratch stop` strands its shells.
- **`TestCLITerminalResumeDecisionTable` is red too** (`cli_launch_test.go:358`,
  `resume argv=["--default"]`): deterministic on a clean `main` (`02d9f792`,
  single test, `-count=1`, no shards). Not the peer-stop line above and not
  recorded anywhere else — found by `feat/tmux-app` (2026-09-13). `make ci`
  cannot be green until both are resolved.
