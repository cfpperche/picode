-- 048: Work browser history (desktop browser plan, slice 3). One row per
-- navigation the app saw; the newest row for a URL is updated in place when
-- the same page is re-reported, so revisits bump rather than pile up.
CREATE TABLE browser_history (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  url TEXT NOT NULL,
  title TEXT NOT NULL DEFAULT '',
  host TEXT NOT NULL DEFAULT '',
  typed INTEGER NOT NULL DEFAULT 0,
  visited_at TEXT NOT NULL
);
CREATE INDEX idx_browser_history_visited ON browser_history(visited_at DESC);
