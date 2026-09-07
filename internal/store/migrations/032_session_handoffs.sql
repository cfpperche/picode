-- Cross-CLI session handoffs (ADR-0087): one row per translated session,
-- the lineage between a source conversation and the native session (or
-- brief) it continued as in another Agent CLI.
CREATE TABLE IF NOT EXISTS session_handoffs (
  id          TEXT PRIMARY KEY,
  source_cli  TEXT NOT NULL,
  source_id   TEXT NOT NULL,
  source_path TEXT NOT NULL DEFAULT '',
  target_cli  TEXT NOT NULL,
  target_id   TEXT NOT NULL DEFAULT '',
  target_path TEXT NOT NULL DEFAULT '',
  mode        TEXT NOT NULL,
  "window"    TEXT NOT NULL,
  tools       TEXT NOT NULL DEFAULT '',
  manifest    TEXT NOT NULL DEFAULT '{}',
  terminal_id TEXT NOT NULL DEFAULT '',
  agent_id    TEXT NOT NULL DEFAULT '',
  created_at  TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS session_handoffs_source ON session_handoffs(source_cli, source_id);
CREATE INDEX IF NOT EXISTS session_handoffs_target ON session_handoffs(target_cli, target_id);
