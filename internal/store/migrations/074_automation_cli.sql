-- The CLI an automation's start runs use (ADR-0217): pi keeps the managed
-- runtime; claude-code, codex, grok and hermes run through their own TUI.
ALTER TABLE automations ADD COLUMN cli TEXT NOT NULL DEFAULT 'pi';
