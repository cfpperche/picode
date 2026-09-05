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
