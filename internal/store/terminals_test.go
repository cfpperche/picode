package store

import (
	"os"
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

func TestCreateTerminalBadCwd(t *testing.T) {
	s := openTest(t)
	if _, err := s.CreateTerminal("x", "/no/such/picode-term-cwd"); err == nil {
		t.Fatal("want error")
	}
}
