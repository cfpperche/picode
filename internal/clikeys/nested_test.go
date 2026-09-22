package clikeys

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/clisettings"
)

// A nested key map writes one row inside a table named by the row's context —
// the shape Codex's `[tui.keymap.<context>.<action>]` has (P3). The engine is
// the flat one; the declaration is what differs, which is the whole point of
// FlatMap.Path. The fixture is a temporary TOML file, not a vendor's.
func nestedIn(t *testing.T, body string) (FlatMap, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.toml")
	if body != "" {
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	m := FlatMap{
		CLI:     "testnested",
		Catalog: []Action{{ID: "submit", Group: "composer"}, {ID: "open", Group: "global"}},
		File:    func() (string, clisettings.Format, error) { return path, clisettings.FormatTOML, nil },
		Path:    func(a Action) []string { return []string{"tui", "keymap", a.Group, a.ID} },
	}
	return m, path
}

func TestNestedMapWritesOneContextRow(t *testing.T) {
	m, path := nestedIn(t, "model = \"gpt-5\"\n")
	report, err := ReadFlat(m)
	if err != nil {
		t.Fatal(err)
	}
	if err := WriteFlat(m, "submit", []string{"ctrl+m"}, false, report.Revision); err != nil {
		t.Fatal(err)
	}
	got := read(t, path)
	if !strings.Contains(got, "[tui.keymap.composer]") || !strings.Contains(got, `submit = ["ctrl+m"]`) {
		t.Fatalf("the row did not land in its context table:\n%s", got)
	}
	if !strings.Contains(got, `model = "gpt-5"`) {
		t.Fatalf("the file lost a key PiCode does not manage:\n%s", got)
	}
	// The map reads it back, and only that row: `open` is still unset.
	report, err = ReadFlat(m)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(report.Values["submit"], ",") != "ctrl+m" {
		t.Fatalf("read back %+v", report.Values)
	}
	if _, set := report.Values["open"]; set {
		t.Fatalf("an unset row reported a value: %+v", report.Values)
	}
	// Reset hands the row back and takes the table with it.
	if err := WriteFlat(m, "submit", nil, true, report.Revision); err != nil {
		t.Fatal(err)
	}
	if got := read(t, path); strings.Contains(got, "[tui.keymap.composer]") {
		t.Fatalf("an emptied table must go:\n%s", got)
	}
}

// Two context tables coexist: a write into one must not disturb the other, and
// a row that was already set reads back with its own chords.
func TestNestedMapKeepsOtherContextsIntact(t *testing.T) {
	m, path := nestedIn(t, "[tui.keymap.global]\nopen = [\"ctrl+o\"]\n")
	if err := WriteFlat(m, "submit", []string{"ctrl+m"}, false, ""); err != nil {
		t.Fatal(err)
	}
	got := read(t, path)
	if !strings.Contains(got, `open = ["ctrl+o"]`) || !strings.Contains(got, "[tui.keymap.composer]") || !strings.Contains(got, `submit = ["ctrl+m"]`) {
		t.Fatalf("both contexts must survive:\n%s", got)
	}
	report, err := ReadFlat(m)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(report.Values["open"], ",") != "ctrl+o" {
		t.Fatalf("the pre-existing row must read: %+v", report.Values)
	}
	// A second row in the table that exists joins it rather than opening a
	// second one.
	withSecond := m
	withSecond.Catalog = append(append([]Action{}, m.Catalog...), Action{ID: "close", Group: "global"})
	report, err = ReadFlat(withSecond)
	if err != nil {
		t.Fatal(err)
	}
	if err := WriteFlat(withSecond, "close", []string{"alt+o"}, false, report.Revision); err != nil {
		t.Fatal(err)
	}
	got = read(t, path)
	if strings.Count(got, "[tui.keymap.global]") != 1 {
		t.Fatalf("a second table was opened:\n%s", got)
	}
	if !strings.Contains(got, `close = ["alt+o"]`) || !strings.Contains(got, `open = ["ctrl+o"]`) {
		t.Fatalf("the sibling row did not survive:\n%s", got)
	}
}
