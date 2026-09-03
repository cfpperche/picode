package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/store"
)

func wiringTestServer(t *testing.T) (*httptest.Server, string) {
	t.Helper()
	root := t.TempDir()
	t.Setenv("HOME", root)
	t.Setenv("USERPROFILE", root)
	st, err := store.Open(filepath.Join(root, "data", "picode.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	dataDir := filepath.Join(root, "data")
	ts := httptest.NewServer(New("127.0.0.1:0", Deps{
		Store: st, DataDir: dataDir, TermStates: NewTermStates(),
	}).Handler)
	t.Cleanup(ts.Close)
	return ts, dataDir
}

// User content written like a human would: a model line, a custom Stop
// group, unrelated keys. Every wiring operation must preserve it.
func userClaudeSettings(t *testing.T) string {
	p, err := claudeSettingsPath()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	doc := map[string]any{
		"model": "claude-opus-4-8",
		"hooks": map[string]any{
			"Stop": []any{
				map[string]any{"hooks": []any{map[string]any{"type": "command", "command": "/usr/bin/say done"}}},
			},
		},
	}
	raw, _ := json.MarshalIndent(doc, "", "  ")
	if err := os.WriteFile(p, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func readDoc(t *testing.T, p string) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("settings not JSON: %v", err)
	}
	return doc
}

func TestClaudeWiringEnableDisablePreservesUserContent(t *testing.T) {
	ts, dataDir := wiringTestServer(t)
	p := userClaudeSettings(t)

	// Enable: our three events appear, the user's Stop group and model stay.
	if res := postJSON(t, ts, "/api/terminals/wiring/claude-code/enable", map[string]any{}); res.StatusCode != http.StatusOK {
		t.Fatalf("enable = %d", res.StatusCode)
	}
	doc := readDoc(t, p)
	if doc["model"] != "claude-opus-4-8" {
		t.Fatalf("user model lost: %+v", doc)
	}
	hooks := doc["hooks"].(map[string]any)
	for _, ev := range []string{"UserPromptSubmit", "Stop", "Notification"} {
		groups, _ := hooks[ev].([]any)
		nOurs, nTotal := 0, len(groups)
		for _, g := range groups {
			if groupHasMarker(g) {
				nOurs++
				hooksMap := g.(map[string]any)["hooks"].([]any)
				cmd, _ := hooksMap[0].(map[string]any)["command"].(string)
				if !strings.HasPrefix(cmd, filepath.Join(dataDir, wiringMarker)) {
					t.Fatalf("command %q does not use the data-dir reporter", cmd)
				}
			}
		}
		if nOurs != 1 {
			t.Fatalf("%s: %d marker groups, want 1 (total %d)", ev, nOurs, nTotal)
		}
	}
	// The user's own Stop command is still there, next to ours.
	stop := hooks["Stop"].([]any)
	if len(stop) != 2 {
		t.Fatalf("Stop groups = %d, want 2 (user + ours)", len(stop))
	}

	// Double enable is idempotent: still exactly one marker group per event.
	if res := postJSON(t, ts, "/api/terminals/wiring/claude-code/enable", map[string]any{}); res.StatusCode != http.StatusOK {
		t.Fatalf("re-enable = %d", res.StatusCode)
	}
	doc = readDoc(t, p)
	hooks = doc["hooks"].(map[string]any)
	for _, ev := range []string{"UserPromptSubmit", "Stop", "Notification"} {
		groups := hooks[ev].([]any)
		n := 0
		for _, g := range groups {
			if groupHasMarker(g) {
				n++
			}
		}
		if n != 1 {
			t.Fatalf("%s: %d marker groups after double enable, want 1", ev, n)
		}
	}

	// Status agrees.
	res := do(t, ts.Client(), mustGet(t, ts.URL+"/api/terminals/wiring"))
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", res.StatusCode)
	}
	var page struct {
		Clis []wiringRow `json:"clis"`
	}
	_ = json.NewDecoder(res.Body).Decode(&page)
	var claude *wiringRow
	for i := range page.Clis {
		if page.Clis[i].ID == "claude-code" {
			claude = &page.Clis[i]
		}
	}
	if claude == nil || !claude.Wired {
		t.Fatalf("claude row = %+v, want wired", claude)
	}

	// The reporter exists and is executable.
	info, err := os.Stat(filepath.Join(dataDir, wiringMarker))
	if err != nil {
		t.Fatalf("reporter script missing: %v", err)
	}
	if info.Mode().Perm()&0o111 == 0 {
		t.Fatal("reporter script is not executable")
	}

	// Disable: our entries go, the user's Stop group and model stay, and a
	// hooks map that only held ours disappears with the key.
	if res := postJSON(t, ts, "/api/terminals/wiring/claude-code/disable", map[string]any{}); res.StatusCode != http.StatusOK {
		t.Fatalf("disable = %d", res.StatusCode)
	}
	doc = readDoc(t, p)
	if doc["model"] != "claude-opus-4-8" {
		t.Fatalf("user model lost on disable: %+v", doc)
	}
	hooks = doc["hooks"].(map[string]any)
	stop = hooks["Stop"].([]any)
	if len(stop) != 1 || groupHasMarker(stop[0]) {
		t.Fatalf("Stop after disable = %+v, want only the user group", stop)
	}
	for _, ev := range []string{"UserPromptSubmit", "Notification"} {
		if _, ok := hooks[ev]; ok {
			t.Fatalf("%s should be gone after disable", ev)
		}
	}
	page = struct {
		Clis []wiringRow `json:"clis"`
	}{}
	res = do(t, ts.Client(), mustGet(t, ts.URL+"/api/terminals/wiring"))
	_ = json.NewDecoder(res.Body).Decode(&page)
	for _, row := range page.Clis {
		if row.ID == "claude-code" && row.Wired {
			t.Fatal("status still wired after disable")
		}
	}
}

func TestWiringRefusesUnknownCLI(t *testing.T) {
	ts, _ := wiringTestServer(t)
	res := postJSON(t, ts, "/api/terminals/wiring/codex/enable", map[string]any{})
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("codex enable = %d, want 400 with visible reason", res.StatusCode)
	}
	var body map[string]any
	_ = json.NewDecoder(res.Body).Decode(&body)
	if msg, _ := body["error"].(string); !strings.Contains(msg, "guide") {
		t.Fatalf("error message %q should point at the guide", msg)
	}
}

func TestClaudeWiringRefusesCorruptSettings(t *testing.T) {
	ts, _ := wiringTestServer(t)
	p := userClaudeSettings(t)
	if err := os.WriteFile(p, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	res := postJSON(t, ts, "/api/terminals/wiring/claude-code/enable", map[string]any{})
	if res.StatusCode != http.StatusInternalServerError {
		t.Fatalf("corrupt settings = %d, want 500", res.StatusCode)
	}
	var body map[string]any
	_ = json.NewDecoder(res.Body).Decode(&body)
	if msg, _ := body["error"].(string); !strings.Contains(msg, "not valid JSON") {
		t.Fatalf("error %q should name the JSON problem", msg)
	}
}
