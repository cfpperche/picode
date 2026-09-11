package store

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"math"
	"sort"
	"strings"
	"unicode/utf8"
)

// Matrix limits (ADR-0108). Every one refuses with the limit named in the
// message; nothing is truncated (the pins v2 review, finding 1: a
// byte-sliced title once left invalid UTF-8 in the database).
const (
	MaxMatrices     = 64
	MaxMatrixPanels = 500
	MaxMatrixName   = 80 // runes
	// Edges (ADR-0116, migration 044): a link is much cheaper than a panel
	// and a dense board draws more of them than it holds panels, so the cap
	// is twice the panel cap. internal/store/matrix_edges.go is the model.
	MaxMatrixEdges  = 1000
	MatrixCols      = 12 // grid mode: fixed, not stored
	MinMatrixPanelW = 4  // columns
	MinMatrixPanelH = 8  // rows

	// Canvas mode (ADR-0113): the unit is 8 px, x and y may be negative,
	// and the plane is bounded only so a panel cannot be dragged out of
	// reach of every viewport.
	MinCanvasPanelW = 32     // units (256 px)
	MinCanvasPanelH = 28     // units (224 px)
	MaxCanvasPanel  = 4096   // units, w and h
	MaxCanvasCoord  = 100000 // units, |x| and |y|
)

// Layout modes (ADR-0113, amending ADR-0108). The mode is a property of the
// matrix and decides what a panel's x/y/w/h mean; `compact` keeps meaning
// only in grid mode.
const (
	MatrixModeGrid   = "grid"
	MatrixModeCanvas = "canvas"
)

// The switch transform's factors: a grid cell is 8 canvas units wide
// (colWidth / 8) and 3 tall (rowHeight 24 px / 8).
const (
	canvasPerCol = 8
	canvasPerRow = 3
)

// Compaction modes. The v1 grid compacts vertically (rows without end);
// "none" is the v2 free canvas the column already admits.
const (
	MatrixCompactVertical = "vertical"
	MatrixCompactNone     = "none"
)

// Matrix is the summary a list, a feed event and a PATCH answer carry.
type Matrix struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Compact    string `json:"compact"`
	Mode       string `json:"mode"`
	CreatedAt  string `json:"createdAt"`
	UpdatedAt  string `json:"updatedAt"`
	PanelCount int    `json:"panelCount"`
}

// MatrixPanel is one slot on a matrix: a binding (kind, ref) at a cell
// rectangle. Its id is random and never derived from ref, so the same
// terminal on two matrices is two panels.
type MatrixPanel struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"`
	Ref       string `json:"ref"`
	X         int    `json:"x"`
	Y         int    `json:"y"`
	W         int    `json:"w"`
	H         int    `json:"h"`
	CreatedAt string `json:"createdAt"`
}

// MatrixDetail is what GET /api/matrices/{id} answers: the summary plus
// every panel and every edge — one read still opens a matrix (500 panels
// ≈ 50 KB; an edge is four short strings).
type MatrixDetail struct {
	Matrix
	Panels []MatrixPanel `json:"panels"`
	Edges  []MatrixEdge  `json:"edges"`
}

// MatrixPatch is UpdateMatrix's optional fields; nil leaves a column alone.
type MatrixPatch struct {
	Name    *string
	Compact *string
}

// PanelPlacement is one row of a layout patch: where a panel the matrix
// already holds goes.
type PanelPlacement struct {
	ID string `json:"id"`
	X  int    `json:"x"`
	Y  int    `json:"y"`
	W  int    `json:"w"`
	H  int    `json:"h"`
}

// MatrixLayout is the matrix.layout event and the PATCH …/layout answer:
// exactly the subset that moved, never the whole matrix.
type MatrixLayout struct {
	ID        string           `json:"id"`
	UpdatedAt string           `json:"updatedAt"`
	Panels    []PanelPlacement `json:"panels"`
}

// MatrixModeChanged is the matrix.mode event and the PATCH answer when the
// mode changes: the summary (carrying the new mode) plus exactly the panels
// the switch transform moved, in the shape PATCH …/layout answers with, so
// the client keeps one reducer path.
type MatrixModeChanged struct {
	Matrix
	Panels []PanelPlacement `json:"panels"`
}

// MatrixPanelAdded is the matrix.panel.added event and the POST …/panels
// answer.
type MatrixPanelAdded struct {
	ID        string      `json:"id"`
	UpdatedAt string      `json:"updatedAt"`
	Panel     MatrixPanel `json:"panel"`
}

// MatrixPanelRemoved is the matrix.panel.removed event.
type MatrixPanelRemoved struct {
	ID        string `json:"id"`
	UpdatedAt string `json:"updatedAt"`
	PanelID   string `json:"panelId"`
}

// conflictError and notFoundError read as their message alone (the UI
// shows it verbatim) and still answer errors.Is against ErrConflict /
// ErrNotFound, the way invalidError does for ErrInvalid.
type conflictError struct{ msg string }

func (e conflictError) Error() string        { return e.msg }
func (e conflictError) Is(target error) bool { return target == ErrConflict }

type notFoundError struct{ msg string }

func (e notFoundError) Error() string        { return e.msg }
func (e notFoundError) Is(target error) bool { return target == ErrNotFound }

var (
	errMatrixNotFound = notFoundError{"matrix not found"}
	errPanelNotFound  = notFoundError{"panel not found"}
	errMatrixConflict = conflictError{"This matrix changed elsewhere. Reload to see the latest version."}
)

// normalizeMatrixName trims and validates; it never truncates.
func normalizeMatrixName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", invalid("name is required")
	}
	if !utf8.ValidString(name) {
		return "", invalid("name is not valid UTF-8")
	}
	if utf8.RuneCountInString(name) > MaxMatrixName {
		return "", invalid("name is too long (max %d characters)", MaxMatrixName)
	}
	return name, nil
}

func validateCompact(c string) error {
	if c != MatrixCompactVertical && c != MatrixCompactNone {
		return invalid("compact must be %s or %s", MatrixCompactVertical, MatrixCompactNone)
	}
	return nil
}

// Panel kinds (ADR-0108; the note/file/diff bodies are C3 of
// docs/plans/matrix-canvas.md). `kind` is an open text column, so a kind is
// a validator edit and never a migration. The store stays ignorant of what a
// ref points at — only its *shape* is a rule, exactly as a deleted terminal
// leaves its panel behind for the UI to render as gone.
const (
	MatrixKindAgent    = "agent"
	MatrixKindTerminal = "terminal"
	MatrixKindNote     = "note" // ref: a pin id
	MatrixKindFile     = "file" // ref: <owner>:<id>:<path>
	MatrixKindDiff     = "diff" // ref: <owner>:<id>:<path>
)

// The refusals are the contract, repeated word for word in
// web/shared/domain/matrix.js so the UI can refuse before asking.
const (
	matrixKindMsg = "kind must be agent, terminal, note, file or diff"
	matrixRefMsg  = "ref must be <owner>:<id>:<path> with owner t, a or w"
)

func validatePanelBinding(kind, ref string) error {
	switch kind {
	case MatrixKindAgent, MatrixKindTerminal, MatrixKindNote, MatrixKindFile, MatrixKindDiff:
	default:
		return invalid(matrixKindMsg)
	}
	if ref == "" {
		return invalid("ref is required")
	}
	if kind == MatrixKindFile || kind == MatrixKindDiff {
		return validateOwnerRef(ref)
	}
	return nil
}

// validateOwnerRef checks the shape a file or a diff panel binds by:
// "<owner>:<id>:<path>", the desktop's own owner letters (t terminal,
// a agent, w workspace — web/desktop/src/lib/routes.js). A path may hold
// colons of its own; only the first two separate. Whether that owner or
// that file exists is not the store's business.
func validateOwnerRef(ref string) error {
	letter, rest, ok := strings.Cut(ref, ":")
	if !ok {
		return invalid(matrixRefMsg)
	}
	id, path, ok := strings.Cut(rest, ":")
	if !ok || (letter != "t" && letter != "a" && letter != "w") || id == "" || path == "" {
		return invalid(matrixRefMsg)
	}
	return nil
}

func validateMode(m string) error {
	if m != MatrixModeGrid && m != MatrixModeCanvas {
		return invalid("mode must be %s or %s", MatrixModeGrid, MatrixModeCanvas)
	}
	return nil
}

// validatePlacement is the rectangle rule of the matrix's own mode
// (ADR-0113): grid is 12 columns and panels of at least 4×8 cells, rows
// without end; canvas is a plane of 8 px units where x and y may be
// negative, a panel is at least 32×28 units, and both are bounded so a
// panel cannot be lost. The web contract (web/shared/domain/matrix.js)
// repeats these words so the UI can refuse before asking.
func validatePlacement(mode string, x, y, w, h int) error {
	if mode == MatrixModeCanvas {
		switch {
		case x < -MaxCanvasCoord || x > MaxCanvasCoord:
			return invalid("x must be between -%d and %d canvas units", MaxCanvasCoord, MaxCanvasCoord)
		case y < -MaxCanvasCoord || y > MaxCanvasCoord:
			return invalid("y must be between -%d and %d canvas units", MaxCanvasCoord, MaxCanvasCoord)
		case w < MinCanvasPanelW:
			return invalid("w must be at least %d canvas units", MinCanvasPanelW)
		case h < MinCanvasPanelH:
			return invalid("h must be at least %d canvas units", MinCanvasPanelH)
		case w > MaxCanvasPanel:
			return invalid("w must be at most %d canvas units", MaxCanvasPanel)
		case h > MaxCanvasPanel:
			return invalid("h must be at most %d canvas units", MaxCanvasPanel)
		}
		return nil
	}
	switch {
	case x < 0:
		return invalid("x must be 0 or more")
	case y < 0:
		return invalid("y must be 0 or more")
	case w < MinMatrixPanelW:
		return invalid("w must be at least %d columns", MinMatrixPanelW)
	case h < MinMatrixPanelH:
		return invalid("h must be at least %d rows", MinMatrixPanelH)
	case x+w > MatrixCols:
		return invalid("x + w must be at most %d columns", MatrixCols)
	}
	return nil
}

// ---- the switch transform (ADR-0113) -------------------------------------
//
// One documented conversion in each direction, repeated as pure functions in
// web/shared/domain/matrix.js (gridToCanvas / canvasToGrid) so the UI can
// preview a switch before asking for it.

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// roundDiv rounds half away from zero, the way math.Round does and the way
// the web contract's round() does, so both sides land on the same cell.
func roundDiv(v, f int) int { return int(math.Round(float64(v) / float64(f))) }

// gridToCanvas scales one grid rectangle into canvas units and clamps it to
// the canvas rules. A panel at the 8-row minimum becomes 24 units tall,
// under the canvas minimum of 28, so it grows — and can then overlap the
// panel below by up to 4 units. Canvas mode is free placement and allows
// that; switching back packs it out.
func gridToCanvas(r PanelPlacement) PanelPlacement {
	return PanelPlacement{
		ID: r.ID,
		X:  clampInt(r.X*canvasPerCol, -MaxCanvasCoord, MaxCanvasCoord),
		Y:  clampInt(r.Y*canvasPerRow, -MaxCanvasCoord, MaxCanvasCoord),
		W:  clampInt(r.W*canvasPerCol, MinCanvasPanelW, MaxCanvasPanel),
		H:  clampInt(r.H*canvasPerRow, MinCanvasPanelH, MaxCanvasPanel),
	}
}

// canvasToGridRect divides by the same factors, rounds and clamps into the
// 12-column rules. It says nothing about overlap — canvasToGrid packs.
func canvasToGridRect(r PanelPlacement) PanelPlacement {
	out := PanelPlacement{
		ID: r.ID,
		X:  roundDiv(r.X, canvasPerCol),
		Y:  roundDiv(r.Y, canvasPerRow),
		W:  clampInt(roundDiv(r.W, canvasPerCol), MinMatrixPanelW, MatrixCols),
		H:  roundDiv(r.H, canvasPerRow),
	}
	if out.H < MinMatrixPanelH {
		out.H = MinMatrixPanelH
	}
	out.X = clampInt(out.X, 0, MatrixCols-out.W)
	if out.Y < 0 {
		out.Y = 0
	}
	return out
}

// nextGridSlot is the first free w×h slot scanning rows top-down and columns
// left-to-right, else the row under everything — the same arithmetic as
// nextSlot in web/shared/domain/matrix.js. Only a placed panel's edges can
// start a first-free slot (a free rectangle slides up and left until it
// touches one), so the scan visits edges, not cells.
func nextGridSlot(placed []PanelPlacement, w, h int) (int, int) {
	ys, xs := map[int]bool{0: true}, map[int]bool{0: true}
	bottom := 0
	for _, r := range placed {
		if r.Y+r.H > bottom {
			bottom = r.Y + r.H
		}
		ys[r.Y+r.H] = true
		xs[r.X+r.W] = true
	}
	rows, cols := sortedKeys(ys), []int{}
	for _, x := range sortedKeys(xs) {
		if x+w <= MatrixCols {
			cols = append(cols, x)
		}
	}
	for _, y := range rows {
		for _, x := range cols {
			cand := PanelPlacement{X: x, Y: y, W: w, H: h}
			free := true
			for _, r := range placed {
				if rectsOverlap(cand, r) {
					free = false
					break
				}
			}
			if free {
				return x, y
			}
		}
	}
	return 0, bottom
}

func sortedKeys(set map[int]bool) []int {
	out := make([]int, 0, len(set))
	for v := range set {
		out = append(out, v)
	}
	sort.Ints(out)
	return out
}

func rectsOverlap(a, b PanelPlacement) bool {
	return a.X < b.X+b.W && b.X < a.X+a.W && a.Y < b.Y+b.H && b.Y < a.Y+a.H
}

// canvasToGrid converts every rectangle and packs it into the grid: reading
// order (top-down, then left-to-right, ties by id), each panel at the first
// free slot. Rounding alone would leave panels overlapping, and overlap
// after a switch is a bug, not a tolerance — the pack is what makes the
// switch total. The result keeps the input's order, so the caller can
// compare rectangle by rectangle.
func canvasToGrid(rects []PanelPlacement) []PanelPlacement {
	order := make([]PanelPlacement, 0, len(rects))
	for _, r := range rects {
		order = append(order, canvasToGridRect(r))
	}
	sort.SliceStable(order, func(i, j int) bool {
		a, b := order[i], order[j]
		if a.Y != b.Y {
			return a.Y < b.Y
		}
		if a.X != b.X {
			return a.X < b.X
		}
		return a.ID < b.ID
	})
	placed := make([]PanelPlacement, 0, len(order))
	at := make(map[string]PanelPlacement, len(order))
	for _, r := range order {
		r.X, r.Y = nextGridSlot(placed, r.W, r.H)
		placed = append(placed, r)
		at[r.ID] = r
	}
	out := make([]PanelPlacement, 0, len(rects))
	for _, r := range rects {
		out = append(out, at[r.ID])
	}
	return out
}

// switchPanels converts every panel into the target mode's units and returns
// exactly the ones whose rectangle changed — what the event carries and what
// the transaction writes.
func switchPanels(mode string, panels []MatrixPanel) []PanelPlacement {
	rects := make([]PanelPlacement, 0, len(panels))
	for _, p := range panels {
		rects = append(rects, PanelPlacement{ID: p.ID, X: p.X, Y: p.Y, W: p.W, H: p.H})
	}
	next := make([]PanelPlacement, 0, len(rects))
	if mode == MatrixModeCanvas {
		for _, r := range rects {
			next = append(next, gridToCanvas(r))
		}
	} else {
		next = canvasToGrid(rects)
	}
	moved := []PanelPlacement{}
	for i, r := range next {
		if r != rects[i] {
			moved = append(moved, r)
		}
	}
	return moved
}

// newPanelID is a slot id with 48 random bits: a matrix holds up to 500
// panels, and newID's 24-bit suffix on a fixed prefix would collide once in
// a few hundred matrices.
func newPanelID() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return "panel-" + hex.EncodeToString(b)
}

// matrixTx runs fn in one transaction: the events fn appends are announced
// on commit and dropped on rollback (ADR-0048). The store has one SQLite
// connection, so fn must read through tx, never s.db.
func (s *Store) matrixTx(fn func(tx *sql.Tx) error) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		s.rollback(tx)
		return err
	}
	return s.commit(tx)
}

// matrixGuard reads the row's updated_at and mode inside the transaction,
// before anything is written: a missing matrix is ErrNotFound; when the
// caller sent the updatedAt it last saw and the row moved on, ErrConflict.
// It answers the mode because every rectangle is validated against the
// matrix's own mode, and the store has one SQLite connection — a second read
// through s.db inside the transaction would deadlock.
func matrixGuard(tx *sql.Tx, id, ifUpdatedAt string) (string, error) {
	var cur, mode string
	err := tx.QueryRow(`SELECT updated_at, mode FROM matrices WHERE id = ?`, id).Scan(&cur, &mode)
	if err == sql.ErrNoRows {
		return "", errMatrixNotFound
	}
	if err != nil {
		return "", fmt.Errorf("store: matrix: %w", err)
	}
	if ifUpdatedAt != "" && cur != ifUpdatedAt {
		return "", errMatrixConflict
	}
	return mode, nil
}

func touchMatrix(tx *sql.Tx, id, now string) error {
	if _, err := tx.Exec(`UPDATE matrices SET updated_at = ? WHERE id = ?`, now, id); err != nil {
		return fmt.Errorf("store: touch matrix: %w", err)
	}
	return nil
}

const matrixCols = `id, name, compact, mode, created_at, updated_at,
		(SELECT COUNT(1) FROM matrix_panels p WHERE p.matrix_id = matrices.id)`

func scanMatrix(row interface{ Scan(...any) error }, m *Matrix) error {
	return row.Scan(&m.ID, &m.Name, &m.Compact, &m.Mode, &m.CreatedAt, &m.UpdatedAt, &m.PanelCount)
}

// rowQuerier is the intersection of *sql.DB and *sql.Tx the reads use.
type rowQuerier interface {
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
}

func matrixSummary(q rowQuerier, id string) (Matrix, error) {
	var m Matrix
	err := scanMatrix(q.QueryRow(`SELECT `+matrixCols+` FROM matrices WHERE id = ?`, id), &m)
	if err == sql.ErrNoRows {
		return Matrix{}, errMatrixNotFound
	}
	if err != nil {
		return Matrix{}, fmt.Errorf("store: matrix summary: %w", err)
	}
	return m, nil
}

func matrixPanels(q rowQuerier, id string) ([]MatrixPanel, error) {
	rows, err := q.Query(`SELECT id, kind, ref, x, y, w, h, created_at FROM matrix_panels WHERE matrix_id = ? ORDER BY y, x, created_at`, id)
	if err != nil {
		return nil, fmt.Errorf("store: matrix panels: %w", err)
	}
	defer rows.Close()
	out := []MatrixPanel{}
	for rows.Next() {
		var p MatrixPanel
		if err := rows.Scan(&p.ID, &p.Kind, &p.Ref, &p.X, &p.Y, &p.W, &p.H, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("store: scan panel: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// ListMatrices is the switcher's list: by name (ASCII case-folded), then
// by creation, so a drag never reorders the tabs.
func (s *Store) ListMatrices() ([]Matrix, error) {
	rows, err := s.db.Query(`SELECT ` + matrixCols + ` FROM matrices ORDER BY lower(name), created_at`)
	if err != nil {
		return nil, fmt.Errorf("store: list matrices: %w", err)
	}
	defer rows.Close()
	out := []Matrix{}
	for rows.Next() {
		var m Matrix
		if err := scanMatrix(rows, &m); err != nil {
			return nil, fmt.Errorf("store: scan matrix: %w", err)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// GetMatrix is the summary plus every panel, ordered by row then column,
// plus every edge (ADR-0116) so one read still opens a matrix.
func (s *Store) GetMatrix(id string) (MatrixDetail, error) {
	var d MatrixDetail
	err := s.matrixTx(func(tx *sql.Tx) error {
		m, err := matrixSummary(tx, id)
		if err != nil {
			return err
		}
		panels, err := matrixPanels(tx, id)
		if err != nil {
			return err
		}
		edges, err := matrixEdges(tx, id)
		if err != nil {
			return err
		}
		m.PanelCount = len(panels)
		d = MatrixDetail{Matrix: m, Panels: panels, Edges: edges}
		return nil
	})
	return d, err
}

// CreateMatrix adds an empty matrix (grid mode, vertical compaction) and
// announces matrix.created with the summary. The 65th is refused.
func (s *Store) CreateMatrix(name string) (Matrix, error) {
	name, err := normalizeMatrixName(name)
	if err != nil {
		return Matrix{}, err
	}
	var m Matrix
	err = s.matrixTx(func(tx *sql.Tx) error {
		var n int
		if err := tx.QueryRow(`SELECT COUNT(1) FROM matrices`).Scan(&n); err != nil {
			return fmt.Errorf("store: count matrices: %w", err)
		}
		if n >= MaxMatrices {
			return invalid("limit: %d matrices", MaxMatrices)
		}
		now := nowUTC()
		m = Matrix{ID: newID(name, "matrix"), Name: name, Compact: MatrixCompactVertical, Mode: MatrixModeGrid, CreatedAt: now, UpdatedAt: now}
		if _, err := tx.Exec(`INSERT INTO matrices (id, name, compact, mode, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
			m.ID, m.Name, m.Compact, m.Mode, m.CreatedAt, m.UpdatedAt); err != nil {
			return fmt.Errorf("store: create matrix: %w", err)
		}
		return s.AppendEventTx(tx, "matrix.created", nil, nil, m)
	})
	return m, err
}

// UpdateMatrix renames and/or changes the compaction. ifUpdatedAt, when
// not empty, is the updatedAt the caller last saw: a row that moved on
// answers ErrConflict and nothing is written. Announces matrix.updated.
// The layout mode is SetMatrixMode's: it moves every panel, so it is its
// own transaction and its own event.
func (s *Store) UpdateMatrix(id string, p MatrixPatch, ifUpdatedAt string) (Matrix, error) {
	if p.Name == nil && p.Compact == nil {
		return Matrix{}, invalid("nothing to update: send name or compact")
	}
	var name, compact any // nil keeps the column (COALESCE)
	if p.Name != nil {
		n, err := normalizeMatrixName(*p.Name)
		if err != nil {
			return Matrix{}, err
		}
		name = n
	}
	if p.Compact != nil {
		if err := validateCompact(*p.Compact); err != nil {
			return Matrix{}, err
		}
		compact = *p.Compact
	}
	var m Matrix
	err := s.matrixTx(func(tx *sql.Tx) error {
		if _, err := matrixGuard(tx, id, ifUpdatedAt); err != nil {
			return err
		}
		if _, err := tx.Exec(`UPDATE matrices SET name = COALESCE(?, name), compact = COALESCE(?, compact), updated_at = ? WHERE id = ?`,
			name, compact, nowUTC(), id); err != nil {
			return fmt.Errorf("store: update matrix: %w", err)
		}
		var err error
		if m, err = matrixSummary(tx, id); err != nil {
			return err
		}
		return s.AppendEventTx(tx, "matrix.updated", nil, nil, m)
	})
	return m, err
}

// PatchMatrixLayout moves the listed panels — the changed subset, not the
// whole matrix — in one transaction: every id must be on this matrix and
// every rectangle legal in the matrix's current mode, or nothing is
// written. One matrix.layout event carries exactly the subset.
func (s *Store) PatchMatrixLayout(id string, panels []PanelPlacement, ifUpdatedAt string) (MatrixLayout, error) {
	if len(panels) == 0 {
		return MatrixLayout{}, invalid("panels is required")
	}
	seen := make(map[string]bool, len(panels))
	for _, p := range panels {
		if p.ID == "" {
			return MatrixLayout{}, invalid("panel id is required")
		}
		if seen[p.ID] {
			return MatrixLayout{}, invalid("panel %s is listed twice", p.ID)
		}
		seen[p.ID] = true
	}
	var out MatrixLayout
	err := s.matrixTx(func(tx *sql.Tx) error {
		mode, err := matrixGuard(tx, id, ifUpdatedAt)
		if err != nil {
			return err
		}
		// Every rectangle is checked against this matrix's mode before any
		// row is written: a canvas rectangle cannot land in a grid matrix,
		// and one bad row refuses the whole batch.
		for _, p := range panels {
			if err := validatePlacement(mode, p.X, p.Y, p.W, p.H); err != nil {
				return fmt.Errorf("panel %s: %w", p.ID, err)
			}
		}
		for _, p := range panels {
			res, err := tx.Exec(`UPDATE matrix_panels SET x = ?, y = ?, w = ?, h = ? WHERE id = ? AND matrix_id = ?`,
				p.X, p.Y, p.W, p.H, p.ID, id)
			if err != nil {
				return fmt.Errorf("store: move panel: %w", err)
			}
			if n, _ := res.RowsAffected(); n == 0 {
				return invalid("panel %s is not on this matrix", p.ID)
			}
		}
		now := nowUTC()
		if err := touchMatrix(tx, id, now); err != nil {
			return err
		}
		out = MatrixLayout{ID: id, UpdatedAt: now, Panels: panels}
		return s.AppendEventTx(tx, "matrix.layout", nil, nil, out)
	})
	return out, err
}

// SetMatrixMode switches a matrix between grid and canvas layout
// (ADR-0113) in one transaction: the column, every panel's rectangle by the
// documented transform, updated_at, and one matrix.mode event carrying the
// new summary plus exactly the panels that moved. Switching to the mode the
// matrix already has is a no-op — nothing is written, nothing is announced
// and updated_at does not move — so a client that re-sends the mode it is
// already showing cannot make two browsers refetch.
func (s *Store) SetMatrixMode(id, mode, ifUpdatedAt string) (MatrixModeChanged, error) {
	if err := validateMode(mode); err != nil {
		return MatrixModeChanged{}, err
	}
	var out MatrixModeChanged
	err := s.matrixTx(func(tx *sql.Tx) error {
		cur, err := matrixGuard(tx, id, ifUpdatedAt)
		if err != nil {
			return err
		}
		if cur == mode {
			m, err := matrixSummary(tx, id)
			if err != nil {
				return err
			}
			out = MatrixModeChanged{Matrix: m, Panels: []PanelPlacement{}}
			return nil
		}
		panels, err := matrixPanels(tx, id)
		if err != nil {
			return err
		}
		moved := switchPanels(mode, panels)
		for _, p := range moved {
			if _, err := tx.Exec(`UPDATE matrix_panels SET x = ?, y = ?, w = ?, h = ? WHERE id = ? AND matrix_id = ?`,
				p.X, p.Y, p.W, p.H, p.ID, id); err != nil {
				return fmt.Errorf("store: switch panel: %w", err)
			}
		}
		now := nowUTC()
		if _, err := tx.Exec(`UPDATE matrices SET mode = ?, updated_at = ? WHERE id = ?`, mode, now, id); err != nil {
			return fmt.Errorf("store: switch matrix mode: %w", err)
		}
		m, err := matrixSummary(tx, id)
		if err != nil {
			return err
		}
		out = MatrixModeChanged{Matrix: m, Panels: moved}
		return s.AppendEventTx(tx, "matrix.mode", nil, nil, out)
	})
	return out, err
}

// AddMatrixPanel binds an agent or a terminal to a new slot. The client
// places (x, y, w, h); the store validates the rule of this matrix's mode,
// the cap and the binding — one (kind, ref) per matrix, refused by the unique index and
// read back as ErrConflict. Announces matrix.panel.added.
func (s *Store) AddMatrixPanel(id, kind, ref string, x, y, w, h int) (MatrixPanelAdded, error) {
	ref = strings.TrimSpace(ref)
	if err := validatePanelBinding(kind, ref); err != nil {
		return MatrixPanelAdded{}, err
	}
	var out MatrixPanelAdded
	err := s.matrixTx(func(tx *sql.Tx) error {
		mode, err := matrixGuard(tx, id, "")
		if err != nil {
			return err
		}
		if err := validatePlacement(mode, x, y, w, h); err != nil {
			return err
		}
		var n int
		if err := tx.QueryRow(`SELECT COUNT(1) FROM matrix_panels WHERE matrix_id = ?`, id).Scan(&n); err != nil {
			return fmt.Errorf("store: count panels: %w", err)
		}
		if n >= MaxMatrixPanels {
			return invalid("limit: %d panels per matrix", MaxMatrixPanels)
		}
		now := nowUTC()
		p := MatrixPanel{ID: newPanelID(), Kind: kind, Ref: ref, X: x, Y: y, W: w, H: h, CreatedAt: now}
		if _, err := tx.Exec(`INSERT INTO matrix_panels (id, matrix_id, kind, ref, x, y, w, h, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			p.ID, id, p.Kind, p.Ref, p.X, p.Y, p.W, p.H, p.CreatedAt); err != nil {
			// The unique index refused the binding; every other failure
			// keeps its real diagnostic.
			var dup int
			if e := tx.QueryRow(`SELECT COUNT(1) FROM matrix_panels WHERE matrix_id = ? AND kind = ? AND ref = ?`, id, kind, ref).Scan(&dup); e == nil && dup > 0 {
				return conflictError{fmt.Sprintf("This %s is already on this matrix", kind)}
			}
			return fmt.Errorf("store: add panel: %w", err)
		}
		if err := touchMatrix(tx, id, now); err != nil {
			return err
		}
		out = MatrixPanelAdded{ID: id, UpdatedAt: now, Panel: p}
		return s.AppendEventTx(tx, "matrix.panel.added", nil, nil, out)
	})
	return out, err
}

// RemoveMatrixPanel frees the slot and announces matrix.panel.removed.
func (s *Store) RemoveMatrixPanel(id, panelID string) error {
	return s.matrixTx(func(tx *sql.Tx) error {
		if _, err := matrixGuard(tx, id, ""); err != nil {
			return err
		}
		res, err := tx.Exec(`DELETE FROM matrix_panels WHERE id = ? AND matrix_id = ?`, panelID, id)
		if err != nil {
			return fmt.Errorf("store: remove panel: %w", err)
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return errPanelNotFound
		}
		now := nowUTC()
		if err := touchMatrix(tx, id, now); err != nil {
			return err
		}
		return s.AppendEventTx(tx, "matrix.panel.removed", nil, nil, MatrixPanelRemoved{ID: id, UpdatedAt: now, PanelID: panelID})
	})
}

// DeleteMatrix removes the matrix; its panels go with it (ON DELETE
// CASCADE) without an event each. Announces matrix.deleted {id}.
func (s *Store) DeleteMatrix(id string) error {
	return s.matrixTx(func(tx *sql.Tx) error {
		res, err := tx.Exec(`DELETE FROM matrices WHERE id = ?`, id)
		if err != nil {
			return fmt.Errorf("store: delete matrix: %w", err)
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return errMatrixNotFound
		}
		return s.AppendEventTx(tx, "matrix.deleted", nil, nil, idData(id))
	})
}
