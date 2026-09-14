package snips

import "testing"

// A starter that does not parse is a broken product surface that ships in
// every binary: the editor would open it and refuse to save it.
func TestTemplatesParseAndAreDistinct(t *testing.T) {
	seen := map[string]bool{}
	for _, tpl := range Templates() {
		if tpl.ID == "" || tpl.Title == "" || tpl.Description == "" || tpl.Body == "" {
			t.Fatalf("template %q has an empty field: %+v", tpl.ID, tpl)
		}
		if seen[tpl.ID] {
			t.Fatalf("duplicate template id %q", tpl.ID)
		}
		seen[tpl.ID] = true
		if tpl.Kind != "prompt" && tpl.Kind != "shell" {
			t.Fatalf("template %q: kind %q is not prompt or shell", tpl.ID, tpl.Kind)
		}
		if got := Slug(tpl.Title); got == "" {
			t.Fatalf("template %q: title %q has no slug", tpl.ID, tpl.Title)
		}
		phs, err := Parse(tpl.Body)
		if err != nil {
			t.Fatalf("template %q does not parse: %v", tpl.ID, err)
		}
		if len(phs) > MaxPlaceholders {
			t.Fatalf("template %q has %d placeholders", tpl.ID, len(phs))
		}
		for _, tag := range tpl.Tags {
			if tag == "" {
				t.Fatalf("template %q has an empty tag", tpl.ID)
			}
		}
	}
	if len(seen) < 6 {
		t.Fatalf("expected at least six starters, got %d", len(seen))
	}
}

// Every starter must expand with empty values: the ones with defaults fill
// in, and the ones without say what is missing instead of panicking.
func TestTemplatesExpandWithoutInput(t *testing.T) {
	for _, tpl := range Templates() {
		out, missing, err := Expand(tpl.Body, nil, nil)
		if err != nil {
			t.Fatalf("template %q does not expand: %v", tpl.ID, err)
		}
		if out == "" {
			t.Fatalf("template %q expands to nothing", tpl.ID)
		}
		for _, name := range missing {
			if Reserved[name] || name == "" {
				t.Fatalf("template %q reports reserved name %q as missing", tpl.ID, name)
			}
		}
	}
}
