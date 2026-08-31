package store

import (
	"os"
	"strings"
	"testing"
)

func TestTerminalsCRUD(t *testing.T) {
	s := openTest(t)
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	a, err := s.CreateTerminal("", "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if a.Name != "Terminal" || a.Cwd != home || a.ID == "" {
		t.Fatalf("first = %+v", a)
	}
	b, err := s.CreateTerminal("", home)
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if b.Name != "Terminal 2" {
		t.Fatalf("name=%q", b.Name)
	}
	list, err := s.ListTerminals()
	if err != nil || len(list) != 2 {
		t.Fatalf("list=%+v %v", list, err)
	}
	got, err := s.GetTerminal(a.ID)
	if err != nil || got.ID != a.ID {
		t.Fatalf("get: %+v %v", got, err)
	}
	if err := s.DeleteTerminal(a.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetTerminal(a.ID); err != ErrNotFound {
		t.Fatalf("gone: %v", err)
	}
	if err := s.DeleteTerminal(a.ID); err != ErrNotFound {
		t.Fatalf("double delete: %v", err)
	}
}

func TestRenameTerminal(t *testing.T) {
	s := openTest(t)
	a, err := s.CreateTerminal("", "")
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.RenameTerminal(a.ID, "  build  ")
	if err != nil || got.Name != "build" {
		t.Fatalf("rename = %+v %v", got, err)
	}
	if _, err := s.RenameTerminal(a.ID, "   "); err == nil {
		t.Fatal("empty name")
	}
	if _, err := s.RenameTerminal("nope", "x"); err != ErrNotFound {
		t.Fatalf("missing: %v", err)
	}
}

func TestListTerminalsByName(t *testing.T) {
	s := openTest(t)
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateTerminal("Pi/PiCode", home); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateTerminal("Claude/PiCode", home); err != nil {
		t.Fatal(err)
	}
	list, err := s.ListTerminals()
	if err != nil || len(list) != 2 {
		t.Fatalf("list=%+v %v", list, err)
	}
	if list[0].Name != "Claude/PiCode" || list[1].Name != "Pi/PiCode" {
		t.Fatalf("order=%q %q", list[0].Name, list[1].Name)
	}
}

func TestCreateTerminalBadCwd(t *testing.T) {
	s := openTest(t)
	if _, err := s.CreateTerminal("x", "/no/such/picode-term-cwd"); err == nil {
		t.Fatal("want error")
	}
}

func TestCreateTerminalDefaultsToFreeWorkspace(t *testing.T) {
	s := openTest(t)
	a, err := s.CreateTerminal("", "")
	if err != nil {
		t.Fatal(err)
	}
	if a.WorkspaceID != FreeWorkspaceID {
		t.Fatalf("workspace=%q", a.WorkspaceID)
	}
}

func TestCreateTerminalInWorkspaceDefaultsToItsFolder(t *testing.T) {
	s := openTest(t)
	proj := t.TempDir()
	w, _, err := s.AddWorkspace("App", proj)
	if err != nil {
		t.Fatal(err)
	}
	tm, err := s.CreateTerminalIn(w.ID, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if tm.WorkspaceID != w.ID || tm.Cwd != w.Path {
		t.Fatalf("terminal=%+v want workspace=%q cwd=%q", tm, w.ID, w.Path)
	}
	got, err := s.GetTerminal(tm.ID)
	if err != nil || got.WorkspaceID != w.ID {
		t.Fatalf("get=%+v %v", got, err)
	}
}

func TestCreateTerminalInUnknownWorkspace(t *testing.T) {
	s := openTest(t)
	_, err := s.CreateTerminalIn("ws-nope", "", "")
	if err == nil || !strings.Contains(err.Error(), "doesn't exist") {
		t.Fatalf("err=%v", err)
	}
}

func TestListWorkspaceTerminals(t *testing.T) {
	s := openTest(t)
	proj := t.TempDir()
	w, _, err := s.AddWorkspace("App", proj)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateTerminal("free one", ""); err != nil {
		t.Fatal(err)
	}
	owned, err := s.CreateTerminalIn(w.ID, "owned", "")
	if err != nil {
		t.Fatal(err)
	}
	list, err := s.ListWorkspaceTerminals(w.ID)
	if err != nil || len(list) != 1 || list[0].ID != owned.ID {
		t.Fatalf("workspace list=%+v %v", list, err)
	}
	all, err := s.ListTerminals()
	if err != nil || len(all) != 2 {
		t.Fatalf("all=%+v %v", all, err)
	}
}

func TestRemoveWorkspaceDeletesItsTerminals(t *testing.T) {
	s := openTest(t)
	proj := t.TempDir()
	w, _, err := s.AddWorkspace("App", proj)
	if err != nil {
		t.Fatal(err)
	}
	owned, err := s.CreateTerminalIn(w.ID, "owned", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SetTerminalSettings(owned.ID, map[string]string{"mouse": "off"}); err != nil {
		t.Fatal(err)
	}
	free, err := s.CreateTerminal("free one", "")
	if err != nil {
		t.Fatal(err)
	}
	removed, err := s.RemoveWorkspace(w.ID)
	if err != nil || !removed {
		t.Fatalf("remove=%v %v", removed, err)
	}
	if _, err := s.GetTerminal(owned.ID); err != ErrNotFound {
		t.Fatalf("owned terminal survived: %v", err)
	}
	if v, err := s.TerminalSettings(owned.ID); err != nil || len(v) != 0 {
		t.Fatalf("owned settings survived: %+v %v", v, err)
	}
	// The free terminal must survive — a filtered ListTerminals or an
	// over-eager cascade here would break tmux overrides for everyone.
	if _, err := s.GetTerminal(free.ID); err != nil {
		t.Fatalf("free terminal gone: %v", err)
	}
}
