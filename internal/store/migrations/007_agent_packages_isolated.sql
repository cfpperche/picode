-- When set, this agent starts with --no-extensions/skills/prompts/themes
-- and only its own packages (-e). Machine and workspace packages stay out.
ALTER TABLE agents ADD COLUMN packages_isolated INTEGER NOT NULL DEFAULT 0;
