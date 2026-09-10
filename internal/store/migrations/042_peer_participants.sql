-- ADR-0110: participation outlives a conversation, credentials never do.
CREATE TABLE peer_participants (
    owner_key TEXT PRIMARY KEY,
    agent_id TEXT REFERENCES agents(id) ON DELETE CASCADE,
    terminal_id TEXT REFERENCES terminals(id) ON DELETE CASCADE,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    cli TEXT NOT NULL,
    enabled INTEGER NOT NULL DEFAULT 0,
    revision INTEGER NOT NULL DEFAULT 1,
    phase TEXT NOT NULL DEFAULT 'disabled',
    problem TEXT NOT NULL DEFAULT '',
    applied_session TEXT NOT NULL DEFAULT '',
    applied_connection TEXT NOT NULL DEFAULT '',
    CHECK ((agent_id IS NULL) != (terminal_id IS NULL))
);
CREATE TABLE peer_checks (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    sender_id TEXT NOT NULL REFERENCES peer_connections(id) ON DELETE CASCADE,
    recipient_id TEXT NOT NULL REFERENCES peer_connections(id) ON DELETE CASCADE,
    phase TEXT NOT NULL DEFAULT 'pending',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
