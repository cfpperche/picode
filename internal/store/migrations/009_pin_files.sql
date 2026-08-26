CREATE TABLE pin_files (
  id TEXT PRIMARY KEY,
  pin_id TEXT NOT NULL REFERENCES pins(id) ON DELETE CASCADE,
  kind TEXT NOT NULL,
  name TEXT NOT NULL,
  mime TEXT NOT NULL,
  size INTEGER NOT NULL,
  created_at TEXT NOT NULL
);
CREATE INDEX pin_files_pin ON pin_files(pin_id);
