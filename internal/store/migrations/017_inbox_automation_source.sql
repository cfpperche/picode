-- inbox_items.source_kind gains 'automation' (ADR-0046). SQLite cannot
-- alter a CHECK constraint, so the table is rebuilt in place; indexes
-- from 014 are recreated with the same names.
CREATE TABLE inbox_items_new (
  id                TEXT PRIMARY KEY,
  kind              TEXT NOT NULL CHECK (kind IN ('fyi','question','approval','result')),
  source_kind       TEXT NOT NULL CHECK (source_kind IN ('agent','terminal','system','automation')),
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
INSERT INTO inbox_items_new SELECT id, kind, source_kind, source_id, workspace_id, reason, title, body,
  blocking, allowed_responses, state, snoozed_until, response, responded_at, created_at, updated_at
  FROM inbox_items;
DROP TABLE inbox_items;
ALTER TABLE inbox_items_new RENAME TO inbox_items;
CREATE INDEX idx_inbox_state_blocking ON inbox_items(state, blocking);
CREATE INDEX idx_inbox_created ON inbox_items(created_at DESC);
