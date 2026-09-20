package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func seedFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func patchJSON(t *testing.T, ts *httptest.Server, path string, body any) *http.Response {
	t.Helper()
	return postJSONMethod(t, ts, http.MethodPatch, path, body)
}

func putJSON(t *testing.T, ts *httptest.Server, path string, body any) *http.Response {
	t.Helper()
	return postJSONMethod(t, ts, http.MethodPut, path, body)
}

func deleteReq(t *testing.T, ts *httptest.Server, path string) *http.Response {
	t.Helper()
	req, _ := http.NewRequest(http.MethodDelete, ts.URL+path, nil)
	return do(t, ts.Client(), req)
}

func getJSONBody(t *testing.T, ts *httptest.Server, url string) (int, map[string]any) {
	t.Helper()
	res, err := ts.Client().Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var body map[string]any
	_ = json.NewDecoder(res.Body).Decode(&body)
	return res.StatusCode, body
}

// TestCLISettingsAPIReadsAndWritesNativeFiles walks the round trip the pane
// makes: read a guest CLI's own config, change one key, and see the change in
// the file the CLI reads.
func TestCLISettingsAPIReadsAndWritesNativeFiles(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	ts := newTestServer(t, "cat")

	config := filepath.Join(home, ".codex", "config.toml")
	seedFile(t, config, "model = \"gpt-6\"\nsandbox_mode = \"read-only\"   # kept tight\n")

	status, body := getJSONBody(t, ts, ts.URL+"/api/cli-settings?cli=codex")
	if status != http.StatusOK {
		t.Fatalf("GET status %d: %v", status, body)
	}
	layers, _ := body["layers"].([]any)
	if len(layers) != 1 {
		t.Fatalf("codex has one config file, got %d layers", len(layers))
	}
	layer, _ := layers[0].(map[string]any)
	values, _ := layer["values"].(map[string]any)
	if values["model"] != "gpt-6" || values["sandbox_mode"] != "read-only" {
		t.Fatalf("values = %#v", values)
	}
	revision, _ := layer["revision"].(string)
	if revision == "" {
		t.Fatal("a layer that exists owes the editor a revision")
	}
	if fields, _ := body["fields"].([]any); len(fields) == 0 {
		t.Fatal("the pane needs the field schema")
	}

	res := patchJSON(t, ts, "/api/cli-settings", map[string]any{
		"cli": "codex", "scope": "user", "revision": revision,
		"set": map[string]any{"sandbox_mode": "workspace-write"},
	})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("PATCH status %d: %s", res.StatusCode, thisBody(res))
	}
	got, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), `sandbox_mode = "workspace-write"   # kept tight`) {
		t.Fatalf("the save moved more than the value:\n%s", got)
	}
}

// TestCLISettingsAPIRefusesAStaleWrite: the CLI writes this file too, so a save
// built on an older read answers 409 and the file is untouched.
func TestCLISettingsAPIRefusesAStaleWrite(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	ts := newTestServer(t, "cat")

	config := filepath.Join(home, ".hermes", "config.yaml")
	seedFile(t, config, "memory:\n  memory_enabled: true\n")
	_, body := getJSONBody(t, ts, ts.URL+"/api/cli-settings?cli=hermes")
	layers, _ := body["layers"].([]any)
	layer, _ := layers[0].(map[string]any)
	revision, _ := layer["revision"].(string)

	// Another writer moves first.
	seedFile(t, config, "memory:\n  memory_enabled: true\n  nudge_interval: 10\n")

	res := patchJSON(t, ts, "/api/cli-settings", map[string]any{
		"cli": "hermes", "scope": "user", "revision": revision,
		"set": map[string]any{"memory.memory_enabled": false},
	})
	if res.StatusCode != http.StatusConflict {
		t.Fatalf("want 409, got %d: %s", res.StatusCode, thisBody(res))
	}
	got, _ := os.ReadFile(config)
	if !strings.Contains(string(got), "nudge_interval: 10") || !strings.Contains(string(got), "memory_enabled: true") {
		t.Fatalf("the other writer's file was disturbed:\n%s", got)
	}
}

// TestCLISettingsAPIRefusesPiAndUnknownCLIs: Pi keeps its own editor and its
// own trust rules (ADR-0101); a CLI with no declaration has no editor.
func TestCLISettingsAPIRefusesPiAndUnknownCLIs(t *testing.T) {
	ts := newTestServer(t, "cat")
	for _, cli := range []string{"pi", "", "aider"} {
		status, _ := getJSONBody(t, ts, ts.URL+"/api/cli-settings?cli="+cli)
		if status != http.StatusBadRequest {
			t.Errorf("cli=%q: want 400, got %d", cli, status)
		}
	}
}

// TestCLIMemoryAPIReportsEveryTierHonestly is the pane's contract: what it may
// do with a CLI's memory comes from the driver, and a CLI without one says so.
func TestCLIMemoryAPIReportsEveryTierHonestly(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	ts := newTestServer(t, "cat")

	for cli, tier := range map[string]string{
		"claude-code": "editable",
		"hermes":      "editable",
		"muse":        "editable",
		"omp":         "editable",
		"grok":        "readonly",
		"codex":       "readonly",
		"pi":          "none",
		"opencode":    "none",
		"agy":         "unknown",
	} {
		status, body := getJSONBody(t, ts, ts.URL+"/api/cli-memory?cli="+cli)
		if status != http.StatusOK {
			t.Errorf("%s: status %d", cli, status)
			continue
		}
		rep, _ := body["report"].(map[string]any)
		if rep["tier"] != tier {
			t.Errorf("%s: tier %v, want %s", cli, rep["tier"], tier)
		}
		if tier == "none" || tier == "unknown" {
			if note, _ := rep["note"].(string); note == "" {
				t.Errorf("%s: a CLI with no memory owes the reader one line", cli)
			}
		}
	}
}

// TestCLIMemoryAPIListsReadsAndDeletes on an editable store, and masks a
// credential the agent wrote down.
func TestCLIMemoryAPIListsReadsAndDeletes(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	ts := newTestServer(t, "cat")

	dir := filepath.Join(home, ".hermes", "memories")
	seedFile(t, filepath.Join(dir, "MEMORY.md"), "The owner runs make close before landing.\n")
	seedFile(t, filepath.Join(dir, "USER.md"), "Prefers Portuguese. Token ghp_abcdefghijklmnopqrstuvwxyz012345 was shown once.\n")

	status, body := getJSONBody(t, ts, ts.URL+"/api/cli-memory?cli=hermes")
	if status != http.StatusOK {
		t.Fatalf("status %d: %v", status, body)
	}
	items, _ := body["items"].([]any)
	if len(items) != 2 {
		t.Fatalf("want 2 items, got %d: %v", len(items), items)
	}
	first, _ := items[0].(map[string]any)
	if first["index"] != true {
		t.Fatalf("the index sorts first, got %#v", first)
	}

	status, read := getJSONBody(t, ts, ts.URL+"/api/cli-memory/item?cli=hermes&scope=global&id=USER.md")
	if status != http.StatusOK {
		t.Fatalf("read status %d: %v", status, read)
	}
	text, _ := read["body"].(string)
	if strings.Contains(text, "ghp_abcdefghijklmnopqrstuvwxyz012345") {
		t.Fatalf("a credential was echoed: %q", text)
	}
	if !strings.Contains(text, "Prefers Portuguese.") {
		t.Fatalf("prose was over-masked: %q", text)
	}

	res := deleteReq(t, ts, "/api/cli-memory/item?cli=hermes&scope=global&id=USER.md")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("delete status %d: %s", res.StatusCode, thisBody(res))
	}
	if _, err := os.Stat(filepath.Join(dir, "USER.md")); !os.IsNotExist(err) {
		t.Fatal("delete did not remove the file")
	}
	// The index is the map to everything else.
	res = deleteReq(t, ts, "/api/cli-memory/item?cli=hermes&scope=global&id=MEMORY.md")
	if res.StatusCode == http.StatusOK {
		t.Fatal("the index must not be deletable")
	}
}

// TestCLIMemoryAPIRefusesWritesOnGeneratedStores: Grok and Codex own theirs.
func TestCLIMemoryAPIRefusesWritesOnGeneratedStores(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	ts := newTestServer(t, "cat")

	dir := filepath.Join(home, ".grok", "memory-v2", "global")
	seedFile(t, filepath.Join(dir, "MEMORY.md"), "# Global memory index\n\n> Generated by Grok. Do not edit this file directly.\n")
	seedFile(t, filepath.Join(dir, "testing.md"), "Run make ci-scoped.\n")

	status, body := getJSONBody(t, ts, ts.URL+"/api/cli-memory?cli=grok&scope=global")
	if status != http.StatusOK {
		t.Fatalf("status %d", status)
	}
	rep, _ := body["report"].(map[string]any)
	if rep["clear"] != "grok memory clear" {
		t.Fatalf("a read-only store must name the vendor command, got %v", rep["clear"])
	}
	items, _ := body["items"].([]any)
	if len(items) != 2 {
		t.Fatalf("want 2 items, got %d", len(items))
	}
	index, _ := items[0].(map[string]any)
	if index["generated"] != true {
		t.Fatalf("the generated banner must be reported: %#v", index)
	}

	res := putJSON(t, ts, "/api/cli-memory/item", map[string]any{
		"cli": "grok", "scope": "global", "id": "testing.md", "body": "changed",
	})
	if res.StatusCode == http.StatusOK {
		t.Fatal("a write into a generated store must be refused")
	}
	res = deleteReq(t, ts, "/api/cli-memory/item?cli=grok&scope=global&id=testing.md")
	if res.StatusCode == http.StatusOK {
		t.Fatal("a delete in a generated store must be refused")
	}
	if _, err := os.Stat(filepath.Join(dir, "testing.md")); err != nil {
		t.Fatal("the file must survive a refused delete")
	}
}

// TestCLIMemoryAPIRefusesPathsOutsideTheStore.
func TestCLIMemoryAPIRefusesPathsOutsideTheStore(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	ts := newTestServer(t, "cat")
	seedFile(t, filepath.Join(home, ".hermes", "memories", "MEMORY.md"), "index\n")
	seedFile(t, filepath.Join(home, "secret.md"), "not a memory\n")

	for _, id := range []string{"..%2f..%2fsecret.md", "%2Fetc%2Fpasswd"} {
		status, _ := getJSONBody(t, ts, ts.URL+"/api/cli-memory/item?cli=hermes&scope=global&id="+id)
		if status == http.StatusOK {
			t.Errorf("%s was readable", id)
		}
	}
}
