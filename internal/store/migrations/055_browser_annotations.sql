CREATE TABLE IF NOT EXISTS browser_annotations (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    terminal_id  TEXT NOT NULL DEFAULT '',
    workspace_id TEXT NOT NULL DEFAULT '',
    url          TEXT NOT NULL DEFAULT '',
    title        TEXT NOT NULL DEFAULT '',
    host         TEXT NOT NULL DEFAULT '',
    selector     TEXT NOT NULL DEFAULT '',
    comment      TEXT NOT NULL DEFAULT '',
    dom          TEXT NOT NULL DEFAULT '',
    css          TEXT NOT NULL DEFAULT '',
    shot         TEXT NOT NULL DEFAULT '',
    note         TEXT NOT NULL DEFAULT '',
    created_at   TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_browser_annotations_created ON browser_annotations(created_at DESC);
