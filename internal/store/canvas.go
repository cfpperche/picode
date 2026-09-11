package store

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"unicode/utf8"
)

// Canvas limits (ADR-0108, renamed by ADR-0118). Every one refuses with the
// limit named in the message; nothing is truncated (the pins v2 review,
// finding 1: a byte-sliced title once left invalid UTF-8 in the database).
const (
	MaxCanvases     = 64
	MaxCanvasPanels = 500
	MaxCanvasName   = 80 // runes
	// Edges (ADR-0116, migration 044): a link is much cheaper than a panel
	// and a dense board draws more of them than it holds panels, so the cap
	// is twice the panel cap. internal/store/canvas_edges.go is the model.
	MaxCanvasEdges = 1000

	// The plane (ADR-0113's canvas rules, the only rules since ADR-0118):
	// the unit is 8 px, x and y may be negative, and the plane is bounded
	// only so a panel cannot be dragged out of reach of every viewport.
	MinCanvasPanelW = 32     // units (256 px)
	MinCanvasPanelH = 28     // units (224 px)
	MaxCanvasPanel  = 4096   // units, w and h
	MaxCanvasCoord  = 100000 // units, |x| and |y|
)

// Compaction modes. `compact` is the column the v1 grid compacted by; a
// canvas is free placement, so nothing compacts it and the column only
// records what the owner asked for.
const (
	CanvasCompactVertical = "vertical"
	CanvasCompactNone     = "none"
)

// Canvas is the summary a list, a feed event and a PATCH answer carry.
type Canvas struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Compact    string `json:"compact"`
	CreatedAt  string `json:"createdAt"`
	UpdatedAt  string `json:"updatedAt"`
	PanelCount int    `json:"panelCount"`
}

// CanvasPanel is one panel on a canvas: a binding (kind, ref) at a
// rectangle of 8 px units. Its id is random and never derived from ref, so
// the same terminal on two canvases is two panels.
type CanvasPanel struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"`
	Ref       string `json:"ref"`
	X         int    `json:"x"`
	Y         int    `json:"y"`
	W         int    `json:"w"`
	H         int    `json:"h"`
	CreatedAt string `json:"createdAt"`
}

// CanvasDetail is what GET /api/canvases/{id} answers: the summary plus
// every panel and every edge — one read still opens a canvas (500 panels
// ≈ 50 KB; an edge is four short strings).
type CanvasDetail struct {
	Canvas
	Panels []CanvasPanel `json:"panels"`
	Edges  []CanvasEdge  `json:"edges"`
}

// CanvasPatch is UpdateCanvas's optional fields; nil leaves a column alone.
type CanvasPatch struct {
	Name    *string
	Compact *string
}

// PanelPlacement is one row of a layout patch: where a panel the canvas
// already holds goes.
type PanelPlacement struct {
	ID string `json:"id"`
	X  int    `json:"x"`
	Y  int    `json:"y"`
	W  int    `json:"w"`
	H  int    `json:"h"`
}

// CanvasLayout is the canvas.layout event and the PATCH …/layout answer:
// exactly the subset that moved, never the whole canvas.
type CanvasLayout struct {
	ID        string           `json:"id"`
	UpdatedAt string           `json:"updatedAt"`
	Panels    []PanelPlacement `json:"panels"`
}

// CanvasPanelAdded is the canvas.panel.added event and the POST …/panels
// answer.
type CanvasPanelAdded struct {
	ID        string      `json:"id"`
	UpdatedAt string      `json:"updatedAt"`
	Panel     CanvasPanel `json:"panel"`
}

// CanvasPanelRemoved is the canvas.panel.removed event.
type CanvasPanelRemoved struct {
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
	errCanvasNotFound = notFoundError{"canvas not found"}
	errPanelNotFound  = notFoundError{"panel not found"}
	errCanvasConflict = conflictError{"This canvas changed elsewhere. Reload to see the latest version."}
)

// normalizeCanvasName trims and validates; it never truncates.
func normalizeCanvasName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", invalid("name is required")
	}
	if !utf8.ValidString(name) {
		return "", invalid("name is not valid UTF-8")
	}
	if utf8.RuneCountInString(name) > MaxCanvasName {
		return "", invalid("name is too long (max %d characters)", MaxCanvasName)
	}
	return name, nil
}

func validateCompact(c string) error {
	if c != CanvasCompactVertical && c != CanvasCompactNone {
		return invalid("compact must be %s or %s", CanvasCompactVertical, CanvasCompactNone)
	}
	return nil
}

// Panel kinds (ADR-0108; the note/file/diff bodies are C3 of
// docs/plans/matrix-canvas.md). `kind` is an open text column, so a kind is
// a validator edit and never a migration. The store stays ignorant of what a
// ref points at — only its *shape* is a rule, exactly as a deleted terminal
// leaves its panel behind for the UI to render as gone.
const (
	CanvasKindAgent    = "agent"
	CanvasKindTerminal = "terminal"
	CanvasKindNote     = "note" // ref: a pin id
	CanvasKindFile     = "file" // ref: <owner>:<id>:<path>
	CanvasKindDiff     = "diff" // ref: <owner>:<id>:<path>
)

// The refusals are the contract, repeated word for word in
// web/shared/domain/canvas.js so the UI can refuse before asking.
const (
	canvasKindMsg = "kind must be agent, terminal, note, file or diff"
	canvasRefMsg  = "ref must be <owner>:<id>:<path> with owner t, a or w"
)

func validatePanelBinding(kind, ref string) error {
	switch kind {
	case CanvasKindAgent, CanvasKindTerminal, CanvasKindNote, CanvasKindFile, CanvasKindDiff:
	default:
		return invalid(canvasKindMsg)
	}
	if ref == "" {
		return invalid("ref is required")
	}
	if kind == CanvasKindFile || kind == CanvasKindDiff {
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
		return invalid(canvasRefMsg)
	}
	id, path, ok := strings.Cut(rest, ":")
	if !ok || (letter != "t" && letter != "a" && letter != "w") || id == "" || path == "" {
		return invalid(canvasRefMsg)
	}
	return nil
}

// validatePlacement is the rectangle rule of the plane (ADR-0118 left one):
// x and y may be negative, a panel is at least 32×28 units, and both are
// bounded so a panel cannot be lost off the edge. The web contract
// (web/shared/domain/canvas.js) repeats these words so the UI can refuse
// before asking.
func validatePlacement(x, y, w, h int) error {
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

// newPanelID is a panel id with 48 random bits: a canvas holds up to 500
// panels, and newID's 24-bit suffix on a fixed prefix would collide once in
// a few hundred canvases.
func newPanelID() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return "panel-" + hex.EncodeToString(b)
}

// canvasTx runs fn in one transaction: the events fn appends are announced
// on commit and dropped on rollback (ADR-0048). The store has one SQLite
// connection, so fn must read through tx, never s.db.
func (s *Store) canvasTx(fn func(tx *sql.Tx) error) error {
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

// canvasGuard reads the row's updated_at inside the transaction, before
// anything is written: a missing canvas is ErrNotFound; when the caller sent
// the updatedAt it last saw and the row moved on, ErrConflict. The store has
// one SQLite connection, so a second read through s.db inside the
// transaction would deadlock — every read a mutator needs goes through tx.
func canvasGuard(tx *sql.Tx, id, ifUpdatedAt string) error {
	var cur string
	err := tx.QueryRow(`SELECT updated_at FROM canvases WHERE id = ?`, id).Scan(&cur)
	if err == sql.ErrNoRows {
		return errCanvasNotFound
	}
	if err != nil {
		return fmt.Errorf("store: canvas: %w", err)
	}
	if ifUpdatedAt != "" && cur != ifUpdatedAt {
		return errCanvasConflict
	}
	return nil
}

func touchCanvas(tx *sql.Tx, id, now string) error {
	if _, err := tx.Exec(`UPDATE canvases SET updated_at = ? WHERE id = ?`, now, id); err != nil {
		return fmt.Errorf("store: touch canvas: %w", err)
	}
	return nil
}

const canvasCols = `id, name, compact, created_at, updated_at,
		(SELECT COUNT(1) FROM canvas_panels p WHERE p.canvas_id = canvases.id)`

func scanCanvas(row interface{ Scan(...any) error }, c *Canvas) error {
	return row.Scan(&c.ID, &c.Name, &c.Compact, &c.CreatedAt, &c.UpdatedAt, &c.PanelCount)
}

// rowQuerier is the intersection of *sql.DB and *sql.Tx the reads use.
type rowQuerier interface {
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
}

func canvasSummary(q rowQuerier, id string) (Canvas, error) {
	var c Canvas
	err := scanCanvas(q.QueryRow(`SELECT `+canvasCols+` FROM canvases WHERE id = ?`, id), &c)
	if err == sql.ErrNoRows {
		return Canvas{}, errCanvasNotFound
	}
	if err != nil {
		return Canvas{}, fmt.Errorf("store: canvas summary: %w", err)
	}
	return c, nil
}

func canvasPanels(q rowQuerier, id string) ([]CanvasPanel, error) {
	rows, err := q.Query(`SELECT id, kind, ref, x, y, w, h, created_at FROM canvas_panels WHERE canvas_id = ? ORDER BY y, x, created_at`, id)
	if err != nil {
		return nil, fmt.Errorf("store: canvas panels: %w", err)
	}
	defer rows.Close()
	out := []CanvasPanel{}
	for rows.Next() {
		var p CanvasPanel
		if err := rows.Scan(&p.ID, &p.Kind, &p.Ref, &p.X, &p.Y, &p.W, &p.H, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("store: scan panel: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// ListCanvases is the switcher's list: by name (ASCII case-folded), then
// by creation, so a drag never reorders the tabs.
func (s *Store) ListCanvases() ([]Canvas, error) {
	rows, err := s.db.Query(`SELECT ` + canvasCols + ` FROM canvases ORDER BY lower(name), created_at`)
	if err != nil {
		return nil, fmt.Errorf("store: list canvases: %w", err)
	}
	defer rows.Close()
	out := []Canvas{}
	for rows.Next() {
		var c Canvas
		if err := scanCanvas(rows, &c); err != nil {
			return nil, fmt.Errorf("store: scan canvas: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// GetCanvas is the summary plus every panel, ordered by row then column,
// plus every edge (ADR-0116) so one read still opens a canvas.
func (s *Store) GetCanvas(id string) (CanvasDetail, error) {
	var d CanvasDetail
	err := s.canvasTx(func(tx *sql.Tx) error {
		c, err := canvasSummary(tx, id)
		if err != nil {
			return err
		}
		panels, err := canvasPanels(tx, id)
		if err != nil {
			return err
		}
		edges, err := canvasEdges(tx, id)
		if err != nil {
			return err
		}
		c.PanelCount = len(panels)
		d = CanvasDetail{Canvas: c, Panels: panels, Edges: edges}
		return nil
	})
	return d, err
}

// CreateCanvas adds an empty canvas and announces canvas.created with the
// summary. The 65th is refused.
func (s *Store) CreateCanvas(name string) (Canvas, error) {
	name, err := normalizeCanvasName(name)
	if err != nil {
		return Canvas{}, err
	}
	var c Canvas
	err = s.canvasTx(func(tx *sql.Tx) error {
		var n int
		if err := tx.QueryRow(`SELECT COUNT(1) FROM canvases`).Scan(&n); err != nil {
			return fmt.Errorf("store: count canvases: %w", err)
		}
		if n >= MaxCanvases {
			return invalid("limit: %d canvases", MaxCanvases)
		}
		now := nowUTC()
		c = Canvas{ID: newID(name, "canvas"), Name: name, Compact: CanvasCompactVertical, CreatedAt: now, UpdatedAt: now}
		if _, err := tx.Exec(`INSERT INTO canvases (id, name, compact, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
			c.ID, c.Name, c.Compact, c.CreatedAt, c.UpdatedAt); err != nil {
			return fmt.Errorf("store: create canvas: %w", err)
		}
		return s.AppendEventTx(tx, "canvas.created", nil, nil, c)
	})
	return c, err
}

// UpdateCanvas renames and/or changes the compaction. ifUpdatedAt, when
// not empty, is the updatedAt the caller last saw: a row that moved on
// answers ErrConflict and nothing is written. Announces canvas.updated.
func (s *Store) UpdateCanvas(id string, p CanvasPatch, ifUpdatedAt string) (Canvas, error) {
	if p.Name == nil && p.Compact == nil {
		return Canvas{}, invalid("nothing to update: send name or compact")
	}
	var name, compact any // nil keeps the column (COALESCE)
	if p.Name != nil {
		n, err := normalizeCanvasName(*p.Name)
		if err != nil {
			return Canvas{}, err
		}
		name = n
	}
	if p.Compact != nil {
		if err := validateCompact(*p.Compact); err != nil {
			return Canvas{}, err
		}
		compact = *p.Compact
	}
	var c Canvas
	err := s.canvasTx(func(tx *sql.Tx) error {
		if err := canvasGuard(tx, id, ifUpdatedAt); err != nil {
			return err
		}
		if _, err := tx.Exec(`UPDATE canvases SET name = COALESCE(?, name), compact = COALESCE(?, compact), updated_at = ? WHERE id = ?`,
			name, compact, nowUTC(), id); err != nil {
			return fmt.Errorf("store: update canvas: %w", err)
		}
		var err error
		if c, err = canvasSummary(tx, id); err != nil {
			return err
		}
		return s.AppendEventTx(tx, "canvas.updated", nil, nil, c)
	})
	return c, err
}

// PatchCanvasLayout moves the listed panels — the changed subset, not the
// whole canvas — in one transaction: every id must be on this canvas and
// every rectangle legal, or nothing is written. One canvas.layout event
// carries exactly the subset.
func (s *Store) PatchCanvasLayout(id string, panels []PanelPlacement, ifUpdatedAt string) (CanvasLayout, error) {
	if len(panels) == 0 {
		return CanvasLayout{}, invalid("panels is required")
	}
	seen := make(map[string]bool, len(panels))
	for _, p := range panels {
		if p.ID == "" {
			return CanvasLayout{}, invalid("panel id is required")
		}
		if seen[p.ID] {
			return CanvasLayout{}, invalid("panel %s is listed twice", p.ID)
		}
		seen[p.ID] = true
	}
	var out CanvasLayout
	err := s.canvasTx(func(tx *sql.Tx) error {
		if err := canvasGuard(tx, id, ifUpdatedAt); err != nil {
			return err
		}
		// Every rectangle is checked before any row is written: one bad row
		// refuses the whole batch.
		for _, p := range panels {
			if err := validatePlacement(p.X, p.Y, p.W, p.H); err != nil {
				return fmt.Errorf("panel %s: %w", p.ID, err)
			}
		}
		for _, p := range panels {
			res, err := tx.Exec(`UPDATE canvas_panels SET x = ?, y = ?, w = ?, h = ? WHERE id = ? AND canvas_id = ?`,
				p.X, p.Y, p.W, p.H, p.ID, id)
			if err != nil {
				return fmt.Errorf("store: move panel: %w", err)
			}
			if n, _ := res.RowsAffected(); n == 0 {
				return invalid("panel %s is not on this canvas", p.ID)
			}
		}
		now := nowUTC()
		if err := touchCanvas(tx, id, now); err != nil {
			return err
		}
		out = CanvasLayout{ID: id, UpdatedAt: now, Panels: panels}
		return s.AppendEventTx(tx, "canvas.layout", nil, nil, out)
	})
	return out, err
}

// AddCanvasPanel binds an agent or a terminal to a new panel. The client
// places (x, y, w, h); the store validates the rectangle, the cap and the
// binding — one (kind, ref) per canvas, refused by the unique index and read
// back as ErrConflict. Announces canvas.panel.added.
func (s *Store) AddCanvasPanel(id, kind, ref string, x, y, w, h int) (CanvasPanelAdded, error) {
	ref = strings.TrimSpace(ref)
	if err := validatePanelBinding(kind, ref); err != nil {
		return CanvasPanelAdded{}, err
	}
	var out CanvasPanelAdded
	err := s.canvasTx(func(tx *sql.Tx) error {
		if err := canvasGuard(tx, id, ""); err != nil {
			return err
		}
		if err := validatePlacement(x, y, w, h); err != nil {
			return err
		}
		var n int
		if err := tx.QueryRow(`SELECT COUNT(1) FROM canvas_panels WHERE canvas_id = ?`, id).Scan(&n); err != nil {
			return fmt.Errorf("store: count panels: %w", err)
		}
		if n >= MaxCanvasPanels {
			return invalid("limit: %d panels per canvas", MaxCanvasPanels)
		}
		now := nowUTC()
		p := CanvasPanel{ID: newPanelID(), Kind: kind, Ref: ref, X: x, Y: y, W: w, H: h, CreatedAt: now}
		if _, err := tx.Exec(`INSERT INTO canvas_panels (id, canvas_id, kind, ref, x, y, w, h, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			p.ID, id, p.Kind, p.Ref, p.X, p.Y, p.W, p.H, p.CreatedAt); err != nil {
			// The unique index refused the binding; every other failure
			// keeps its real diagnostic.
			var dup int
			if e := tx.QueryRow(`SELECT COUNT(1) FROM canvas_panels WHERE canvas_id = ? AND kind = ? AND ref = ?`, id, kind, ref).Scan(&dup); e == nil && dup > 0 {
				return conflictError{fmt.Sprintf("This %s is already on this canvas", kind)}
			}
			return fmt.Errorf("store: add panel: %w", err)
		}
		if err := touchCanvas(tx, id, now); err != nil {
			return err
		}
		out = CanvasPanelAdded{ID: id, UpdatedAt: now, Panel: p}
		return s.AppendEventTx(tx, "canvas.panel.added", nil, nil, out)
	})
	return out, err
}

// RemoveCanvasPanel frees the panel and announces canvas.panel.removed.
func (s *Store) RemoveCanvasPanel(id, panelID string) error {
	return s.canvasTx(func(tx *sql.Tx) error {
		if err := canvasGuard(tx, id, ""); err != nil {
			return err
		}
		res, err := tx.Exec(`DELETE FROM canvas_panels WHERE id = ? AND canvas_id = ?`, panelID, id)
		if err != nil {
			return fmt.Errorf("store: remove panel: %w", err)
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return errPanelNotFound
		}
		now := nowUTC()
		if err := touchCanvas(tx, id, now); err != nil {
			return err
		}
		return s.AppendEventTx(tx, "canvas.panel.removed", nil, nil, CanvasPanelRemoved{ID: id, UpdatedAt: now, PanelID: panelID})
	})
}

// DeleteCanvas removes the canvas; its panels go with it (ON DELETE
// CASCADE) without an event each. Announces canvas.deleted {id}.
func (s *Store) DeleteCanvas(id string) error {
	return s.canvasTx(func(tx *sql.Tx) error {
		res, err := tx.Exec(`DELETE FROM canvases WHERE id = ?`, id)
		if err != nil {
			return fmt.Errorf("store: delete canvas: %w", err)
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return errCanvasNotFound
		}
		return s.AppendEventTx(tx, "canvas.deleted", nil, nil, idData(id))
	})
}
