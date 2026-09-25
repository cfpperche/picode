# tmux app (ADR-0133, amended 2026-09-14)

> Part of [PiCode's architecture](../architecture.md) (ADR-0105: one file per subsystem). Edit here; the index only links.

The **tmux app** is a first-party **primitives app** (`internal/apps/tmux.go`)
on the Docker mold: one manifest, `View`/`Action` over the frozen vocabulary,
rendered by every shell's `AppSurface` — desktop and phone. It shows **every
session on the tmux server the daemon talks to** — PiCode's own sessions, the
ones no store row claims, and the user's own tmux sessions — and it can remove
one, one confirmed session at a time. Plan:
[`docs/plans/tmux-app.md`](../plans/tmux-app.md); decision: ADR-0133 (surface
amended 2026-09-14: primitives, not a native component — the native body did
not apply the mount's `hidden` and drew over other tabs, and its chrome was a
second UI standard).

## The read

`View(ctx, h, path)` over three paths:

| Path | Screen |
|---|---|
| `""` | the inventory: three collapsible list groups — **PiCode** (live work, each row opens its detail), **Not in PiCode's records** (leftovers; each row opens a detail that carries the removal), **Not PiCode's** (the user's own tmux, read-only, no path). Row: title (owner name when known, else the session name), command, relative start time (`At`, host-formatted), kind, attached clients, panes, live/exited badge. |
| `item/<session>` | one session's screen: identity (`name`, `$id`, started, folder, command, windows/panes/clients), the owner sentence its scope earns, and — for a leftover of *this* instance only — **Remove this session**. A claimed session links to its own surface instead: `#/term/<id>` or `#/agent/<id>` by markdown, no `Goto` growth; another instance's session gets the sentence naming it and no door (ADR-0141). |
| `"server"` | the server's facts (version, running, socket, sessions · attached, clients, keyboard mode with the Preferences pointer) and the absence list: terminals this daemon has rows for whose session is **not** on the server, `lost at restart` from the flight recorder (ADR-0085). |

The read costs **one tmux subprocess** for the whole inventory
(`list-panes -a -F`, `internal/tmux/server.go`) plus the server facts — no call
per session, no call per pane, and no timer: `View` is fetched by the shell on
open and refetched on the change feed's terminal events, exactly like the
Docker app. No tmux binary, or no server running, is the view's `Empty` line —
an honest blankslate, not a 500.

**The screen** is the host's page frame since 2026-09-14
([parity plan](../plans/app-surface-parity.md)): Sessions/Server as an
underline nav inside the card, the filter in the card toolbar, a session's
detail a stacked path with a back link above it. `Tabs[0].Badge` is the plain
session count in every shell, and every tab badge is empty at zero
(`countBadge`: never a "0" pill) — the long "N · M unclaimed" sentence was the one
badge in the product that ignored `primitives.Tab`'s own "short count"
contract, and the unclaimed fact is already the leftovers group's title and
each row's `no record` badge.

A session with two panes stays ONE row: the active pane supplies the details
and `panes` counts them, so a split pane cannot duplicate a row or inflate a
count.

## Attribution: the name is the hint, the marker is the receipt

Ownership is resolved in two steps, and the split is the whole safety model:

1. **The name**, in the list. `picode-sh-<id>` means a terminal, `picode-<id>`
   means an agent, anything else is not PiCode's namespace. The id is looked up
   in `Host.Store` (`GetTerminal`, `GetAgent`); a match is `PiCode`, no match is
   `Not in PiCode's records`, a foreign name is `Not PiCode's`. The step is
   lossy by construction — session names are sanitized to `[a-z0-9-]` — which
   is why a miss is reported as an uncertainty and never as "nobody's".
2. **The markers**, before any action, from the session's own environment:
   `PICODE_TERM_ID` / `PICODE_AGENT_ID`, injected at creation
   (`internal/server/terminals.go`, `internal/store/agents.go`, the CLI launch
   path). A marker that resolves to a terminal or an agent this daemon still
   holds refuses the removal and names the surface that owns the session —
   including in the detail view, where the removal button is replaced by that
   sentence.

`internal/apps/tmux_test.go:TestTmuxMarkerNamesMatchTheirWriters` pins the
marker strings (against the store's constant where one exists, literally
otherwise), because a rename in one of the writers would silently turn every
session into an "unclaimed" one — or, for `PICODE_INSTANCE`/`PICODE_TERM_URL`,
silently re-open the reap to another instance's work.

**Whose session is it: the instance stamp (ADR-0141).** A PiCode-shaped
session missing from the store may be a leftover, a harness's, or **another
PiCode instance's on the same machine** — measured 2026-09-14: a script over
the socket read seven live sessions as unclaimed, and all seven belonged to a
second instance it had failed to reach. ADR-0139 made that ambiguity visible
in every instance (each one now reads the default server's fleet beside its
own), so a scratch could see production's twelve terminals, one click from a
removal. The session itself now settles it: the Manager stamps every session
it creates with `PICODE_INSTANCE` = its data directory (`filepath.Dir` of its
socket, one funnel — `NewSessionEnvSize`, after the caller's env, so no caller
can claim another identity), the receipt reads it beside `PICODE_TERM_URL`,
and a session stamped by a different directory — or, before the stamp existed,
carrying a different port — is *another instance's work*: named on screen, no
removal door, and `reap` refuses it even if a request arrives by hand. A
session with neither stamp nor URL (an agent created before this version)
keeps the old sentence and the human's judgement; the list still carries no
per-row verdict, because ADR-0133 rule 1 keeps it one tmux call.

## The one action

`Action(ctx, h, {action: "reap", args: {name, sessionId, created, panePid}})`.
The args are the **receipt**: they were rendered into the view the human
confirmed against, and the action re-reads tmux before killing. The decision
table, row by row — every row is a test in
`internal/apps/tmux_test.go:TestTmuxReapDecisionTable`:

| Name | Marker's owner in the store | Receipt | Result |
|---|---|---|---|
| not `picode-*` | — | — | refused — the user's own session is never this app's business |
| `picode-*` | yes | any | refused — names the terminal/agent and its door |
| `picode-*` | no | missing, or `sessionId`/`created`/`panePid` moved | refused — "changed since this page was drawn" |
| `picode-*` | no | matches, stamped by another instance (or another instance's port) | refused — "belongs to another PiCode instance", named |
| `picode-*` | no | matches, stamped by this instance | killed once by exact name, one `tmux.session.reaped` event |
| session gone | — | — | refused — "no longer running" |
| unknown action | — | — | refused |

No prefix sweep, no bulk action, no automatic removal: the 2026-09-06 debt
(*"never kill by prefix"*) is what this table encodes. The removal is
irreversible, so the answer reports the audit honestly — the toast says so when
the record could not be written (the session is gone either way). The
`tmux.changed` ephemeral of the first delivery is gone with the custom
renderer: the action returns the refreshed view it replaced, and no other
surface consumed it.

## Runtime loss detection (2026-09-15)

`ServerInfo.Running` is the liveness probe, and a daemon-side watch
(`internal/server/tmux_watch.go`, `StartTmuxServerWatch`, 15 s) reads it once
per tick. The
boot diff above only runs at boot, so on 2026-09-15 a server
death under a live daemon — the third incident in ten days — was invisible
until hours later; the watcher closes that window while it is happening:

| Previous tick | This tick | Condition | Action |
|---|---|---|---|
| running | absent | sessions > 0 | `terminal.server_lost` (count, socket, last-seen) + journal line |
| running | absent | sessions == 0 | nothing — normal `exit-empty` |
| absent | running | — | `terminal.server_back` + journal line |
| running | running | drop ≥ 5 | journal line only (mass closes are also a user action) |

The session count is the last known value while the server answered: a
listing that fails carries the previous number forward, so one failed read
cannot fake a loss. Events go through `Store.AppendEvent` (ADR-0048), so the
change feed carries them; a server that restarts between two ticks is caught
by the drop line rather than the transitions.

## Host growth (ADR-0109's sanctioned direction)

`apps.Host` gains `Tmux TmuxServer` (an interface over the manager's read +
kill methods, so tests script the server instead of spawning one) and
`LostSessions map[string]bool` (ADR-0085's boot diff). The app never imports
`internal/server`.

## Sockets (ADR-0139 follow-up, 2026-09-15)

**A session's folder is known from its first instant (2026-09-15).**
`#{pane_current_path}` for a pane tmux has not polled yet is the *server's*
directory — the cwd of the client that started the tmux server, i.e. this
daemon — so a read taken in the same instant as `new-session` answered with the
wrong folder, and `#{pane_pid}` is already set while it does (measured: 33 of 40
creations in a loop; the first read of a relative path through
`/api/terminals/{id}/text` therefore 404'd, once, under a sharded CI run). The
Manager therefore remembers the folder it created (or respawned) each session in
— `bornCwd`, a five-second grace — and answers with that until tmux reports a
path that is not the server's, at which point the record is dropped and the live
path rules (a shell that `cd`s right away shows its new folder immediately). A
daemon with no record for the session reads tmux, exactly as before.

**A socket tmux cannot create is an error, not silence (2026-09-15).** With
`-S` inside a directory that did not exist yet, `new-session` printed
`error creating <path> (No such file or directory)` and **exited 0**, so the
caller reported a terminal that had never started. Two halves fix it:
`NewWithSocket` makes the socket's directory (tmux creates the socket, not its
parent — its own default path handles that for the shared server only), and
`runStdin` promotes tmux's `error creating …`/`error connecting to …` text to
the error it means even when the exit status is 0. `serverInfo` reads a live
socket only when the probe comes back clean, so a dead socket no longer reads
as running-with-a-garbage-path either.

A third tab answers the operator question the dedicated socket created:
which tmux servers exist on this machine, what lives on each, and where new
sessions land. `Manager.MachineSockets` lists every socket file in the user's
tmux directory (the default socket and every `-L` name) plus this instance's
own `-S` path; a socket somewhere else is undiscoverable by design (there is
no registry), and the block says so. Liveness is a plain unix-socket dial —
never a tmux client, which would start a transient server on a dead socket
just to report it dead. Counts come from one `list-sessions` on the live
ones; the `picode-` prefix is the cheap name hint, the same one the
inventory uses.

Rows: name (`· this instance` for ours), badge running / no server, the
subtitle `N session(s) · M PiCode`, and the socket path in the meta strip.
The tab badge is the number of running servers (none at zero). The group
header's own item count is the socket count; the block carries no second
"N socket(s)" meta line.

## What the app does not do

It does not list terminals or agents as a fleet (that is the sidebar's job —
ADR-0109's doors), it does not offer tmux **settings** (Preferences → Terminal
owns the live catalog, ADR-0025), it does not open a second attach for a
terminal that has one (one xterm per terminal), it never creates a session,
and it never acts on a session outside PiCode's namespace.

## Version note

PiCode requires tmux ≥ 3.5 and the read uses `session_id`, `pane_id`,
`pane_dead_status`, `session_attached`, `pane_active` and `show-environment -t`
— all present in 3.2+, so nothing here raises the floor. `pane_dead_status` is
only meaningful with `remain-on-exit`, which PiCode does **not** set: a dead
pane therefore usually means a dead session (the "exited N" badge is for the
case where the user's own `tmux.conf` turned it on).