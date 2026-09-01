# ADR-0040: Per-agent `--session-dir` extends ownership into pi's own TUI

- **Status**: accepted (amends ADR-0039; ADR-0039's own Decision, table,
  and migration are unchanged and still correct — this adds a second,
  complementary mechanism for a surface ADR-0039's DB-filter cannot reach)
- **Date**: 2026-09-01

## Context

ADR-0039 fixed cross-agent session bleed in PiCode's *own* chat "Search
sessions" picker (`handleListSessions`) by historizing ownership in a new
`agent_sessions` table and filtering the HTTP response. It explicitly
rejected giving each agent a private `--session-dir`, reasoning that
`internal/session`'s cwd-keyed functions are used machine-wide and that
`--session-dir` leans on upstream pi behavior with known rough edges
(pi-mono #320, #2024).

That fix only reaches PiCode's own UI. The same agent, viewed through its
own interactive pi TUI in tmux — the same process, reached a different
way — still shows every session in the shared cwd bucket via pi's native
"Resume Session (Current Folder)" picker, because that picker reads
`~/.pi/agent/sessions/<cwd>/` directly off disk, with no knowledge of
PiCode's HTTP API or `agent_sessions` table at all. A database-side filter
cannot reach a picker rendered entirely inside a process PiCode does not
control the internals of. An agent can be driven from the chat or from
its own pi TUI — two doors onto the same entity, not two concepts — and
both need to agree. Terminals stay out of scope entirely: they have no
session concept and never call any of this code.

Three facts, confirmed live against installed `pi 0.84.4`, change the
calculus on the rejected alternative:

1. `pi --session-dir <X> --session-id <uuid>` (fresh start): the file lands
   directly in `<X>`, no further cwd-encoded sub-bucketing.
2. `pi --session-dir <X> --continue`: finds and appends to the file just
   created in `<X>`, and the *default* cwd bucket is never touched —
   `--session-dir` governs lookup, not just storage, matching its own
   `--help` text ("Directory for session storage and lookup").
3. `pi --session <path-outside-X> --session-dir <X>`: an explicit
   `--session <path>` always wins over `--session-dir` for that spawn,
   appending at the path's original location; `<X>` stays untouched. No
   physical migration of existing sessions is required — only future fresh
   starts land in the new private directory, mirroring this ADR's own
   accepted-limitation precedent of not reconstructing pre-fix history.

A fourth fact, also confirmed live, closes the one gap the "shared
`CLIFlags()`" plumbing opens: `pi --mode rpc --no-session --session-dir <X>`
starts cleanly and leaves `<X>` empty — `--session-dir` is inert alongside
`--no-session`, so the ephemeral headless spawn `internal/rpc/auth.go`
uses for MCP OAuth is unaffected by inheriting the flag for free.

Separately, nesting the private directory *inside* `internal/session`'s
existing `Root()` (`~/.pi/agent/sessions/<agentID>/`) means `ListRoot`'s
loop — which walks every immediate subdirectory with no naming-shape
filter — and `UnderRoot` pick it up with zero code changes. This is what
keeps the machine-wide views (`/sessions/all`, the delete-anywhere
validator, the "from a Pi session" adopt picker) untouched, closing the
"ripples through everything" concern ADR-0039's rejection raised.

## Decision

`Agent.CLIFlags()` (`internal/store/agents.go`) unconditionally appends
`--session-dir <session.AgentDir(a.ID)>` on every spawn, for every agent
with a persisted id. `session.AgentDir(agentID)` is
`filepath.Join(session.Root(), agentID)`, added to `internal/session`
(the only package with zero internal imports that already owns `Root()`,
avoiding a `store → server` import cycle). Because `CLIFlagsForSpawn` and
all three spawn chokepoints (`Deps.spawnFlags` for tmux, `rpc.Runtime.Start`
for managed mode, and `rpc/auth.go`'s headless OAuth spawn) already build
on `CLIFlags()`, this reaches every spawn path through one function.

`safeSessionPath` (four call sites in `internal/server`) generalizes from
a single `(cwd, path)` root to `(path, dirs ...string)`, so a session
inside an agent's private dir validates the same way one in the shared
cwd bucket does. `handleManageSessions` and `sweepOrphanSessions` gain a
new `workspaceSessionDirs`/per-agent-dir union so the workspace-scoped
manage view and the age-based orphan sweep see private dirs too — the
purge-on-delete flow (`internal/server/cleanup.go`) gets the matching
treatment (`session.DirStatsAt`, `session.RemoveAgentDir`) so "delete
sessions too" on an agent or workspace removal doesn't silently leave an
`AgentDir` behind.

Bundled in the same change: the orphan sweep now also skips any path
present in `agent_sessions` (`store.AllAgentSessionPaths`), not just an
agent's *current* `session_path`. This closes a latent gap that existed
from the moment ADR-0039 shipped — an agent's older, non-current session
is resumable in its own chat picker, but the sweep never knew that and
could delete it out from under the user once it aged past the cleanup
threshold.

## Consequences

- **Easier**: pi's own in-TUI "Resume Session" picker, reached by a human
  attached to the tmux pane rather than through PiCode's chat, now shows
  only sessions that agent itself created going forward — no PiCode-side
  filtering possible, because pi's own directory-scoped lookup does the
  work. Zero new migrations or schema — `AgentDir` is a pure function of
  the agent id, which already exists. Terminals required no changes at
  all: `ensureShell` never calls `CLIFlags()`.
- **Harder**: four existing `internal/store/store_test.go` assertions that
  treated "no explicit config" as "zero CLI flags" needed their expected
  lengths updated — every real agent now has at least the `--session-dir`
  pair. `safeSessionPath`'s signature changed from `(cwd, path)` to
  `(path, dirs...)`; its four callers were updated in the same change.
- **Accepted limitation**: pi's native picker is scoped by directory, not
  by a queryable history — an agent with session history from before this
  shipped (sitting in the shared cwd bucket) sees that history in
  PiCode's own chat picker (unaffected, DB-driven, ADR-0039) but **not**
  in pi's own native picker, until its next fresh start populates its
  `AgentDir`. Its *current* session keeps resuming correctly regardless
  (fact 3), so nothing breaks — pi's native "browse other sessions" list
  is just narrower than PiCode's own until it accumulates post-fix
  history. Same direction ADR-0039 itself already accepted erring in.
- **If wrong**: dropping the one `--session-dir` append in `CLIFlags()`
  reverts every spawn to today's shared-bucket behavior; the generalized
  `safeSessionPath`/`workspaceSessionDirs` degrade to their old
  single-root behavior for any agent whose `AgentDir` is empty — reversible
  in either direction without a migration.

## Alternatives considered

| Alternative | Why not |
|---|---|
| Give each agent a private `--session-dir` *(ADR-0039's own rejected alternative)* | **Revisited here.** Rejected in ADR-0039 for chat-picker attribution, where the DB filter is sufficient and simpler. Turns out necessary anyway for a surface that filter cannot reach at all: pi's own native picker, which reads straight off disk with zero PiCode awareness. Nesting `AgentDir(id)` under `Root()` (confirmed live) keeps `ListAll`/`ListRoot`/`UnderRoot` unchanged, closing the "ripples through everything" concern; the pi-mono #320/#2024 rough edges concern `--resume`/`--continue` inside a *shared* bucket, not an isolated per-agent `--session-dir`, and were not observed in live testing of that exact combination. |
| Physically migrate each agent's existing session(s) into its new `AgentDir` at upgrade time | Rejected for the same reason ADR-0039 rejected reconstructing pre-fix history: the accepted limitation above is honest and reversible; a migration step adds file-move failure modes (partial moves, permissions, a session mid-write) for a gap that self-heals on next fresh start and never breaks a resume (fact 3). |
| Leave the `sweepOrphanSessions`/`agent_sessions` gap for a separate change | Real gap, found while researching this same table, and it predates this ADR — but small, additive, and in the same code area, so bundled in rather than left as a known data-loss risk. |
