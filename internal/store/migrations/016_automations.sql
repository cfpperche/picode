-- Automations (ADR-0044): trigger(s) + prompt + bounds; every run is an
-- ordinary pi session on a per-automation agent. Source columns are plain
-- text with no foreign keys so an automation and its runs outlive a
-- deleted workspace or agent (the same rule as inbox_items, 014).
CREATE TABLE automations (
  id                  TEXT PRIMARY KEY,
  name                TEXT NOT NULL,
  enabled             INTEGER NOT NULL DEFAULT 1,
  workspace_id        TEXT NOT NULL DEFAULT 'ws_free',
  action              TEXT NOT NULL CHECK (action IN ('start','message')),
  target_agent_id     TEXT,            -- action=message: the agent to queue into
  agent_id            TEXT,            -- action=start: lazily created, reused per run
  prompt              TEXT NOT NULL,
  provider            TEXT,
  model               TEXT,
  thinking            TEXT,
  cron                TEXT,            -- 5-field expression, NULL = no schedule
  webhook_hash        TEXT,            -- sha256 hex of the secret, NULL = no webhook
  max_cost_usd        REAL,            -- NULL = unlimited
  max_runs            INTEGER,         -- with max_runs_window_min: N runs per window
  max_runs_window_min INTEGER,
  last_fired_at       TEXT,            -- RFC3339, drives due + catch-up
  created_at          TEXT NOT NULL,
  updated_at          TEXT NOT NULL
);

CREATE TABLE automation_runs (
  id            TEXT PRIMARY KEY,
  automation_id TEXT NOT NULL,
  trigger       TEXT NOT NULL CHECK (trigger IN ('schedule','webhook','manual','catch-up')),
  status        TEXT NOT NULL CHECK (status IN ('running','done','failed','skipped')),
  reason        TEXT NOT NULL DEFAULT '',
  session_path  TEXT,
  cost_usd      REAL NOT NULL DEFAULT 0,
  fired_at      TEXT NOT NULL,
  finished_at   TEXT
);
CREATE INDEX idx_automation_runs_by_automation ON automation_runs(automation_id, fired_at DESC);
CREATE INDEX idx_automation_runs_status ON automation_runs(status);
