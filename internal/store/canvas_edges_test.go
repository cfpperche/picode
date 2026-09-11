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

// linkedPair is one canvas with a panel for each of two enrolled sessions in
// *different* workspaces, joined by one edge — the union's whole point.
type linkedPair struct {
	s                 *Store
	left, right       PeerConnection
	leftTok, rightTok string
	leftAgent         string
	rightAgent        string
	canvas            string
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
	m, err := s.CreateCanvas("Pairing")
	if err != nil {
		t.Fatal(err)
	}
	lp, err := s.AddCanvasPanel(m.ID, CanvasKindAgent, leftAgent, 0, 0, 32, 28)
	if err != nil {
		t.Fatal(err)
	}
	rp, err := s.AddCanvasPanel(m.ID, CanvasKindAgent, rightAgent, 40, 0, 32, 28)
	if err != nil {
		t.Fatal(err)
	}
	e, err := s.AddCanvasEdge(m.ID, lp.Panel.ID, rp.Panel.ID)
	if err != nil {
		t.Fatal(err)
	}
	return linkedPair{
		s: s, left: left, right: right, leftTok: leftTok, rightTok: rightTok,
		leftAgent: leftAgent, rightAgent: rightAgent,
		canvas: m.ID, leftPanel: lp.Panel.ID, rightPanel: rp.Panel.ID, edge: e.Edge.ID,
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

// Row: add an edge between two agent/terminal panels of one canvas → the
// row, the event, and both directions dedupe to one row (the reverse is a
// 409 naming the rule).
func TestAddCanvasEdgeOrdersThePairAndRefusesTheReverse(t *testing.T) {
	s := openTest(t)
	m, err := s.CreateCanvas("Ops")
	if err != nil {
		t.Fatal(err)
	}
	a, err := s.AddCanvasPanel(m.ID, CanvasKindTerminal, "t-1", 0, 0, 32, 28)
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.AddCanvasPanel(m.ID, CanvasKindAgent, "a-1", 40, 0, 32, 28)
	if err != nil {
		t.Fatal(err)
	}
	evs := recordCanvasEvents(s)
	// Drawn from b to a: the store stores the pair ordered.
	added, err := s.AddCanvasEdge(m.ID, b.Panel.ID, a.Panel.ID)
	if err != nil {
		t.Fatalf("AddCanvasEdge: %v", err)
	}
	lo, hi := orderEdge(a.Panel.ID, b.Panel.ID)
	if added.Edge.APanel != lo || added.Edge.BPanel != hi {
		t.Fatalf("edge = %+v, want ordered (%s, %s)", added.Edge, lo, hi)
	}
	if added.ID != m.ID || added.UpdatedAt == "" || added.UpdatedAt != added.Edge.CreatedAt {
		t.Fatalf("answer = %+v", added)
	}
	var payload CanvasEdgeAdded
	decodeEvent(t, lastEvent(t, evs, "canvas.edge.added"), &payload)
	if payload.Edge.ID != added.Edge.ID || payload.UpdatedAt != added.UpdatedAt {
		t.Fatalf("event = %+v, want %+v", payload, added)
	}

	// Row: the same edge reversed → 409 "already linked", still one row.
	if _, err = s.AddCanvasEdge(m.ID, a.Panel.ID, b.Panel.ID); !errors.Is(err, ErrConflict) || err.Error() != canvasEdgeDupMsg {
		t.Fatalf("reverse = %v, want conflict %q", err, canvasEdgeDupMsg)
	}
	list, err := s.ListCanvasEdges(m.ID)
	if err != nil || len(list) != 1 || list[0].ID != added.Edge.ID {
		t.Fatalf("edges = %v %v", list, err)
	}
	if n := countType(evs, "canvas.edge.added"); n != 1 {
		t.Fatalf("edge.added events = %d, want 1", n)
	}
	// One read still opens a canvas: the detail carries the edges.
	d, err := s.GetCanvas(m.ID)
	if err != nil || len(d.Edges) != 1 || d.Edges[0].ID != added.Edge.ID {
		t.Fatalf("detail edges = %v %v", d.Edges, err)
	}
}

// Rows: same panel twice, a panel of another canvas, a kind with no mailbox
// (the message names the kind), and the two ids that are required.
func TestAddCanvasEdgeRefusals(t *testing.T) {
	s := openTest(t)
	m, err := s.CreateCanvas("Ops")
	if err != nil {
		t.Fatal(err)
	}
	other, err := s.CreateCanvas("Other")
	if err != nil {
		t.Fatal(err)
	}
	mk := func(canvas, kind, ref string, x int) string {
		t.Helper()
		p, err := s.AddCanvasPanel(canvas, kind, ref, x, 0, 32, 28)
		if err != nil {
			t.Fatal(err)
		}
		return p.Panel.ID
	}
	term := mk(m.ID, CanvasKindTerminal, "t-1", 0)
	agent := mk(m.ID, CanvasKindAgent, "a-1", 4)
	note := mk(m.ID, CanvasKindNote, "pin-1", 8)
	file := mk(m.ID, CanvasKindFile, "t:term-1:/a.go", 0)
	elsewhere := mk(other.ID, CanvasKindAgent, "a-2", 0)

	for _, tc := range []struct {
		name, a, b, want string
	}{
		{"both ids required", term, "", canvasEdgeBothMsg},
		{"a panel to itself", term, term, canvasEdgeSameMsg},
		{"a panel from another canvas", term, elsewhere, "panel " + elsewhere + " is not on this canvas"},
		{"a panel that does not exist", term, "panel-gone", "panel panel-gone is not on this canvas"},
		{"a note panel has no mailbox", agent, note, "panel " + note + " is a note panel and has no mailbox: " + canvasEdgeKindsMsg},
		{"a file panel has no mailbox", agent, file, "panel " + file + " is a file panel and has no mailbox: " + canvasEdgeKindsMsg},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := s.AddCanvasEdge(m.ID, tc.a, tc.b)
			if !errors.Is(err, ErrInvalid) || err.Error() != tc.want {
				t.Fatalf("err = %v, want invalid %q", err, tc.want)
			}
		})
	}
	// A canvas that does not exist is ErrNotFound, not a refusal about panels.
	if _, err = s.AddCanvasEdge("nope", term, agent); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown canvas = %v", err)
	}
	list, err := s.ListCanvasEdges(m.ID)
	if err != nil || len(list) != 0 {
		t.Fatalf("refusals wrote a row: %v %v", list, err)
	}
}

// Row: the cap is reached → 400 naming the cap. 46 panels make 1035 pairs,
// so the fill runs through the store — the cap counts rows the store wrote,
// and the pair asked for afterwards is one it has never seen.
func TestAddCanvasEdgeCap(t *testing.T) {
	s := openTest(t)
	m, err := s.CreateCanvas("Ops")
	if err != nil {
		t.Fatal(err)
	}
	const n = 46 // n*(n-1)/2 = 1035 pairs > MaxCanvasEdges
	panels := make([]string, 0, n)
	for i := range n {
		p, err := s.AddCanvasPanel(m.ID, CanvasKindTerminal, fmt.Sprintf("t-%d", i), 0, 0, 32, 28)
		if err != nil {
			t.Fatalf("panel %d: %v", i, err)
		}
		panels = append(panels, p.Panel.ID)
	}
	drawn := 0
	for i := 0; i < n && drawn < MaxCanvasEdges; i++ {
		for j := i + 1; j < n && drawn < MaxCanvasEdges; j++ {
			if _, err := s.AddCanvasEdge(m.ID, panels[i], panels[j]); err != nil {
				t.Fatalf("fill (%d,%d): %v", i, j, err)
			}
			drawn++
		}
	}
	// The last pair in that order is still free, so the refusal is the cap
	// and not the duplicate.
	_, err = s.AddCanvasEdge(m.ID, panels[n-2], panels[n-1])
	want := fmt.Sprintf("limit: %d edges per canvas", MaxCanvasEdges)
	if !errors.Is(err, ErrInvalid) || err.Error() != want {
		t.Fatalf("at the cap = %v, want invalid %q", err, want)
	}
	list, err := s.ListCanvasEdges(m.ID)
	if err != nil || len(list) != MaxCanvasEdges {
		t.Fatalf("edges = %d %v, want %d", len(list), err, MaxCanvasEdges)
	}
}

// RemoveCanvasEdge: the row, the event, the bumped updatedAt, and a second
// removal that is ErrNotFound rather than a silent success.
func TestRemoveCanvasEdge(t *testing.T) {
	s := openTest(t)
	p := linkPair(t, s)
	evs := recordCanvasEvents(s)
	if err := s.RemoveCanvasEdge(p.canvas, p.edge); err != nil {
		t.Fatalf("RemoveCanvasEdge: %v", err)
	}
	var payload CanvasEdgeRemoved
	decodeEvent(t, lastEvent(t, evs, "canvas.edge.removed"), &payload)
	if payload.ID != p.canvas || payload.EdgeID != p.edge || payload.UpdatedAt == "" {
		t.Fatalf("event = %+v", payload)
	}
	if err := s.RemoveCanvasEdge(p.canvas, p.edge); !errors.Is(err, ErrNotFound) {
		t.Fatalf("second remove = %v, want not found", err)
	}
	list, err := s.ListCanvasEdges(p.canvas)
	if err != nil || len(list) != 0 {
		t.Fatalf("edges = %v %v", list, err)
	}
}

// Rows: a removed panel and a deleted canvas take their edges with them —
// migration 044 cascades at both levels, so no reader has to remember.
func TestCanvasEdgesCascade(t *testing.T) {
	t.Run("a removed panel takes its edges", func(t *testing.T) {
		s := openTest(t)
		p := linkPair(t, s)
		if err := s.RemoveCanvasPanel(p.canvas, p.leftPanel); err != nil {
			t.Fatal(err)
		}
		list, err := s.ListCanvasEdges(p.canvas)
		if err != nil || len(list) != 0 {
			t.Fatalf("edges after panel removal = %v %v", list, err)
		}
	})
	t.Run("a deleted canvas takes its edges", func(t *testing.T) {
		s := openTest(t)
		p := linkPair(t, s)
		if err := s.DeleteCanvas(p.canvas); err != nil {
			t.Fatal(err)
		}
		var n int
		if err := s.db.QueryRow(`SELECT COUNT(1) FROM canvas_edges`).Scan(&n); err != nil || n != 0 {
			t.Fatalf("edges after canvas deletion = %d %v", n, err)
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
		if err := s.db.QueryRow(`SELECT COUNT(1) FROM canvas_edges`).Scan(&n); err != nil || n != 0 {
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
		if err := s.RemoveCanvasEdge(p.canvas, p.edge); err != nil {
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
		if err := s.RemoveCanvasPanel(p.canvas, p.rightPanel); err != nil {
			t.Fatal(err)
		}
		wantContacts(t, s, p.leftTok)
		wantContacts(t, s, p.rightTok)
	})

	t.Run("canvas deleted: no contact, edges gone", func(t *testing.T) {
		s := openTest(t)
		p := linkPair(t, s)
		if err := s.DeleteCanvas(p.canvas); err != nil {
			t.Fatal(err)
		}
		wantContacts(t, s, p.leftTok)
		wantContacts(t, s, p.rightTok)
	})

	t.Run("two edges in two canvases for the same pair: listed once", func(t *testing.T) {
		s := openTest(t)
		p := linkPair(t, s)
		second, err := s.CreateCanvas("Second")
		if err != nil {
			t.Fatal(err)
		}
		l, err := s.AddCanvasPanel(second.ID, CanvasKindAgent, p.leftAgent, 0, 0, 32, 28)
		if err != nil {
			t.Fatal(err)
		}
		r, err := s.AddCanvasPanel(second.ID, CanvasKindAgent, p.rightAgent, 40, 0, 32, 28)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = s.AddCanvasEdge(second.ID, l.Panel.ID, r.Panel.ID); err != nil {
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
		list, err := s.ListCanvasEdges(p.canvas)
		if err != nil || len(list) != 1 {
			t.Fatalf("edges = %v %v", list, err)
		}
		wantContacts(t, s, p.leftTok)
	})

	t.Run("an edge between two panels of my own session grants nothing", func(t *testing.T) {
		s := openTest(t)
		left, leftTok, leftAgent := edgePeer(t, s, "Left")
		_, _, rightAgent := edgePeer(t, s, "Right")
		m, err := s.CreateCanvas("Self")
		if err != nil {
			t.Fatal(err)
		}
		a, err := s.AddCanvasPanel(m.ID, CanvasKindAgent, leftAgent, 0, 0, 32, 28)
		if err != nil {
			t.Fatal(err)
		}
		b, err := s.AddCanvasPanel(m.ID, CanvasKindAgent, rightAgent, 40, 0, 32, 28)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = s.AddCanvasEdge(m.ID, a.Panel.ID, b.Panel.ID); err != nil {
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
	if err := s.RemoveCanvasEdge(p.canvas, p.edge); err != nil {
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
