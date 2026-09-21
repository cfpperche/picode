package clikeys

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// TestJSListMatchesTheKeyboardRegistry is the seam between the registry (what a
// CLI's map is) and web/shared/domain/cliKeys.js (what the pane says about it):
// same ids, same state, same pickup. It is the same pattern
// TestJSListMatchesTheCatalog holds for the guest settings table, and it is the
// only thing keeping a CLI added on one side from being invisible on the other.
func TestJSListMatchesTheKeyboardRegistry(t *testing.T) {
	body, err := os.ReadFile("../../web/shared/domain/cliKeys.js")
	if err != nil {
		t.Fatal(err)
	}
	block := regexp.MustCompile(`(?s)KEYBOARD_CLIS = \[(.*?)\n\];`).FindSubmatch(body)
	if block == nil {
		t.Fatal("KEYBOARD_CLIS not found in web/shared/domain/cliKeys.js")
	}
	row := regexp.MustCompile(`\{ id: "([^"]+)", label: "([^"]+)", state: "([^"]+)", pickup: "([^"]+)"`)
	js := map[string][2]string{}
	order := []string{}
	for _, m := range row.FindAllStringSubmatch(string(block[1]), -1) {
		if _, seen := js[m[1]]; !seen {
			order = append(order, m[1])
		}
		js[m[1]] = [2]string{m[3], m[4]}
	}
	if len(js) != len(Registry) {
		t.Fatalf("the UI lists %d CLIs, the registry has %d", len(js), len(Registry))
	}
	for _, cli := range Registry {
		got, ok := js[cli.ID]
		if !ok {
			t.Fatalf("%s is in the registry and not in the UI list", cli.ID)
		}
		if got[0] != string(cli.State) || got[1] != string(cli.Pickup) {
			t.Fatalf("%s: UI says state=%s pickup=%s, registry says state=%s pickup=%s",
				cli.ID, got[0], got[1], cli.State, cli.Pickup)
		}
	}
	goOrder := append([]string{}, Supported()...)
	if strings.Join(order, ",") != strings.Join(goOrder, ",") {
		t.Fatalf("catalog order differs:\n  go: %v\n  js: %v", goOrder, order)
	}
}

// A registry row is a claim about someone else's software, so it must say where
// the claim came from — the rule internal/clisettings follows for its defaults.
func TestEveryRowCitesItsSource(t *testing.T) {
	for _, cli := range Registry {
		if strings.TrimSpace(cli.Source) == "" {
			t.Fatalf("%s declares vendor behaviour with no source", cli.ID)
		}
		if cli.Source == "" || len(cli.Source) < 20 {
			t.Fatalf("%s: source reads as a placeholder", cli.ID)
		}
	}
}

// The three states are three different answers, and only `shipped` means PiCode
// writes this CLI's map. A row that claims `shipped` also has to say how the
// CLI takes a change.
func TestOnlyShippedRowsPromiseAnEditor(t *testing.T) {
	for _, cli := range Registry {
		if cli.State == Shipped && cli.Pickup == PickupUnknown {
			t.Fatalf("%s is shipped, so its pickup cannot be unknown", cli.ID)
		}
		if cli.State == Refused && cli.Keymap != None {
			t.Fatalf("%s refuses remapping but declares keymap %q", cli.ID, cli.Keymap)
		}
		if cli.State != Refused && cli.Keymap == None {
			t.Fatalf("%s declares no key map but is not refused", cli.ID)
		}
	}
}

// Registry order is the sidebar's catalog order, and For() answers for every
// row — the pane opens a CLI by id and nothing else.
func TestRegistryIdsAreUniqueAndAddressable(t *testing.T) {
	seen := map[string]bool{}
	for i, cli := range Registry {
		if seen[cli.ID] {
			t.Fatalf("duplicate id %q", cli.ID)
		}
		seen[cli.ID] = true
		if For(cli.ID) != &Registry[i] {
			t.Fatalf("For(%q) does not return the row at %d", cli.ID, i)
		}
	}
	if Supported()[0] != "pi" {
		t.Fatalf("the pane's list starts with %q, not pi", Supported()[0])
	}
	if For("nonesuch") != nil {
		t.Fatal("For() invented a CLI")
	}
}
