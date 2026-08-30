package store

import (
	"path/filepath"
	"testing"
)

func settingsStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "picode.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestTerminalSettingsUnsetScopeIsEmptyNotAnError(t *testing.T) {
	s := settingsStore(t)
	got, err := s.TerminalSettings("global")
	if err != nil {
		t.Fatalf("reading a scope nobody wrote must not fail: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("got %v, want empty", got)
	}
}

func TestTerminalSettingsRoundTrip(t *testing.T) {
	s := settingsStore(t)
	if err := s.SetTerminalSettings("global", map[string]string{"mouse": "off"}); err != nil {
		t.Fatalf("set: %v", err)
	}
	got, err := s.TerminalSettings("global")
	if err != nil || got["mouse"] != "off" {
		t.Fatalf("got %v (%v), want mouse=off", got, err)
	}
}

// Storing nothing must leave nothing: a terminal that overrides no field
// should not keep a row that a later read has to interpret.
func TestTerminalSettingsEmptyMapDeletesTheRow(t *testing.T) {
	s := settingsStore(t)
	_ = s.SetTerminalSettings("term-abc123", map[string]string{"mouse": "off"})
	if err := s.SetTerminalSettings("term-abc123", map[string]string{}); err != nil {
		t.Fatalf("set empty: %v", err)
	}
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(1) FROM terminal_settings WHERE scope = ?`, "term-abc123").Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 0 {
		t.Fatalf("row count = %d, want the row gone", n)
	}
}

// A row we cannot parse must read as unset, not as a permanent error. The
// alternative locks the user out of the very panel that could fix it.
func TestTerminalSettingsUnparsableRowReadsAsUnset(t *testing.T) {
	s := settingsStore(t)
	if _, err := s.db.Exec(`INSERT INTO terminal_settings (scope, settings, updated_at) VALUES (?, ?, ?)`,
		"global", "{not json", nowUTC()); err != nil {
		t.Fatalf("seed: %v", err)
	}
	got, err := s.TerminalSettings("global")
	if err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("got %v, want empty", got)
	}
	if err := s.SetTerminalSettings("global", map[string]string{"mouse": "on"}); err != nil {
		t.Fatalf("a bad row must not block the next write: %v", err)
	}
}

func TestDeletingATerminalTakesItsOverridesWithIt(t *testing.T) {
	s := settingsStore(t)
	term, err := s.CreateTerminal("Shell", t.TempDir())
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := s.SetTerminalSettings(term.ID, map[string]string{"mouse": "off"}); err != nil {
		t.Fatalf("set: %v", err)
	}
	if err := s.DeleteTerminal(term.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	got, _ := s.TerminalSettings(term.ID)
	if len(got) != 0 {
		t.Fatalf("overrides outlived their terminal: %v", got)
	}
}
