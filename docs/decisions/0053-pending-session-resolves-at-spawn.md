# ADR-0053: A pending session resolves to its file at spawn — mode switches resume, never mint

- **Status**: accepted (amends ADR-0039 and ADR-0040; both decisions stand —
  this fixes the follow-through gap between them and adds one mechanism)
- **Date**: 2026-09-02

## Context

The owner hit three symptoms on one freshly created agent: its composer
status bar showed **another agent's** context, spend and cache the moment
it was created; its chat picker said "No sessions yet" after it had
already chatted; and opening its TUI started a **new** session instead of
continuing the chat's thread — after which a message sent from the chat
killed the TUI's work (run modes are exclusive, ADR-0006, so the kill
itself is by design; the loss of the thread was not).

Root cause chain, all in code that ADR-0039/0040 touched or left behind:

1. **The picker never saw private-dir sessions.** ADR-0040 routes every
   fresh spawn into `session.AgentDir(agentID)`, but `handleListSessions`
   still listed only `session.List(cwd)` — the shared cwd bucket. An
   actively-used fresh agent's own session was invisible to its picker,
   and (because ADR-0039's lazy backfill of pending `--session-id` → path
   lives inside that same handler) `agents.session_path` stayed NULL no
   matter how much the agent had chatted.
2. **Every run-mode switch minted a competitor.** With `session_path`
   NULL, each spawn site (`Deps.spawnFlags` for the TUI, `Runtime.Start`
   for managed) took ADR-0039's fresh-start branch and minted a *new*
   pending `--session-id`. Chat → TUI → chat walked across three
   different sessions; each hop abandoned the previous thread.
3. **The status bar was never agent-scoped.** `GET /api/workspaces/{id}/status`
   supports `?agent=`, but the desktop app fetched it without the
   parameter, so `loadWS` fell back to `DefaultAgent` — the workspace's
   *first-created* agent. Every later agent's screen showed that first
   agent's context %, tokens, cache and cost; a fresh agent did not start
   at zero but with a sibling's spend.

## Decision

Before any spawn mints a fresh `--session-id`, it first tries to adopt:
`store.ResolvePendingAgentSession(agentID)` matches the agent's pending
session ids (historized by `NewPendingAgentSession`, ADR-0039) against
the files actually on disk in its private dir (ADR-0040), resolves every
match's row, backfills `agents.session_path` when it is empty, and
returns the newest match. `Deps.spawnFlags` (interactive/tmux) and
`rpc.Runtime.Start` (managed) call it first; an adopted path becomes a
plain `--session <path>` resume. Adoption is safe to skip: no pending
ids, or no files yet, just means a genuinely fresh start.

`handleListSessions` now lists the union of the cwd bucket and the
agent's private dir (`session.ListDirs`), so the picker and its built-in
backfill see where ADR-0040 actually writes. The desktop status bar
fetches `/status?agent=<selected>` via `workspaceStatusPath()`, making
the bar — and the Sessions chip's cost, which derives from it — follow
the selected agent; a free agent (no workspace) still gets no bar, as
before.

The full spawn decision table (tested row by row in
`internal/server/sessions_test.go`):

| session_path | pending ids | file on disk | spawn argv | session_path after |
|---|---|---|---|---|
| set | any | any | `--session <path>` | unchanged |
| empty | yes | yes | `--session <newest file>` | backfilled to file |
| empty | yes | no | `--session-id <fresh mint>` | still empty |
| empty | no | — | `--session-id <fresh mint>` | still empty |

Picker rows (cwd bucket vs private dir × owner) keep ADR-0039's
isolation table; private-dir rows are added, cross-agent visibility
still denied, unowned files still hidden.

## Consequences

- **Easier**: the chat and the agent's own TUI are two doors onto one
  thread. Opening the TUI resumes the chat's session; sending from the
  chat later resumes the TUI's session — including turns typed while the
  other door was open. The kill on mode switch (ADR-0006) now costs at
  most the in-flight turn, never the thread.
- **Easier**: a fresh agent reads as fresh — zeroed bar, empty picker —
  instead of inheriting the first agent's numbers; and its picker shows
  its real (private-dir) history.
- **Harder**: `ResolvePendingAgentSession` does directory I/O
  (`ListDirs` + `Summarize`) on the rare spawn-of-a-sessionless-agent
  path; acceptable, and only until the first resolve lands.
- **Accepted limitation**: pending ids from *before* ADR-0040 whose
  files landed in the shared cwd bucket are not adopted by the resolver
  (it scans the private dir only). The picker's union still shows them,
  and ADR-0039's in-picker backfill still resolves them there; a
  pre-fix upgrade with such a row sees one extra mint instead of a
  resume — the same direction ADR-0039/0040 already accepted erring in.
- **Known test gap**: `Runtime.Start`'s adoption branch is the same
  three lines as the tested `spawnFlags` branch, but is not covered by a
  live managed-spawn test (that needs a real pi RPC process); locked in
  at the `spawnFlags` chokepoint instead.
- **If wrong**: dropping the `ResolvePendingAgentSession` calls reverts
  every spawn to ADR-0039's always-mint behavior; dropping the
  `ListDirs` union reverts the picker to the cwd bucket; dropping the
  `?agent=` param reverts the bar to the workspace default. All three
  are single-point reverts with no schema or data migration.

## Alternatives considered

| Alternative | Why not |
|---|---|
| Backfill `session_path` from the managed runtime (`GetState` on settle) | Helps only the managed side; the TUI writes sessions with no event channel at all (the very reason ADR-0040 exists). The resolve-at-spawn covers both doors from one place, with no goroutine or RPC round-trip. Worth adding later as defense-in-depth, not as the fix. |
| Keep the picker cwd-only and "fix" the bar by removing it for non-first agents | Throws away the user-visible signal ADR-0041's dashboard model promises per agent; the bar is per-agent truth, and the server already supported scoping — the desktop app was simply not passing it. |
| Warn/confirm before a chat send kills a working TUI | Considered; refused for now. With adoption in place the loss is bounded to the in-flight turn, the exclusive-mode kill is ADR-0006 working as decided, and a guard would strand a user whose TUI is busy with no way to take the agent back from the chat. Revisit if the owner wants a confirm. |
| Make run modes non-exclusive (TUI attach to the managed process) | A much larger protocol change (pi TUI has no attach mode to a `--mode rpc` process); out of scope for a correctness fix. |
