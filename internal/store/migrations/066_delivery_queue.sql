-- ADR-0182: the integration queue. An entry is a durable *intent* that the
-- reviewed revision be integrated into its target, never an execution result:
-- eligibility (branch head, target movement, evidence) is derived from Git and
-- receipts at read time, and the row records only what the queue itself did.
CREATE TABLE delivery_queue (
    seq INTEGER PRIMARY KEY AUTOINCREMENT,
    id TEXT NOT NULL UNIQUE,
    repo TEXT NOT NULL,
    delivery_id TEXT NOT NULL,
    actor TEXT NOT NULL,
    state TEXT NOT NULL,
    body TEXT NOT NULL
);
CREATE INDEX delivery_queue_repo ON delivery_queue(repo, seq);
CREATE INDEX delivery_queue_active ON delivery_queue(delivery_id, state);
CREATE TABLE delivery_queue_requests (
    repo TEXT NOT NULL,
    actor TEXT NOT NULL,
    request_id TEXT NOT NULL,
    payload TEXT NOT NULL,
    result TEXT NOT NULL,
    PRIMARY KEY(repo, actor, request_id)
);
