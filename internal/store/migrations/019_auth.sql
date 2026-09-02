-- Authentication (ADR-0049): browser and token principals, and one-time
-- pairing codes. Secrets are stored as sha256 hex, never in clear.
CREATE TABLE auth_sessions (
  id           TEXT PRIMARY KEY,
  secret_hash  TEXT NOT NULL UNIQUE,
  kind         TEXT NOT NULL CHECK (kind IN ('browser','token')),
  device_id    TEXT NOT NULL DEFAULT '',
  label        TEXT NOT NULL DEFAULT '',
  ip           TEXT NOT NULL DEFAULT '',
  created_at   TEXT NOT NULL,
  last_seen_at TEXT NOT NULL,
  expires_at   TEXT,
  revoked_at   TEXT
);
CREATE TABLE auth_pairings (
  code_hash  TEXT PRIMARY KEY,
  created_by TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  expires_at TEXT NOT NULL,
  used_at    TEXT
);
