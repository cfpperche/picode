package server

// omp's custom-provider surface (the owner's amendment to ADR-0169): the
// roster carries the door and the definitions, PUT/DELETE write omp's own
// models.yml, and verify/listing spend the saved key on the gateway — never
// on anything else.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ompBody builds one PUT payload the way the form sends it: cli omp, the
// definition, and the key only when the form holds one.
func ompBody(baseURL, key string) string {
	b := `{
		"cli": "omp",
		"baseUrl": "` + baseURL + `/v1",
		"api": "openai-completions",
		"compat": {"supportsDeveloperRole": true, "supportsReasoningEffort": false, "supportsUsageInStreaming": true},
		"models": [{"id": "m1", "reasoning": true, "contextWindow": 128000, "maxTokens": 8192}]`
	if key != "" {
		b += `,"key": "` + key + `"`
	}
	return b + `}`
}

func ompYML(t *testing.T, home string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(home, ".omp", "agent", "models.yml"))
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func ompRosterRow(t *testing.T, ts *httptest.Server, id string) map[string]any {
	t.Helper()
	roster := cliRequest(t, ts, "GET", "/api/credentials?cli=omp", nil, 200)
	for _, p := range roster["providers"].([]any) {
		row := p.(map[string]any)
		if row["id"] == id {
			return row
		}
	}
	return nil
}

func TestOMPCustomProviderLifecycle(t *testing.T) {
	ts, home, req := newCustomProviderServer(t)

	// Create: the definition and its key land in omp's models.yml, one file,
	// never in pi's models.json or the vault.
	if status, b := req(http.MethodPut, "/api/providers/custom/cheaperinference", ompBody("https://api.cheaperinference.com", "ci_live_secret123")); status != http.StatusOK {
		t.Fatalf("PUT status %d: %s", status, b)
	}
	yml := ompYML(t, home)
	if !strings.Contains(yml, "apiKey: ci_live_secret123") || !strings.Contains(yml, "cheaperinference") {
		t.Fatalf("models.yml is missing the definition or key:\n%s", yml)
	}
	if _, err := os.Stat(filepath.Join(home, ".pi", "agent", "models.json")); err == nil {
		t.Fatal("pi's models.json was created by an omp write")
	}

	// The roster carries the door and the definition row, with the keyed flag
	// but never the key itself.
	roster := cliRequest(t, ts, "GET", "/api/credentials?cli=omp", nil, 200)
	door, _ := roster["custom"].(map[string]any)
	if door == nil || door["available"] != true {
		t.Fatalf("omp roster carries no custom door: %v", roster["custom"])
	}
	if href, _ := door["href"].(string); href != "#/clis/omp/providers/custom" {
		t.Fatalf("custom href = %q", href)
	}
	row := ompRosterRow(t, ts, "cheaperinference")
	if row == nil {
		t.Fatal("the custom definition is not on the omp roster")
	}
	if row["custom"] != true {
		t.Fatalf("row is not marked custom: %v", row)
	}
	def, _ := row["definition"].(map[string]any)
	if def == nil || def["baseUrl"] == "" {
		t.Fatalf("row carries no definition: %v", row)
	}
	if def["keyed"] != true {
		t.Fatalf("definition is not marked keyed: %v", def)
	}
	encoded, _ := json.Marshal(roster)
	if strings.Contains(string(encoded), "ci_live_secret123") {
		t.Fatalf("the key leaked into the roster: %s", encoded)
	}

	// Edit with a blank key keeps the stored credential.
	if status, b := req(http.MethodPut, "/api/providers/custom/cheaperinference", ompBody("https://moved.example.com", "")); status != http.StatusOK {
		t.Fatalf("edit status %d: %s", status, b)
	}
	yml = ompYML(t, home)
	if !strings.Contains(yml, "apiKey: ci_live_secret123") {
		t.Fatalf("the blank edit dropped the key:\n%s", yml)
	}
	if !strings.Contains(yml, "https://moved.example.com/v1") {
		t.Fatalf("the edit did not land:\n%s", yml)
	}

	// Remove takes the definition and its key, and only from omp's file.
	if status, b := req(http.MethodDelete, "/api/providers/custom/cheaperinference?cli=omp", ""); status != http.StatusOK {
		t.Fatalf("delete status %d: %s", status, b)
	}
	if ompRosterRow(t, ts, "cheaperinference") != nil {
		t.Fatal("the removed definition is still on the roster")
	}
	if yml := ompYML(t, home); strings.Contains(yml, "cheaperinference") {
		t.Fatalf("the definition survived the delete:\n%s", yml)
	}
}

func TestOMPVerifySpendsTheSavedKey(t *testing.T) {
	_, _, req := newCustomProviderServer(t)
	var auth, path, method string
	var body map[string]any
	gw := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path, auth = r.Method, r.URL.Path, r.Header.Get("Authorization")
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&body)
		}
		w.Write([]byte(`{"choices":[{"message":{"content":"hi"}}],"usage":{"prompt_tokens":3,"completion_tokens":1}}`))
	}))
	defer gw.Close()

	if status, b := req(http.MethodPut, "/api/providers/custom/gw", ompBody(gw.URL, "sk-omp-saved")); status != http.StatusOK {
		t.Fatalf("PUT status %d: %s", status, b)
	}

	// The saved definition is the row's verify: no key in the body, the one
	// in models.yml travels to the gateway the definition names.
	status, raw := req(http.MethodPost, "/api/providers/gw/verify", `{"cli":"omp"}`)
	var res map[string]any
	if err := json.Unmarshal([]byte(raw), &res); err != nil {
		t.Fatalf("body is not JSON: %s", raw)
	}
	if status != http.StatusOK || res["ok"] != true {
		t.Fatalf("status %d res %v", status, res)
	}
	if method != http.MethodPost || path != "/v1/chat/completions" || auth != "Bearer sk-omp-saved" {
		t.Fatalf("the saved key did not reach the endpoint: %s %s %q", method, path, auth)
	}
	if body["max_tokens"] != float64(1) {
		t.Fatalf("the probe was not minimal: %v", body)
	}
	encoded, _ := json.Marshal(res)
	if strings.Contains(string(encoded), "sk-omp-saved") {
		t.Fatalf("the key leaked into the verdict: %s", encoded)
	}
}

func TestOMPModelsListingUsesTheSavedKey(t *testing.T) {
	_, _, req := newCustomProviderServer(t)
	var auth string
	gw := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		w.Write([]byte(`{"object":"list","data":[{"id":"m1"},{"id":"m2"}]}`))
	}))
	defer gw.Close()

	if status, b := req(http.MethodPut, "/api/providers/custom/gw", ompBody(gw.URL, "sk-list-key")); status != http.StatusOK {
		t.Fatalf("PUT status %d: %s", status, b)
	}

	status, raw := req(http.MethodPost, "/api/providers/custom/models",
		`{"cli":"omp","id":"gw","baseUrl":"`+gw.URL+`/v1","api":"openai-completions"}`)
	if status != http.StatusOK {
		t.Fatalf("list status %d: %s", status, raw)
	}
	if auth != "Bearer sk-list-key" {
		t.Fatalf("the listing did not use the saved key: %q", auth)
	}
}

func TestOMPCustomRefusals(t *testing.T) {
	ts, home, req := newCustomProviderServer(t)

	// Another guest CLI has no custom surface: the door does not lie.
	codexBody := strings.Replace(ompBody("https://x.example.com", "k"), `"cli": "omp"`, `"cli": "codex"`, 1)
	if status, b := req(http.MethodPut, "/api/providers/custom/gw", codexBody); status != http.StatusBadRequest {
		t.Fatalf("codex PUT status %d: %s", status, b)
	}

	// omp's built-ins are refused on write and have no verify.
	if status, b := req(http.MethodPut, "/api/providers/custom/openai", ompBody("https://x.example.com", "k")); status != http.StatusBadRequest {
		t.Fatalf("built-in id status %d: %s", status, b)
	}
	if status, b := req(http.MethodPost, "/api/providers/anthropic/verify", `{"cli":"omp"}`); status != http.StatusBadRequest {
		t.Fatalf("omp built-in verify status %d: %s", status, b)
	}
	if _, err := os.Stat(filepath.Join(home, ".omp", "agent", "models.yml")); !os.IsNotExist(err) {
		t.Fatal("a refused write created models.yml")
	}
	_ = ts
}
