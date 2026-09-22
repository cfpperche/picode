package clikeys

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// The Antigravity catalog: 36 actions in ten namespaces, every one bound (the
// vendor's own file has no unbound action), each keeping the file's own chord
// spelling (`pgdown`, `esc`, `ctrl+_`).
func TestAgyCatalogIsWellFormed(t *testing.T) {
	if len(AgyCatalog) != 36 {
		t.Fatalf("the vendor's file carries 36 actions, the catalog has %d", len(AgyCatalog))
	}
	seen, groups := map[string]bool{}, []string{}
	for _, a := range AgyCatalog {
		if a.ID == "" || a.Group == "" || a.Label == "" {
			t.Fatalf("row %+v is missing a field", a)
		}
		if seen[a.ID] {
			t.Fatalf("%s appears twice", a.ID)
		}
		seen[a.ID] = true
		if len(groups) == 0 || groups[len(groups)-1] != a.Group {
			groups = append(groups, a.Group)
		}
		if len(a.Defaults) == 0 {
			t.Fatalf("%s ships unbound, which the vendor's file does not do", a.ID)
		}
	}
	if len(groups) != 10 {
		t.Fatalf("the file uses 10 namespaces, the catalog has %d groups: %v", len(groups), groups)
	}
	// A chord the pane cannot capture is still in the catalog, spelled the way
	// the vendor spells it.
	if a, ok := action(AgyCatalog, "edit.undo"); !ok || !slices.Contains(a.Defaults, "ctrl+_") {
		t.Fatalf("edit.undo lost the vendor's chords: %+v", a)
	}
}

func writeAgyHome(t *testing.T, body string) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := filepath.Join(home, ".gemini", "antigravity-cli")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "keybindings.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return filepath.Join(dir, "keybindings.json")
}

// The map is a flat JSON document the engine already knows how to splice; the
// declaration only adds where it lives and the one chord name the pane spells
// differently (`escape` for the file's `esc`).
func TestAgyWritesAndRendersTheFileVocabulary(t *testing.T) {
	path := writeAgyHome(t, "{\n  \"cli.clear_screen\": [\"ctrl+l\"],\n  \"cli.escape\": [\"ctrl+c\", \"esc\"]\n}\n")
	report, err := ReadFlat(AgyMap)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(report.Values["cli.escape"], ",") != "ctrl+c,esc" {
		t.Fatalf("read back %+v", report.Values)
	}
	// A captured chord in the pane's spelling lands in the file's own.
	if err := WriteFlat(AgyMap, "cli.escape", []string{"ctrl+escape"}, false, report.Revision); err != nil {
		t.Fatal(err)
	}
	got := read(t, path)
	if !strings.Contains(got, `"ctrl+esc"`) {
		t.Fatalf("the chord was not rendered in the file's vocabulary:\n%s", got)
	}
	// Reset removes the row, which for an override file means "back to the
	// built-in binding", exactly what the vendor documents.
	report, err = ReadFlat(AgyMap)
	if err != nil {
		t.Fatal(err)
	}
	if err := WriteFlat(AgyMap, "cli.escape", nil, true, report.Revision); err != nil {
		t.Fatal(err)
	}
	report, err = ReadFlat(AgyMap)
	if err != nil {
		t.Fatal(err)
	}
	if _, set := report.Values["cli.escape"]; set {
		t.Fatalf("the row survived a reset: %+v", report.Values)
	}
}
