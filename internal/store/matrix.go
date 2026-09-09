package store

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
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
	MatrixCols      = 12 // fixed in v1, not stored
	MinMatrixPanelW = 4  // columns
	MinMatrixPanelH = 8  // rows
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
// every panel — one read on open (500 panels ≈ 50 KB).
type MatrixDetail struct {
	Matrix
	Panels []MatrixPanel `json:"panels"`
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

func validatePanelBinding(kind, ref string) error {
	if kind != "agent" && kind != "terminal" {
		return invalid("kind must be agent or terminal")
	}
	if ref == "" {
		return invalid("ref is required")
	}
	return nil
}

// validatePlacement is the grid rule: 12 columns, panels of at least 4×8
// cells, rows without end. The web contract (web/shared/domain/matrix.js)
// repeats these words so the UI can refuse before asking.
func validatePlacement(x, y, w, h int) error {
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

// matrixGuard reads the row's updated_at inside the transaction, before
// anything is written: a missing matrix is ErrNotFound; when the caller
// sent the updatedAt it last saw and the row moved on, ErrConflict.
func matrixGuard(tx *sql.Tx, id, ifUpdatedAt string) error {
	var cur string
	err := tx.QueryRow(`SELECT updated_at FROM matrices WHERE id = ?`, id).Scan(&cur)
	if err == sql.ErrNoRows {
		return errMatrixNotFound
	}
	if err != nil {
		return fmt.Errorf("store: matrix: %w", err)
	}
	if ifUpdatedAt != "" && cur != ifUpdatedAt {
		return errMatrixConflict
	}
	return nil
}

func touchMatrix(tx *sql.Tx, id, now string) error {
	if _, err := tx.Exec(`UPDATE matrices SET updated_at = ? WHERE id = ?`, now, id); err != nil {
		return fmt.Errorf("store: touch matrix: %w", err)
	}
	return nil
}

const matrixCols = `id, name, compact, created_at, updated_at,
		(SELECT COUNT(1) FROM matrix_panels p WHERE p.matrix_id = matrices.id)`

func scanMatrix(row interface{ Scan(...any) error }, m *Matrix) error {
	return row.Scan(&m.ID, &m.Name, &m.Compact, &m.CreatedAt, &m.UpdatedAt, &m.PanelCount)
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

// GetMatrix is the summary plus every panel, ordered by row then column.
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
		m.PanelCount = len(panels)
		d = MatrixDetail{Matrix: m, Panels: panels}
		return nil
	})
	return d, err
}

// CreateMatrix adds an empty matrix (vertical compaction) and announces
// matrix.created with the summary. The 65th is refused.
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
		m = Matrix{ID: newID(name, "matrix"), Name: name, Compact: MatrixCompactVertical, CreatedAt: now, UpdatedAt: now}
		if _, err := tx.Exec(`INSERT INTO matrices (id, name, compact, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
			m.ID, m.Name, m.Compact, m.CreatedAt, m.UpdatedAt); err != nil {
			return fmt.Errorf("store: create matrix: %w", err)
		}
		return s.AppendEventTx(tx, "matrix.created", nil, nil, m)
	})
	return m, err
}

// UpdateMatrix renames and/or changes the compaction. ifUpdatedAt, when
// not empty, is the updatedAt the caller last saw: a row that moved on
// answers ErrConflict and nothing is written. Announces matrix.updated.
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
		if err := matrixGuard(tx, id, ifUpdatedAt); err != nil {
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
// every rectangle inside the grid, or nothing is written. One matrix.layout
// event carries exactly the subset.
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
		if err := validatePlacement(p.X, p.Y, p.W, p.H); err != nil {
			return MatrixLayout{}, fmt.Errorf("panel %s: %w", p.ID, err)
		}
	}
	var out MatrixLayout
	err := s.matrixTx(func(tx *sql.Tx) error {
		if err := matrixGuard(tx, id, ifUpdatedAt); err != nil {
			return err
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

// AddMatrixPanel binds an agent or a terminal to a new slot. The client
// places (x, y, w, h); the store validates the grid rule, the cap and the
// binding — one (kind, ref) per matrix, refused by the unique index and
// read back as ErrConflict. Announces matrix.panel.added.
func (s *Store) AddMatrixPanel(id, kind, ref string, x, y, w, h int) (MatrixPanelAdded, error) {
	ref = strings.TrimSpace(ref)
	if err := validatePanelBinding(kind, ref); err != nil {
		return MatrixPanelAdded{}, err
	}
	if err := validatePlacement(x, y, w, h); err != nil {
		return MatrixPanelAdded{}, err
	}
	var out MatrixPanelAdded
	err := s.matrixTx(func(tx *sql.Tx) error {
		if err := matrixGuard(tx, id, ""); err != nil {
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
		if err := matrixGuard(tx, id, ""); err != nil {
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
