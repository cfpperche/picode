-- Pins list v2 (docs/plans/pins-v2.md slice 4): keep on top and archive.
-- "starred" rather than "pinned" — a pinned pin reads twice.
ALTER TABLE pins ADD COLUMN starred INTEGER NOT NULL DEFAULT 0;
ALTER TABLE pins ADD COLUMN archived_at TEXT;
