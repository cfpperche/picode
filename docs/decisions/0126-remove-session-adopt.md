# ADR-0126: Remove session adoption — agents are born only from new sessions

- **Status**: accepted
- **Date**: 2026-09-12
- **Boundary**: protocol — a public API route (`POST /api/clis/{cli}/sessions/adopt`) and its UI entry point are removed; persistence keeps the adopted copies as ordinary files

## Context

ADR-0021 let a Pi TUI session JSONL become a PiCode agent by copying the
file and pointing a new stopped agent at the copy ("From a Pi session" in
the create dialog). In practice the flow barely survived contact with the
product: agents adopted this way are born mid-conversation with someone
else's provider/model/thinking frozen in the file, the fork diverges from
the live TUI at the first turn, and the create dialog grew a second
creation path (picker, search, empty states) that the owner never wants
again. ADR-0021 itself recorded the exit: "If wrong: hide From a Pi
session; copies stay as ordinary files." The owner has now decided it was
wrong, and wants the removal to be total — no orphan endpoint.

## Decision

Remove the adoption path end to end. The create dialog offers only
new-session creation (workspace agent, free agent). The
`POST /api/clis/{cli}/sessions/adopt` route, its handler, and
`session.CopyFile` are deleted; `GET /api/clis/{cli}/sessions` stays —
the session-management surface (list, delete, auto-clean) still reads it.
`adoptHome` stays: the CLI handoff (ADR-0088) reuses it. The ADR-0053
spawn-time adoption of a pending session file is untouched — that is a
pointer-heal for an agent's own thread, not a birth path. Agents already
created by adoption keep working; their copies are ordinary session files.

## Consequences

- **Easier**: one creation story — every agent starts from a fresh
  session with the config the user just picked; the create dialog loses
  its second mode; one less mutation endpoint to guard (the pi-only
  adopt guards and their tests go with it).
- **Harder**: a user with an existing TUI conversation must start over
  (or keep using the TUI); there is no bridge. Existing adopted agents
  still resume their copies.
- **Who breaks if we're wrong**: users who relied on adopt re-create the
  flow as a script around `pi --resume` themselves; the JSONL files were
  never modified, so nothing on disk is lost.

## Alternatives considered

| Alternative | Why not |
|---|---|
| Hide the button, keep the API (ADR-0021's literal exit) | An undocumented mutation endpoint nothing calls is debt; simplicity is a product feature |
| Keep adopt, move it to the sessions-management surface | Relocates the second creation path instead of removing it |
| Convert adopt into "import with prompt" | Same surface with extra steps; the owner asked for removal, not redesign |
