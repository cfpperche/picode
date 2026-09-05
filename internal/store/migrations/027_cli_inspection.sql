CREATE TABLE cli_profiles (
    id TEXT PRIMARY KEY,
    cli TEXT NOT NULL,
    name TEXT NOT NULL,
    config TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE TABLE cli_checks (
    cli TEXT PRIMARY KEY,
    diagnostic TEXT NOT NULL
);
ALTER TABLE terminal_launches ADD COLUMN attempt TEXT;
