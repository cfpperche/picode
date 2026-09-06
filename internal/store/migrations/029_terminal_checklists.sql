-- Internal checklists for coding-CLI terminals (ADR-0055 extended to
-- Agent CLI terminals). The pi-checklist extension running inside a PiCode
-- terminal publishes under PICODE_TERM_ID; one row per terminal, same shape
-- as agent_checklists. Rows die with the terminal.
CREATE TABLE terminal_checklists (
  terminal_id TEXT PRIMARY KEY,
  session_id  TEXT NOT NULL DEFAULT '',
  items       TEXT NOT NULL, -- JSON array of {text, status}
  absent      INTEGER NOT NULL DEFAULT 0, -- the turn ended without one, or a change was refused
  updated_at  TEXT NOT NULL
);
