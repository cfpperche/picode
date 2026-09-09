# Cross-CLI session handoff (ADR-0088)

> Part of [PiCode's architecture](../architecture.md) (ADR-0105: one file per subsystem). Edit here; the index only links.

Any session of an Agent CLI can continue in another one from the Sessions
tab ("Continue in <CLI>…"). `internal/transcript` is the portable model —
ordered events (message, tool call, tool result, thinking, compaction,
context), a header and a manifest of what a reader could not carry.
`internal/clisession` adds optional capabilities next to `Source`:
`Reader` (native → timeline), `Writer` (timeline → a new native session,
create-only, atomic, re-read before it counts) and `Prompter` (launch
arguments for an initial prompt). They are discovered by type assertion,
never by a switch, and `GET /api/clis` advertises them as
`sessions: {list, read, write, prompt}`; the web derives the targets from
that. All six CLIs read and write; prompters exist for every CLI but
Hermes, which cannot be started with an initial prompt.

Writers publish two ways. Claude Code, Codex, pi (as an adopted managed
agent) and Grok get a new session artifact created in their own store,
atomically and never over an existing one. OpenCode and Hermes are SQLite
stores their CLI holds open, so a writer publishes through the vendor's
own importer instead (ADR-0094): `WriteRequest.Run` executes the CLI in
the session's folder with its configured executable and environment
(`opencode import`, `hermes sessions import --from claude`). Every route
ends in a read-back, block for block where the importer lands what it was
given and by arrival where it rewrites, as Hermes does.

`POST /api/clis/{cli}/sessions/handoff/preview` and `…/handoff` share one
plan: window (since the last compaction, or all), repair (every tool call
answered), prepare (no thinking, no injected context), a handoff note as
the first user message, then the target's Writer — or, in brief mode, a
deterministic markdown brief under `<dataDir>/handoffs/<id>/brief.md` and
the target started with a one-line prompt. Lineage lives in
`session_handoffs` (event `session.handoff`) and is shown on both session
listings. The installed target version comes from the setup check (run on
demand); an unknown format refuses native and keeps brief.
