package mcp

import "testing"

// Presets render as one-click catalog cards; every entry must be directly
// addable (a remote URL or a runnable command) or the card would create a
// broken server entry.
func TestPresetsAreComplete(t *testing.T) {
	seen := map[string]bool{}
	for _, p := range Presets() {
		if p.ID == "" || p.Name == "" || p.Summary == "" {
			t.Fatalf("preset missing identity: %#v", p)
		}
		if seen[p.ID] {
			t.Fatalf("duplicate preset id %q", p.ID)
		}
		seen[p.ID] = true
		if p.Entry.URL == "" && p.Entry.Command == "" {
			t.Fatalf("preset %q has neither url nor command", p.ID)
		}
	}
	if !seen["gmail"] {
		t.Fatal("gmail preset missing from catalog")
	}
}

// PiCode's own tool cards (ADR-0154) lead the catalog for CLI agents and are
// runnable as written: the daemon's binary, `mcp <family>`.
func TestToolPresetsLeadTheCatalogAndPiHidesThem(t *testing.T) {
	all := Presets()
	if len(all) < 2 || all[0].ID != "picode-computer" || all[1].ID != "picode-browser" {
		t.Fatalf("catalog head = %v", all[:2])
	}
	for _, p := range all[:len(ToolPresets())] {
		if !IsToolPreset(p.ID) || p.Entry.Command == "" || len(p.Entry.Args) != 2 || p.Entry.Args[0] != "mcp" {
			t.Fatalf("tool preset %q = %+v", p.ID, p.Entry)
		}
	}
	for _, p := range WithoutToolPresets(all) {
		if IsToolPreset(p.ID) {
			t.Fatalf("pi still sees %s", p.ID)
		}
	}
	if len(WithoutToolPresets(all)) != len(all)-len(ToolPresets()) {
		t.Fatal("only the tool cards are hidden")
	}
}
