-- ADR-0045 amendment 2026-09-09: many schedules per automation. The
-- schedule leaves the automations row: one row per rule, each with its
-- own zone, switch and last fire, so two rules never share a catch-up or
-- a jitter. No FK on automation_id (same rule as automation_runs: the
-- store deletes children in the automation's own transaction).
CREATE TABLE automation_schedules (
  id            TEXT PRIMARY KEY,
  automation_id TEXT NOT NULL,
  label         TEXT NOT NULL DEFAULT '',
  cron          TEXT NOT NULL,              -- 5-field, internal/cron
  tz            TEXT NOT NULL DEFAULT '',   -- IANA zone; '' = the daemon's local zone
  enabled       INTEGER NOT NULL DEFAULT 1,
  last_fired_at TEXT,                       -- RFC3339 UTC, drives due + catch-up
  position      INTEGER NOT NULL DEFAULT 0,
  created_at    TEXT NOT NULL,
  updated_at    TEXT NOT NULL
);
CREATE INDEX idx_automation_schedules_by_automation ON automation_schedules(automation_id, position, created_at);

-- Every existing cron becomes the automation's first schedule and keeps
-- its last fire, so the migration itself catches nothing up.
INSERT INTO automation_schedules (id, automation_id, label, cron, tz, enabled, last_fired_at, position, created_at, updated_at)
  SELECT 'sch-' || id, id, '', cron, '', 1, last_fired_at, 0, created_at, updated_at
  FROM automations WHERE cron IS NOT NULL AND cron != '';

ALTER TABLE automations DROP COLUMN cron;
ALTER TABLE automations DROP COLUMN last_fired_at;

-- Which rule fired; NULL for webhook, Run now and rows older than this.
ALTER TABLE automation_runs ADD COLUMN schedule_id TEXT;
