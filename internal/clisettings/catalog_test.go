package clisettings

import (
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/climemory"
)

// The pane asks the server for a CLI's schema, so the list it uses to decide
// whether to ask must be the list the server answers for. Two files, one
// truth: this test is the seam.
func TestJSListMatchesTheCatalog(t *testing.T) {
	body, err := os.ReadFile("../../web/shared/domain/cliNative.js")
	if err != nil {
		t.Fatal(err)
	}
	match := regexp.MustCompile(`NATIVE_SETTINGS_CLIS = \[([^\]]*)\]`).FindSubmatch(body)
	if match == nil {
		t.Fatal("NATIVE_SETTINGS_CLIS not found in web/shared/domain/cliNative.js")
	}
	var js []string
	for _, part := range strings.Split(string(match[1]), ",") {
		id := strings.Trim(strings.TrimSpace(part), `"`)
		if id != "" {
			js = append(js, id)
		}
	}
	got := append([]string{}, Supported()...)
	sort.Strings(got)
	sort.Strings(js)
	if strings.Join(got, ",") != strings.Join(js, ",") {
		t.Fatalf("the UI list and the Go catalog disagree:\n  go: %v\n  js: %v", got, js)
	}
	for _, id := range got {
		if id == "pi" {
			t.Fatal("Pi keeps its own editor (ADR-0101) and must not be in this catalog")
		}
	}
}

// Every CLI PiCode launches answers the Memory pane, and every CLI with a
// settings schema is one PiCode launches. A CLI added to the launch catalog
// without a memory answer would render a blank pane.
func TestEveryManagedCLIAnswersMemory(t *testing.T) {
	managed := []string{"pi", "claude-code", "codex", "grok", "hermes", "opencode", "muse", "agy", "omp"}
	for _, cli := range managed {
		if climemory.For(cli) == nil {
			t.Errorf("%s has no memory driver: the pane would render nothing", cli)
		}
	}
	for _, cli := range Supported() {
		found := false
		for _, m := range managed {
			if m == cli {
				found = true
			}
		}
		if !found {
			t.Errorf("%s has a settings schema but is not a managed CLI", cli)
		}
	}
}

// Every declared field carries the copy the pane renders: a label, and either a
// fallback saying what the CLI does when the key is unset or help saying what
// the key is. A row with neither is a blank the reader cannot act on.
func TestEveryFieldCarriesItsCopy(t *testing.T) {
	for _, cli := range Supported() {
		for _, f := range For(cli).fields {
			switch {
			case f.Label == "":
				t.Errorf("%s/%s has no label", cli, f.Key)
			case f.Fallback == "" && f.Help == "":
				t.Errorf("%s/%s says neither what it does nor what happens when it is unset", cli, f.Key)
			case f.Kind == KindSelect && len(f.Options) == 0:
				t.Errorf("%s/%s is a select with no options", cli, f.Key)
			}
			if f.Danger != "" && f.Kind == KindSelect {
				found := false
				for _, o := range f.Options {
					if o.Value == f.Danger {
						found = true
					}
				}
				if !found {
					t.Errorf("%s/%s names a dangerous value %q that is not one of its options", cli, f.Key, f.Danger)
				}
			}
		}
	}
}

// TestDangerNotesAreDistinctWithinAGroup: two dangerous rows in one group
// printed the same 17 words twice (visual review, 2026-09-20). Each dangerous
// choice says what it specifically costs.
func TestDangerNotesAreDistinctWithinAGroup(t *testing.T) {
	for _, cli := range Supported() {
		seen := map[string]string{}
		for _, f := range For(cli).fields {
			if f.Danger == "" {
				continue
			}
			if f.DangerNote == "" {
				t.Errorf("%s/%s marks a dangerous value with no note", cli, f.Key)
				continue
			}
			key := f.Group + "|" + f.DangerNote
			if other, dup := seen[key]; dup {
				t.Errorf("%s: %s and %s print the same warning in group %q", cli, other, f.Key, f.Group)
			}
			seen[key] = f.Key
		}
	}
}
