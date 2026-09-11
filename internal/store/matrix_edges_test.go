package store

import (
	"errors"
	"fmt"
	"path/filepath"
	"testing"
)

// ADR-0116's decision table, row by row. An edge grants exactly ADR-0104's
// mailbox contact: these tests prove what it grants, what it refuses, and
// that the grant dies with the line — it is derived on every read and never
// written into peer_connections.

// edgePeer enrols the default agent of a fresh workspace and answers the
// connection, its bearer and the agent's id (the panel `ref` that binds to
// it). Each call gets its own workspace, so two of them are exactly the
// cross-workspace pair the mailbox refuses without an edge.
func edgePeer(t *testing.T, s *Store, name string) (PeerConnection, string, string) {
	t.Helper()
	w, a, err := addWorkspaceWithAgent(s, name, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	session := filepath.Join(w.Path, name+".jsonl")
	if _, err = s.UpdateAgent(a.ID, AgentPatch{SessionPath: &session}); err != nil {
		t.Fatal(err)
	}
	p, token, err := s.EnablePeer("agent", a.ID, session)
	if err != nil {
		t.Fatal(err)
	}
	return p, token, a.ID
}

// linkedPair is one matrix with a panel for each of two enrolled sessions in
// *different* workspaces, joined by one edge — the union's whole point.
type linkedPair struct {
	s                 *Store
	left, right       PeerConnection
	leftTok, rightTok string
	leftAgent         string
	rightAgent        string
	matrix            string
	leftPanel         string
	rightPanel        string
	edge              string
}

func linkPair(t *testing.T, s *Store) linkedPair {
	t.Helper()
	left, leftTok, leftAgent := edgePeer(t, s, "Left")
	right, rightTok, rightAgent := edgePeer(t, s, "Right")
	if left.WorkspaceID == right.WorkspaceID {
		t.Fatal("fixture: both peers landed in one workspace")
	}
	m, err := s.CreateMatrix("Pairing")
	if err != nil {
		t.Fatal(err)
	}
	lp, err := s.AddMatrixPanel(m.ID, MatrixKindAgent, leftAgent, 0, 0, 4, 8)
	if err != nil {
		t.Fatal(err)
	}
	rp, err := s.AddMatrixPanel(m.ID, MatrixKindAgent, rightAgent, 4, 0, 4, 8)
	if err != nil {
		t.Fatal(err)
	}
	e, err := s.AddMatrixEdge(m.ID, lp.Panel.ID, rp.Panel.ID)
	if err != nil {
		t.Fatal(err)
	}
	return linkedPair{
		s: s, left: left, right: right, leftTok: leftTok, rightTok: rightTok,
		leftAgent: leftAgent, rightAgent: rightAgent,
		matrix: m.ID, leftPanel: lp.Panel.ID, rightPanel: rp.Panel.ID, edge: e.Edge.ID,
	}
}

func contactIDs(t *testing.T, s *Store, token string) []string {
	t.Helper()
	list, err := s.PeerContacts(token)
	if err != nil {
		t.Fatalf("PeerContacts: %v", err)
	}
	out := make([]string, 0, len(list))
	for _, p := range list {
		out = append(out, p.ID)
	}
	return out
}

func wantContacts(t *testing.T, s *Store, token string, want ...string) {
	t.Helper()
	got := contactIDs(t, s, token)
	if len(got) != len(want) {
		t.Fatalf("contacts = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("contacts = %v, want %v", got, want)
		}
	}
}

// Row: add an edge between two agent/terminal panels of one matrix → the
// row, the event, and both directions dedupe to one row (the reverse is a
// 409 naming the rule).
func TestAddMatrixEdgeOrdersThePairAndRefusesTheReverse(t *testing.T) {
	s := openTest(t)
	m, err := s.CreateMatrix("Ops")
	if err != nil {
		t.Fatal(err)
	}
	a, err := s.AddMatrixPanel(m.ID, MatrixKindTerminal, "t-1", 0, 0, 4, 8)
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.AddMatrixPanel(m.ID, MatrixKindAgent, "a-1", 4, 0, 4, 8)
	if err != nil {
		t.Fatal(err)
	}
	evs := recordMatrixEvents(s)
	// Drawn from b to a: the store stores the pair ordered.
	added, err := s.AddMatrixEdge(m.ID, b.Panel.ID, a.Panel.ID)
	if err != nil {
		t.Fatalf("AddMatrixEdge: %v", err)
	}
	lo, hi := orderEdge(a.Panel.ID, b.Panel.ID)
	if added.Edge.APanel != lo || added.Edge.BPanel != hi {
		t.Fatalf("edge = %+v, want ordered (%s, %s)", added.Edge, lo, hi)
	}
	if added.ID != m.ID || added.UpdatedAt == "" || added.UpdatedAt != added.Edge.CreatedAt {
		t.Fatalf("answer = %+v", added)
	}
	var payload MatrixEdgeAdded
	decodeEvent(t, lastEvent(t, evs, "matrix.edge.added"), &payload)
	if payload.Edge.ID != added.Edge.ID || payload.UpdatedAt != added.UpdatedAt {
		t.Fatalf("event = %+v, want %+v", payload, added)
	}

	// Row: the same edge reversed → 409 "already linked", still one row.
	if _, err = s.AddMatrixEdge(m.ID, a.Panel.ID, b.Panel.ID); !errors.Is(err, ErrConflict) || err.Error() != matrixEdgeDupMsg {
		t.Fatalf("reverse = %v, want conflict %q", err, matrixEdgeDupMsg)
	}
	list, err := s.ListMatrixEdges(m.ID)
	if err != nil || len(list) != 1 || list[0].ID != added.Edge.ID {
		t.Fatalf("edges = %v %v", list, err)
	}
	if n := countType(evs, "matrix.edge.added"); n != 1 {
		t.Fatalf("edge.added events = %d, want 1", n)
	}
	// One read still opens a matrix: the detail carries the edges.
	d, err := s.GetMatrix(m.ID)
	if err != nil || len(d.Edges) != 1 || d.Edges[0].ID != added.Edge.ID {
		t.Fatalf("detail edges = %v %v", d.Edges, err)
	}
}

// Rows: same panel twice, a panel of another matrix, a kind with no mailbox
// (the message names the kind), and the two ids that are required.
func TestAddMatrixEdgeRefusals(t *testing.T) {
	s := openTest(t)
	m, err := s.CreateMatrix("Ops")
	if err != nil {
		t.Fatal(err)
	}
	other, err := s.CreateMatrix("Other")
	if err != nil {
		t.Fatal(err)
	}
	mk := func(matrix, kind, ref string, x int) string {
		t.Helper()
		p, err := s.AddMatrixPanel(matrix, kind, ref, x, 0, 4, 8)
		if err != nil {
			t.Fatal(err)
		}
		return p.Panel.ID
	}
	term := mk(m.ID, MatrixKindTerminal, "t-1", 0)
	agent := mk(m.ID, MatrixKindAgent, "a-1", 4)
	note := mk(m.ID, MatrixKindNote, "pin-1", 8)
	file := mk(m.ID, MatrixKindFile, "t:term-1:/a.go", 0)
	elsewhere := mk(other.ID, MatrixKindAgent, "a-2", 0)

	for _, tc := range []struct {
		name, a, b, want string
	}{
		{"both ids required", term, "", matrixEdgeBothMsg},
		{"a panel to itself", term, term, matrixEdgeSameMsg},
		{"a panel from another matrix", term, elsewhere, "panel " + elsewhere + " is not on this matrix"},
		{"a panel that does not exist", term, "panel-gone", "panel panel-gone is not on this matrix"},
		{"a note panel has no mailbox", agent, note, "panel " + note + " is a note panel and has no mailbox: " + matrixEdgeKindsMsg},
		{"a file panel has no mailbox", agent, file, "panel " + file + " is a file panel and has no mailbox: " + matrixEdgeKindsMsg},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := s.AddMatrixEdge(m.ID, tc.a, tc.b)
			if !errors.Is(err, ErrInvalid) || err.Error() != tc.want {
				t.Fatalf("err = %v, want invalid %q", err, tc.want)
			}
		})
	}
	// A matrix that does not exist is ErrNotFound, not a refusal about panels.
	if _, err = s.AddMatrixEdge("nope", term, agent); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown matrix = %v", err)
	}
	list, err := s.ListMatrixEdges(m.ID)
	if err != nil || len(list) != 0 {
		t.Fatalf("refusals wrote a row: %v %v", list, err)
	}
}

// Row: the cap is reached → 400 naming the cap. 46 panels make 1035 pairs,
// so the fill runs through the store — the cap counts rows the store wrote,
// and the pair asked for afterwards is one it has never seen.
func TestAddMatrixEdgeCap(t *testing.T) {
	s := openTest(t)
	m, err := s.CreateMatrix("Ops")
	if err != nil {
		t.Fatal(err)
	}
	const n = 46 // n*(n-1)/2 = 1035 pairs > MaxMatrixEdges
	panels := make([]string, 0, n)
	for i := range n {
		p, err := s.AddMatrixPanel(m.ID, MatrixKindTerminal, fmt.Sprintf("t-%d", i), 0, 0, 4, 8)
		if err != nil {
			t.Fatalf("panel %d: %v", i, err)
		}
		panels = append(panels, p.Panel.ID)
	}
	drawn := 0
	for i := 0; i < n && drawn < MaxMatrixEdges; i++ {
		for j := i + 1; j < n && drawn < MaxMatrixEdges; j++ {
			if _, err := s.AddMatrixEdge(m.ID, panels[i], panels[j]); err != nil {
				t.Fatalf("fill (%d,%d): %v", i, j, err)
			}
			drawn++
		}
	}
	// The last pair in that order is still free, so the refusal is the cap
	// and not the duplicate.
	_, err = s.AddMatrixEdge(m.ID, panels[n-2], panels[n-1])
	want := fmt.Sprintf("limit: %d edges per matrix", MaxMatrixEdges)
	if !errors.Is(err, ErrInvalid) || err.Error() != want {
		t.Fatalf("at the cap = %v, want invalid %q", err, want)
	}
	list, err := s.ListMatrixEdges(m.ID)
	if err != nil || len(list) != MaxMatrixEdges {
		t.Fatalf("edges = %d %v, want %d", len(list), err, MaxMatrixEdges)
	}
}

// RemoveMatrixEdge: the row, the event, the bumped updatedAt, and a second
// removal that is ErrNotFound rather than a silent success.
func TestRemoveMatrixEdge(t *testing.T) {
	s := openTest(t)
	p := linkPair(t, s)
	evs := recordMatrixEvents(s)
	if err := s.RemoveMatrixEdge(p.matrix, p.edge); err != nil {
		t.Fatalf("RemoveMatrixEdge: %v", err)
	}
	var payload MatrixEdgeRemoved
	decodeEvent(t, lastEvent(t, evs, "matrix.edge.removed"), &payload)
	if payload.ID != p.matrix || payload.EdgeID != p.edge || payload.UpdatedAt == "" {
		t.Fatalf("event = %+v", payload)
	}
	if err := s.RemoveMatrixEdge(p.matrix, p.edge); !errors.Is(err, ErrNotFound) {
		t.Fatalf("second remove = %v, want not found", err)
	}
	list, err := s.ListMatrixEdges(p.matrix)
	if err != nil || len(list) != 0 {
		t.Fatalf("edges = %v %v", list, err)
	}
}

// Rows: a removed panel and a deleted matrix take their edges with them —
// migration 044 cascades at both levels, so no reader has to remember.
func TestMatrixEdgesCascade(t *testing.T) {
	t.Run("a removed panel takes its edges", func(t *testing.T) {
		s := openTest(t)
		p := linkPair(t, s)
		if err := s.RemoveMatrixPanel(p.matrix, p.leftPanel); err != nil {
			t.Fatal(err)
		}
		list, err := s.ListMatrixEdges(p.matrix)
		if err != nil || len(list) != 0 {
			t.Fatalf("edges after panel removal = %v %v", list, err)
		}
	})
	t.Run("a deleted matrix takes its edges", func(t *testing.T) {
		s := openTest(t)
		p := linkPair(t, s)
		if err := s.DeleteMatrix(p.matrix); err != nil {
			t.Fatal(err)
		}
		var n int
		if err := s.db.QueryRow(`SELECT COUNT(1) FROM matrix_edges`).Scan(&n); err != nil || n != 0 {
			t.Fatalf("edges after matrix deletion = %d %v", n, err)
		}
	})
}

// The contact union (ADR-0116 §3), row by row. Every row is a *read*: the
// grant is derived here and nowhere else, so each row ends by asking the
// mailbox who it can reach.
func TestPeerContactsUnion(t *testing.T) {
	t.Run("same workspace, no edge: unchanged", func(t *testing.T) {
		s := openTest(t)
		a, at, b, bt := peerFixture(t, s)
		wantContacts(t, s, at, b.ID)
		wantContacts(t, s, bt, a.ID)
		// And no edge was needed for it.
		var n int
		if err := s.db.QueryRow(`SELECT COUNT(1) FROM matrix_edges`).Scan(&n); err != nil || n != 0 {
			t.Fatalf("edges = %d %v", n, err)
		}
	})

	t.Run("different workspaces, live edge: each side sees the other once", func(t *testing.T) {
		s := openTest(t)
		p := linkPair(t, s)
		wantContacts(t, s, p.leftTok, p.right.ID)
		wantContacts(t, s, p.rightTok, p.left.ID)
	})

	t.Run("different workspaces, no edge: neither side sees the other", func(t *testing.T) {
		s := openTest(t)
		_, leftTok, _ := edgePeer(t, s, "Left")
		_, rightTok, _ := edgePeer(t, s, "Right")
		wantContacts(t, s, leftTok)
		wantContacts(t, s, rightTok)
	})

	t.Run("edge removed: neither side sees the other", func(t *testing.T) {
		s := openTest(t)
		p := linkPair(t, s)
		if err := s.RemoveMatrixEdge(p.matrix, p.edge); err != nil {
			t.Fatal(err)
		}
		wantContacts(t, s, p.leftTok)
		wantContacts(t, s, p.rightTok)
	})

	t.Run("edge present, one connection revoked: no contact", func(t *testing.T) {
		s := openTest(t)
		p := linkPair(t, s)
		if err := s.RevokePeer(p.right.ID); err != nil {
			t.Fatal(err)
		}
		wantContacts(t, s, p.leftTok)
	})

	t.Run("edge present, one session changed: no contact", func(t *testing.T) {
		s := openTest(t)
		p := linkPair(t, s)
		moved := filepath.Join(t.TempDir(), "moved.jsonl")
		if _, err := s.UpdateAgent(p.rightAgent, AgentPatch{SessionPath: &moved}); err != nil {
			t.Fatal(err)
		}
		wantContacts(t, s, p.leftTok)
	})

	t.Run("edge present, one panel removed: no contact", func(t *testing.T) {
		s := openTest(t)
		p := linkPair(t, s)
		if err := s.RemoveMatrixPanel(p.matrix, p.rightPanel); err != nil {
			t.Fatal(err)
		}
		wantContacts(t, s, p.leftTok)
		wantContacts(t, s, p.rightTok)
	})

	t.Run("matrix deleted: no contact, edges gone", func(t *testing.T) {
		s := openTest(t)
		p := linkPair(t, s)
		if err := s.DeleteMatrix(p.matrix); err != nil {
			t.Fatal(err)
		}
		wantContacts(t, s, p.leftTok)
		wantContacts(t, s, p.rightTok)
	})

	t.Run("two edges in two matrices for the same pair: listed once", func(t *testing.T) {
		s := openTest(t)
		p := linkPair(t, s)
		second, err := s.CreateMatrix("Second")
		if err != nil {
			t.Fatal(err)
		}
		l, err := s.AddMatrixPanel(second.ID, MatrixKindAgent, p.leftAgent, 0, 0, 4, 8)
		if err != nil {
			t.Fatal(err)
		}
		r, err := s.AddMatrixPanel(second.ID, MatrixKindAgent, p.rightAgent, 4, 0, 4, 8)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = s.AddMatrixEdge(second.ID, l.Panel.ID, r.Panel.ID); err != nil {
			t.Fatal(err)
		}
		wantContacts(t, s, p.leftTok, p.right.ID)
		wantContacts(t, s, p.rightTok, p.left.ID)
	})

	t.Run("edge to a panel whose target was deleted from the fleet: no contact", func(t *testing.T) {
		s := openTest(t)
		p := linkPair(t, s)
		if err := s.DeleteAgent(p.rightAgent); err != nil {
			t.Fatal(err)
		}
		// The panel outlives its target (no foreign key to agents, ADR-0108),
		// and the connection cascades away with the agent, so the edge
		// resolves to nobody.
		list, err := s.ListMatrixEdges(p.matrix)
		if err != nil || len(list) != 1 {
			t.Fatalf("edges = %v %v", list, err)
		}
		wantContacts(t, s, p.leftTok)
	})

	t.Run("an edge between two panels of my own session grants nothing", func(t *testing.T) {
		s := openTest(t)
		left, leftTok, leftAgent := edgePeer(t, s, "Left")
		_, _, rightAgent := edgePeer(t, s, "Right")
		m, err := s.CreateMatrix("Self")
		if err != nil {
			t.Fatal(err)
		}
		a, err := s.AddMatrixPanel(m.ID, MatrixKindAgent, leftAgent, 0, 0, 4, 8)
		if err != nil {
			t.Fatal(err)
		}
		b, err := s.AddMatrixPanel(m.ID, MatrixKindAgent, rightAgent, 4, 0, 4, 8)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = s.AddMatrixEdge(m.ID, a.Panel.ID, b.Panel.ID); err != nil {
			t.Fatal(err)
		}
		// The far end is somebody else; my own connection is never in my list.
		for _, id := range contactIDs(t, s, leftTok) {
			if id == left.ID {
				t.Fatal("the caller listed itself")
			}
		}
	})
}

// The grant is a *contact*, so the send obeys the same union — a listed
// contact the caller cannot write to would be a line that means nothing.
// And no edge ever writes into peer_connections.
func TestPeerSendFollowsTheEdge(t *testing.T) {
	s := openTest(t)
	p := linkPair(t, s)
	if _, err := s.SendPeerMessage(p.leftTok, p.right.ID, "r1", "hello", ""); err != nil {
		t.Fatalf("send across a live edge: %v", err)
	}
	if _, err := s.SendPeerMessage(p.rightTok, p.left.ID, "r2", "hi back", ""); err != nil {
		t.Fatalf("send back across a live edge: %v", err)
	}
	if err := s.RemoveMatrixEdge(p.matrix, p.edge); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SendPeerMessage(p.leftTok, p.right.ID, "r3", "still there?", ""); !errors.Is(err, ErrPeerDenied) {
		t.Fatalf("send after the edge was removed = %v, want denied", err)
	}
	// Nothing about the grant was cached: the connection rows are exactly
	// what ADR-0104 wrote, in their own workspaces.
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(1) FROM peer_connections WHERE workspace_id = ?`, p.left.WorkspaceID).Scan(&n); err != nil || n != 1 {
		t.Fatalf("left workspace connections = %d %v", n, err)
	}
}
