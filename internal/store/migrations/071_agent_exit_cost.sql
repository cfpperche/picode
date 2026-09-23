-- ADR-0194 follow-up: what the removed agent's sessions cost, as JSON
-- (recorded, estimated part, unpriced turns, tokens, turns, which sessions).
-- NULL = not measured (a CLI whose sessions are not files, or no session
-- known) — never a zero.
ALTER TABLE agent_exits ADD COLUMN cost TEXT;
