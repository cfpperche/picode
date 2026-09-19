package store

import (
	"errors"
	"testing"

	"github.com/cfpperche/picode/internal/clilaunch"
)

func TestAgentByTerminalAndDeleteRemovesGuest(t *testing.T) {
	s := openTest(t)
	proj := t.TempDir()
	w, err := s.AddWorkspace("App", proj)
	if err != nil {
		t.Fatal(err)
	}
	a, err := s.AddAgentWithCLI(w.ID, "claude-code", "Claude", "")
	if err != nil {
		t.Fatal(err)
	}
	tm, err := s.CreateTerminalIn(w.ID, "Claude", proj)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SetTerminalLaunch(tm.ID, "claude-code", clilaunch.Overrides{}); err != nil {
		t.Fatal(err)
	}
	tid := tm.ID
	a, err = s.UpdateAgent(a.ID, AgentPatch{TerminalID: &tid})
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.AgentByTerminal(tm.ID)
	if err != nil || got.ID != a.ID {
		t.Fatalf("by terminal = %+v err %v", got, err)
	}
	if _, err := s.AgentByTerminal("missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing: %v", err)
	}

	if err := s.DeleteTerminal(tm.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetAgent(a.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("guest agent survived terminal delete: %v", err)
	}
}

func TestMigrateManagedCLIsOntoAgents(t *testing.T) {
	s := openTest(t)
	proj := t.TempDir()
	w, err := s.AddWorkspace("App", proj)
	if err != nil {
		t.Fatal(err)
	}
	tm, err := s.CreateTerminalIn(w.ID, "Claude", proj)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SetTerminalLaunch(tm.ID, "claude-code", clilaunch.Overrides{}); err != nil {
		t.Fatal(err)
	}

	if _, err := s.db.Exec(`CREATE TABLE managed_clis (
		id TEXT PRIMARY KEY,
		workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
		cli TEXT NOT NULL,
		terminal_id TEXT NOT NULL UNIQUE REFERENCES terminals(id) ON DELETE CASCADE,
		name TEXT NOT NULL,
		created_at TEXT NOT NULL
	)`); err != nil {
		t.Fatalf("recreate managed_clis: %v", err)
	}
	if _, err := s.db.Exec(`INSERT INTO managed_clis (id, workspace_id, cli, terminal_id, name, created_at)
		VALUES ('mcli-claude', ?, 'claude-code', ?, 'Claude', ?)`, w.ID, tm.ID, nowUTC()); err != nil {
		t.Fatalf("seed row: %v", err)
	}
	if _, err := s.db.Exec(`DELETE FROM schema_migrations WHERE version = 61`); err != nil {
		t.Fatal(err)
	}
	if err := s.migrate(); err != nil {
		t.Fatalf("migrate 061: %v", err)
	}
	a, err := s.GetAgent("mcli-claude")
	if err != nil {
		t.Fatalf("copied agent: %v", err)
	}
	if a.CLI != "claude-code" || a.TerminalID == nil || *a.TerminalID != tm.ID {
		t.Fatalf("copied = %+v", a)
	}
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(1) FROM sqlite_master WHERE name = 'managed_clis'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatal("managed_clis table still present")
	}
}
