package server

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakePiScript prints a list-models table naming the custom provider, so
// catalog.Load answers without a real pi binary.
func fakePiScript(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fakepi")
	script := "#!/bin/sh\n" +
		"if [ \"$1\" = \"--list-models\" ]; then\n" +
		"  echo \"provider model context max-out thinking images\"\n" +
		"  echo \"cheaperinference gpt-5.4 200K 32K yes no\"\n" +
		"fi\n"
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	return path
}

// newCustomProviderServer points HOME at a scratch dir and serves with the
// fake pi. Returns the server, the scratch home and a JSON request helper.
func newCustomProviderServer(t *testing.T) (*httptest.Server, string, func(method, path, body string) (int, string)) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	ts := newTestServer(t, fakePiScript(t))
	doReq := func(method, path, body string) (int, string) {
		req, err := http.NewRequest(method, ts.URL+path, bytes.NewReader([]byte(body)))
		if err != nil {
			t.Fatal(err)
		}
		res := do(t, ts.Client(), req)
		b, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatal(err)
		}
		return res.StatusCode, string(b)
	}
	return ts, home, doReq
}

func readHomeFile(t *testing.T, home, rel string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(home, rel))
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func TestCustomProviderLifecycle(t *testing.T) {
	_, home, req := newCustomProviderServer(t)

	// Create with key: definition lands in models.json, key in auth.json.
	status, body := req(http.MethodPut, "/api/providers/custom/cheaperinference", `{
		"baseUrl": "https://api.cheaperinference.com/v1",
		"api": "openai-completions",
		"compat": {"supportsDeveloperRole": false, "supportsReasoningEffort": false},
		"models": [{"id": "gpt-5.4", "contextWindow": 400000, "maxTokens": 32000}],
		"key": "ci_live_secret123"
	}`)
	if status != http.StatusOK {
		t.Fatalf("PUT status %d: %s", status, body)
	}
	modelsJSON := readHomeFile(t, home, ".pi/agent/models.json")
	if !strings.Contains(modelsJSON, "api.cheaperinference.com") {
		t.Fatalf("definition missing: %s", modelsJSON)
	}
	if strings.Contains(modelsJSON, "ci_live_secret123") {
		t.Fatal("key leaked into models.json")
	}
	if !strings.Contains(readHomeFile(t, home, ".pi/agent/auth.json"), "ci_live_secret123") {
		t.Fatal("key missing from auth.json")
	}

	// Catalog shows it signed in, custom, with config — and never the key.
	status, body = req(http.MethodGet, "/api/catalog", "")
	if status != http.StatusOK {
		t.Fatalf("catalog status %d", status)
	}
	if !strings.Contains(body, `"custom":true`) || !strings.Contains(body, "cheaperinference") {
		t.Fatalf("custom provider missing from catalog: %s", body)
	}
	if strings.Contains(body, "ci_live_secret123") {
		t.Fatal("catalog leaked key material")
	}

	// Edit without key: definition updates, credential survives.
	status, body = req(http.MethodPut, "/api/providers/custom/cheaperinference", `{
		"baseUrl": "https://api.cheaperinference.com",
		"api": "anthropic-messages",
		"compat": {"supportsDeveloperRole": false, "supportsReasoningEffort": false},
		"models": [{"id": "claude-opus-5"}]
	}`)
	if status != http.StatusOK {
		t.Fatalf("edit status %d: %s", status, body)
	}
	if !strings.Contains(readHomeFile(t, home, ".pi/agent/models.json"), "anthropic-messages") {
		t.Fatal("edit not applied")
	}
	if !strings.Contains(readHomeFile(t, home, ".pi/agent/auth.json"), "ci_live_secret123") {
		t.Fatal("edit dropped the key")
	}

	// Delete removes definition and credential; second delete is 404.
	if status, body = req(http.MethodDelete, "/api/providers/custom/cheaperinference", ""); status != http.StatusOK {
		t.Fatalf("DELETE status %d: %s", status, body)
	}
	if strings.Contains(readHomeFile(t, home, ".pi/agent/models.json"), "cheaperinference") {
		t.Fatal("definition survived delete")
	}
	if strings.Contains(readHomeFile(t, home, ".pi/agent/auth.json"), "cheaperinference") {
		t.Fatal("credential survived delete")
	}
	if status, _ = req(http.MethodDelete, "/api/providers/custom/cheaperinference", ""); status != http.StatusNotFound {
		t.Fatalf("second DELETE status %d, want 404", status)
	}
}

func TestCustomProviderRefusals(t *testing.T) {
	_, _, req := newCustomProviderServer(t)
	body := `{"baseUrl":"https://x.com/v1","api":"openai-completions","models":[{"id":"m"}],"key":"k"}`
	for _, tc := range []struct {
		name, path, payload string
		want                int
	}{
		{"builtin collision", "/api/providers/custom/anthropic", body, http.StatusBadRequest},
		{"bad url", "/api/providers/custom/gw", `{"baseUrl":"notaurl","api":"openai-completions","models":[{"id":"m"}]}`, http.StatusBadRequest},
		{"no models", "/api/providers/custom/gw", `{"baseUrl":"https://x.com/v1","api":"openai-completions","models":[]}`, http.StatusBadRequest},
		{"bad json", "/api/providers/custom/gw", `{`, http.StatusBadRequest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if status, got := req(http.MethodPut, tc.path, tc.payload); status != tc.want {
				t.Fatalf("status %d, want %d: %s", status, tc.want, got)
			}
		})
	}
	if status, _ := req(http.MethodDelete, "/api/providers/custom/anthropic", ""); status != http.StatusNotFound {
		t.Fatalf("delete builtin status %d, want 404", status)
	}
}

// A reasoning model's level selection reaches models.json as pi's
// thinkingLevelMap, comes back in the catalog for the form prefill, and an
// invented level is refused before the file is touched.
func TestCustomProviderThinkingLevels(t *testing.T) {
	_, home, req := newCustomProviderServer(t)
	status, body := req(http.MethodPut, "/api/providers/custom/cheaperinference", `{
		"baseUrl": "https://api.cheaperinference.com/v1",
		"api": "openai-completions",
		"compat": {"supportsDeveloperRole": false, "supportsReasoningEffort": true},
		"models": [{"id": "deepseek-v4.1-flash", "reasoning": true, "thinkingLevels": ["high", "max"]}],
		"key": "ci_live_secret123"
	}`)
	if status != http.StatusOK {
		t.Fatalf("PUT status %d: %s", status, body)
	}
	modelsJSON := readHomeFile(t, home, ".pi/agent/models.json")
	for _, want := range []string{`"reasoning": true`, `"max": "max"`, `"minimal": null`, `"xhigh": null`} {
		if !strings.Contains(modelsJSON, want) {
			t.Fatalf("models.json missing %s: %s", want, modelsJSON)
		}
	}

	// The catalog hands the map back so Edit prefills the same chips.
	status, body = req(http.MethodGet, "/api/catalog", "")
	if status != http.StatusOK {
		t.Fatalf("catalog status %d", status)
	}
	if !strings.Contains(body, `"thinkingLevelMap"`) {
		t.Fatalf("catalog dropped the level map: %s", body)
	}

	bad := `{"baseUrl":"https://x.com/v1","api":"openai-completions","models":[{"id":"m","thinkingLevels":["turbo"]}]}`
	if status, body = req(http.MethodPut, "/api/providers/custom/othergw", bad); status != http.StatusBadRequest {
		t.Fatalf("bad level status %d: %s", status, body)
	}
	if strings.Contains(readHomeFile(t, home, ".pi/agent/models.json"), "othergw") {
		t.Fatal("a refused definition reached the file")
	}
}

// The thinking format is the one string compat key the form owns: it lands in
// models.json beside the bools, comes back in the catalog for the prefill, and
// an unknown format is refused before the file is touched.
func TestCustomProviderThinkingFormat(t *testing.T) {
	_, home, req := newCustomProviderServer(t)
	status, body := req(http.MethodPut, "/api/providers/custom/gw", `{
		"baseUrl": "https://api.example.com/v1",
		"api": "openai-completions",
		"compat": {"supportsDeveloperRole": false, "supportsReasoningEffort": true},
		"thinkingFormat": "deepseek",
		"models": [{"id": "deepseek-v4.1-flash", "reasoning": true, "thinkingLevels": ["high", "max"]}]
	}`)
	if status != http.StatusOK {
		t.Fatalf("PUT status %d: %s", status, body)
	}
	modelsJSON := readHomeFile(t, home, ".pi/agent/models.json")
	if !strings.Contains(modelsJSON, `"thinkingFormat": "deepseek"`) {
		t.Fatalf("format missing from models.json: %s", modelsJSON)
	}

	status, body = req(http.MethodGet, "/api/catalog", "")
	if status != http.StatusOK {
		t.Fatalf("catalog status %d", status)
	}
	if !strings.Contains(body, `"thinkingFormat":"deepseek"`) {
		t.Fatalf("catalog dropped the format: %s", body)
	}

	// A hand-edited string compat key must not hide the provider (it used to:
	// the whole entry failed to decode and the row vanished from the GUI).
	if !strings.Contains(body, `"custom":true`) {
		t.Fatalf("custom row missing: %s", body)
	}

	bad := `{"baseUrl":"https://x.com/v1","api":"openai-completions","thinkingFormat":"mind-meld","models":[{"id":"m"}]}`
	if status, body = req(http.MethodPut, "/api/providers/custom/othergw", bad); status != http.StatusBadRequest {
		t.Fatalf("bad format status %d: %s", status, body)
	}
	if strings.Contains(readHomeFile(t, home, ".pi/agent/models.json"), "othergw") {
		t.Fatal("a refused definition reached the file")
	}
}

// The unsigned definition still shows in the catalog (available, not signed
// in) so it can be picked up again from Add provider.
func TestCustomProviderUnsignedShowsInCatalog(t *testing.T) {
	_, home, req := newCustomProviderServer(t)
	status, body := req(http.MethodPut, "/api/providers/custom/cheaperinference", `{
		"baseUrl": "https://api.cheaperinference.com/v1",
		"api": "openai-completions",
		"models": [{"id": "gpt-5.4"}]
	}`)
	if status != http.StatusOK {
		t.Fatalf("PUT status %d: %s", status, body)
	}
	status, body = req(http.MethodGet, "/api/catalog", "")
	if status != http.StatusOK {
		t.Fatalf("catalog status %d", status)
	}
	if !strings.Contains(body, `"custom":true`) {
		t.Fatalf("custom flag missing: %s", body)
	}
	if !strings.Contains(body, `"signedIn":false`) {
		t.Fatalf("expected unsigned custom provider: %s", body)
	}
	_ = readHomeFile(t, home, ".pi/agent/models.json")
}
