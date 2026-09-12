-- ADR-0116: an edge is the owner's recorded intent that two sessions may
-- exchange messages. It grants exactly ADR-0104's mailbox contact and
-- nothing else — never a transcript, a scrollback or a session file.
--
-- The pair is stored **ordered** (`a_panel < b_panel`, the store sorts
-- before writing): the same edge drawn in either direction then collides on
-- UNIQUE(matrix_id, a_panel, b_panel), so an undirected edge needs no second
-- row and no second index. A drawn arrow would promise a one-way restriction
-- the mailbox cannot keep (ADR-0116 §6), so there is no direction column.
--
-- The endpoints are **panels**, not sessions. A panel already carries its
-- own (kind, ref); re-pointing a panel is a new binding that must be drawn
-- again, and the grant follows the line the owner can see rather than an id
-- it once resolved to. matrix_panels has no foreign key to agents or
-- terminals on purpose (041), so an edge's end can outlive its target — the
-- contact union resolves (kind, ref) against peer_connections on every read
-- and a target deleted from the fleet takes its connection with it.
--
-- **Foreign keys: yes, at both levels, ON DELETE CASCADE.** matrix_panels(id)
-- is a primary key, so SQLite accepts it, and the honest reason is the
-- grant: an edge to a dead panel must not grant, and a cascade makes that
-- true in the schema instead of in every future reader. Deleting the matrix,
-- or either panel, removes the edge in the same transaction that removed the
-- row. Nothing outlives the line you can see (ADR-0116 §4). The unique index
-- leads with matrix_id, so it serves the per-matrix read and the cascade from
-- matrices; the cascade from matrix_panels and the contact union both look a
-- panel id up as an *endpoint*, which that index cannot serve, so each end
-- gets its own index. The FK does not say the two panels are on the same
-- matrix — that is AddMatrixEdge's refusal, checked in its transaction.
--
-- Cap: 1000 edges per matrix (MaxMatrixEdges), in the shape of the 500-panel
-- cap; the refusal names it.
CREATE TABLE matrix_edges (
    id TEXT PRIMARY KEY,
    matrix_id TEXT NOT NULL REFERENCES matrices(id) ON DELETE CASCADE,
    a_panel TEXT NOT NULL REFERENCES matrix_panels(id) ON DELETE CASCADE,
    b_panel TEXT NOT NULL REFERENCES matrix_panels(id) ON DELETE CASCADE,
    created_at TEXT NOT NULL,
    UNIQUE(matrix_id, a_panel, b_panel)
);
CREATE INDEX matrix_edge_a ON matrix_edges(a_panel);
CREATE INDEX matrix_edge_b ON matrix_edges(b_panel);
