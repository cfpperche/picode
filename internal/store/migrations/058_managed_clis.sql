-- ADR-0159: a workspace may bind one Agent CLI terminal as a managed
-- principal. The terminal, launch settings and vendor files stay where
-- ADR-0069 put them. This row is the identity seam, not a second runtime.
CREATE TABLE managed_clis (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    cli TEXT NOT NULL,
    terminal_id TEXT NOT NULL UNIQUE REFERENCES terminals(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    created_at TEXT NOT NULL,
    CHECK (length(name) BETWEEN 1 AND 200),
    CHECK (length(cli) BETWEEN 1 AND 64)
);
CREATE INDEX managed_clis_workspace ON managed_clis(workspace_id);
