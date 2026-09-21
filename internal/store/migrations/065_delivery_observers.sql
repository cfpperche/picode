-- ADR-0170 D2: one local observer per workspace, bound to a resolved repository key.
CREATE TABLE delivery_observers (
    workspace_id TEXT PRIMARY KEY REFERENCES workspaces(id) ON DELETE CASCADE,
    repo TEXT NOT NULL,
    observer TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE INDEX delivery_observers_repo ON delivery_observers(repo);
