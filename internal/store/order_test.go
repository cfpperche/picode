package store

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSidebarOrderAppendsAndReorders(t *testing.T) {
	s := openTest(t)
	aDir, bDir, cDir := t.TempDir(), t.TempDir(), t.TempDir()

	// Created as B then A: the list follows insertion, not the name.
	b, err := s.AddWorkspace("B", bDir)
	if err != nil {
		t.Fatal(err)
	}
	a, err := s.AddWorkspace("A", aDir)
	if err != nil {
		t.Fatal(err)
	}
	if got := workspaceIDs(t, s); !sameIDs(got, []string{b.ID, a.ID}) {
		t.Fatalf("append order = %v", got)
	}

	var events []string
	s.OnEvent = func(ev Event) { events = append(events, ev.Type) }
	if err := s.ReorderWorkspaces([]string{a.ID, b.ID}); err != nil {
		t.Fatal(err)
	}
	if got := workspaceIDs(t, s); !sameIDs(got, []string{a.ID, b.ID}) {
		t.Fatalf("reordered = %v", got)
	}
	if !sameIDs(events, []string{"workspace.reordered"}) {
		t.Fatalf("events = %v", events)
	}
	events = nil
	if err := s.ReorderWorkspaces([]string{a.ID, b.ID}); err != nil {
		t.Fatal(err)
	}
	if len(events) != 0 {
		t.Fatalf("same order emitted %v", events)
	}

	c, err := s.AddWorkspace("C", cDir)
	if err != nil {
		t.Fatal(err)
	}
	if got := workspaceIDs(t, s); !sameIDs(got, []string{a.ID, b.ID, c.ID}) {
		t.Fatalf("new workspace = %v, want appended", got)
	}

	// Agents in a workspace append, then reorder. The oldest stays DefaultAgent.
	first, err := s.AddAgent(a.ID, "first", "")
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.AddAgent(a.ID, "second", "")
	if err != nil {
		t.Fatal(err)
	}
	if got := agentIDs(t, s, a.ID); !sameIDs(got, []string{first.ID, second.ID}) {
		t.Fatalf("agents = %v", got)
	}
	if err := s.ReorderAgents(a.ID, []string{second.ID, first.ID}); err != nil {
		t.Fatal(err)
	}
	if got := agentIDs(t, s, a.ID); !sameIDs(got, []string{second.ID, first.ID}) {
		t.Fatalf("agents reordered = %v", got)
	}
	def, err := s.DefaultAgent(a.ID)
	if err != nil || def.ID != first.ID {
		t.Fatalf("default agent = %+v, want oldest %s (%v)", def, first.ID, err)
	}

	// Shell terminals reorder. A terminal bound to an agent is not a row.
	t1, err := s.CreateTerminalIn(a.ID, "zeta", aDir)
	if err != nil {
		t.Fatal(err)
	}
	t2, err := s.CreateTerminalIn(a.ID, "alpha", aDir)
	if err != nil {
		t.Fatal(err)
	}
	owned, err := s.CreateTerminalIn(a.ID, "owned", aDir)
	if err != nil {
		t.Fatal(err)
	}
	tid := owned.ID
	if _, err := s.UpdateAgent(first.ID, AgentPatch{TerminalID: &tid}); err != nil {
		t.Fatal(err)
	}
	if got := shellTerminalIDs(t, s, a.ID); !sameIDs(got, []string{t1.ID, t2.ID}) {
		t.Fatalf("shell terminals = %v", got)
	}
	if err := s.ReorderTerminals(a.ID, []string{t2.ID, t1.ID}); err != nil {
		t.Fatal(err)
	}
	if got := shellTerminalIDs(t, s, a.ID); !sameIDs(got, []string{t2.ID, t1.ID}) {
		t.Fatalf("terminals reordered = %v", got)
	}
}

func TestSidebarOrderRejects(t *testing.T) {
	s := openTest(t)
	a, err := s.AddWorkspace("A", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.AddWorkspace("B", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	before := workspaceIDs(t, s)

	cases := []struct {
		name string
		run  func() error
	}{
		{"empty", func() error { return s.ReorderWorkspaces(nil) }},
		{"duplicate", func() error { return s.ReorderWorkspaces([]string{a.ID, a.ID}) }},
		{"missing", func() error { return s.ReorderWorkspaces([]string{a.ID}) }},
		{"unknown", func() error { return s.ReorderWorkspaces([]string{a.ID, b.ID, "nope"}) }},
		{"free workspace", func() error { return s.ReorderWorkspaces([]string{a.ID, FreeWorkspaceID}) }},
		{"blank id", func() error { return s.ReorderWorkspaces([]string{a.ID, ""}) }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.run()
			var oe *OrderError
			if !errors.As(err, &oe) {
				t.Fatalf("err = %v, want OrderError", err)
			}
			if got := workspaceIDs(t, s); !sameIDs(got, before) {
				t.Fatalf("list changed to %v", got)
			}
		})
	}

	t.Run("unknown workspace", func(t *testing.T) {
		err := s.ReorderAgents("missing", []string{"x"})
		if !errors.Is(err, ErrNotFound) {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("agent from another workspace", func(t *testing.T) {
		ag, err := s.AddAgent(a.ID, "only", "")
		if err != nil {
			t.Fatal(err)
		}
		err = s.ReorderAgents(b.ID, []string{ag.ID})
		var oe *OrderError
		if !errors.As(err, &oe) {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("terminal bound to an agent", func(t *testing.T) {
		dir := a.Path
		shell, err := s.CreateTerminalIn(a.ID, "shell", dir)
		if err != nil {
			t.Fatal(err)
		}
		owned, err := s.CreateTerminalIn(a.ID, "owned", dir)
		if err != nil {
			t.Fatal(err)
		}
		ag, err := s.AddAgent(a.ID, "owner", "")
		if err != nil {
			t.Fatal(err)
		}
		tid := owned.ID
		if _, err := s.UpdateAgent(ag.ID, AgentPatch{TerminalID: &tid}); err != nil {
			t.Fatal(err)
		}
		err = s.ReorderTerminals(a.ID, []string{owned.ID, shell.ID})
		var oe *OrderError
		if !errors.As(err, &oe) || !strings.Contains(oe.Error(), "belongs to an agent") {
			t.Fatalf("err = %v", err)
		}
		err = s.ReorderTerminals(a.ID, []string{shell.ID, owned.ID})
		if !errors.As(err, &oe) {
			t.Fatalf("mixed list err = %v", err)
		}
	})
}

func TestSidebarOrderBackfillKeepsTheOldSort(t *testing.T) {
	s := openTest(t)
	// Rows that predate the column all sat at position 0. Re-running the
	// ranking statements (every statement in the migration except ALTER)
	// must reproduce the sidebar's old sort.
	insert := []struct {
		q    string
		args []any
	}{
		{`INSERT INTO workspaces (id, name, path, created_at, position) VALUES (?, ?, ?, ?, 0)`, []any{"c", "C", filepath.Join(t.TempDir()), "2026-01-03T00:00:00Z"}},
		{`INSERT INTO workspaces (id, name, path, created_at, position) VALUES (?, ?, ?, ?, 0)`, []any{"a", "A", filepath.Join(t.TempDir()), "2026-01-01T00:00:00Z"}},
		{`INSERT INTO workspaces (id, name, path, created_at, position) VALUES (?, ?, ?, ?, 0)`, []any{"b", "B", filepath.Join(t.TempDir()), "2026-01-02T00:00:00Z"}},
		{`INSERT INTO agents (id, workspace_id, name, created_at, position) VALUES (?, 'a', 'later', '2026-02-02T00:00:00Z', 0)`, nil},
		{`INSERT INTO agents (id, workspace_id, name, created_at, position) VALUES (?, 'a', 'earlier', '2026-02-01T00:00:00Z', 0)`, nil},
		{`INSERT INTO agents (id, workspace_id, name, created_at, position) VALUES ('z-free', 'ws_free', 'Zed', '2026-01-01T00:00:00Z', 0)`, nil},
		{`INSERT INTO agents (id, workspace_id, name, created_at, position) VALUES ('m-free', 'ws_free', 'amy', '2026-03-01T00:00:00Z', 0)`, nil},
		{`INSERT INTO terminals (id, name, cwd, workspace_id, created_at, position) VALUES ('tb', 'beta', ?, 'a', '2026-01-01T00:00:00Z', 0)`, []any{t.TempDir()}},
		{`INSERT INTO terminals (id, name, cwd, workspace_id, created_at, position) VALUES ('ta', 'Alpha', ?, 'a', '2026-01-02T00:00:00Z', 0)`, []any{t.TempDir()}},
	}
	// The two workspace agents need ids. Fill them in.
	insert[3].args = []any{"ag-later"}
	insert[4].args = []any{"ag-earlier"}
	for _, row := range insert {
		if _, err := s.db.Exec(row.q, row.args...); err != nil {
			t.Fatalf("seed: %v\n%s", err, row.q)
		}
	}
	body, err := os.ReadFile("migrations/064_sidebar_order.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, stmt := range strings.Split(string(body), ";") {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" || strings.HasPrefix(stmt, "ALTER") || strings.HasPrefix(stmt, "--") {
			continue
		}
		// A comment block may precede the statement.
		if i := strings.LastIndex(stmt, "\n"); i >= 0 && strings.HasPrefix(strings.TrimSpace(stmt[:i+1]), "--") {
			// keep going; Exec accepts leading comments
		}
		if _, err := s.db.Exec(stmt); err != nil {
			t.Fatalf("rank: %v\n%s", err, stmt)
		}
	}
	if got := workspaceIDs(t, s); !sameIDs(got, []string{"a", "b", "c"}) {
		t.Fatalf("workspaces = %v", got)
	}
	if got := agentIDs(t, s, "a"); !sameIDs(got, []string{"ag-earlier", "ag-later"}) {
		t.Fatalf("agents = %v", got)
	}
	if got := agentIDs(t, s, FreeWorkspaceID); !sameIDs(got, []string{"m-free", "z-free"}) {
		t.Fatalf("free agents = %v, want name order", got)
	}
	rows, err := s.db.Query(`SELECT id FROM terminals WHERE workspace_id = 'a' ORDER BY position, id`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var terms []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		terms = append(terms, id)
	}
	if !sameIDs(terms, []string{"ta", "tb"}) {
		t.Fatalf("terminals = %v, want Alpha then beta", terms)
	}
}

func workspaceIDs(t *testing.T, s *Store) []string {
	t.Helper()
	list, err := s.ListWorkspaces()
	if err != nil {
		t.Fatal(err)
	}
	out := make([]string, len(list))
	for i, w := range list {
		out[i] = w.ID
	}
	return out
}

func agentIDs(t *testing.T, s *Store, ws string) []string {
	t.Helper()
	list, err := s.ListAgents(ws)
	if err != nil {
		t.Fatal(err)
	}
	out := make([]string, len(list))
	for i, a := range list {
		out[i] = a.ID
	}
	return out
}

func shellTerminalIDs(t *testing.T, s *Store, ws string) []string {
	t.Helper()
	list, err := s.ListWorkspaceTerminals(ws)
	if err != nil {
		t.Fatal(err)
	}
	owned := map[string]bool{}
	agents, err := s.ListAgents(ws)
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range agents {
		if a.TerminalID != nil {
			owned[*a.TerminalID] = true
		}
	}
	var out []string
	for _, tm := range list {
		if !owned[tm.ID] {
			out = append(out, tm.ID)
		}
	}
	return out
}
