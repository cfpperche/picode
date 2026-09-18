CREATE TABLE webapps (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    url TEXT NOT NULL UNIQUE,
    icon BLOB,
    icon_mime TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    CHECK (length(name) BETWEEN 1 AND 800),
    CHECK (length(url) BETWEEN 1 AND 8192),
    CHECK (length(icon) <= 262144)
);
