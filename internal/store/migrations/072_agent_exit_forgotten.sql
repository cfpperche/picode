-- ADR-0205: the agent history is the exits whose transcript still exists.
-- forgotten_at is the person's "remove from history": the exit stays in
-- the catalog (Outcomes) and the files are untouched. NULL = listed.
ALTER TABLE agent_exits ADD COLUMN forgotten_at TEXT;
