package clikeys

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The OpenCode catalog: every key the loader's own Definitions table declares
// except the leader-key configuration and the object-form paste binding, in ten
// contiguous groups, each keeping the vendor's chord spelling (including
// `<leader>` two-stroke sequences and the literal "none" for ships-disabled).
func TestOpenCodeCatalogIsWellFormed(t *testing.T) {
	if len(OpenCodeCatalog) != 162 {
		t.Fatalf("the vendor's Definitions table declares 162 actions, the catalog has %d", len(OpenCodeCatalog))
	}
	seen, groups := map[string]bool{}, []string{}
	unbound := 0
	for _, a := range OpenCodeCatalog {
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
		if a.ID == "leader" || a.ID == "input_paste" {
			t.Fatalf("%s must not be in the catalog (config key / object form)", a.ID)
		}
		if len(a.Defaults) == 0 {
			unbound++
		}
	}
	if unbound == 0 {
		t.Fatal("the vendor's own table ships actions disabled; the catalog says none")
	}
	if len(groups) < 20 {
		t.Fatalf("the namespaces collapsed: %v", groups)
	}
}

func writeOpenCodeHome(t *testing.T, body string) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := filepath.Join(home, ".config", "opencode")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "tui.json")
	if body != "" {
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return path
}

// The map is a flat `keybinds` object nested one level down, the vendor's
// "none" reads as unbound, and a write lands inside the existing object without
// disturbing a sibling key PiCode does not manage.
func TestOpenCodeWritesOneRow(t *testing.T) {
	path := writeOpenCodeHome(t, "{\n  \"$schema\": \"https://opencode.ai/tui.json\",\n  \"keybinds\": {\n    \"app_exit\": \"ctrl+c,ctrl+d,<leader>q\"\n  }\n}\n")
	report, err := ReadFlat(OpenCodeMap)
	if err != nil {
		t.Fatal(err)
	}
	if report.File != path || !report.Exists {
		t.Fatalf("file=%q exists=%v", report.File, report.Exists)
	}
	// The vendor's "none" is a disabled action: reported unbound, not a chord
	// named none.
	if strings.Join(report.Values["app_debug"], ",") != "" {
		t.Fatalf("a disabled action must read as unset: %+v", report.Values["app_debug"])
	}
	if err := WriteFlat(OpenCodeMap, "command_list", []string{"ctrl+alt+p"}, false, report.Revision); err != nil {
		t.Fatal(err)
	}
	got := read(t, path)
	if !strings.Contains(got, `"command_list": ["ctrl+alt+p"]`) {
		t.Fatalf("the row did not land in keybinds:\n%s", got)
	}
	if !strings.Contains(got, "app_exit") || !strings.Contains(got, "$schema") {
		t.Fatalf("the sibling key and $schema did not survive:\n%s", got)
	}
	// Reset hands the row back: the key leaves keybinds, the rest stays.
	report, err = ReadFlat(OpenCodeMap)
	if err != nil {
		t.Fatal(err)
	}
	if err := WriteFlat(OpenCodeMap, "command_list", nil, true, report.Revision); err != nil {
		t.Fatal(err)
	}
	got = read(t, path)
	if strings.Contains(got, "command_list") || !strings.Contains(got, "app_exit") {
		t.Fatalf("the reset disturbed the file:\n%s", got)
	}
}

// A project tui.json deep-merges over the user's rows and wins (the loader's
// file order), so the declaration edits the user file and says so rather than
// pretending it is the only layer.
func TestOpenCodePrefersTheJsoncSibling(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := filepath.Join(home, ".config", "opencode")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "tui.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	jsonc := filepath.Join(dir, "tui.jsonc")
	if err := os.WriteFile(jsonc, []byte("// mine\n{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := ReadFlat(OpenCodeMap)
	if err != nil {
		t.Fatal(err)
	}
	if report.File != jsonc {
		t.Fatalf("a .jsonc sibling wins in the same directory, got %s", report.File)
	}
}
