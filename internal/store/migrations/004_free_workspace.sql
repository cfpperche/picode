-- Reserved workspace for unbound ("free") agents. Hidden from the
-- workspace list. Path is a sentinel, not a real folder (ADR-0011).
INSERT OR IGNORE INTO workspaces (id, name, path, created_at)
VALUES ('ws_free', 'Free', '__picode_free__', '2026-08-25T00:00:00Z');
