-- ADR-0108: a matrix is a named grid; each panel is one row — a slot bound
-- to an agent or a terminal by (kind, ref) — never one JSON blob, so a drag
-- rewrites only the rows that moved and the unique index refuses a duplicate
-- binding. No foreign key to agents or terminals on purpose: the store stays
-- ignorant of the binding, a deleted target leaves its panel and the UI
-- shows it gone. The cascade from matrices runs on PRAGMA foreign_keys=1
-- (set by the DSN); the unique index leads with matrix_id, so it also
-- serves the per-matrix read and the cascade — no second index.
-- (040 is taken by feat/unified-messages, hence 041.)
CREATE TABLE matrices (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    compact TEXT NOT NULL DEFAULT 'vertical',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE TABLE matrix_panels (
    id TEXT PRIMARY KEY,
    matrix_id TEXT NOT NULL REFERENCES matrices(id) ON DELETE CASCADE,
    kind TEXT NOT NULL,
    ref TEXT NOT NULL,
    x INTEGER NOT NULL,
    y INTEGER NOT NULL,
    w INTEGER NOT NULL,
    h INTEGER NOT NULL,
    created_at TEXT NOT NULL,
    UNIQUE(matrix_id, kind, ref)
);
