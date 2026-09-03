-- Internal checklists (ADR-0055). Per-agent obligation level, passed to
-- pi-checklist as PICODE_CHECKLIST; and the latest list the extension
-- published, one row per agent (the session file is the history).
ALTER TABLE agents ADD COLUMN checklist TEXT NOT NULL DEFAULT 'changes';

CREATE TABLE agent_checklists (
  agent_id   TEXT PRIMARY KEY,
  session_id TEXT NOT NULL DEFAULT '',
  items      TEXT NOT NULL, -- JSON array of {text, status}
  absent     INTEGER NOT NULL DEFAULT 0, -- the turn ended without one, or a change was refused
  updated_at TEXT NOT NULL
);
