package clikeys

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// The Codex catalog and its declaration, measured against the vendor's own
// artifacts: what can be checked mechanically is checked here, because a row is
// a claim about software PiCode does not own (ADR-0174).
func TestCodexCatalogIsWellFormed(t *testing.T) {
	if len(CodexCatalog) != 149 {
		t.Fatalf("the config struct accepts 149 keymap keys, the catalog has %d", len(CodexCatalog))
	}
	seen, groups := map[string]bool{}, []string{}
	unbound, multi := 0, 0
	for _, a := range CodexCatalog {
		if a.ID == "" || a.Label == "" || a.Group == "" {
			t.Fatalf("row %+v is missing a field", a)
		}
		if seen[a.ID] {
			// The inventory lists `chat` twice (the voice surface shares the
			// config name): the catalog must still hold one row per action.
			t.Fatalf("%s appears twice", a.ID)
		}
		seen[a.ID] = true
		if len(groups) == 0 || groups[len(groups)-1] != a.Group {
			groups = append(groups, a.Group)
		}
		if len(a.Defaults) == 0 {
			unbound++
		}
		if len(a.Defaults) > 1 {
			multi++
		}
		for _, chord := range a.Defaults {
			assertCodexChord(t, a.ID, chord)
		}
	}
	if len(groups) != 12 {
		t.Fatalf("the vendor declares 12 contexts, the catalog has %d: %v", len(groups), groups)
	}
	want := []string{"global", "chat", "composer", "editor", "vim_normal", "vim_search", "vim_operator", "vim_text_object", "pager", "list", "agents", "approval"}
	if strings.Join(groups, ",") != strings.Join(want, ",") {
		t.Fatalf("contexts are out of order:\n got %v\nwant %v", groups, want)
	}
	if unbound != 11 || multi != 53 {
		t.Fatalf("unbound=%d multi-chord=%d: the vendor's own table says 11 and 53", unbound, multi)
	}
	if _, ok := action(CodexCatalog, "composer.submit"); !ok {
		t.Fatal("composer.submit is an action codex has")
	}
	// The three fallback slots are the rows the live check found missing: codex
	// accepted and resolved them (an unknown action there makes it refuse the
	// file and print this very list). They ship with no binding of their own.
	for _, key := range []string{"global.submit", "global.queue", "global.toggle_shortcuts"} {
		a, ok := action(CodexCatalog, key)
		if !ok {
			t.Fatalf("%s is a key codex accepts and the catalog omits it", key)
		}
		if a.Defaults != nil {
			t.Fatalf("%s ships unset: %+v", key, a.Defaults)
		}
	}
	global := []string{}
	for _, a := range CodexCatalog {
		if a.Group == "global" {
			global = append(global, a.ID)
		}
	}
	// Pinned from codex 0.155.1 itself: an unknown action in this context makes
	// the CLI refuse the file and print exactly these names (live, 2026-09-21).
	wantGlobal := []string{"global.open_agents", "global.open_transcript", "global.open_external_editor", "global.copy", "global.clear_terminal", "global.toggle_vim_mode", "global.toggle_fast_mode", "global.toggle_raw_output", "global.toggle_side_conversation", "global.queue", "global.submit", "global.toggle_shortcuts"}
	if strings.Join(global, ",") != strings.Join(wantGlobal, ",") {
		t.Fatalf("the global context drifted from what codex accepts:\n got %v\nwant %v", global, wantGlobal)
	}
}

// assertCodexChord checks a chord against codex's own vocabulary: lower-case
// modifier words joined by `-`, in the order the parser's own test uses
// (`ctrl-alt-shift-a`), then one key — never PiCode's `+`.
func assertCodexChord(t *testing.T, id, chord string) {
	t.Helper()
	if chord == "" || strings.ContainsAny(chord, " +") {
		t.Fatalf("%s binds %q, which is not codex's spelling", id, chord)
	}
	// The modifiers are the leading tokens, in the parser's own order; the key
	// that follows may itself carry a hyphen (`page-up`).
	order := []string{"ctrl", "alt", "shift"}
	parts := strings.Split(chord, "-")
	last := -1
	i := 0
	for ; i < len(parts); i++ {
		at := slices.Index(order, parts[i])
		if at < 0 {
			break
		}
		if at <= last {
			t.Fatalf("%s binds %q: %q is out of the parser's order %v", id, chord, parts[i], order)
		}
		last = at
	}
	if key := strings.Join(parts[i:], "-"); key == "" {
		t.Fatalf("%s binds %q with no key", id, chord)
	}
}

// The map is written through the same engine as a flat one; the declaration is
// what makes it nested, and the file it writes is the settings file, so a key
// PiCode does not manage must survive untouched.
func TestCodexWritesOneContextRow(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".codex"), 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(home, ".codex", "config.toml")
	if err := os.WriteFile(path, []byte("model = \"gpt-5\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := ReadFlat(CodexMap)
	if err != nil {
		t.Fatal(err)
	}
	if report.File != path || len(report.Values) != 0 {
		t.Fatalf("a machine with no keymap rows reads clean: %+v", report)
	}
	if err := WriteFlat(CodexMap, "composer.submit", []string{"ctrl-m"}, false, report.Revision); err != nil {
		t.Fatal(err)
	}
	got := read(t, path)
	if !strings.Contains(got, "[tui.keymap.composer]") || !strings.Contains(got, `submit = ["ctrl-m"]`) {
		t.Fatalf("the row did not land in its context table:\n%s", got)
	}
	if !strings.Contains(got, `model = "gpt-5"`) {
		t.Fatalf("the settings key did not survive:\n%s", got)
	}
	// The other 145 rows are still on their defaults, and only this one is set.
	report, err = ReadFlat(CodexMap)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Values) != 1 || strings.Join(report.Values["composer.submit"], ",") != "ctrl-m" {
		t.Fatalf("read back %+v", report.Values)
	}
	// Reset all clears every row the file sets, wherever the declaration puts
	// it: this is the path a nested map needs and a flat one hides.
	if err := ResetFlat(CodexMap, report.Revision); err != nil {
		t.Fatalf("reset all: %v", err)
	}
	after, err := ReadFlat(CodexMap)
	if err != nil {
		t.Fatal(err)
	}
	if len(after.Values) != 0 {
		t.Fatalf("Reset all left rows behind: %+v", after.Values)
	}
	if got := read(t, path); strings.Contains(got, "[tui.keymap.composer]") {
		t.Fatalf("an emptied table must go:\n%s", got)
	}

	// A single row reset hands it back to codex's default and takes the table
	// with it when it was the last one.
	report, err = ReadFlat(CodexMap)
	if err != nil {
		t.Fatal(err)
	}
	if err := WriteFlat(CodexMap, "composer.submit", []string{"ctrl-m"}, false, report.Revision); err != nil {
		t.Fatal(err)
	}
	report, err = ReadFlat(CodexMap)
	if err != nil {
		t.Fatal(err)
	}
	if err := WriteFlat(CodexMap, "composer.submit", nil, true, report.Revision); err != nil {
		t.Fatal(err)
	}
	if got := read(t, path); strings.Contains(got, "[tui.keymap.composer]") {
		t.Fatalf("an emptied table must go:\n%s", got)
	}
}
