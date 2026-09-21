package clipkgs

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRemoveArrayElement is the golden table for PiCode's only plugin-list
// write (ADR-0167): the user's own opencode.json(c), edited around the removed
// element and nothing else.
func TestRemoveArrayElement(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		element string
		want    string
		wantErr error
	}{
		{
			name:    "third of three",
			in:      "{\n  \"$schema\": \"https://opencode.ai/config.json\",\n  \"plugin\": [\"a\", \"b\", \"c\"],\n  \"model\": \"x\"\n}\n",
			element: "b",
			want:    "{\n  \"$schema\": \"https://opencode.ai/config.json\",\n  \"plugin\": [\"a\", \"c\"],\n  \"model\": \"x\"\n}\n",
		},
		{
			name:    "first of two",
			in:      "{\"plugin\": [\"a\", \"b\"]}",
			element: "a",
			want:    "{\"plugin\": [\"b\"]}",
		},
		{
			name:    "last of two takes the comma with it",
			in:      "{\"plugin\": [\"a\", \"b\"]}",
			element: "b",
			want:    "{\"plugin\": [\"a\"]}",
		},
		{
			name:    "sole element",
			in:      "{\"plugin\": [\"a\"]}",
			element: "a",
			want:    "{\"plugin\": []}",
		},
		{
			name:    "trailing comma survives the edit",
			in:      "{\"plugin\": [\"a\", \"b\",]}",
			element: "a",
			want:    "{\"plugin\": [\"b\",]}",
		},
		{
			name:    "scoped package with a version",
			in:      "{\"plugin\": [\"@scope/pkg@1.2.3\", \"keep\"]}",
			element: "@scope/pkg@1.2.3",
			want:    "{\"plugin\": [\"keep\"]}",
		},
		{
			name:    "a bracket inside another string does not end the array",
			in:      "{\"note\": \"] still a string\", \"plugin\": [\"a\", \"b\"]}",
			element: "a",
			want:    "{\"note\": \"] still a string\", \"plugin\": [\"b\"]}",
		},
		{
			name:    "a nested plugin key is not the top-level one",
			in:      "{\"nested\": {\"plugin\": [\"x\"]}, \"plugin\": [\"a\", \"b\"]}",
			element: "a",
			want:    "{\"nested\": {\"plugin\": [\"x\"]}, \"plugin\": [\"b\"]}",
		},
		{
			name:    "comments outside the removed span survive",
			in:      "{\n  // keep this note\n  \"plugin\": [\n    // about a\n    \"a\",\n    \"b\" // about b\n  ]\n}\n",
			element: "b",
			want:    "{\n  // keep this note\n  \"plugin\": [\n    // about a\n    \"a\" // about b\n  ]\n}\n",
		},
		{
			name:    "a module the file does not name is not an edit",
			in:      "{\"plugin\": [\"a\"]}",
			element: "zzz",
			wantErr: errNoElement,
		},
		{
			name:    "an unknown shape is refused, not guessed",
			in:      "{\"plugin\": \"a\"}",
			element: "a",
			wantErr: errNoElement,
		},
		{
			name:    "no plugin key at all",
			in:      "{\"model\": \"x\"}",
			element: "a",
			wantErr: errNoElement,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := removeArrayElement([]byte(tc.in), "plugin", tc.element)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("err = %v, want %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("removeArrayElement: %v", err)
			}
			if string(got) != tc.want {
				t.Fatalf("got:\n%s\nwant:\n%s", got, tc.want)
			}
		})
	}
}

// TestOpenCodeRosterReadsBothHalves pins the file-based roster: the npm modules
// the config names (jsonc first, the vendor's conflict winner) and the local
// files that load by their presence.
func TestOpenCodeRosterReadsBothHalves(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	cfg := filepath.Join(dir, "opencode")
	if err := os.MkdirAll(filepath.Join(cfg, "plugins"), 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(cfg, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("opencode.json", `{"$schema":"https://opencode.ai/config.json","plugin":["a","@scope/pkg@1.2.3"]}`)
	write(filepath.Join("plugins", "notify.ts"), "export const P = 1\n")
	if err := os.WriteFile(filepath.Join(cfg, "plugins", "ignore.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	rows, note, err := opencodeRoster(context.Background(), Paths{}, "user", false)
	if err != nil {
		t.Fatal(err)
	}
	if note != "" {
		t.Fatalf("unexpected note %q", note)
	}
	got := map[string]Row{}
	for _, r := range rows {
		got[r.Name] = r
	}
	if len(got) != 3 {
		t.Fatalf("rows = %+v", rows)
	}
	if r := got["a"]; r.SourceKind != "npm" || !r.Enabled || !r.Installed || r.Source != "a" {
		t.Fatalf("npm row = %+v", r)
	}
	if r := got["@scope/pkg"]; r.Version != "1.2.3" {
		t.Fatalf("version was not read off the spec: %+v", r)
	}
	if r := got["notify.ts"]; r.SourceKind != "local-file" || r.Note == "" {
		t.Fatalf("local file row = %+v", r)
	}
	if _, ok := got["ignore.txt"]; ok {
		t.Fatal("a non-plugin file was listed")
	}
}

// TestOpenCodeRosterRefusesABrokenFile: a malformed user file is an error that
// names it, never an empty list (ADR-0150's rule).
func TestOpenCodeRosterRefusesABrokenFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	cfg := filepath.Join(dir, "opencode")
	if err := os.MkdirAll(cfg, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfg, "opencode.json"), []byte("{ not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, _, err := opencodeRoster(context.Background(), Paths{}, "user", false)
	if err == nil || !strings.Contains(err.Error(), "opencode.json") {
		t.Fatalf("a broken config must be named: %v", err)
	}
}

// TestOpenCodeRemoveWritesTheFileTheModuleLivesIn: the jsonc wins a conflict,
// so removal follows the vendor's own precedence.
func TestOpenCodeRemoveWritesTheFileTheModuleLivesIn(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	cfg := filepath.Join(dir, "opencode")
	if err := os.MkdirAll(cfg, 0o755); err != nil {
		t.Fatal(err)
	}
	jsonc := filepath.Join(cfg, "opencode.jsonc")
	plain := filepath.Join(cfg, "opencode.json")
	if err := os.WriteFile(jsonc, []byte("{\n  \"model\": \"x\",\n  \"plugin\": [\"a\", \"b\"]\n}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(plain, []byte("{\"plugin\": [\"a\", \"b\"]}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := opencodeRemove(context.Background(), Paths{}, Target{Name: "a", Scope: "user"}); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(jsonc)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "{\n  \"model\": \"x\",\n  \"plugin\": [\"b\"]\n}\n" {
		t.Fatalf("jsonc = %q", got)
	}
	info, err := os.Stat(jsonc)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode changed to %v", info.Mode().Perm())
	}
	other, err := os.ReadFile(plain)
	if err != nil {
		t.Fatal(err)
	}
	if string(other) != "{\"plugin\": [\"a\", \"b\"]}" {
		t.Fatalf("the second file was touched: %q", other)
	}
}

// TestOpenCodeRemoveRefusesAModuleNoFileNames keeps the refusal honest: the
// pane shows a conflict, not a silent success.
func TestOpenCodeRemoveRefusesAModuleNoFileNames(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	cfg := filepath.Join(dir, "opencode")
	if err := os.MkdirAll(cfg, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfg, "opencode.json"), []byte("{\"plugin\": [\"a\"]}"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := opencodeRemove(context.Background(), Paths{}, Target{Name: "zzz", Scope: "user"})
	if !errors.Is(err, ErrStale) {
		t.Fatalf("err = %v, want ErrStale", err)
	}
}

// TestPackageNameAndVersion pins the spec parsing the pane's columns depend on.
func TestPackageNameAndVersion(t *testing.T) {
	cases := []struct{ spec, name, version string }{
		{"opencode-wakatime", "opencode-wakatime", ""},
		{"npm:opencode-wakatime", "opencode-wakatime", ""},
		{"@scope/pkg@1.2.3", "@scope/pkg", "1.2.3"},
		{"npm:@scope/pkg@^1.2.3", "@scope/pkg", "^1.2.3"},
		{"@scope/pkg", "@scope/pkg", ""},
		{"pkg@v2.0.0", "pkg", "2.0.0"},
	}
	for _, tc := range cases {
		if got := packageName(tc.spec); got != tc.name {
			t.Fatalf("packageName(%q) = %q, want %q", tc.spec, got, tc.name)
		}
		if got := packageVersion(tc.spec); got != tc.version {
			t.Fatalf("packageVersion(%q) = %q, want %q", tc.spec, got, tc.version)
		}
	}
}

// TestOpenCodeProjectInstallsAreVisible is the divergence the live harness
// found (2026-09-20): `opencode plugin <module>` with no -g writes
// <cwd>/.opencode/opencode.json, not <cwd>/opencode.json. A roster that read
// only the documented project file showed nothing after a successful install,
// and a removal would have edited a file the vendor never wrote.
func TestOpenCodeProjectInstallsAreVisible(t *testing.T) {
	ws := t.TempDir()
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	if err := os.MkdirAll(filepath.Join(ws, ".opencode"), 0o755); err != nil {
		t.Fatal(err)
	}
	vendorFile := filepath.Join(ws, ".opencode", "opencode.json")
	documented := filepath.Join(ws, "opencode.json")
	if err := os.WriteFile(vendorFile, []byte("{\n  \"plugin\": [\"opencode-wakatime\"]\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(documented, []byte("{\"model\": \"x\"}"), 0o644); err != nil {
		t.Fatal(err)
	}

	rows, _, err := opencodeRoster(context.Background(), Paths{Cwd: ws}, "project", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Name != "opencode-wakatime" {
		t.Fatalf("a vendor-written project install is invisible: %+v", rows)
	}

	if err := opencodeRemove(context.Background(), Paths{Cwd: ws}, Target{Name: "opencode-wakatime", Scope: "project"}); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(vendorFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "{\n  \"plugin\": []\n}\n" {
		t.Fatalf("removal did not edit the file the install landed in: %q", got)
	}
	untouched, err := os.ReadFile(documented)
	if err != nil {
		t.Fatal(err)
	}
	if string(untouched) != "{\"model\": \"x\"}" {
		t.Fatalf("the documented project file was touched: %q", untouched)
	}
}
