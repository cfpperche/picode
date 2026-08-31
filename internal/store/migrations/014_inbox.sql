-- Human-facing mailbox (ADR-0037): agents, terminals and PiCode itself
-- file items the operator triages. Deliberately NOT the legacy `messages`
-- table (001_init): that one is agent→agent with to_agent_id NOT NULL
-- ON DELETE CASCADE — it cannot address the human and its rows die with
-- their agent. Inbox items outlive their sources, so source columns are
-- plain text with no foreign keys.
CREATE TABLE inbox_items (
  id                TEXT PRIMARY KEY,
  kind              TEXT NOT NULL CHECK (kind IN ('fyi','question','approval','result')),
  source_kind       TEXT NOT NULL CHECK (source_kind IN ('agent','terminal','system')),
  source_id         TEXT NOT NULL DEFAULT '',
  workspace_id      TEXT NOT NULL DEFAULT '',
  reason            TEXT NOT NULL,
  title             TEXT NOT NULL,
  body              TEXT NOT NULL DEFAULT '',
  blocking          INTEGER NOT NULL DEFAULT 0,
  allowed_responses TEXT NOT NULL DEFAULT '[]',
  state             TEXT NOT NULL CHECK (state IN ('unread','read','done')) DEFAULT 'unread',
  snoozed_until     TEXT,
  response          TEXT,
  responded_at      TEXT,
  created_at        TEXT NOT NULL,
  updated_at        TEXT NOT NULL
);
CREATE INDEX idx_inbox_state_blocking ON inbox_items(state, blocking);
CREATE INDEX idx_inbox_created ON inbox_items(created_at DESC);
