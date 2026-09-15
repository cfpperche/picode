-- 050: Site permissions for the work browser (slice 3, Browser permissions).
-- One row per (origin, kind): what the user (or the policy) decided. The
-- shell reports each decision as it is made; the Site settings dialog reads,
-- changes and prunes them.
CREATE TABLE browser_permissions (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  origin TEXT NOT NULL,
  kind TEXT NOT NULL,
  decision TEXT NOT NULL,
  decided_at TEXT NOT NULL
);
CREATE UNIQUE INDEX idx_browser_permissions_site ON browser_permissions(origin, kind);
CREATE INDEX idx_browser_permissions_kind ON browser_permissions(kind);
