# Terminals

## Next

## Debts

- Terminal menus (2026-09-09): the phone's rows are hand-built, not from `termRowMenu.js` (they gained **Send to terminal…** / **Run command…** on 2026-09-13), so desktop rows must be mirrored by hand; **Continue in…** waits on that merge.
- Scrollbars: the web terminal draws none (a tmux client has no scrollback — `term-scrollbar.test.mjs`); a draggable bar means taking the tmux client off the alternate screen, a decision.

## Traps

- **`$TMUX` beats `TMUX_TMPDIR`** (measured 2026-09-15): a client inside a session talks to the server in `$TMUX` regardless of `TMUX_TMPDIR`. Use `tmux -L` for scratch; scrub `TMUX`/`TMUX_PANE`, assert `#{socket_path}` in fixtures (ADR-0138).
- Two gaps the 2026-09-15 incidents exposed: a tmux server mass death leaves **no journal trace** (ADR-0085 reconstructs at next boot), and the guard toggle is **API-only** (default on) — needs a UI home.

- Tab strip backlog: keyboard close of `.mtab-close`; live needs-you arrow; `scrollbar-width: thin` vs webkit.
- CLI pane-death (ADR-0085): an owned OpenCode QA stop left two processes after its pane closed; ADR-0084 pins nothing for terminals stopped before it.
- `TestCLIAdapterPreviewMatchesExecution` failed once in a full worktree run with a live scratch server and passed alone (2026-09-13) — not diagnosed.
- CLI launch fixtures: `waitCLIFile` only waits for the path to exist — a second launch into the same file needs the file removed (or a wait on its own argv).

- tmux fixtures: a **fixed session name shared across concurrent test binaries** (the heavy package runs in four shards) was the cause of the 2026-09-13 flakes — `TestPaneRootSurvivesSIGHUP` and `TestPeerStopStubbornChildStaysPending` went red under concurrent closes. Process-unique fixture names, plus a ready-marker handshake so the pane's script has installed its traps before a stop signals it, are the fixes. If a tmux fixture reddens again, suspect a shared fixed name first.
- tmux: never kill by prefix (2026-09-06 sweep: 29 sessions); guard tests: no `kill-server`, exact-name cleanup, deadlines on every exec.