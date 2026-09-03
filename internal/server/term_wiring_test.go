package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
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
	dataDir := filepath.Join(root, "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatal(err)
	}
	st, err := store.Open(filepath.Join(dataDir, "picode.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	ts := httptest.NewServer(New("127.0.0.1:0", Deps{
		Store: st, DataDir: dataDir, TermStates: NewTermStates(),
	}).Handler)
	t.Cleanup(ts.Close)
	return ts, dataDir
}

func TestInterceptDoesNotWriteUserClaudeSettings(t *testing.T) {
	ts, dataDir := wiringTestServer(t)
	home, _ := os.UserHomeDir()
	userSettings := filepath.Join(home, ".claude", "settings.json")

	if res := postJSON(t, ts, "/api/terminals/wiring/claude-code/enable", map[string]any{}); res.StatusCode != http.StatusOK {
		t.Fatalf("enable = %d", res.StatusCode)
	}
	if _, err := os.Stat(userSettings); !os.IsNotExist(err) {
		t.Fatalf("must not create %s: %v", userSettings, err)
	}
	wrap := wrapperPath(dataDir, "claude")
	body, err := os.ReadFile(wrap)
	if err != nil {
		t.Fatalf("wrapper missing: %v", err)
	}
	if !strings.Contains(string(body), "--settings") {
		t.Fatalf("wrapper does not inject --settings:\n%s", body)
	}
	if !strings.Contains(string(body), claudeSettingsFile(dataDir)) {
		t.Fatal("wrapper must point at the data-dir settings file")
	}
	rawSettings, err := os.ReadFile(claudeSettingsFile(dataDir))
	if err != nil {
		t.Fatalf("intercept settings missing: %v", err)
	}
	if !strings.Contains(string(rawSettings), "TaskCompleted") || !strings.Contains(string(rawSettings), " auto claude-code") {
		t.Fatalf("settings should map Stop/TaskCompleted via auto:\n%s", rawSettings)
	}
	pathEnv := interceptSessionPath(dataDir)
	if !strings.HasPrefix(pathEnv, "PATH="+interceptBinDir(dataDir)) {
		t.Fatalf("session PATH = %q", pathEnv)
	}

	var page struct {
		Clis []wiringRow `json:"clis"`
	}
	res := do(t, ts.Client(), mustGet(t, ts.URL+"/api/terminals/wiring"))
	_ = json.NewDecoder(res.Body).Decode(&page)
	var row *wiringRow
	for i := range page.Clis {
		if page.Clis[i].ID == "claude-code" {
			row = &page.Clis[i]
		}
	}
	if row == nil || !row.Wired {
		t.Fatalf("status = %+v, want wired", page.Clis)
	}

	if res := postJSON(t, ts, "/api/terminals/wiring/claude-code/disable", map[string]any{}); res.StatusCode != http.StatusOK {
		t.Fatalf("disable = %d", res.StatusCode)
	}
	if _, err := os.Stat(wrap); !os.IsNotExist(err) {
		t.Fatal("wrapper should be gone after disable")
	}
	if interceptSessionPath(dataDir) != "" {
		t.Fatal("PATH must not be prepended when nothing is intercepting")
	}
	if interceptBinEnv(dataDir) != "" {
		t.Fatal("PICODE_INTERCEPT_BIN must be empty when nothing is intercepting")
	}
}

func TestInterceptCodexAndGrok(t *testing.T) {
	ts, dataDir := wiringTestServer(t)
	if res := postJSON(t, ts, "/api/terminals/wiring/codex/enable", map[string]any{}); res.StatusCode != http.StatusOK {
		t.Fatalf("codex enable = %d", res.StatusCode)
	}
	body, _ := os.ReadFile(wrapperPath(dataDir, "codex"))
	if !strings.Contains(string(body), "-c") || !strings.Contains(string(body), "notify=") {
		t.Fatalf("codex wrapper:\n%s", body)
	}
	if res := postJSON(t, ts, "/api/terminals/wiring/grok/enable", map[string]any{}); res.StatusCode != http.StatusOK {
		t.Fatalf("grok enable = %d", res.StatusCode)
	}
	body, _ = os.ReadFile(wrapperPath(dataDir, "grok"))
	if !strings.Contains(string(body), "GROK_HOME=") {
		t.Fatalf("grok wrapper:\n%s", body)
	}
	hookJSON := filepath.Join(grokHomeDir(dataDir), "hooks", "picode.json")
	raw, err := os.ReadFile(hookJSON)
	if err != nil {
		t.Fatalf("grok hooks missing: %v", err)
	}
	if !strings.Contains(string(raw), "UserPromptSubmit") {
		t.Fatalf("grok hooks: %s", raw)
	}
}

func TestInterceptRefusesUnknownCLI(t *testing.T) {
	ts, _ := wiringTestServer(t)
	res := postJSON(t, ts, "/api/terminals/wiring/antigravity/enable", map[string]any{})
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("unknown CLI = %d, want 400", res.StatusCode)
	}
}

func TestStripLegacyUserClaudeHooks(t *testing.T) {
	ts, _ := wiringTestServer(t)
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
				map[string]any{"hooks": []any{map[string]any{"type": "command", "command": "/x/picode-hook idle claude-code"}}},
			},
		},
	}
	raw, _ := json.MarshalIndent(doc, "", "  ")
	if err := os.WriteFile(p, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if res := postJSON(t, ts, "/api/terminals/wiring/claude-code/enable", map[string]any{}); res.StatusCode != http.StatusOK {
		t.Fatalf("enable = %d", res.StatusCode)
	}
	got, _ := os.ReadFile(p)
	var after map[string]any
	_ = json.Unmarshal(got, &after)
	if after["model"] != "claude-opus-4-8" {
		t.Fatalf("user model lost: %s", got)
	}
	if strings.Contains(string(got), wiringMarker) {
		t.Fatalf("legacy marker still in user settings: %s", got)
	}
	stop := after["hooks"].(map[string]any)["Stop"].([]any)
	if len(stop) != 1 {
		t.Fatalf("Stop groups = %d, want only the user group", len(stop))
	}
}

func TestHookMapPy(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not on PATH")
	}
	dir := t.TempDir()
	if _, err := ensureHookScript(dir); err != nil {
		t.Fatal(err)
	}
	run := func(in string) string {
		t.Helper()
		cmd := exec.Command("python3", filepath.Join(dir, "picode-hook-map.py"))
		cmd.Stdin = strings.NewReader(in)
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("map %q: %v", in, err)
		}
		return string(out)
	}
	if got := run(`{"hook_event_name":"UserPromptSubmit"}`); got != "working\n" {
		t.Fatalf("prompt = %q", got)
	}
	if got := run(`{"hook_event_name":"Stop"}`); got != "idle\n" {
		t.Fatalf("stop = %q", got)
	}
	if got := run(`{"hook_event_name":"TaskCompleted"}`); got != "idle\n" {
		t.Fatalf("task = %q", got)
	}
	if got := run(`{"type":"agent-turn-complete"}`); got != "idle\n" {
		t.Fatalf("codex = %q", got)
	}
	if got := run(`{"hook_event_name":"Notification","notification_type":"permission_prompt"}`); got != "needs-you\n" {
		t.Fatalf("perm = %q", got)
	}
}

func TestHookScriptTalksToLocalhostInsecure(t *testing.T) {
	if !strings.Contains(hookScriptTmpl, "curl -fsSk") {
		t.Fatal("reporter must curl -k: mkcert is localhost, WSL has no CA")
	}
	if !strings.Contains(hookScriptTmpl, "https://localhost:") {
		t.Fatal("reporter must rewrite 127.0.0.1 → localhost for the cert SAN")
	}
}

func TestClaudeSetWiringRefusesEnable(t *testing.T) {
	_, err := claudeSetWiring("/nope", "", true)
	if err == nil || !strings.Contains(err.Error(), "refusing") {
		t.Fatalf("enable must refuse writing user settings: %v", err)
	}
}
