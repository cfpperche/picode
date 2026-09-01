-- Append-only record of every pi session id/path an agent has owned
-- (ADR-0039). Parallel to agents.session_path (the *current* pick only) —
-- lets the per-agent resume picker (handleListSessions) filter
-- session.List(cwd) down to sessions this specific agent actually
-- created, resumed, forked, cloned, adopted or imported, instead of every
-- JSONL any process ever wrote into the cwd's shared pi bucket.
--
-- Either column may be the only one known at insert time:
--   - A fresh spawn with no prior session_path mints a session_id up
--     front (--session-id) and historizes it before pi has written
--     anything: pi's filename carries a timestamp prefix PiCode does not
--     control, so the path isn't knowable in advance, but the id is.
--   - A resume/fork/clone/adopt/import already has a concrete path; its
--     id is filled in lazily, best-effort, by ResolveAgentSessionID —
--     never required for correctness, the filter matches on path too.
-- Nullable columns, not '' sentinels: SQLite's unique index treats
-- multiple NULLs as distinct, so several not-yet-resolved rows for one
-- agent never collide.
CREATE TABLE agent_sessions (
  id            TEXT PRIMARY KEY,
  agent_id      TEXT NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
  session_id    TEXT,
  session_path  TEXT,
  first_seen_at TEXT NOT NULL
);
CREATE INDEX idx_agent_sessions_agent ON agent_sessions(agent_id);
CREATE UNIQUE INDEX idx_agent_sessions_agent_sid ON agent_sessions(agent_id, session_id);
CREATE UNIQUE INDEX idx_agent_sessions_agent_path ON agent_sessions(agent_id, session_path);

-- Backfill: each agent's *current* session_path becomes its first
-- historized row, so nothing an agent is actively using today disappears
-- from its own picker after upgrade. session_id is left NULL (resolved
-- lazily, same as a fresh pre-assigned session, on next read).
--
-- Deliberately NOT reconstructed: any *earlier* session this agent used
-- before the current one, already overwritten in agents.session_path, or
-- abandoned outright in a shared-cwd setup predating this fix. There is
-- no historical record to recover it from. It simply stops appearing in
-- this agent's picker going forward — an accepted, documented limitation
-- (ADR-0039), not a bug: those files stay on disk, visible and adoptable
-- via the machine-wide "All sessions" view.
INSERT INTO agent_sessions (id, agent_id, session_id, session_path, first_seen_at)
SELECT 'backfill-' || id, id, NULL, session_path, COALESCE(last_started_at, created_at)
FROM agents
WHERE session_path IS NOT NULL AND TRIM(session_path) != '';
