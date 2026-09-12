-- ADR-0118: Canvas replaces Matrix — one engine, one name. Two things
-- happen here, in one transaction, and only the first cannot be undone.
--
-- **What converts (irreversible).** ADR-0113 gave a matrix two layout
-- engines and a `mode` column to say which one a rectangle was written in.
-- Grid mode is gone, so every panel of a matrix still in `grid` is rewritten
-- in canvas units once, with the factors ADR-0113 defined and tested: a grid
-- cell is 8 canvas units wide and 3 tall (colWidth / 8, rowHeight 24 px / 8).
-- The result is clamped to the canvas minimums (w >= 32, h >= 28), which is
-- why a panel at the old 8-row minimum lands 28 units tall and not 24 — the
-- same clamp the switch transform applied. A matrix already in `canvas` is
-- untouched: its rectangles already mean canvas units. Going back would mean
-- restoring the column and packing every panel into 12 columns again, so
-- this half is one-way in practice; nothing in the code path reverses it.
--
-- **What is renamed (mechanical).** matrices -> canvases, matrix_panels ->
-- canvas_panels, matrix_edges -> canvas_edges, with the matrix_id column on
-- both children and the two endpoint indexes. SQLite's ALTER TABLE ... RENAME
-- TO carries the foreign keys that point at the renamed table and the indexes
-- defined on it (legacy_alter_table is off by default, so referencing FK
-- clauses in other tables are rewritten too); TestCanvasRenameKeepsKeysAndIndexes
-- proves the cascade still fires rather than trusting that. The two named
-- indexes are dropped and recreated because a rename cannot rename an index.
--
-- **What is dropped.** matrices.mode. `compact` stays: it is still the
-- column UpdateCanvas patches, even though a canvas is free placement and
-- nothing compacts it today.

UPDATE matrix_panels
SET x = MAX(MIN(x * 8, 100000), -100000),
    y = MAX(MIN(y * 3, 100000), -100000),
    w = MIN(MAX(w * 8, 32), 4096),
    h = MIN(MAX(h * 3, 28), 4096)
WHERE matrix_id IN (SELECT id FROM matrices WHERE mode = 'grid');

ALTER TABLE matrices RENAME TO canvases;
ALTER TABLE matrix_panels RENAME TO canvas_panels;
ALTER TABLE matrix_edges RENAME TO canvas_edges;
ALTER TABLE canvas_panels RENAME COLUMN matrix_id TO canvas_id;
ALTER TABLE canvas_edges RENAME COLUMN matrix_id TO canvas_id;

DROP INDEX matrix_edge_a;
DROP INDEX matrix_edge_b;
CREATE INDEX canvas_edge_a ON canvas_edges(a_panel);
CREATE INDEX canvas_edge_b ON canvas_edges(b_panel);

ALTER TABLE canvases DROP COLUMN mode;
