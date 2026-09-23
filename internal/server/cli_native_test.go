package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/climodels"
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

// TestCLISettingsAPICarriesTheRoleMatrix: omp's role rows are computed per
// request from its own catalog plus the files, and a role the pane writes lands
// in the file the CLI reads — including the workspace layer, which omp's own
// `config set` cannot reach (measured 2026-09-22: it writes the global file
// wherever it runs, and `--scope` is not an option).
func TestCLISettingsAPICarriesTheRoleMatrix(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	ts := newTestServer(t, "cat")

	config := filepath.Join(home, ".omp", "agent", "config.yml")
	seedFile(t, config, "modelRoles:\n  default: deepseek/deepseek-flash:max\nsymbolPreset: ascii\n")

	status, body := getJSONBody(t, ts, ts.URL+"/api/cli-settings?cli=omp")
	if status != http.StatusOK {
		t.Fatalf("GET status %d: %v", status, body)
	}
	roles, _ := body["roles"].(map[string]any)
	if roles == nil {
		t.Fatal("omp declares a role matrix; the pane needs its prefixes and catalog")
	}
	if roles["rolePrefix"] != "modelRoles." || roles["chainPrefix"] != "retry.fallbackChains." {
		t.Errorf("prefixes = %v / %v", roles["rolePrefix"], roles["chainPrefix"])
	}
	if catalog, _ := roles["catalog"].([]any); len(catalog) != 15 {
		t.Errorf("the vendor has 15 built-in roles, the report carries %d", len(catalog))
	}
	fields, _ := body["fields"].([]any)
	var sawDefault, sawJudge bool
	for _, raw := range fields {
		f, _ := raw.(map[string]any)
		switch f["key"] {
		case "modelRoles.default":
			sawDefault = true
			if f["kind"] != "role" || f["tag"] != "DEFAULT" {
				t.Errorf("the default row is %#v", f)
			}
		case "modelRoles.judge":
			sawJudge = true
		}
	}
	if !sawDefault || !sawJudge {
		t.Error("every role the CLI has is a row, assigned or not")
	}
	layers, _ := body["layers"].([]any)
	layer, _ := layers[0].(map[string]any)
	values, _ := layer["values"].(map[string]any)
	if values["modelRoles.default"] != "deepseek/deepseek-flash:max" {
		t.Errorf("values = %#v", values)
	}
	revision, _ := layer["revision"].(string)

	res := patchJSON(t, ts, "/api/cli-settings", map[string]any{
		"cli": "omp", "scope": "user", "revision": revision,
		"set": map[string]any{
			"modelRoles.plan":              "anthropic/claude-opus-4-8:high",
			"retry.fallbackChains.default": []any{"@smol", "openai/gpt-5"},
		},
	})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("PATCH status %d: %s", res.StatusCode, thisBody(res))
	}
	got, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"plan: anthropic/claude-opus-4-8:high",
		"default: [\"@smol\", \"openai/gpt-5\"]",
		"symbolPreset: ascii",
	} {
		if !strings.Contains(string(got), want) {
			t.Errorf("the file is missing %q:\n%s", want, got)
		}
	}
}

// A key outside the vendor's own naming rule is refused by name, and the file
// is left exactly as it was.
func TestCLISettingsAPIRefusesARoleTheCLIWouldNotAccept(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	ts := newTestServer(t, "cat")
	config := filepath.Join(home, ".omp", "agent", "config.yml")
	seedFile(t, config, "symbolPreset: ascii\n")

	for _, key := range []string{"modelRoles.9lives", "retry.fallbackChains.a: b", "modelRoles."} {
		res := patchJSON(t, ts, "/api/cli-settings", map[string]any{
			"cli": "omp", "scope": "user",
			"set": map[string]any{key: "openai/gpt-5"},
		})
		if res.StatusCode != http.StatusBadRequest {
			t.Errorf("%s: want 400, got %d: %s", key, res.StatusCode, thisBody(res))
		}
	}
	got, _ := os.ReadFile(config)
	if string(got) != "symbolPreset: ascii\n" {
		t.Errorf("a refused write changed the file:\n%s", got)
	}
}

// The model picker asks the CLI, and only a CLI PiCode has measured a way to
// ask answers at all.
func TestCLIModelsAPIRefusesCLIsItCannotAsk(t *testing.T) {
	ts := newTestServer(t, "cat")
	for _, cli := range []string{"", "codex", "claude-code"} {
		status, _ := getJSONBody(t, ts, ts.URL+"/api/cli-models?cli="+cli)
		if status != http.StatusBadRequest {
			t.Errorf("cli=%q: want 400, got %d", cli, status)
		}
	}
}

func TestCLIDoctorAPIRefusesCLIsWithoutChecks(t *testing.T) {
	ts := newTestServer(t, "cat")
	for _, cli := range []string{"pi", "", "codex"} {
		status, _ := getJSONBody(t, ts, ts.URL+"/api/cli-doctor?cli="+cli)
		if status != http.StatusBadRequest {
			t.Errorf("cli=%q: want 400, got %d", cli, status)
		}
	}
}

// Pi has a reader (ADR-0009 amendment): the API runs the daemon's configured
// Pi, not whatever "pi" is on PATH.
func TestCLIModelsAPIAsksTheConfiguredPi(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "my-pi")
	table := "provider model context max-out thinking images\nanthropic claude-haiku-4-5 200K 64K yes yes\n"
	if err := os.WriteFile(script, []byte("#!/bin/sh\nprintf '"+table+"'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { climodels.Forget("pi") })
	ts := newTestServer(t, script)
	status, body := getJSONBody(t, ts, ts.URL+"/api/cli-models?cli=pi&fresh=1")
	if status != http.StatusOK {
		t.Fatalf("status %d: %v", status, body)
	}
	models, _ := body["models"].([]any)
	if len(models) != 1 || models[0].(map[string]any)["selector"] != "anthropic/claude-haiku-4-5" {
		t.Fatalf("models = %v", body["models"])
	}
}
