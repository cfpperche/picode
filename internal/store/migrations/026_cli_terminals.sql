CREATE TABLE cli_configs (
    id TEXT PRIMARY KEY,
    config TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE TABLE terminal_launches (
    terminal_id TEXT PRIMARY KEY REFERENCES terminals(id) ON DELETE CASCADE,
    cli TEXT NOT NULL,
    overrides TEXT NOT NULL,
    applied TEXT,
    updated_at TEXT NOT NULL
);
