-- Skills for one agent alone (ADR-0196 slice 4): a JSON list of
-- {name, digest, source, dir}. The content lives in PiCode's digest-addressed
-- cache (<data>/skills/cache/<digest>/<name>); each CLI receives it at launch
-- through its own flag (Pi --skill, Omp --config, Claude Code --plugin-dir).
ALTER TABLE agents ADD COLUMN skills TEXT NOT NULL DEFAULT '[]';
