# Terminals

## Next

## Debts

- Agent CLIs Terminals ⋯ on the phone is still hand-built (Launch settings / Restart / Stop / Remove), not `termRowMenu`. Work rows use the shared menu (`surface: "phone"`) since 2026-09-19.
- Scrollbars: the web terminal draws none (a tmux client has no scrollback — `term-scrollbar.test.mjs`); a draggable bar means taking the tmux client off the alternate screen, a decision.

## Traps

- **`$TMUX` beats `TMUX_TMPDIR`** (measured 2026-09-15): a client inside a session talks to the server in `$TMUX` regardless of `TMUX_TMPDIR`. Use `tmux -L` for scratch; scrub `TMUX`/`TMUX_PANE`, assert `#{socket_path}` in fixtures (ADR-0138).
- The tmux guard's toggle is **desktop-only** (Preferences ▸ Terminal ▸ Safety): the mobile shell has no terminal-settings page, so the guard stays default-on there. The server-loss watch (phase 1 of the plan) closed the other gap — a mass death now lands in the journal and the feed while it happens.

- Tab strip backlog: keyboard close of `.mtab-close`; live needs-you arrow; `scrollbar-width: thin` vs webkit.
- CLI pane-death (ADR-0085): an owned OpenCode QA stop left two processes after its pane closed; ADR-0084 pins nothing for terminals stopped before it.
- `TestCLIAdapterPreviewMatchesExecution` failed once in a full worktree run with a live scratch server and passed alone (2026-09-13) — not diagnosed.
- CLI launch fixtures: `waitCLIFile` only waits for the path to exist — a second launch into the same file needs the file removed (or a wait on its own argv).

- tmux fixtures: a **fixed session name shared across concurrent test binaries** (the heavy package runs in four shards) was the cause of the 2026-09-13 flakes — `TestPaneRootSurvivesSIGHUP` and `TestPeerStopStubbornChildStaysPending` went red under concurrent closes. Process-unique fixture names, plus a ready-marker handshake so the pane's script has installed its traps before a stop signals it, are the fixes. If a tmux fixture reddens again, suspect a shared fixed name first.
- tmux: never kill by prefix (2026-09-06 sweep: 29 sessions); guard tests: no `kill-server`, exact-name cleanup, deadlines on every exec.
- Suites that start tmux servers go through `internal/tmuxtest` (`TestMain`): private `TMUX_TMPDIR`, `TMUX`/`TMUX_PANE` scrubbed, the private server killed twice with a grace between (a straggler watcher finished a tick after `m.Run` and leaked a session on 2026-09-15 — the env is deliberately not restored). No name-matching orphan sweep exists; leftovers die with the private server.