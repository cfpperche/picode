-- ADR-0173: the sidebar order is a stored position, not a sort.
-- Rows that already exist keep the order the sidebar showed on the day
-- this landed: workspaces by name, agents by created_at (free agents by
-- name, which is what that tab showed), terminals by name.

ALTER TABLE workspaces ADD COLUMN position INTEGER NOT NULL DEFAULT 0;
ALTER TABLE agents ADD COLUMN position INTEGER NOT NULL DEFAULT 0;
ALTER TABLE terminals ADD COLUMN position INTEGER NOT NULL DEFAULT 0;

WITH ranked AS (
  SELECT id, ROW_NUMBER() OVER (ORDER BY name, id) - 1 AS pos
  FROM workspaces
  WHERE id != 'ws_free'
)
UPDATE workspaces
SET position = (SELECT pos FROM ranked WHERE ranked.id = workspaces.id)
WHERE id != 'ws_free';

WITH ranked AS (
  SELECT id, ROW_NUMBER() OVER (PARTITION BY workspace_id ORDER BY created_at, id) - 1 AS pos
  FROM agents
)
UPDATE agents
SET position = (SELECT pos FROM ranked WHERE ranked.id = agents.id);

WITH ranked AS (
  SELECT id, ROW_NUMBER() OVER (ORDER BY name COLLATE NOCASE, id) - 1 AS pos
  FROM agents
  WHERE workspace_id = 'ws_free'
)
UPDATE agents
SET position = (SELECT pos FROM ranked WHERE ranked.id = agents.id)
WHERE workspace_id = 'ws_free';

WITH ranked AS (
  SELECT id, ROW_NUMBER() OVER (PARTITION BY workspace_id ORDER BY name COLLATE NOCASE, id) - 1 AS pos
  FROM terminals
)
UPDATE terminals
SET position = (SELECT pos FROM ranked WHERE ranked.id = terminals.id);
