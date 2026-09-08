-- ADR-0100: pin reminders. inbox_items.kind gains 'reminder' and
-- source_kind gains 'pin'; SQLite cannot alter a CHECK, so the table is
-- rebuilt in place exactly as 017 did (columns as of 023, indexes kept).
CREATE TABLE inbox_items_new (
  id                TEXT PRIMARY KEY,
  kind              TEXT NOT NULL CHECK (kind IN ('fyi','question','approval','result','reminder')),
  source_kind       TEXT NOT NULL CHECK (source_kind IN ('agent','terminal','system','automation','pin')),
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
  updated_at        TEXT NOT NULL,
  session_path      TEXT NOT NULL DEFAULT ''
);
INSERT INTO inbox_items_new SELECT id, kind, source_kind, source_id, workspace_id, reason, title, body,
  blocking, allowed_responses, state, snoozed_until, response, responded_at, created_at, updated_at, session_path
  FROM inbox_items;
DROP TABLE inbox_items;
ALTER TABLE inbox_items_new RENAME TO inbox_items;
CREATE INDEX idx_inbox_state_blocking ON inbox_items(state, blocking);
CREATE INDEX idx_inbox_created ON inbox_items(created_at DESC);

-- One reminder per pin in v2 (the UI shows one chip); the table allows more.
CREATE TABLE pin_reminders (
  id            TEXT PRIMARY KEY,
  pin_id        TEXT NOT NULL REFERENCES pins(id) ON DELETE CASCADE,
  kind          TEXT NOT NULL CHECK (kind IN ('once','interval','cron')),
  at            TEXT,                       -- once: RFC3339 UTC
  interval_min  INTEGER NOT NULL DEFAULT 0, -- interval: minutes
  cron          TEXT NOT NULL DEFAULT '',   -- cron: 5-field, internal/cron
  anchor        TEXT NOT NULL CHECK (anchor IN ('schedule','completion')) DEFAULT 'schedule',
  tz            TEXT NOT NULL,              -- IANA zone the wall clock belongs to
  next_at       TEXT,                       -- UTC, materialized; NULL = nothing owed
  last_fired_at TEXT,
  enabled       INTEGER NOT NULL DEFAULT 1,
  created_at    TEXT NOT NULL,
  updated_at    TEXT NOT NULL
);
CREATE UNIQUE INDEX pin_reminders_pin ON pin_reminders(pin_id);
CREATE INDEX pin_reminders_due ON pin_reminders(enabled, next_at);
