-- ADR-0194: a person's removal of an agent writes an exit record in the same
-- transaction as the delete, and the agent counts its own turns.

-- turns: transitions into work. Rows that exist now were never counted, so
-- they read NULL ("not measured"); rows born after this read 0 and count.
ALTER TABLE agents ADD COLUMN turns INTEGER DEFAULT 0;
UPDATE agents SET turns = NULL;
ALTER TABLE agents ADD COLUMN first_worked_at TEXT;
ALTER TABLE agents ADD COLUMN last_worked_at TEXT;

-- No foreign keys: the agent row is gone by design, and removing a workspace
-- keeps its exits (the catalog outlives what it describes).
CREATE TABLE agent_exits (
  id                TEXT PRIMARY KEY,
  agent_id          TEXT NOT NULL,
  agent_name        TEXT NOT NULL DEFAULT '',
  workspace_id      TEXT NOT NULL DEFAULT '',
  workspace_name    TEXT NOT NULL DEFAULT '',
  cli               TEXT NOT NULL,
  provider          TEXT NOT NULL DEFAULT '',
  model             TEXT NOT NULL DEFAULT '',
  config            TEXT NOT NULL DEFAULT '{}',  -- JSON: the frozen setup
  created_at        TEXT NOT NULL,               -- the agent's birth
  removed_at        TEXT NOT NULL,
  lifetime_s        INTEGER NOT NULL DEFAULT 0,
  turns             INTEGER,                     -- NULL = not measured
  first_worked_at   TEXT,
  last_worked_at    TEXT,
  signals           TEXT NOT NULL DEFAULT '{}',  -- JSON: status, inbox, checklist
  sessions          TEXT NOT NULL DEFAULT '{}',  -- JSON: where the sessions were
  sessions_purged   INTEGER NOT NULL DEFAULT 0,
  work_purged       INTEGER NOT NULL DEFAULT 0,
  origin            TEXT NOT NULL DEFAULT 'api', -- desktop | mobile | api
  asked             INTEGER NOT NULL DEFAULT 0,
  ask_skip          TEXT NOT NULL DEFAULT '',    -- off | idle | brief | client
  outcome           TEXT NOT NULL DEFAULT '',    -- '' = no answer
  reasons           TEXT NOT NULL DEFAULT '[]',
  note              TEXT NOT NULL DEFAULT '',
  labeled_at        TEXT,
  taxonomy          INTEGER NOT NULL DEFAULT 1,
  undone_at         TEXT,
  restored_agent_id TEXT NOT NULL DEFAULT ''
);
CREATE INDEX idx_agent_exits_removed ON agent_exits(removed_at DESC);
CREATE INDEX idx_agent_exits_workspace ON agent_exits(workspace_id, removed_at DESC);
