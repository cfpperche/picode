-- Per-agent working directory (ADR-0011). Workspace agents leave this NULL
-- and inherit the folder. Unbound agents get ~/.picode/work/<name>/ — never $HOME.
ALTER TABLE agents ADD COLUMN work_path TEXT;
