package server

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/cfpperche/picode/internal/rpc"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

// Omp loads extensions from its own settings, so a row for one is not a plugin
// row: the read merges the two sources, and a mutation of a workspace entry is
// a write PiCode performs itself — the CLI has no verb for that file, for a
// removal or for a disable. These tests pin the route's half of that: the row
// reaches the pane, and the mutation answers the CLI's fresh list instead of
// reserving a job that runs no command at all.

// ompSettings is the workspace settings file as these tests read it back.
type ompSettings struct {
	Extensions []string       `json:"extensions"`
	Disabled   []string       `json:"disabledExtensions"`
	Theme      map[string]any `json:"theme"`
}

func readOmpSettings(t *testing.T, file string) ompSettings {
	t.Helper()
	b, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	var doc ompSettings
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatalf("the written settings file does not parse: %v\n%s", err, b)
	}
	return doc
}

// ompExtensionFixture is what both extension route tests start from: a
// workspace whose `.omp/settings.json` is body, an `omp` whose plugin store is
// empty and whose config reads answer nothing (so the workspace file is the
// whole row), and the test server over them.
func ompExtensionFixture(t *testing.T, body string) (*httptest.Server, store.Workspace, string) {
	t.Helper()
	ws := t.TempDir()
	settings := filepath.Join(ws, ".omp", "settings.json")
	if err := os.MkdirAll(filepath.Dir(settings), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(settings, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	stubVendor(t, "omp", `case "$*" in
  *"config get disabledExtensions"*) printf '%s' '{"key":"disabledExtensions","value":[],"type":"array","description":""}' ;;
  *"config get extensions"*) printf '%s' '{"key":"extensions","value":[],"type":"array","description":""}' ;;
  *) printf '%s' '{"npm":[],"marketplace":[]}' ;;
esac`)

	st, err := store.Open(filepath.Join(t.TempDir(), "picode.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	workspace, err := st.AddWorkspace("ws", ws)
	if err != nil {
		t.Fatalf("workspace: %v", err)
	}
	ts := httptest.NewServer(New("127.0.0.1:0", Deps{
		Store:    st,
		Tmux:     tmux.New(),
		Runtime:  rpc.NewRuntime("cat", st, nil),
		AgentCmd: "cat",
		DataDir:  t.TempDir(),
	}).Handler)
	t.Cleanup(ts.Close)
	return ts, workspace, settings
}

// TestCLIPackagesOmpExtensions is the whole arc over the real routes: the
// workspace entry appears in the omp read, and removing it writes the settings
// file — answered with the CLI's own fresh list, not a job.
func TestCLIPackagesOmpExtensions(t *testing.T) {
	ts, workspace, settings := ompExtensionFixture(t, `{
  "extensions": ["pkgs/ext/tool.ts"],
  "theme": {"keep": true}
}`)

	base := "/api/cli-packages?cli=omp&scope=project&workspace=" + workspace.ID + "&refresh=1"
	v := cliRequest(t, ts, "GET", base, nil, 200)
	row := rowBySource(t, v, "pkgs/ext/tool.ts")
	if row["name"] != "tool" || row["scope"] != "project" || row["sourceKind"] != "extension" {
		t.Errorf("row = %v, want the workspace's own extension", row)
	}
	if row["enabled"] != true {
		t.Errorf("row = %v, want it enabled (nothing disables it)", row)
	}

	// The pane sends the row's name and source back. The mutation is a local
	// write with no argv, so the answer is the CLI's fresh list — a 202 job
	// here would run nothing and fail.
	v = cliRequest(t, ts, "POST", "/api/cli-packages/remove", map[string]any{
		"cli": "omp", "scope": "project", "workspace": workspace.ID,
		"name": "tool", "source": "pkgs/ext/tool.ts", "requestKey": "req-ext-1",
	}, 200)
	rows, _ := v["rows"].([]any)
	for _, r := range rows {
		if m, _ := r.(map[string]any); m["source"] == "pkgs/ext/tool.ts" {
			t.Errorf("the row is still there after its removal: %v", m)
		}
	}

	doc := readOmpSettings(t, settings)
	if len(doc.Extensions) != 0 {
		t.Errorf("extensions = %v, want the entry gone", doc.Extensions)
	}
	if doc.Theme["keep"] != true {
		t.Errorf("theme = %v, want every other key untouched", doc.Theme)
	}
}

// TestCLIPackagesOmpExtensionToggle is the same arc for the control the row
// draws: Enable/Disable on an extension runs the CLI's plugin verb, which does
// not know an extension, so the route answers by writing what the CLI itself
// reads — `disabledExtensions` — and hands back the CLI's fresh list. Both
// directions, over the real file.
func TestCLIPackagesOmpExtensionToggle(t *testing.T) {
	ts, workspace, settings := ompExtensionFixture(t, `{
  "extensions": ["pkgs/ext/tool.ts"],
  "theme": {"keep": true}
}`)
	body := map[string]any{
		"cli": "omp", "scope": "project", "workspace": workspace.ID,
		"name": "tool", "source": "pkgs/ext/tool.ts", "on": false,
	}

	// Off: the row comes back disabled, which is the CLI's own list answering.
	v := cliRequest(t, ts, "POST", "/api/cli-packages/toggle", body, 200)
	if row := rowBySource(t, v, "pkgs/ext/tool.ts"); row["enabled"] != false {
		t.Errorf("row = %v, want the row the CLI's own list reports as off", row)
	}
	doc := readOmpSettings(t, settings)
	if len(doc.Disabled) != 1 || doc.Disabled[0] != "extension-module:tool" {
		t.Errorf("disabledExtensions = %v, want the id the toggle wrote", doc.Disabled)
	}
	if doc.Theme["keep"] != true {
		t.Errorf("theme = %v, want every other key untouched", doc.Theme)
	}

	// On: the id leaves the array, and the row is enabled again.
	body["on"] = true
	v = cliRequest(t, ts, "POST", "/api/cli-packages/toggle", body, 200)
	if row := rowBySource(t, v, "pkgs/ext/tool.ts"); row["enabled"] != true {
		t.Errorf("row = %v, want the row enabled again", row)
	}
	doc = readOmpSettings(t, settings)
	if len(doc.Disabled) != 0 {
		t.Errorf("disabledExtensions = %v, want the id gone", doc.Disabled)
	}
	if len(doc.Extensions) != 1 || doc.Extensions[0] != "pkgs/ext/tool.ts" {
		t.Errorf("extensions = %v, want the entry the enable did not touch", doc.Extensions)
	}
}

// rowBySource finds one row of a /api/cli-packages answer by its source.
func rowBySource(t *testing.T, view map[string]any, source string) map[string]any {
	t.Helper()
	rows, _ := view["rows"].([]any)
	for _, r := range rows {
		m, _ := r.(map[string]any)
		if m["source"] == source {
			return m
		}
	}
	t.Fatalf("no row for %q in %v", source, view["rows"])
	return nil
}
