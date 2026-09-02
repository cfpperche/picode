-- Actuation batches (ADR-0053): one validated picode-act block per agent
-- turn, executed by the human's extension. Like automation_runs (016),
-- plain text columns with no foreign keys: a batch outlives a deleted
-- agent row and expires on its own.
CREATE TABLE act_batches (
  id         TEXT PRIMARY KEY,
  agent_id   TEXT NOT NULL,
  origin     TEXT NOT NULL,
  actions    TEXT NOT NULL, -- JSON array of acts
  state      TEXT NOT NULL CHECK (state IN ('pending','claimed','done','expired')),
  round      INTEGER NOT NULL DEFAULT 1,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE INDEX idx_act_batches_by_agent ON act_batches(agent_id, created_at DESC);
