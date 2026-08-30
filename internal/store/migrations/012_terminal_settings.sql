-- Terminal behaviour (tmux options), per ADR-0024. One row holds the global
-- defaults under the reserved scope 'global'; every other row is keyed by a
-- terminal id and holds ONLY the fields that terminal overrides. A missing
-- row and an empty object mean the same thing: inherit everything.
--
-- The settings column is a JSON object rather than a column per flag so that
-- adding a flag is a registry change in Go, not a migration.
CREATE TABLE terminal_settings (
  scope      TEXT PRIMARY KEY,
  settings   TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
