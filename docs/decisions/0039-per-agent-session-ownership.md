# ADR-0039: Per-agent session ownership, tracked in PiCode, not pi's layout

- **Status**: accepted (amends ADR-0006's session-visibility clause)
- **Date**: 2026-09-01

## Context

Pi buckets session JSONL files by cwd (`~/.pi/agent/sessions/--<cwd>--/`),
not by who created them. Agents commonly share a cwd — `AgentCwd` defaults
to the workspace path, and nothing enforces uniqueness — and a Terminal
running bare `pi` by hand writes into that same bucket. ADR-0006 itself
documented the consequence at the time as accepted: "Sessions remain
per-process as pi creates them (keyed by cwd); both modes see the
workspace's session history." The owner hit this directly: an Agent tab's
**Search sessions** picker (`handleListSessions`, backing
`SessionBar.jsx`) showed a session that actually belonged to a Terminal
sharing the same folder — because the handler only used `agent=<id>` to
resolve a cwd, then listed every `.jsonl` pi had ever written there,
unfiltered.

Pi's session filename encodes a timestamp PiCode does not control
(`<ts>_<id>.jsonl`), so the exact path of a session that does not exist
yet cannot be predicted. But pi's `--session-id <id>` flag — "use exact
project session id, creating it if missing", confirmed live against the
installed `pi 0.84.4` and not gated behind `--mode` — lets PiCode choose
the *id* up front and pass it at spawn. That turns attribution into an
exact match on an id PiCode itself minted, not a guess based on timing.

## Decision

PiCode mints and historizes a session id before every fresh spawn (no
current `session_path`), in both run modes, and records every session
id/path an agent is ever pointed at — first spawn, resume, fork, clone,
adopt, import — in a new append-only `agent_sessions` table
(`internal/store/agent_sessions.go`, migration
`014_agent_session_history.sql`). This is parallel to, not a replacement
for, `agents.session_path` (which keeps its existing "current pointer
only" contract). `handleListSessions` filters `session.List(cwd)` to rows
owned by the requesting agent, always including its current session as a
safety net. Pi's on-disk layout, `internal/session`'s cwd-keyed path
functions, and the machine-wide session views (`/sessions/manage`,
`/sessions/all`) are all untouched — those exist specifically to show
every session for cleanup and stay that way.

## Consequences

- **Easier**: Terminals need no changes at all — they have no
  `session_path`, are never handed `PICODE_AGENT_ID`, and never call
  `handleListSessions`. A human's bare `pi` in a Terminal is
  architecturally identical to an external session, exactly ADR-0021's
  adopt-by-copy scenario, never implicitly shared. Filtering is an exact
  id/path match, not a heuristic — no false negatives from spawn timing,
  no false positives from guessing which new file is "probably mine".
- **Harder**: nine call sites spawn `pi`; the fix reaches all of them
  through two chokepoints instead of repeating the mint-and-pass logic
  nine times — `rpc.Runtime.Start` for managed mode, a new
  `Deps.spawnFlags` helper for the five interactive/tmux call sites.
  Depends on the installed `pi` supporting `--session-id` (ADR-0003's
  recommended baseline, v0.80.4+, does not guarantee it) — this degrades
  to today's unattributed-auto-session behavior if the flag is rejected,
  not a spawn failure.
- **Accepted limitation**: on upgrade, only each agent's *current*
  `session_path` is backfilled into `agent_sessions` (the migration's own
  `INSERT ... SELECT`). Any earlier session an agent used before that one
  — already overwritten, or abandoned in a shared-cwd setup predating
  this fix — has no historical record to recover it from, and simply
  stops appearing in that agent's picker. This is the correct direction
  to err in, given the bug was unrelated sessions over-appearing; those
  files stay on disk, visible and adoptable via the machine-wide "All
  sessions" view.
- **If wrong**: `agent_sessions` is additive and read only by the
  picker's filter; dropping the table or the filter reverts to today's
  behavior with no data loss elsewhere.

## Alternatives considered

| Alternative | Why not |
|---|---|
| Give each agent a private `--session-dir` | `internal/session`'s `DirName`/`Dir`/`Root` are cwd-keyed and used machine-wide (`ListAll`, `tree.go`, `cleanup.go`); a private-dir-per-agent exception would ripple through all of them. It also leans on upstream pi behavior with known rough edges in exactly this area (pi-mono #320 "--resume doesn't look at the right session directory", #2024 "/resume from All scope doesn't update cwd"). ADR-0005 already commits `internal/session` to being a read-only index of pi's *existing* layout — this alternative asks pi to change what it owns. |
| Directory-diff / snapshot-at-launch reconciliation heuristic | The fallback considered before confirming `--session-id`. Once confirmed, rejected: exact-id matching is strictly simpler *and* strictly more reliable than "which new file in this bucket, near this timestamp, is probably mine" — no scenario favors the heuristic. |
| Fork or patch pi's session writer | ADR-0003 and ADR-0028 both treat "we orchestrate pi, we do not fork it" as a standing non-goal. |
| Tag ownership inside a pi extension (the `pi-roles`/`PI_ROLES_AGENT` pattern, ADR-0033) | Extensions are opt-in packages configured per workspace (`.pi/settings.json`, ADR-0028/0033). Session isolation has to hold for every agent unconditionally, including ones with no packages installed — it belongs in PiCode's own backend, not in something a user could uninstall. |
