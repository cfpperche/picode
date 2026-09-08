-- pin_files.updated_at: an edited sketch keeps its id, so the browser needs
-- a value that changes to bust its one-hour cache of the preview bytes.
ALTER TABLE pin_files ADD COLUMN updated_at TEXT NOT NULL DEFAULT '';
UPDATE pin_files SET updated_at = created_at WHERE updated_at = '';
