-- ADR-0104: a connection capability is separate from its durable inbox.
CREATE TABLE peer_connections (
    id TEXT PRIMARY KEY,
    agent_id TEXT REFERENCES agents(id) ON DELETE CASCADE,
    terminal_id TEXT REFERENCES terminals(id) ON DELETE CASCADE,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    session_key TEXT NOT NULL,
    cli TEXT NOT NULL,
    label TEXT NOT NULL,
    token_hash TEXT NOT NULL UNIQUE,
    created_at TEXT NOT NULL,
    revoked_at TEXT,
    CHECK ((agent_id IS NULL) != (terminal_id IS NULL))
);
CREATE INDEX peer_owner_agent ON peer_connections(agent_id);
CREATE INDEX peer_owner_terminal ON peer_connections(terminal_id);
CREATE TABLE peer_messages (
    seq INTEGER PRIMARY KEY AUTOINCREMENT,
    id TEXT NOT NULL UNIQUE,
    sender_id TEXT NOT NULL REFERENCES peer_connections(id) ON DELETE CASCADE,
    recipient_id TEXT NOT NULL REFERENCES peer_connections(id) ON DELETE CASCADE,
    request_id TEXT NOT NULL,
    body TEXT NOT NULL,
    reply_to TEXT,
    created_at TEXT NOT NULL,
    acked_at TEXT,
    UNIQUE(sender_id, request_id)
);
CREATE INDEX peer_inbox ON peer_messages(recipient_id, acked_at, seq);
CREATE INDEX peer_sent ON peer_messages(sender_id, seq);
