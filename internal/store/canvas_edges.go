package store

// Canvas edges (ADR-0116). An edge is the owner's recorded intent that two
// sessions may exchange messages: it grants exactly ADR-0104's mailbox
// contact and nothing else. No transcript, no scrollback, no session file —
// not here, not anywhere. Only the owner API writes one (internal/server/
// canvas.go); the MCP surface (internal/communication) gains no verb, so an
// agent can neither create an edge nor discover that one could exist.
//
// The grant is **derived** on every contact read by peerEdgeContacts below
// and is never copied into peer_connections: deleting the canvas, either
// panel, or the edge removes it with no second write (migration 044
// cascades at both levels).

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
)

// CanvasEdge is one undirected link between two panels of one canvas. The
// pair is stored ordered (APanel < BPanel) so the same edge drawn either way
// collides on the unique index; a client may send the two ids in any order.
type CanvasEdge struct {
	ID        string `json:"id"`
	APanel    string `json:"aPanel"`
	BPanel    string `json:"bPanel"`
	CreatedAt string `json:"createdAt"`
}

// CanvasEdgeAdded is the canvas.edge.added event and the POST …/edges
// answer, in the shape canvas.panel.added uses.
type CanvasEdgeAdded struct {
	ID        string     `json:"id"`
	UpdatedAt string     `json:"updatedAt"`
	Edge      CanvasEdge `json:"edge"`
}

// CanvasEdgeRemoved is the canvas.edge.removed event.
type CanvasEdgeRemoved struct {
	ID        string `json:"id"`
	UpdatedAt string `json:"updatedAt"`
	EdgeID    string `json:"edgeId"`
}

var errEdgeNotFound = notFoundError{"edge not found"}

// The refusals are the contract, repeated word for word in
// web/shared/domain/canvas.js so the UI can refuse before asking.
const (
	canvasEdgeSameMsg  = "an edge needs two different panels"
	canvasEdgeBothMsg  = "aPanel and bPanel are required"
	canvasEdgeDupMsg   = "These panels are already linked"
	canvasEdgeKindsMsg = "an edge links agent or terminal panels"
)

// edgeHasMailbox is the kind rule: only a session has a mailbox. A note, a
// file or a diff is not one, so it cannot hold ADR-0104's contact.
func edgeHasMailbox(kind string) bool {
	return kind == CanvasKindAgent || kind == CanvasKindTerminal
}

// newEdgeID is a random link id, 48 bits like a panel's: a canvas holds up
// to MaxCanvasEdges of them.
func newEdgeID() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return "edge-" + hex.EncodeToString(b)
}

// orderEdge sorts the pair the way the table stores it, so the same edge
// drawn in either direction is the same row.
func orderEdge(a, b string) (string, string) {
	if a > b {
		return b, a
	}
	return a, b
}

func canvasEdges(q rowQuerier, id string) ([]CanvasEdge, error) {
	rows, err := q.Query(`SELECT id, a_panel, b_panel, created_at FROM canvas_edges WHERE canvas_id = ? ORDER BY created_at, id`, id)
	if err != nil {
		return nil, fmt.Errorf("store: canvas edges: %w", err)
	}
	defer rows.Close()
	out := []CanvasEdge{}
	for rows.Next() {
		var e CanvasEdge
		if err := rows.Scan(&e.ID, &e.APanel, &e.BPanel, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("store: scan edge: %w", err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// ListCanvasEdges is every edge of one canvas, oldest first. A canvas that
// does not exist is ErrNotFound, not an empty list.
func (s *Store) ListCanvasEdges(id string) ([]CanvasEdge, error) {
	var out []CanvasEdge
	err := s.canvasTx(func(tx *sql.Tx) error {
		if err := canvasGuard(tx, id, ""); err != nil {
			return err
		}
		var err error
		out, err = canvasEdges(tx, id)
		return err
	})
	return out, err
}

// edgePanelKind reads the kind of a panel that must be on this canvas. A
// panel of another canvas and a panel that does not exist are the same
// refusal: the caller sent an id this canvas does not hold.
func edgePanelKind(tx *sql.Tx, canvasID, panelID string) (string, error) {
	var kind string
	err := tx.QueryRow(`SELECT kind FROM canvas_panels WHERE id = ? AND canvas_id = ?`, panelID, canvasID).Scan(&kind)
	if err == sql.ErrNoRows {
		return "", invalid("panel %s is not on this canvas", panelID)
	}
	if err != nil {
		return "", fmt.Errorf("store: panel kind: %w", err)
	}
	return kind, nil
}

// AddCanvasEdge links two panels of one canvas. Both must be on this
// canvas, be different, and be an `agent` or a `terminal` panel — a note has
// no mailbox, so the refusal names the kind. The pair is ordered before the
// write, so drawing the same edge backwards is the 409 the unique index
// refuses. Announces canvas.edge.added with the payload the route answers.
func (s *Store) AddCanvasEdge(id, aPanel, bPanel string) (CanvasEdgeAdded, error) {
	if aPanel == "" || bPanel == "" {
		return CanvasEdgeAdded{}, invalid(canvasEdgeBothMsg)
	}
	if aPanel == bPanel {
		return CanvasEdgeAdded{}, invalid(canvasEdgeSameMsg)
	}
	var out CanvasEdgeAdded
	err := s.canvasTx(func(tx *sql.Tx) error {
		if err := canvasGuard(tx, id, ""); err != nil {
			return err
		}
		a, b := orderEdge(aPanel, bPanel)
		for _, p := range []string{a, b} {
			kind, err := edgePanelKind(tx, id, p)
			if err != nil {
				return err
			}
			if !edgeHasMailbox(kind) {
				return invalid("panel %s is a %s panel and has no mailbox: %s", p, kind, canvasEdgeKindsMsg)
			}
		}
		var n int
		if err := tx.QueryRow(`SELECT COUNT(1) FROM canvas_edges WHERE canvas_id = ?`, id).Scan(&n); err != nil {
			return fmt.Errorf("store: count edges: %w", err)
		}
		if n >= MaxCanvasEdges {
			return invalid("limit: %d edges per canvas", MaxCanvasEdges)
		}
		now := nowUTC()
		e := CanvasEdge{ID: newEdgeID(), APanel: a, BPanel: b, CreatedAt: now}
		if _, err := tx.Exec(`INSERT INTO canvas_edges (id, canvas_id, a_panel, b_panel, created_at) VALUES (?, ?, ?, ?, ?)`,
			e.ID, id, e.APanel, e.BPanel, e.CreatedAt); err != nil {
			// The unique index refused the pair; every other failure keeps
			// its real diagnostic.
			var dup int
			if x := tx.QueryRow(`SELECT COUNT(1) FROM canvas_edges WHERE canvas_id = ? AND a_panel = ? AND b_panel = ?`, id, e.APanel, e.BPanel).Scan(&dup); x == nil && dup > 0 {
				return conflictError{canvasEdgeDupMsg}
			}
			return fmt.Errorf("store: add edge: %w", err)
		}
		if err := touchCanvas(tx, id, now); err != nil {
			return err
		}
		out = CanvasEdgeAdded{ID: id, UpdatedAt: now, Edge: e}
		return s.AppendEventTx(tx, "canvas.edge.added", nil, nil, out)
	})
	return out, err
}

// RemoveCanvasEdge deletes one link and, with it, the grant it carried —
// the next contact read derives nothing from a row that is gone. Announces
// canvas.edge.removed.
func (s *Store) RemoveCanvasEdge(id, edgeID string) error {
	return s.canvasTx(func(tx *sql.Tx) error {
		if err := canvasGuard(tx, id, ""); err != nil {
			return err
		}
		res, err := tx.Exec(`DELETE FROM canvas_edges WHERE id = ? AND canvas_id = ?`, edgeID, id)
		if err != nil {
			return fmt.Errorf("store: remove edge: %w", err)
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return errEdgeNotFound
		}
		now := nowUTC()
		if err := touchCanvas(tx, id, now); err != nil {
			return err
		}
		return s.AppendEventTx(tx, "canvas.edge.removed", nil, nil, CanvasEdgeRemoved{ID: id, UpdatedAt: now, EdgeID: edgeID})
	})
}

// ---- the contact union (ADR-0116 §3) -------------------------------------
//
// PeerContacts is the union of *same workspace* (ADR-0104, unchanged) and
// *linked by a live edge*. This is the whole of the second half: one join
// that walks an edge from the caller's own panel to the panel at the far
// end, and resolves that panel's (kind, ref) against peer_connections —
// `agent` matches agent_id, `terminal` matches terminal_id. Every row it
// returns is still filtered by peerCurrent, so a revoked connection or a
// session that moved contributes nothing.
//
// Nothing here writes. If this decision is ever reversed, the table and
// these two queries go and the mailbox is back to workspace scope, with no
// migration of anything else — because nothing was ever copied.

// peerEdgeJoin walks every edge that touches a panel bound to (kind, ref)
// — the caller — and lands on the connection at the far end. Both ends are
// live panel rows by construction: migration 044 cascades an edge away with
// either panel, so a removed panel grants nothing.
const peerEdgeJoin = ` FROM canvas_edges e
 JOIN canvas_panels mine ON mine.id IN (e.a_panel, e.b_panel) AND mine.kind = ? AND mine.ref = ?
 JOIN canvas_panels far ON far.id = CASE WHEN mine.id = e.a_panel THEN e.b_panel ELSE e.a_panel END
 JOIN peer_connections p ON far.kind = CASE WHEN p.agent_id IS NOT NULL THEN 'agent' ELSE 'terminal' END
  AND far.ref = COALESCE(p.agent_id, p.terminal_id)
 WHERE p.revoked_at IS NULL AND p.id <> ?`

// peerEdgeConnCols is peerCols qualified: the join carries canvas_panels,
// which has an id and a created_at of its own.
const peerEdgeConnCols = `p.id,CASE WHEN p.agent_id IS NOT NULL THEN 'agent' ELSE 'terminal' END,COALESCE(p.agent_id,p.terminal_id),p.workspace_id,p.label,p.cli,p.session_key,p.created_at,p.revoked_at`

// peerEdgeContacts lists the connections a live edge links to me, deduped
// (two canvases may draw the same pair; it is one contact). Ordering and the
// peerCurrent filter are the caller's — PeerContacts merges this with the
// workspace list before either.
func peerEdgeContacts(q rowQuerier, me PeerConnection) ([]PeerConnection, error) {
	rows, err := q.Query(`SELECT DISTINCT `+peerEdgeConnCols+peerEdgeJoin, me.Kind, me.PeerOwner.OwnerID, me.ID)
	if err != nil {
		return nil, err
	}
	out := []PeerConnection{}
	for rows.Next() {
		p, e := scanPeer(rows)
		if e != nil {
			rows.Close()
			return nil, e
		}
		out = append(out, p)
	}
	err = rows.Err()
	rows.Close()
	return out, err
}

// peerEdgeLinks answers whether one live edge joins these two connections.
// It is the same walk as peerEdgeContacts with the far end pinned, so a
// send and a contact list can never disagree about who may be reached.
func peerEdgeLinks(q txRunner, me PeerConnection, otherID string) (bool, error) {
	var one int
	err := q.QueryRow(`SELECT 1`+peerEdgeJoin+` AND p.id = ? LIMIT 1`, me.Kind, me.PeerOwner.OwnerID, me.ID, otherID).Scan(&one)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}
