-- ADR-0160 Fatia B: a guest agent's interactive process is a terminal.
-- Pi agents keep a null terminal_id (tmux session named from the agent id).
ALTER TABLE agents ADD COLUMN terminal_id TEXT REFERENCES terminals(id) ON DELETE SET NULL;
CREATE UNIQUE INDEX agents_terminal_id ON agents(terminal_id) WHERE terminal_id IS NOT NULL;
