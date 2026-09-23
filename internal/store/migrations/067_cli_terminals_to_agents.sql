-- ADR-0184 slice 2: a CLI launched for a user always runs as an agent.
-- Every terminal that launches a catalog CLI and no agent binds becomes an
-- agent bound to it — except sign-in terminals, which stay internal (the
-- credential flow names them exactly "<CLI> sign-in"; GLOB is
-- case-sensitive, so a person's "… Sign-In" terminal is not one). A
-- terminal whose workspace row is gone is left alone: agents need one.
-- A Pi agent owns its session (it is reserved on its launch): the pinned
-- conversation, or a lone `--session <path>` launch argument, moves onto
-- session_path, and that argument leaves the launch.
INSERT INTO agents (id, workspace_id, name, created_at, last_status, cli, terminal_id, work_path, session_path)
SELECT 'a-' || t.id, t.workspace_id, t.name, t.created_at, 'never_started', l.cli, t.id,
       CASE WHEN t.workspace_id = 'ws_free' THEN t.cwd
            WHEN t.cwd = w.path THEN NULL
            ELSE t.cwd END,
       CASE WHEN l.cli <> 'pi' THEN NULL
            WHEN json_valid(l.overrides) AND json_extract(l.overrides, '$.args[0]') = '--session'
                 AND json_array_length(json_extract(l.overrides, '$.args')) = 2
                 THEN json_extract(l.overrides, '$.args[1]')
            WHEN json_valid(l.last_session) THEN json_extract(l.last_session, '$.path')
            ELSE NULL END
FROM terminals t
JOIN terminal_launches l ON l.terminal_id = t.id
JOIN workspaces w ON w.id = t.workspace_id
WHERE t.name NOT GLOB '* sign-in'
AND NOT EXISTS (SELECT 1 FROM agents a WHERE a.terminal_id = t.id)
AND NOT EXISTS (SELECT 1 FROM agents a WHERE a.id = 'a-' || t.id);

UPDATE terminal_launches SET overrides = json_remove(overrides, '$.args')
WHERE cli = 'pi' AND json_valid(overrides)
AND json_extract(overrides, '$.args[0]') = '--session'
AND json_array_length(json_extract(overrides, '$.args')) = 2
AND terminal_id IN (SELECT terminal_id FROM agents WHERE id = 'a-' || terminal_id);

-- A terminal's grants follow it onto its agent (an agent's own grant wins),
-- then every terminal grant goes: shells and sign-in terminals hold none.
INSERT OR IGNORE INTO settings (key, value)
SELECT 'browser.policy.' || a.id, s.value
FROM settings s JOIN agents a ON a.terminal_id IS NOT NULL AND s.key = 'browser.policy.term:' || a.terminal_id;
INSERT OR IGNORE INTO settings (key, value)
SELECT 'computer.policy.' || a.id, s.value
FROM settings s JOIN agents a ON a.terminal_id IS NOT NULL AND s.key = 'computer.policy.term:' || a.terminal_id;
DELETE FROM settings WHERE key LIKE 'browser.policy.term:%' OR key LIKE 'computer.policy.term:%';

-- Deliveries and queue entries a terminal owned are its agent's now: the
-- same CLI calls in as the agent (ADR-0184), and must still reach them.
UPDATE delivery_intents
SET principal = (SELECT a.id FROM agents a WHERE 'term:' || a.terminal_id = delivery_intents.principal),
    body = CASE WHEN json_valid(body) AND json_extract(body, '$.principal') IS NOT NULL
                THEN json_set(body, '$.principal', (SELECT a.id FROM agents a WHERE 'term:' || a.terminal_id = delivery_intents.principal))
                ELSE body END
WHERE principal IN (SELECT 'term:' || terminal_id FROM agents WHERE terminal_id IS NOT NULL);
UPDATE OR IGNORE delivery_requests
SET principal = (SELECT a.id FROM agents a WHERE 'term:' || a.terminal_id = delivery_requests.principal)
WHERE principal IN (SELECT 'term:' || terminal_id FROM agents WHERE terminal_id IS NOT NULL);
UPDATE delivery_queue
SET actor = (SELECT a.id FROM agents a WHERE 'term:' || a.terminal_id = delivery_queue.actor)
WHERE actor IN (SELECT 'term:' || terminal_id FROM agents WHERE terminal_id IS NOT NULL);
UPDATE OR IGNORE delivery_queue_requests
SET actor = (SELECT a.id FROM agents a WHERE 'term:' || a.terminal_id = delivery_queue_requests.actor)
WHERE actor IN (SELECT 'term:' || terminal_id FROM agents WHERE terminal_id IS NOT NULL);
