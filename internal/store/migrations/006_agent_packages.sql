-- Packages attached to one agent (every session). Passed as pi -e on start.
-- Not user/project settings.json (ADR-0010).
ALTER TABLE agents ADD COLUMN packages TEXT NOT NULL DEFAULT '[]';
