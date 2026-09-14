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
| `item/<session>` | one session's screen: identity (`name`, `$id`, started, folder, command, windows/panes/clients), the owner sentence its scope earns, and — for a leftover only — **Remove this session**. A claimed session links to its own surface instead: `#/term/<id>` or `#/agent/<id>` by markdown, no `Goto` growth. |
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
session count in every shell — the long "N · M unclaimed" sentence was the one
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
marker strings against the store's constant, because a rename in one of the
three writers would silently turn every session into an "unclaimed" one.

**The uncertainty is in the product, not hidden.** A PiCode-shaped session
missing from the store may be a leftover, a harness's, or **another PiCode
instance's on the same machine** — measured 2026-09-14: a script over the
socket read seven live sessions as unclaimed, and all seven belonged to a
second instance it had failed to reach. No read in this daemon can settle
that; the row says so, and a human confirming one session at a time can.

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
| `picode-*` | no | matches | killed once by exact name, one `tmux.session.reaped` event |
| session gone | — | — | refused — "no longer running" |
| unknown action | — | — | refused |

No prefix sweep, no bulk action, no automatic removal: the 2026-09-06 debt
(*"never kill by prefix"*) is what this table encodes. The removal is
irreversible, so the answer reports the audit honestly — the toast says so when
the record could not be written (the session is gone either way). The
`tmux.changed` ephemeral of the first delivery is gone with the custom
renderer: the action returns the refreshed view it replaced, and no other
surface consumed it.

## Host growth (ADR-0109's sanctioned direction)

`apps.Host` gains `Tmux TmuxServer` (an interface over the manager's read +
kill methods, so tests script the server instead of spawning one) and
`LostSessions map[string]bool` (ADR-0085's boot diff). The app never imports
`internal/server`.

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