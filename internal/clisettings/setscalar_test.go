package clisettings

import (
	"os"
	"path/filepath"
	"testing"
)

// SetScalar keeps every byte it does not own: a comment beside the map, the
// other entries, and the key order (ADR-0196 slice 3's golden rule).
func TestSetScalarPreservesTheDocument(t *testing.T) {
	cases := []struct {
		name, file string
		f          Format
		in         string
		path       []string
		value      any
		want       string
	}{
		{"json insert into an absent map", "s.json", FormatJSON,
			"{\n  \"model\": \"opus\"\n}\n", []string{"skillOverrides", "pdf"}, "off",
			"{\n  \"model\": \"opus\",\n  \"skillOverrides\": {\"pdf\": \"off\"}\n}\n"},
		{"json replace keeps neighbours", "s.json", FormatJSON,
			"{\n  \"skillOverrides\": {\"pdf\": \"on\", \"docx\": \"off\"}\n}\n", []string{"skillOverrides", "pdf"}, "off",
			"{\n  \"skillOverrides\": {\"pdf\": \"off\", \"docx\": \"off\"}\n}\n"},
		{"jsonc keeps comments", "o.json", FormatJSONC,
			"{\n  // mine\n  \"permission\": {\n    \"skill\": {\n      \"a\": \"allow\" // keep\n    }\n  }\n}\n", []string{"permission", "skill", "a"}, "deny",
			"{\n  // mine\n  \"permission\": {\n    \"skill\": {\n      \"a\": \"deny\" // keep\n    }\n  }\n}\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := filepath.Join(t.TempDir(), tc.file)
			if err := os.WriteFile(p, []byte(tc.in), 0o644); err != nil {
				t.Fatal(err)
			}
			d, err := OpenDoc(p, tc.f)
			if err != nil {
				t.Fatal(err)
			}
			if err := d.SetScalar(tc.path, tc.value); err != nil {
				t.Fatal(err)
			}
			if got := string(d.text); got != tc.want {
				t.Fatalf("got\n%s\nwant\n%s", got, tc.want)
			}
		})
	}
}

func TestSetScalarRefusesATable(t *testing.T) {
	p := filepath.Join(t.TempDir(), "o.json")
	_ = os.WriteFile(p, []byte(`{"permission":{"skill":{"a":{"x":1}}}}`), 0o644)
	d, _ := OpenDoc(p, FormatJSON)
	if err := d.SetScalar([]string{"permission", "skill", "a"}, "deny"); err == nil {
		t.Fatal("a table where a scalar belongs must be refused")
	}
}

// Codex's per-skill switch: one element appended, then found again by its
// path and removed, leaving the user's own bytes as they were.
func TestArrayTableRoundTripKeepsTheUsersBytes(t *testing.T) {
	in := "# my codex\nmodel = \"gpt-5\"\n\n[[skills.config]]\nname = \"mine\"\nenabled = false\n\n# profiles below\n[profiles.fast]\nmodel = \"x\"\n"
	p := filepath.Join(t.TempDir(), "config.toml")
	_ = os.WriteFile(p, []byte(in), 0o644)
	d, err := OpenDoc(p, FormatTOML)
	if err != nil {
		t.Fatal(err)
	}
	if err := d.AppendArrayTable("skills.config", [][2]any{{"path", "/h/.agents/skills/pdf/SKILL.md"}, {"enabled", false}}); err != nil {
		t.Fatal(err)
	}
	if got := len(d.ArrayTables("skills.config")); got != 2 {
		t.Fatalf("elements=%d want 2", got)
	}
	n, err := d.RemoveArrayTables("skills.config", func(e map[string]any) bool { return e["path"] == "/h/.agents/skills/pdf/SKILL.md" })
	if err != nil || n != 1 {
		t.Fatalf("removed %d err %v", n, err)
	}
	if string(d.text) != in {
		t.Fatalf("round trip changed the file:\n%q\nwant\n%q", d.text, in)
	}
	// Removing the user's own element keeps the comment introducing the next table.
	if _, err := d.RemoveArrayTables("skills.config", func(e map[string]any) bool { return e["name"] == "mine" }); err != nil {
		t.Fatal(err)
	}
	want := "# my codex\nmodel = \"gpt-5\"\n\n# profiles below\n[profiles.fast]\nmodel = \"x\"\n"
	if string(d.text) != want {
		t.Fatalf("got %q want %q", d.text, want)
	}
}
