# Terminals

## Next

- tmux 3.7c is in PATH (`/usr/local/bin`) while the running server is still 3.6: at the first server restart (reboot, or the last session ending) re-measure `extended-keys-format`, `allow-passthrough`, `display-popup` and the floating-pane gestures on 3.7c, then refresh ADR-0025's 3.6-era numbers (ADR-0164).

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
- The tmux in PATH is now the source build in `/usr/local/bin` (3.7c) shadowing apt's `/usr/bin/tmux` (3.6a), so PATH order decides which binary a terminal uses and an apt upgrade would be invisible; build flags change what the server advertises to pane apps (without `--enable-sixel` the DA1 answer loses attribute 4 — `?1;2c` instead of `?1;2;4c`, measured 2026-09-20).
- `~/.local/bin/pkg-config` is a **mock** ("plausible values" for cargo builds) that precedes the real `/usr/bin/pkg-config` in PATH: any `configure` that trusts it gets invented answers. It bit the tmux build ("libevent: yes" from a lie); `PKG_CONFIG=/usr/bin/pkg-config` is the fix.
- Suites that start tmux servers go through `internal/tmuxtest` (`TestMain`): private `TMUX_TMPDIR`, `TMUX`/`TMUX_PANE` scrubbed, the private server killed twice with a grace between (a straggler watcher finished a tick after `m.Run` and leaked a session on 2026-09-15 — the env is deliberately not restored). No name-matching orphan sweep exists; leftovers die with the private server.
- The tmuxtest teardown is not enough on its own (measured 2026-09-20): 14 `/tmp/picode-tmuxtest-*` dirs, 7 with live servers, sessions back to 2026-09-18, and one pane spinning `while :; do :; done` since 2026-09-19 18:18 — a namespace outlives its suite when the test binary is killed (a shard under a 10-minute timeout, a canceled `make ci`), and the twice-kill reaps the server it started, not the one a dead binary started. The missing piece is a stale sweep (age + no test process) before the next run. Cleaned later on 2026-09-20: the 7 dirs older than 20 minutes went (5 sessions killed by exact name, including the pane spinning since 19/09 18:18) and the 5 belonging to a test run in flight were left alone.
- The 3.7c build in `/usr/local` is not systemd-enabled (Ubuntu's 3.6 is: panes land in their own scopes). Measured no impact on PiCode, which never creates panes and keeps the server in the unit's cgroup (ADR-0002); parity rebuild is `--enable-systemd` plus `libsystemd-dev`.