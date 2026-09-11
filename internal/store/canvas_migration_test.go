package store

import (
	"strings"
	"testing"
)

// ---- migration 045 (ADR-0118) --------------------------------------------
//
// Two properties, one test each, both against the **real embedded
// migration**: the conversion (a grid board is rewritten in canvas units
// once, a canvas board is untouched) and the rename (SQLite carries the
// foreign keys and the unique index through ALTER TABLE … RENAME TO, so the
// cascades still fire under the new names).

// undo045 rebuilds the pre-045 shape on an already-migrated database: the
// three tables under their old names, `matrix_id` back, the two old index
// names, and the `mode` column with the default 043 gave it. Dropping the
// bookkeeping row is what lets Store.migrate run 045 again for real — the
// point of the test is that the shipped SQL does the conversion, not that a
// copy of its arithmetic in Go does.
func undo045(t *testing.T, s *Store) {
	t.Helper()
	for _, stmt := range []string{
		`ALTER TABLE canvases RENAME TO matrices`,
		`ALTER TABLE canvas_panels RENAME TO matrix_panels`,
		`ALTER TABLE canvas_edges RENAME TO matrix_edges`,
		`ALTER TABLE matrix_panels RENAME COLUMN canvas_id TO matrix_id`,
		`ALTER TABLE matrix_edges RENAME COLUMN canvas_id TO matrix_id`,
		`DROP INDEX canvas_edge_a`,
		`DROP INDEX canvas_edge_b`,
		`CREATE INDEX matrix_edge_a ON matrix_edges(a_panel)`,
		`CREATE INDEX matrix_edge_b ON matrix_edges(b_panel)`,
		`ALTER TABLE matrices ADD COLUMN mode TEXT NOT NULL DEFAULT 'grid'`,
		`DELETE FROM schema_migrations WHERE version = 45`,
	} {
		if _, err := s.db.Exec(stmt); err != nil {
			t.Fatalf("undo 045 (%s): %v", stmt, err)
		}
	}
}

type rect struct{ x, y, w, h int }

func readRect(t *testing.T, s *Store, panelID string) rect {
	t.Helper()
	var r rect
	if err := s.db.QueryRow(`SELECT x, y, w, h FROM canvas_panels WHERE id = ?`, panelID).Scan(&r.x, &r.y, &r.w, &r.h); err != nil {
		t.Fatalf("read %s: %v", panelID, err)
	}
	return r
}

// ADR-0118 §2: "coordinates convert once, in the migration". A grid board
// written by the old schema comes back in canvas units with ADR-0113's
// factors (a cell is 8 units wide and 3 tall), clamped to the canvas
// minimums — which is why a panel at the old 8-row minimum lands 28 units
// tall and not 24. A board already in canvas mode is not touched: its
// rectangles already meant units, and multiplying them would squash the
// layout into a corner.
func TestMigration045ConvertsGridPanelsOnce(t *testing.T) {
	s := openTest(t)
	undo045(t, s)

	ins := func(stmt string, args ...any) {
		t.Helper()
		if _, err := s.db.Exec(stmt, args...); err != nil {
			t.Fatalf("%s: %v", stmt, err)
		}
	}
	now := nowUTC()
	ins(`INSERT INTO matrices (id, name, compact, created_at, updated_at, mode) VALUES ('m-grid', 'Grid', 'vertical', ?, ?, 'grid')`, now, now)
	ins(`INSERT INTO matrices (id, name, compact, created_at, updated_at, mode) VALUES ('m-canvas', 'Canvas', 'none', ?, ?, 'canvas')`, now, now)
	// The grid board: the default panel, one at the old minimum height, one
	// full width in the last row.
	ins(`INSERT INTO matrix_panels (id, matrix_id, kind, ref, x, y, w, h, created_at) VALUES ('panel-a', 'm-grid', 'terminal', 't1', 0, 0, 4, 14, ?)`, now)
	ins(`INSERT INTO matrix_panels (id, matrix_id, kind, ref, x, y, w, h, created_at) VALUES ('panel-b', 'm-grid', 'terminal', 't2', 4, 0, 8, 8, ?)`, now)
	ins(`INSERT INTO matrix_panels (id, matrix_id, kind, ref, x, y, w, h, created_at) VALUES ('panel-c', 'm-grid', 'agent', 'a1', 0, 14, 12, 40, ?)`, now)
	// The canvas board, in units already, including a negative coordinate.
	ins(`INSERT INTO matrix_panels (id, matrix_id, kind, ref, x, y, w, h, created_at) VALUES ('panel-d', 'm-canvas', 'terminal', 't3', -40, 17, 32, 42, ?)`, now)
	ins(`INSERT INTO matrix_panels (id, matrix_id, kind, ref, x, y, w, h, created_at) VALUES ('panel-e', 'm-canvas', 'agent', 'a2', 99, 300, 64, 28, ?)`, now)
	ins(`INSERT INTO matrix_edges (id, matrix_id, a_panel, b_panel, created_at) VALUES ('edge-1', 'm-canvas', 'panel-d', 'panel-e', ?)`, now)

	if err := s.migrate(); err != nil {
		t.Fatalf("re-migrate: %v", err)
	}

	// x*8, y*3, w*8, h*3, then clamped to w >= 32 and h >= 28.
	want := map[string]rect{
		"panel-a": {0, 0, 32, 42},  // 4x14 cells
		"panel-b": {32, 0, 64, 28}, // 8 rows is 24 units: the clamp grows it
		"panel-c": {0, 42, 96, 120},
		"panel-d": {-40, 17, 32, 42}, // already canvas: untouched
		"panel-e": {99, 300, 64, 28},
	}
	for id, w := range want {
		if got := readRect(t, s, id); got != w {
			t.Errorf("%s = %+v, want %+v", id, got, w)
		}
	}

	// The board survives as a canvas, edges and all, and `mode` is gone.
	d, err := s.GetCanvas("m-grid")
	if err != nil || d.Name != "Grid" || len(d.Panels) != 3 {
		t.Fatalf("converted board = %+v %v", d, err)
	}
	edges, err := s.ListCanvasEdges("m-canvas")
	if err != nil || len(edges) != 1 || edges[0].APanel != "panel-d" || edges[0].BPanel != "panel-e" {
		t.Fatalf("edges after the rename = %+v %v", edges, err)
	}
	for _, col := range columnsOf(t, s, "canvases") {
		if col == "mode" {
			t.Fatalf("canvases still has a mode column")
		}
	}
	for _, old := range []string{"matrices", "matrix_panels", "matrix_edges"} {
		var n int
		if err := s.db.QueryRow(`SELECT COUNT(1) FROM sqlite_master WHERE type = 'table' AND name = ?`, old).Scan(&n); err != nil || n != 0 {
			t.Errorf("%s still exists (%d, %v)", old, n, err)
		}
	}

	// Running it a second time must not multiply anything again — the
	// bookkeeping row is what makes the conversion once-only.
	if err := s.migrate(); err != nil {
		t.Fatal(err)
	}
	if got := readRect(t, s, "panel-a"); got != want["panel-a"] {
		t.Fatalf("a second migrate converted again: %+v", got)
	}
}

func columnsOf(t *testing.T, s *Store, table string) []string {
	t.Helper()
	rows, err := s.db.Query(`SELECT name FROM pragma_table_info(?)`, table)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatal(err)
		}
		out = append(out, name)
	}
	return out
}

// ADR-0118 §3 and migration 045's comment: SQLite is documented to carry a
// table's indexes and the foreign keys pointing at it through ALTER TABLE …
// RENAME TO. This proves it on the database the migration actually
// produced, rather than trusting the documentation: the two cascades of
// migration 044 still fire under the new names, the unique binding index
// still refuses a duplicate, and both endpoint indexes are there.
func TestCanvasRenameKeepsKeysAndIndexes(t *testing.T) {
	s := openTest(t)
	c, err := s.CreateCanvas("Ops")
	if err != nil {
		t.Fatal(err)
	}
	a := addPanel(t, s, c.ID, "terminal", "t1", 0, 0, 32, 28)
	b := addPanel(t, s, c.ID, "agent", "a1", 40, 0, 32, 28)
	if _, err := s.AddCanvasEdge(c.ID, a.ID, b.ID); err != nil {
		t.Fatal(err)
	}

	// The foreign keys point at the renamed tables, not at ghosts.
	for table, want := range map[string][]string{
		"canvas_panels": {"canvases"},
		"canvas_edges":  {"canvas_panels", "canvas_panels", "canvases"},
	} {
		if got := fkTargets(t, s, table); strings.Join(got, ",") != strings.Join(want, ",") {
			t.Errorf("%s references %v, want %v", table, got, want)
		}
	}
	// Both endpoint indexes came through under their new names.
	for _, idx := range []string{"canvas_edge_a", "canvas_edge_b"} {
		var n int
		if err := s.db.QueryRow(`SELECT COUNT(1) FROM sqlite_master WHERE type = 'index' AND name = ? AND tbl_name = 'canvas_edges'`, idx).Scan(&n); err != nil || n != 1 {
			t.Errorf("index %s = %d (%v)", idx, n, err)
		}
	}
	// The unique binding index still refuses a duplicate (kind, ref) —
	// raw SQL, so it is the index refusing and not the store.
	if _, err := s.db.Exec(`INSERT INTO canvas_panels (id, canvas_id, kind, ref, x, y, w, h, created_at) VALUES ('panel-dup', ?, 'terminal', 't1', 0, 0, 32, 28, ?)`, c.ID, nowUTC()); err == nil {
		t.Fatalf("the unique index did not survive the rename")
	}

	// Cascade one: deleting a panel row takes the edge that ends on it.
	if _, err := s.db.Exec(`DELETE FROM canvas_panels WHERE id = ?`, a.ID); err != nil {
		t.Fatal(err)
	}
	if n := countRows(t, s, `SELECT COUNT(1) FROM canvas_edges`); n != 0 {
		t.Fatalf("edges after deleting a panel = %d, want 0 (the cascade from canvas_panels)", n)
	}
	// Cascade two: deleting the canvas takes its panels.
	if _, err := s.db.Exec(`DELETE FROM canvases WHERE id = ?`, c.ID); err != nil {
		t.Fatal(err)
	}
	if n := countRows(t, s, `SELECT COUNT(1) FROM canvas_panels`); n != 0 {
		t.Fatalf("panels after deleting the canvas = %d, want 0 (the cascade from canvases)", n)
	}
}

func fkTargets(t *testing.T, s *Store, table string) []string {
	t.Helper()
	rows, err := s.db.Query(`SELECT "table" FROM pragma_foreign_key_list(?) ORDER BY "table", "from"`, table)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatal(err)
		}
		out = append(out, name)
	}
	return out
}

func countRows(t *testing.T, s *Store, q string) int {
	t.Helper()
	var n int
	if err := s.db.QueryRow(q).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}
