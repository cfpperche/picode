-- ADR-0160 Fatia C: leftover ADR-0159 bindings become agents, then the
-- table goes. A terminal already on an agents row (Fatia B) is skipped.
INSERT INTO agents (id, workspace_id, name, created_at, last_status, cli, terminal_id)
SELECT m.id, m.workspace_id, m.name, m.created_at, 'never_started', m.cli, m.terminal_id
FROM managed_clis m
WHERE NOT EXISTS (
    SELECT 1 FROM agents a
    WHERE a.terminal_id IS NOT NULL AND a.terminal_id = m.terminal_id
)
AND NOT EXISTS (
    SELECT 1 FROM agents a WHERE a.id = m.id
);
DROP TABLE managed_clis;
