# Cross-CLI session handoff (ADR-0088)

> Part of [PiCode's architecture](../architecture.md) (ADR-0105: one file per subsystem). Edit here; the index only links.

Any session of an Agent CLI can continue in another one from that CLI's
Sessions pane ("Continue in <CLI>…") **and** from the terminal that is
running it: the sidebar `⋯` menu, the Agent CLIs Terminals `⋯` menu, and
the pane's right-click menu. The source is the terminal's pinned
conversation (`lastSession`, ADR-0084), not a session picker. It is still
a handoff, never a live switch — the source terminal keeps running and a
new terminal (or a stopped Pi agent) opens on the target.

`internal/transcript` is the portable model —
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
demand); an unknown format refuses native and keeps brief. A source over
the 64 MB read cap drops native and offers a brief of the last 64 MB of
the file (the recent turns); executing native on that source is still 413.

Where the handoff opens is a choice when the target is also a managed
agent — `sessions.agent` in `GET /api/clis`, pi only today (ADR-0040).
`landing: "agent"` adopts a stopped managed agent: for native it owns the
written session, for brief it starts with the brief queued as its first
prompt. `landing: "terminal"` opens a CLI terminal; for pi native the
session is written into pi's own cwd bucket and the terminal resumes it
through `--session`. Omitting `landing` keeps the shipped default (pi
native → agent, everything else → terminal), and a CLI without an agent
side refuses `landing: "agent"` with 400.

A long-term memory server in the same folder (hooks that inject a next-
session brief) is a different layer: it does not replace Native session,
and Continue does not wrap launches through it. Study:
[2026-09-13-ai-memory](../benchmarks/2026-09-13-ai-memory.md).
