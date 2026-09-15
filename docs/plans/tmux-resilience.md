# tmux resilience — the five open items (2026-09-15)

The tmux guard (ADR-0138) closed the realistic accident: a managed terminal
typing `kill-server` or a pattern kill is refused. This plan closes the five
gaps that remain. Work is phased; each phase is a branch that merges green
before the next starts (ADR-0105).

## Why these five

Three incidents in ten days (29 sessions on 2026-09-06; 140 sessions on
2026-09-13 — recorded in the stalled `feat/tmux-orphans` diff; production
twice on 2026-09-15) shared one property: **the failure was invisible while
it happened and expensive to reconstruct afterwards**. The guard removes the
most likely cause; these items remove the blindness and the remaining
blast radius.

## Phases

### Phase 1 — runtime loss detection (branch `feat/tmux-hardening`)

Today the daemon only reconstructs a lost fleet at its next boot (ADR-0085
boot diff). A server death that the daemon lives through (the 2026-09-15
case: the daemon stayed up, all sessions vanished) is silent in the journal
and leaves no feed event.

- A tmux server watch beside the existing watchers (`StartTmuxServerWatch`,
  same shape as `StartTuiWatch`): one cheap `ServerInfo` probe per tick;
  on a reachable→absent transition, record `terminal.server_lost` (store
  event, ADR-0048) with the last known session count and the socket path,
  and log one journal line with the same facts; on absent→reachable,
  `terminal.server_back`.
- The last known count comes from the last successful `ServerSessions`
  read — the watch keeps it, no extra tmux calls.
- Tests: transition table (alive→alive, alive→absent, absent→absent,
  absent→alive) against a Manager pointed at a temp socket; the event goes
  through the store so `TestEveryMutationAppendsAnEvent` gains its row.
- Acceptance: kill a scratch server's sessions while a scratch daemon runs
  and find the loss in the journal and the feed within one tick.

### Phase 2 — tests stop writing to the live server (branch `feat/tmux-test-isolation`)

`make ci` leaked six sessions onto the production server today
(`picode-sh-feed-fixture-*`, `picode-sh-pi-from-claude-code-race-fix-*`,
removed by exact name). What was built instead of the plan's name-sweep:

- `internal/tmuxtest` (`TestMain` per suite): private `TMUX_TMPDIR`,
  `TMUX`/`TMUX_PANE` scrubbed for the whole test binary, the private server
  killed twice with a 300 ms grace (a straggler watcher ticks after
  `m.Run`), and the environment deliberately **not** restored — restoring it
  re-armed the leak for stragglers on 2026-09-15. Wired into `internal/server`,
  `internal/tmux` (external test package: `tmuxtest` imports it) and
  `internal/term`; `TestSuiteIsTmuxIsolated` fails loudly if a TestMain is
  removed.
- Harvested from `feat/tmux-orphans`: `SocketDirEnv`, `DefaultSocketDir()`,
  `KillIsolatedServer` (refuses the user's directory) + `isolation_test.go`.
  Fixed the harvested flaw: its child env was `append(os.Environ(), …)`, so
  an inherited `$TMUX` still won — `IsolatedEnv(dir)` now scrubs first.
- No name-matching sweep (deviation from the plan): the private server is
  killed whole, so leftovers cannot outlive the run, and no code in this
  repository pattern-kills anything.
- Found while fixing the harvested test: a tmux client with no server starts
  one, and a command needing no session watches it exit — "server exited
  unexpectedly". `serverAbsent` now folds that into "no server" (the tmux app
  renders an empty server instead of an error after a crash).
- Acceptance met: `go test ./internal/server/` and `./internal/tmux/` leave
  zero new sessions on the live server (26 → 26, verified by diff).

### Phase 3 — guard UI toggle (landed: branch `feat/tmux-guard-ui`)

The planned home was the CLIs landing; the landing turned out to always
select the first CLI (never empty), so the toggle lives where terminal-wide
behaviour already lives: **Preferences ▸ Terminal**, a **Safety** section
(browser). Mobile has no terminal-settings page — the toggle is
desktop-only, the guard stays default-on there (debt in
`docs/handoff/open/terminal.md`). Visual review done: on/off/failed-refresh/
failed-toggle states read, overlay audit `ok`, state survives reload, plus
an end-to-end run of `kill-server` refused inside a seeded scratch
terminal.

### Phase 4 — dedicated socket for managed sessions (branch `feat/tmux-socket`)

The ADR-0138 alternative, now with a measured target: managed sessions move
to `tmux -L picode` (data-dir scoped name), so a raw `/usr/bin/tmux
kill-server` on the default socket can no longer reach PiCode's terminals,
and vice versa. This is a **boundary change** (process/attach flow) and gets
its own ADR before code:

- `internal/tmux` Manager gains a socket option; `cmd/picode` wires it from
  the data dir.
- Migration: existing default-socket sessions keep running; the daemon
  adopts both during the transition (read model already lists the server it
  talks to; it needs to list both). New sessions go to the dedicated socket.
- The guard wrapper learns the socket from session env
  (`PICODE_TMUX_SOCKET`) so its ownership probes follow the right server
  instead of the default one.
- The user-facing attach flow (`tmux attach -t picode-…` from their own
  shell) breaks deliberately: docs-site documents the socket flag.
- Acceptance: kill-server on the default socket leaves PiCode terminals
  alive; closing a PiCode terminal leaves the user's sessions alive.

## Decisions and risks

| Item | Boundary | Risk | Mitigation |
|---|---|---|---|
| 1 watch | store event (ADR-0048) | event spam on a flapping server | one event per transition, not per tick |
| 2 tests | none | an isolation helper that still leaks via `$TMUX` | scrub + `#{socket_path}` assertion (the 2026-09-15 lesson) |
| 3 UI | none | none material | visual review mandatory |
| 4 socket | process/attach — ADR first | migration of live sessions; user attach flow | both-socket adoption window; docs-site; opt-in deploy |

## Out of scope (named)

- Restricting the daemon itself (it is the owner of its sessions).
- Agents running outside PiCode terminals (no wrapper on their PATH).
- AGENTS.md rule for `$TMUX` vs `TMUX_TMPDIR` — landed in phase 1 (one
  line, it is a contract edit, not a feature).
