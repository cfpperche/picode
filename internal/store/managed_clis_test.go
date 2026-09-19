package store

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/cfpperche/picode/internal/clilaunch"
	"github.com/cfpperche/picode/internal/grant"
)

func TestAddManagedCLIDecisionTable(t *testing.T) {
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

	m, err := s.AddManagedCLI(w.ID, "claude-code", "Claude", tm.ID)
	if err != nil {
		t.Fatal(err)
	}
	if m.CLI != "claude-code" || m.TerminalID != tm.ID || m.WorkspaceID != w.ID {
		t.Fatalf("binding = %+v", m)
	}
	if p := m.Principal(); p.Kind != grant.KindTerminal || p.ID != tm.ID || p.Key() != "term:"+tm.ID {
		t.Fatalf("principal = %+v key %q", p, p.Key())
	}
	agents, _ := s.ListAgents(w.ID)
	if len(agents) != 0 {
		t.Fatalf("created %d agent rows; guests must not enter agents", len(agents))
	}

	if _, err := s.AddManagedCLI(w.ID, "claude-code", "Again", tm.ID); !errors.Is(err, ErrConflict) {
		t.Fatalf("duplicate bind: %v", err)
	}

	other := filepath.Join(t.TempDir(), "other")
	if err := os.MkdirAll(other, 0o755); err != nil {
		t.Fatal(err)
	}
	w2, err := s.AddWorkspace("Other", other)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.AddManagedCLI(w2.ID, "claude-code", "X", tm.ID); !errors.Is(err, ErrInvalid) {
		t.Fatalf("foreign terminal: %v", err)
	}
	if _, err := s.AddManagedCLI(w.ID, "not-a-cli", "X", tm.ID); !errors.Is(err, ErrInvalid) {
		t.Fatalf("unknown CLI: %v", err)
	}
	if _, err := s.AddManagedCLI(FreeWorkspaceID, "claude-code", "X", tm.ID); !errors.Is(err, ErrInvalid) {
		t.Fatalf("free workspace: %v", err)
	}

	codex, err := s.CreateTerminalIn(w.ID, "Codex", proj)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SetTerminalLaunch(codex.ID, "codex", clilaunch.Overrides{}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AddManagedCLI(w.ID, "claude-code", "Nope", codex.ID); !errors.Is(err, ErrInvalid) {
		t.Fatalf("launch mismatch: %v", err)
	}

	listed, err := s.ListManagedCLIs(w.ID)
	if err != nil || len(listed) != 1 || listed[0].ID != m.ID {
		t.Fatalf("list = %+v err %v", listed, err)
	}

	if err := s.RemoveManagedCLI(m.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetTerminal(tm.ID); err != nil {
		t.Fatalf("unbind must keep the terminal: %v", err)
	}
	if _, err := s.GetManagedCLI(m.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unbind left the row: %v", err)
	}
}

func TestDeleteTerminalRemovesManagedCLI(t *testing.T) {
	s := openTest(t)
	proj := t.TempDir()
	w, _ := s.AddWorkspace("App", proj)
	tm, _ := s.CreateTerminalIn(w.ID, "Claude", proj)
	_ = s.SetTerminalLaunch(tm.ID, "claude-code", clilaunch.Overrides{})
	m, err := s.AddManagedCLI(w.ID, "claude-code", "Claude", tm.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteTerminal(tm.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetManagedCLI(m.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cascade: %v", err)
	}
}

func TestAddManagedCLIPiTUIIsNotAnAgent(t *testing.T) {
	s := openTest(t)
	proj := t.TempDir()
	w, _ := s.AddWorkspace("App", proj)
	tm, _ := s.CreateTerminalIn(w.ID, "Pi", proj)
	_ = s.SetTerminalLaunch(tm.ID, "pi", clilaunch.Overrides{})
	m, err := s.AddManagedCLI(w.ID, "pi", "Pi TUI", tm.ID)
	if err != nil {
		t.Fatal(err)
	}
	agents, _ := s.ListAgents(w.ID)
	if len(agents) != 0 {
		t.Fatalf("pi TUI principal created an agent: %+v", agents)
	}
	if m.Principal().Kind != grant.KindTerminal {
		t.Fatalf("want terminal principal, got %+v", m.Principal())
	}
}
