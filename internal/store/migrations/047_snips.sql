-- 047: PiCode Snippets (docs/architecture/snippets.md, ADR-0130).
-- Named reusable templates. Not internal/snippet (conversation fence runner).
CREATE TABLE snips (
  id TEXT PRIMARY KEY,
  slug TEXT NOT NULL UNIQUE,
  title TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  kind TEXT NOT NULL CHECK (kind IN ('prompt', 'shell')),
  body TEXT NOT NULL DEFAULT '',
  placeholders TEXT NOT NULL DEFAULT '[]',
  tags TEXT NOT NULL DEFAULT '[]',
  starred INTEGER NOT NULL DEFAULT 0,
  archived_at TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE INDEX snips_list ON snips (starred DESC, updated_at DESC);
