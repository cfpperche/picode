package clikeys

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/clisettings"
)

// The flat engine's contract, measured on a real agent directory: PI_CONFIG_DIR
// is the CLI's own override (read out of its bundle), so a test points the
// declaration at a temp dir and exercises exactly the path a machine would.
func ompIn(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	t.Setenv("PI_CONFIG_DIR", root)
	t.Setenv("OMP_PROFILE", "")
	t.Setenv("PI_PROFILE", "")
	return filepath.Join(root, "agent")
}

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// The catalog is a claim about someone else's software, so what can be checked
// mechanically is checked: the ids are unique, every row is named, the group
// headings are consecutive (the pane groups by adjacency, so a row out of place
// would split a group in two), and the five unbound rows are the ones the CLI
// itself ships unbound.
func TestOmpCatalogIsWellFormed(t *testing.T) {
	if len(OmpCatalog) != 70 {
		t.Fatalf("the registry declares 70 actions, the catalog has %d", len(OmpCatalog))
	}
	seen := map[string]bool{}
	groups := []string{}
	unbound := []string{}
	for _, a := range OmpCatalog {
		if a.ID == "" || a.Label == "" || a.Group == "" {
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
			unbound = append(unbound, a.ID)
		}
		for _, chord := range a.Defaults {
			// The CLI's own spelling, checked where a mistake would be silent:
			// the modifiers are lower-case words from the fixed set, and the
			// key after them is there. The key itself keeps the CLI's case
			// (`pageUp` is how Omp spells it).
			parts := strings.Split(chord, "+")
			if chord == "" || strings.Contains(chord, " ") {
				t.Fatalf("%s binds %q, which is not a chord", a.ID, chord)
			}
			for _, mod := range parts[:len(parts)-1] {
				switch mod {
				case "ctrl", "shift", "alt", "super":
				default:
					t.Fatalf("%s binds %q: %q is not one of the CLI's modifiers", a.ID, chord, mod)
				}
			}
			if parts[len(parts)-1] == "" {
				t.Fatalf("%s binds %q with no key", a.ID, chord)
			}
		}
	}
	for i := 1; i < len(groups); i++ {
		for j := range i {
			if groups[i] == groups[j] {
				t.Fatalf("group %q is not contiguous: %v", groups[i], groups)
			}
		}
	}
	if len(unbound) != 5 {
		t.Fatalf("the CLI ships five actions unbound, the catalog says %d: %v", len(unbound), unbound)
	}
	// The one row that binds differently by platform carries the CLI's own
	// platform names, and a real answer for each.
	paste, ok := action(OmpCatalog, "app.clipboard.pasteImage")
	if !ok {
		t.Fatal("app.clipboard.pasteImage is in the registry")
	}
	if len(paste.Alt["win32"]) != 2 || len(paste.Alt["darwin"]) != 2 || len(paste.Defaults) != 1 {
		t.Fatalf("the platform binding was not carried over: %+v", paste)
	}
}

func action(rows []Action, id string) (Action, bool) {
	for _, a := range rows {
		if a.ID == id {
			return a, true
		}
	}
	return Action{}, false
}

// A machine with no file gets one, in the CLI's own format, and the row reads
// back. This is the whole point of P2b: a guest's map is editable.
func TestOmpWriteCreatesTheFileItReads(t *testing.T) {
	dir := ompIn(t)
	m, err := ReadFlat(OmpFlat)
	if err != nil {
		t.Fatal(err)
	}
	if m.Exists || m.Revision != "" || len(m.Values) != 0 {
		t.Fatalf("a fresh machine has no map: %+v", m)
	}
	if m.File != filepath.Join(dir, "keybindings.yml") {
		t.Fatalf("the map is created where the CLI looks: %s", m.File)
	}
	if err := WriteFlat(OmpFlat, "app.exit", []string{"ctrl+q"}, false, m.Revision); err != nil {
		t.Fatal(err)
	}
	got := read(t, m.File)
	if !strings.Contains(got, `app.exit: ["ctrl+q"]`) {
		t.Fatalf("the file says:\n%s", got)
	}
	m2, err := ReadFlat(OmpFlat)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(m2.Values["app.exit"], ",") != "ctrl+q" {
		t.Fatalf("read back %v", m2.Values)
	}
	if m2.Revision == "" {
		t.Fatal("a file that exists has a revision")
	}
	// Reset hands the row back to the CLI, and the file is empty again.
	if err := WriteFlat(OmpFlat, "app.exit", nil, true, m2.Revision); err != nil {
		t.Fatal(err)
	}
	m3, err := ReadFlat(OmpFlat)
	if err != nil {
		t.Fatal(err)
	}
	if _, set := m3.Values["app.exit"]; set {
		t.Fatalf("the row is still set: %+v", m3.Values)
	}
}

// The three file names are a precedence the CLI itself applies: whatever is
// there is what the CLI reads, so it is what PiCode edits — creating `.yml`
// beside a `.json` would shadow the user's own values.
func TestOmpEditsTheFileThatExists(t *testing.T) {
	for _, tc := range []struct{ name, want string }{
		{"keybindings.yml", "keybindings.yml"},
		{"keybindings.yaml", "keybindings.yaml"},
		{"keybindings.json", "keybindings.json"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := ompIn(t)
			writeFile(t, filepath.Join(dir, tc.name), "{}\n")
			m, err := ReadFlat(OmpFlat)
			if err != nil {
				t.Fatal(err)
			}
			if filepath.Base(m.File) != tc.want {
				t.Fatalf("read %s, the CLI would read %s", filepath.Base(m.File), tc.want)
			}
			if err := WriteFlat(OmpFlat, "tui.input.submit", []string{"alt+s"}, false, m.Revision); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(read(t, m.File), "tui.input.submit") {
				t.Fatalf("%s did not take the row", tc.want)
			}
			// Nothing new appears beside it.
			entries, err := os.ReadDir(dir)
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != 1 {
				t.Fatalf("a second file was created: %v", entries)
			}
		})
	}
}

// A legacy JSON map is edited in place and stays valid JSON: the CLI migrates it
// on its own next write, and until then the file is the one that is read.
func TestOmpEditsALegacyJSONMap(t *testing.T) {
	dir := ompIn(t)
	path := filepath.Join(dir, "keybindings.json")
	writeFile(t, path, "{\n  \"app.exit\": \"ctrl+d\"\n}\n")
	m, err := ReadFlat(OmpFlat)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(m.Values["app.exit"], ",") != "ctrl+d" {
		t.Fatalf("a bare string is one chord: %+v", m.Values)
	}
	if err := WriteFlat(OmpFlat, "app.exit", []string{"ctrl+q", "alt+q"}, false, m.Revision); err != nil {
		t.Fatal(err)
	}
	got := read(t, path)
	if !strings.Contains(got, `"app.exit": ["ctrl+q", "alt+q"]`) {
		t.Fatalf("the JSON map says:\n%s", got)
	}
}

// An action the catalog does not declare is refused by name: writing a row the
// CLI would not read is worse than saying no.
func TestOmpRefusesAnUnknownAction(t *testing.T) {
	ompIn(t)
	m, err := ReadFlat(OmpFlat)
	if err != nil {
		t.Fatal(err)
	}
	err = WriteFlat(OmpFlat, "app.nonesuch", []string{"ctrl+q"}, false, m.Revision)
	if err == nil || !strings.Contains(err.Error(), "not an action") {
		t.Fatalf("an unknown action must be refused by name, got %v", err)
	}
}

// A file that moved under the editor is refused, not overwritten: the pane is
// showing a map that is no longer the file's.
func TestOmpRefusesAStaleRevision(t *testing.T) {
	dir := ompIn(t)
	path := filepath.Join(dir, "keybindings.yml")
	writeFile(t, path, "app.exit: ctrl+d\n")
	m, err := ReadFlat(OmpFlat)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, path, "app.exit: ctrl+d\napp.retry: f5\n")
	err = WriteFlat(OmpFlat, "app.exit", []string{"ctrl+q"}, false, m.Revision)
	if !errors.Is(err, clisettings.ErrStale) {
		t.Fatalf("a moved file must be refused as stale, got %v", err)
	}
	if got := read(t, path); !strings.Contains(got, "app.retry") {
		t.Fatal("the other writer's edit was clobbered")
	}
}

// Reset clears every row this file sets *that the catalog knows*, and leaves
// anything else alone: an id from another version, or a comment, is not
// PiCode's to remove.
func TestOmpResetAllLeavesUnknownKeysAlone(t *testing.T) {
	dir := ompIn(t)
	path := filepath.Join(dir, "keybindings.yml")
	writeFile(t, path, "# mine\napp.exit: ctrl+q\napp.fromAFutureVersion: ctrl+u\ntui.input.submit: alt+s\n")
	m, err := ReadFlat(OmpFlat)
	if err != nil {
		t.Fatal(err)
	}
	if err := ResetFlat(OmpFlat, m.Revision); err != nil {
		t.Fatal(err)
	}
	got := read(t, path)
	if strings.Contains(got, "app.exit") || strings.Contains(got, "tui.input.submit") {
		t.Fatalf("a known row survived the reset:\n%s", got)
	}
	if !strings.Contains(got, "app.fromAFutureVersion: ctrl+u") || !strings.Contains(got, "# mine") {
		t.Fatalf("the reset took something that is not its own:\n%s", got)
	}
}

// A row the file holds in a shape PiCode does not rewrite is reported, not
// swallowed: the rest of the map still reads, and a write to that row is
// refused.
func TestOmpReportsARowItWillNotRewrite(t *testing.T) {
	dir := ompIn(t)
	writeFile(t, filepath.Join(dir, "keybindings.yml"), "app.exit:\n  nested: true\napp.retry: f5\n")
	m, err := ReadFlat(OmpFlat)
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Unreadable) != 1 || m.Unreadable[0] != "app.exit" {
		t.Fatalf("the odd row was not reported: %+v", m.Unreadable)
	}
	if strings.Join(m.Values["app.retry"], ",") != "f5" {
		t.Fatalf("the rest of the map must still read: %+v", m.Values)
	}
	if err := WriteFlat(OmpFlat, "app.exit", []string{"ctrl+q"}, false, m.Revision); err == nil {
		t.Fatal("writing a row held as a table must be refused")
	}
	if err := ResetFlat(OmpFlat, m.Revision); err != nil {
		t.Fatal(err)
	}
	if got := read(t, m.File); !strings.Contains(got, "nested: true") {
		t.Fatal("a reset must not remove a row it does not understand")
	}
}
