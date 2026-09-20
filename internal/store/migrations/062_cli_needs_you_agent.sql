-- ADR-0160 Fatia E: a guest's "needs you" Inbox item belongs to the agent,
-- not the terminal. Open items rekey onto the agent bound to their
-- terminal; items whose terminal has no bound agent name a process that is
-- gone, so they close.
UPDATE inbox_items SET source_kind = 'agent',
  source_id = (SELECT a.id FROM agents a WHERE a.terminal_id = inbox_items.source_id)
WHERE reason = 'cli-needs-you' AND source_kind = 'terminal' AND state != 'done'
  AND source_id IN (SELECT t.id FROM terminals t JOIN agents a ON a.terminal_id = t.id);
UPDATE inbox_items SET state = 'done'
WHERE reason = 'cli-needs-you' AND source_kind = 'terminal' AND state != 'done';
