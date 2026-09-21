-- Reserved: 065 held the delivery-observer binding (D2), removed from the
-- product on 2026-09-21 (ADR-0177). No code reads this table, and the file
-- stays so the ledger stays monotonic — reusing the number would make a future
-- migration 065 silently skip every database that already applied this one.
-- ADR-0170 D2: one local observer per workspace, bound to a resolved repository key.
CREATE TABLE delivery_observers (
    workspace_id TEXT PRIMARY KEY REFERENCES workspaces(id) ON DELETE CASCADE,
    repo TEXT NOT NULL,
    observer TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE INDEX delivery_observers_repo ON delivery_observers(repo);
