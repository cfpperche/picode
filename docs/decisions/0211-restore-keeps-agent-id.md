# ADR-0211: A restored agent keeps its id, and Undo is a restore

- **Status**: accepted (owner direction, 2026-09-24)
- **Date**: 2026-09-24
- **Boundary**: protocol + persistence — `POST /api/agent-history/{id}/restore`
  recreates the agent under the removed agent's id and takes `undo`; an agent
  id can now be deleted and created again. Amends ADR-0205 (which minted a
  new id) and ADR-0194 (Undo was a client snapshot).

## Context

ADR-0205 restored a removed agent under a **new** id. Three defects came from
that one choice, all measured:

| Defect | Cause |
|---|---|
| A restored Pi agent's chat was empty (visual QA, 2026-09-23) | Pi's private session folder is `sessions/<agent id>` (ADR-0040); the fix moved the folder |
| The eight-second Undo still had that defect | Undo recreated the agent from a browser snapshot, with a new id, and only re-pointed `sessionPath` |
| Automations, pins and Canvas nodes that named the agent stayed broken | They name the old id |

The removed id is free: `agents` rows that name it cascade on delete, and exits
carry it as plain text.

## Decision

**Restoring creates the agent under the id it had** (`Store.AddAgentAs`;
refused with 409 if a living agent holds it). Its Pi session folder is its own
again, so nothing moves, and whatever still names the id points at it.

**Undo is a restore** with `undo: true`: the agent comes back even with no
conversation to resume (it may never have had one), and a private work folder
the removal purged under the data dir's `work/` is made again. The browser's
snapshot path remains only for a removal whose answer did not carry the exit.
`POST /api/agent-exits/{id}/undo` stays for API callers.

## Consequences

- Undo and the history now give the same result: the same agent, its setup,
  and its conversation resumed on any of the nine CLIs.
- A feed consumer can see `agent.deleted` then `agent.added` for one id. The
  fleet reducer removes and inserts by id; Canvas and checklists drop state on
  delete and rebuild it from the new row.
- Events, sessions and exits keyed by the id now span two lives of one
  agent. An exit per removal keeps each life's record apart.
- If a caller had cached "this id is gone forever", it is now wrong. None in
  this repository did.

## Alternatives considered

- **Keep new ids and move the Pi folder** (ADR-0205 as shipped). Lost: it
  fixes the chat only, and Undo and automations stay broken.
- **A `deleted_at` soft delete.** Lost for the reasons ADR-0205 gives: every
  agent query would have to filter dead rows.
