-- ADR-0184 slice 2: a CLI launched for a user always runs as an agent.
-- Every terminal that launches a catalog CLI and no agent binds becomes an
-- agent bound to it — except sign-in terminals, which stay internal (they
-- are named "<CLI> sign-in" by the credential flow that owns them).
INSERT INTO agents (id, workspace_id, name, created_at, last_status, cli, terminal_id, work_path)
SELECT 'a-' || t.id, t.workspace_id, t.name, t.created_at, 'never_started', l.cli, t.id,
       CASE WHEN t.workspace_id = 'ws_free' THEN t.cwd
            WHEN t.cwd = (SELECT w.path FROM workspaces w WHERE w.id = t.workspace_id) THEN NULL
            ELSE t.cwd END
FROM terminals t
JOIN terminal_launches l ON l.terminal_id = t.id
WHERE t.name NOT LIKE '% sign-in'
AND NOT EXISTS (SELECT 1 FROM agents a WHERE a.terminal_id = t.id)
AND NOT EXISTS (SELECT 1 FROM agents a WHERE a.id = 'a-' || t.id);

-- A terminal's grants follow it onto its agent (an agent's own grant wins),
-- then every terminal grant goes: shells and sign-in terminals hold none.
INSERT OR IGNORE INTO settings (key, value)
SELECT 'browser.policy.' || a.id, s.value
FROM settings s JOIN agents a ON a.terminal_id IS NOT NULL AND s.key = 'browser.policy.term:' || a.terminal_id;
INSERT OR IGNORE INTO settings (key, value)
SELECT 'computer.policy.' || a.id, s.value
FROM settings s JOIN agents a ON a.terminal_id IS NOT NULL AND s.key = 'computer.policy.term:' || a.terminal_id;
DELETE FROM settings WHERE key LIKE 'browser.policy.term:%' OR key LIKE 'computer.policy.term:%';
