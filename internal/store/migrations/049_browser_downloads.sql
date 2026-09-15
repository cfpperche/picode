-- 049: Work browser downloads (desktop browser plan, slice 3). One row per
-- download the shell saw. The shell reports the start (url, path, size) and
-- the outcome (completed/interrupted) by path, so a row is updated in place
-- rather than piling up.
CREATE TABLE browser_downloads (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  url TEXT NOT NULL DEFAULT '',
  name TEXT NOT NULL DEFAULT '',
  path TEXT NOT NULL DEFAULT '',
  host TEXT NOT NULL DEFAULT '',
  total INTEGER NOT NULL DEFAULT 0,
  received INTEGER NOT NULL DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'started',
  started_at TEXT NOT NULL
);
CREATE INDEX idx_browser_downloads_started ON browser_downloads(started_at DESC);
CREATE UNIQUE INDEX idx_browser_downloads_path ON browser_downloads(path) WHERE path != '';
