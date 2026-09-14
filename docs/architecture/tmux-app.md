# tmux app (ADR-0133)

> Part of [PiCode's architecture](../architecture.md) (ADR-0105: one file per subsystem). Edit here; the index only links.

The **tmux app** is a first-party app (`internal/apps/tmux.go`) whose body is
the desktop component `web/browser/src/components/tmux/TmuxSurface.jsx`,
registered by manifest id in `web/browser/src/lib/nativeApps.js` (ADR-0109),
and whose data is a small API family in `internal/server/tmux.go`. It shows
**every session on the tmux server the daemon talks to** — PiCode's own
sessions, sessions no store row claims, and the user's own tmux sessions — and
it can remove one, one confirmed row at a time. Plan:
[`docs/plans/tmux-app.md`](../plans/tmux-app.md); decision: ADR-0133.

## The read

`GET /api/tmux/server` answers one view:

| Field | Meaning |
|---|---|
| `server` | `installed`, `running`, `socketPath`, `clients`, `version`, `extendedKeysFormat`, counts (`sessions`, `picodeSessions`, `unclaimedSessions`, `foreignSessions`, `attachedSessions`), `sampledAt` |
| `sessions[]` | one row per **session** (never per pane): `name`, `sessionId`, `created`, `windows`, `panes`, `attached`, `command`, `cwd`, `panePid`, `dead`, `exitCode`, `kind` (`agent`/`terminal`/`other`), `scope`, and the owner facts when known (`ownerId`, `ownerName`, `workspaceId`) |
| `absent[]` | terminals this daemon has a row for whose session is **not** on the server, with `lostAtRestart` from the flight recorder (ADR-0085) |

It costs **one tmux subprocess** for the whole inventory (`list-panes -a -F`,
`internal/tmux/server.go`) plus the server facts (`display-message`,
`list-clients`, `tmux -V`, `show-options`) — no call per session, no call per
pane, and no timer: the surface refetches from the change feed (`tmux.changed`
after its own removal, `terminal.*` when a session comes or goes). A machine
with no tmux binary, or none running, is a 200 with `installed: false` /
`running: false` and empty lists — an honest empty screen, not a 503 the app
would have to invent copy for.

A session with two panes stays ONE row: the active pane supplies the details
and `panes` counts them, so a split pane cannot duplicate a row or inflate a
count (the reference implementation this learned from had that bug).

## Attribution: the name is a hint, the marker is the receipt

Ownership is resolved in two steps, and the split is the whole safety model:

1. **The name**, in the list. `picode-sh-<id>` means a terminal, `picode-<id>`
   means an agent, anything else is not PiCode's namespace. The id is looked up
   in the store (`GetTerminal`, `GetAgent`); a match is `scope: "yours"`, no
   match is `scope: "unclaimed"`, a foreign name is `scope: "notPiCode"`.
   The step is lossy by construction — session names are sanitized to
   `[a-z0-9-]` — which is exactly why a miss is reported as an uncertainty and
   never as "nobody's".
2. **The markers**, before any action, from the session's own environment:
   `PICODE_TERM_ID` and `PICODE_AGENT_ID` are injected at creation
   (`-e`, `internal/server/terminals.go` and `internal/store/agents.go`; the
   CLI launch path injects them too). `SessionReceipt` reads them together
   with the identity in one pass. A marker that resolves to a terminal or an
   agent this daemon still holds refuses the removal and names the surface
   that owns the session.

`internal/server/tmux_test.go:TestTmuxMarkerNamesMatchTheirWriters` pins the
marker strings against the store's constant, because a rename in one of the
three writers would silently turn every session into an "unclaimed" one.

**The uncertainty is in the product, not hidden.** A PiCode-shaped session
missing from the store may be a leftover of a deleted store, a harness's, or
**another PiCode instance's on the same machine** — measured 2026-09-14: a
script over the socket read seven live sessions as unclaimed, and all seven
belonged to a second instance it had failed to reach. No read in this daemon
can settle that; the row says so, and a human confirming one row at a time can.

## The one action

`POST /api/tmux/sessions/{name}/reap` with `{confirm: true, sessionId, created,
panePid}`. The decision table, row by row — and every row is a test in
`TestTmuxReapDecisionTable`:

| Name | Store knows the marker's owner | Receipt | Result |
|---|---|---|---|
| not `picode-*` | — | — | 400 — the user's own session is never this app's business |
| `picode-*` | yes | any | 409 — names the terminal/agent and its door |
| `picode-*` | no | missing, or `sessionId`/`created`/`panePid` moved | 409 — "changed since this page was drawn" |
| `picode-*` | no | matches, `confirm: true` | 200 — one `kill-session` by exact name, one `tmux.session.reaped` event |
| session gone | — | — | 404 |
| unknown JSON field | — | — | 400 — a field nobody sent is a field nobody meant |

No prefix sweep, no bulk action, no automatic removal: the 2026-09-06 debt
(*"never kill by prefix"*) is what this table encodes. The removal is
irreversible, so the response reports the audit honestly — `audit: "recorded"`
or `"failed"` (the session is gone either way) or `"skipped"` on a host with no
store — rather than dressing a missing record up as a failed action.

The button that starts this lives in the row's **expanded** detail, next to the
sentence explaining what PiCode cannot know, and not in the collapsed row: with
the development machine's 15 leftovers that placement was 15 red buttons
drawing the eye down a list whose job is to be read (found in visual review,
2026-09-14, and moved).

The `tmux.changed` ephemeral event rides the change feed (ADR-0048) so every
open surface refreshes.

## What the app does not do

It does not list terminals or agents as a fleet (that is the sidebar's job —
ADR-0109's doors), it does not offer tmux **settings** (Preferences → Terminal
owns the live catalog, ADR-0025), it does not open a second attach for a
terminal that has one (one xterm per terminal), it never creates a session,
and it draws no phone body: the phone lists the app as desktop-only and
answers its route with one line.

## Version note

PiCode requires tmux ≥ 3.5 and the read uses `session_id`, `pane_id`,
`pane_dead_status`, `session_attached`, `pane_active` and `show-environment -t`
— all present in 3.2+, so nothing here raises the floor. `pane_dead_status` is
only meaningful with `remain-on-exit`, which PiCode does **not** set: a dead
pane therefore usually means a dead session (the "exited N" badge is for the
case where the user's own `tmux.conf` turned it on).