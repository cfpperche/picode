CREATE TABLE workspaces (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    path       TEXT NOT NULL UNIQUE,
    created_at TEXT NOT NULL
);

-- One pi instance in a workspace. v1 guarantees a "default" agent per
-- workspace (auto-created); the M3 wizard creates configured siblings.
-- Config columns are nullable = inherit pi/user defaults.
CREATE TABLE agents (
    id              TEXT PRIMARY KEY,
    workspace_id    TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    created_at      TEXT NOT NULL,
    provider        TEXT,
    model           TEXT,
    thinking        TEXT,
    extra_prompt    TEXT,
    last_started_at TEXT,
    last_status     TEXT NOT NULL DEFAULT 'never_started',
    last_status_at  TEXT
);
CREATE INDEX idx_agents_workspace ON agents(workspace_id);

-- Delivery queue. kind mirrors pi's RPC semantics (prompt/steer/follow_up).
-- status machine: queued -> delivering -> delivered | failed ; cancelled.
CREATE TABLE tasks (
    id           TEXT PRIMARY KEY,
    agent_id     TEXT NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    kind         TEXT NOT NULL CHECK (kind IN ('prompt','steer','follow_up')),
    payload      TEXT NOT NULL,
    source       TEXT NOT NULL DEFAULT 'user',
    status       TEXT NOT NULL DEFAULT 'queued'
                 CHECK (status IN ('queued','delivering','delivered','failed','cancelled')),
    attempts     INTEGER NOT NULL DEFAULT 0,
    last_error   TEXT,
    created_at   TEXT NOT NULL,
    delivered_at TEXT
);
CREATE INDEX idx_tasks_agent_status ON tasks(agent_id, status);
CREATE INDEX idx_tasks_created ON tasks(created_at);

-- Broker inbox (reserved for M4 inter-agent messaging). A delivered message
-- becomes a follow_up task on the target agent; this table keeps the
-- human-readable communication history.
CREATE TABLE messages (
    id            TEXT PRIMARY KEY,
    from_agent_id TEXT REFERENCES agents(id) ON DELETE SET NULL,
    to_agent_id   TEXT NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    body          TEXT NOT NULL,
    created_at    TEXT NOT NULL,
    read_at       TEXT
);
CREATE INDEX idx_messages_inbox ON messages(to_agent_id, read_at);

-- Append-only orchestration audit (started/stopped/task/config events).
-- NOT a chat log: session content lives in pi's own JSONL files (ADR-0005).
CREATE TABLE events (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    agent_id     TEXT REFERENCES agents(id) ON DELETE CASCADE,
    workspace_id TEXT REFERENCES workspaces(id) ON DELETE CASCADE,
    type         TEXT NOT NULL,
    data         TEXT NOT NULL DEFAULT '{}',
    created_at   TEXT NOT NULL
);
CREATE INDEX idx_events_recent ON events(created_at DESC);
CREATE INDEX idx_events_agent ON events(agent_id, created_at DESC);

CREATE TABLE settings (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
