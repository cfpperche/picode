package pikeys

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCatalogIdsAreUniqueAndLabelled(t *testing.T) {
	seen := map[string]bool{}
	for _, a := range Catalog {
		if seen[a.ID] {
			t.Fatalf("duplicate id %q", a.ID)
		}
		seen[a.ID] = true
		if a.Group == "" || a.Label == "" {
			t.Fatalf("%q has no group or label", a.ID)
		}
	}
}

// An alternate is the whole binding on that platform, and "pi binds nothing
// there" is a declared empty list rather than a missing key — app.suspend is
// that row (pi's docs/keybindings.md: Windows terminals have no Unix job
// control, WSL keeps ctrl+z). The pane reads the map by presence, so an empty
// list has to travel as [] and never as null.
func TestAltDeclaresEachPlatformItCovers(t *testing.T) {
	byID := map[string]Action{}
	for _, a := range Catalog {
		byID[a.ID] = a
	}
	for id, a := range byID {
		for platform := range a.Alt {
			if platform != "windows" && platform != "wsl" {
				t.Fatalf("%s: unknown platform %q", id, platform)
			}
		}
	}
	suspend := byID["app.suspend"]
	if suspend.Alt == nil {
		t.Fatal("app.suspend must declare its Windows row")
	}
	raw, err := json.Marshal(suspend)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"windows":[]`) {
		t.Fatalf("an empty platform binding must travel as []: %s", raw)
	}
	if _, declared := suspend.Alt["wsl"]; declared {
		t.Fatal("wsl keeps the base binding, so there is no alternate to declare")
	}
}

// An alternate that repeats the base default is a copy-paste bug: the pane
// would render a second chip the reader cannot tell from the first.
func TestAlternatesDifferFromTheBaseDefault(t *testing.T) {
	for _, a := range Catalog {
		for platform, keys := range a.Alt {
			if sameKeys(keys, a.Defaults) {
				t.Fatalf("%s: the %s alternate repeats the base default", a.ID, platform)
			}
		}
	}
}

func sameKeys(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
