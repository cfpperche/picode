-- ADR-0171: declarations, not evidence or execution authority.
CREATE TABLE delivery_intents (
    seq INTEGER PRIMARY KEY AUTOINCREMENT,
    id TEXT NOT NULL UNIQUE,
    repo TEXT NOT NULL,
    principal TEXT NOT NULL,
    body TEXT NOT NULL
);
CREATE INDEX delivery_intents_repo ON delivery_intents(repo, seq);
CREATE TABLE delivery_requests (
    repo TEXT NOT NULL,
    principal TEXT NOT NULL,
    request_id TEXT NOT NULL,
    payload TEXT NOT NULL,
    result TEXT NOT NULL,
    PRIMARY KEY(repo, principal, request_id)
);
