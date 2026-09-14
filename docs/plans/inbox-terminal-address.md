# Inbox terminal replies — resolving the address at reply time

The debt (2026-09-08, rewritten 2026-09-12): "unmanaged Pi items lack an address
until session update; feed events can be missed across reconnects (ADR-0048)."

## The two halves

**Address half (paid here).** `pi-inbox` 0.2.0 stamps `sourceKind`/`sourceId`
from `PICODE_TERM_ID` and the asking session path onto every new item. Items
that predate the stamp (or where pi could not name a session at ask time) reach
`DeliverTerminalReply` with an empty `SessionPath` and were refused outright —
"this question predates session tracking". But the address exists elsewhere:
`TerminalLaunch.LastSession` pins the native session a terminal was last
running (the "session update" the debt names), written by the same native
session detection that powers one-click resume. The reply can be routed when
**two independent observations agree**: the store's pinned session for that
terminal, and the session the terminal's live receiver is showing right now.

**Feed half (retired, already paid).** Every surface that renders ephemeral
state refetches a snapshot on `feed.open`/`feed.reset`: the Messages view
(`useWorkspaceCommunication` refreshes on `peer.*|terminal.*|agent.*|feed.*`),
reminders (`reminders.js` reconciles on `feed.open`), checklists, and the
mobile poll hooks. Durable `inbox.*`/`peer.*` events replay from the cursor
(ADR-0048); a missed ephemeral event is corrected by the reconnect refetch.
Webhooks are at-least-once with receiver dedupe — a delivery contract, not a
defect. The debt line retires with this plan as the record.

## Decision table

| Item `SessionPath` | Terminal pin (`LastSession`, cli=pi) | Receiver shows | Action |
|---|---|---|---|
| set | any | same session | Deliver (unchanged path) |
| set | any | different / empty | Refuse, item stays open (unchanged rows) |
| empty | none | any | Refuse "predates session tracking" (unchanged) |
| empty | path set, unsafe under cwd | any | Refuse "could not be identified safely" (existing guard) |
| empty | path set, file gone | any | Refuse "no longer exists" (existing guard) |
| empty | path == shown session, file exists | that session | **Deliver — the new row** |
| empty | path != shown session | some session | **Refuse, naming the mismatch (new row)** |

One answer, two sources: the pin (recorded when the session was observed) and
the live receiver hello must name the same file, and the file must still exist
under the terminal's own session tree. A pin that disagrees with the pane is
exactly the 2026-09-11 "showing a different session" hazard, so it refuses
instead of guessing.

Not in scope: `system`-sourced items (`pi (unmanaged)` — pi launched outside
PiCode's launcher) keep the honest refusal; without `PICODE_TERM_ID` nothing
binds the item to a pane, and no receiver ever says hello for it. The accepted
gap "daemon death between park and JSONL row" also stands unchanged.
