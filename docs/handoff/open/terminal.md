# Terminals

## Next

## Debts

- Terminal menus (2026-09-09): the phone's rows are hand-built, not from `termRowMenu.js` (they gained **Send to terminal…** / **Run command…** on 2026-09-13), so desktop rows must be mirrored by hand; **Continue in…** waits on that merge.
- Scrollbars: the web terminal draws none (a tmux client has no scrollback — `term-scrollbar.test.mjs`); a draggable bar means taking the tmux client off the alternate screen, a decision.

## Traps

- **`$TMUX` beats `TMUX_TMPDIR`** (measured 2026-09-15): a client started inside a tmux session talks to the server named in `$TMUX` regardless of `TMUX_TMPDIR` — "isolated" scratch work hits production. Use `tmux -L <name>` for scratch servers, scrub `TMUX`/`TMUX_PANE` from fixture children, and assert `#{socket_path}` before any destructive verb. This cost production twice on 2026-09-15 (the guard branch's own integration test) — the test now skips unless the socket is under its temp dir.
- A tmux server/pane mass death leaves **no journal trace**: the daemon only reconstructs what happened at its next boot (ADR-0085 boot diff). During the 2026-09-15 incidents the death window had to be reconstructed from `~/.pi` session transcripts and the feed — a deliberate "tmux server gone" log line would have answered it in seconds.
- The tmux guard has **no UI toggle** yet (API only: `POST /api/terminals/wiring/tmux-guard/{enable,disable}`, default on) — a row needs a home that is not the per-CLI pane.

- Tab strip backlog: keyboard close of `.mtab-close`; live needs-you arrow; `scrollbar-width: thin` vs webkit.
- CLI pane-death (ADR-0085): an owned OpenCode QA stop left two processes after its pane closed; ADR-0084 pins nothing for terminals stopped before it.
- `TestCLIAdapterPreviewMatchesExecution` failed once in a full worktree run with a live scratch server and passed alone (2026-09-13) — not diagnosed.
- CLI launch fixtures: `waitCLIFile` only waits for the path to exist — a second launch into the same file needs the file removed (or a wait on its own argv).

- tmux fixtures: a **fixed session name shared across concurrent test binaries** (the heavy package runs in four shards) was the cause of the 2026-09-13 flakes — `TestPaneRootSurvivesSIGHUP` and `TestPeerStopStubbornChildStaysPending` went red under concurrent closes. Process-unique fixture names, plus a ready-marker handshake so the pane's script has installed its traps before a stop signals it, are the fixes. If a tmux fixture reddens again, suspect a shared fixed name first.
- tmux: never kill by prefix (a prefix sweep killed 29 sessions in 2026-09-06) — exact names from a fixture's API only; a scratch whose daemon dies before `qa-scratch stop` strands its shells.
- Guard test discipline (present in `internal/server/tmux_guard_test.go`): no test may run `tmux kill-server` — not even against a fixture; cleanup is exact-name only, every exec carries a deadline, and isolation is proven with `#{socket_path}`.