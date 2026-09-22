-- ADR-0184 slice 3: a sign-in terminal is internal — no agent, no grant, off
-- the sidebar, closed by the server. The credential flow named them
-- "<CLI> sign-in"; those rows become kind 'signin'.
ALTER TABLE terminals ADD COLUMN kind TEXT NOT NULL DEFAULT '';
UPDATE terminals SET kind = 'signin'
WHERE name LIKE '% sign-in'
AND id IN (SELECT terminal_id FROM terminal_launches)
AND id NOT IN (SELECT terminal_id FROM agents WHERE terminal_id IS NOT NULL);
