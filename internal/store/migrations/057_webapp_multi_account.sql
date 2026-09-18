-- Multi-account (ADR-0153 amendment): the same address may be installed
-- twice as two independent accounts, so url loses its UNIQUE constraint.
-- SQLite cannot drop an inline autoindex; the table is rebuilt.
CREATE TABLE webapps_new (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    url TEXT NOT NULL,
    start_url TEXT NOT NULL DEFAULT '',
    scope TEXT NOT NULL DEFAULT '',
    display TEXT NOT NULL DEFAULT '',
    theme_color TEXT NOT NULL DEFAULT '',
    partitioned INTEGER NOT NULL DEFAULT 0,
    icon BLOB,
    icon_mime TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    CHECK (length(name) BETWEEN 1 AND 800),
    CHECK (length(url) BETWEEN 1 AND 8192),
    CHECK (length(icon) <= 262144)
);
INSERT INTO webapps_new (id, name, url, start_url, scope, display, theme_color, partitioned, icon, icon_mime, created_at)
    SELECT id, name, url, start_url, scope, display, theme_color, partitioned, icon, icon_mime, created_at FROM webapps;
DROP TABLE webapps;
ALTER TABLE webapps_new RENAME TO webapps;
CREATE INDEX webapps_url ON webapps(url);
