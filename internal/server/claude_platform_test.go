package server

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/store"
)

// Claude Code on a third-party platform (ADR-0189), one test per row:
//
//	PUT a platform             → the wizard's env block; other platforms' keys and the key choice gone; the person's own env and fields kept
//	roster                     → what the platform is, never a secret; no row in use
//	launch                     → no ANTHROPIC_API_KEY (the platform outranks it)
//	Use subscription / key     → platform removed, the person's env kept
//	DELETE                     → platform removed
//	bad form / unreadable file → refused, file untouched
func TestClaudeCodePlatform(t *testing.T) {
	ts, dataDir, home, _ := credentialsServer(t)
	st, err := store.Open(filepath.Join(dataDir, "picode.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	deps := Deps{Store: st, DataDir: dataDir}

	settings := filepath.Join(home, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(settings), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(settings, []byte(`{"model":"opus","env":{"MY_VAR":"kept","CLAUDE_CODE_USE_VERTEX":"1","CLOUD_ML_REGION":"old"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	readEnv := func() (map[string]any, map[string]any) {
		t.Helper()
		raw, err := os.ReadFile(settings)
		if err != nil {
			t.Fatal(err)
		}
		var doc map[string]any
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatal(err)
		}
		env, _ := doc["env"].(map[string]any)
		return doc, env
	}

	// A Console key in use first, so the platform has a choice to end.
	key := cliRequest(t, ts, "POST", "/api/credentials", map[string]any{"provider": "anthropic", "key": "sk-ant-api03-platform-test-key-0123456789"}, 201)
	keyID := key["id"].(string)
	cliRequest(t, ts, "POST", "/api/credentials/anthropic/"+keyID+"/activate", map[string]any{"cli": "claude-code"}, 200)
	// Using the key removed the Vertex setup that was there.
	if _, env := readEnv(); env["CLAUDE_CODE_USE_VERTEX"] != nil || env["MY_VAR"] != "kept" {
		t.Fatalf("Use key: env = %v", env)
	}

	// Row: PUT Bedrock with a Bedrock API key.
	res := cliRequest(t, ts, "PUT", "/api/claude-code/platform", map[string]any{"kind": "bedrock", "auth": "bearer", "region": "us-east-1", "bearerToken": "ABSKsecret"}, 200)
	if p, _ := res["platform"].(map[string]any); p["kind"] != "bedrock" || p["auth"] != "bearer" {
		t.Fatalf("PUT = %v", res)
	}
	doc, env := readEnv()
	if doc["model"] != "opus" || env["MY_VAR"] != "kept" || env["CLAUDE_CODE_USE_BEDROCK"] != "1" ||
		env["AWS_REGION"] != "us-east-1" || env["AWS_BEARER_TOKEN_BEDROCK"] != "ABSKsecret" {
		t.Fatalf("settings after PUT = %v", doc)
	}
	if env := claudeCredentialEnv(deps, map[string]string{}); len(env) != 0 {
		t.Fatalf("a key rides next to a platform: %v", env)
	}

	// Row: roster.
	roster := cliRequest(t, ts, "GET", "/api/credentials?cli=claude-code", nil, 200)
	raw, _ := json.Marshal(roster)
	if strings.Contains(string(raw), "ABSKsecret") {
		t.Fatal("the roster carries the platform's secret")
	}
	if p, _ := roster["platform"].(map[string]any); p["name"] != "Amazon Bedrock" || p["region"] != "us-east-1" {
		t.Fatalf("roster platform = %v", roster["platform"])
	}
	if rosterRowActive(t, rosterFor(t, ts, "claude-code", "anthropic"), keyID) {
		t.Fatal("a key is marked in use under a platform")
	}

	// Row: switching platforms removes the other's keys.
	cliRequest(t, ts, "PUT", "/api/claude-code/platform", map[string]any{"kind": "vertex", "auth": "adc", "project": "my-proj", "region": "us-east5"}, 200)
	if _, env := readEnv(); env["CLAUDE_CODE_USE_BEDROCK"] != nil || env["AWS_BEARER_TOKEN_BEDROCK"] != nil || env["CLAUDE_CODE_USE_VERTEX"] != "1" || env["ANTHROPIC_VERTEX_PROJECT_ID"] != "my-proj" {
		t.Fatalf("Vertex after Bedrock: env = %v", env)
	}

	// Row: Use subscription → platform removed.
	if err := claudeVaultSink(deps)("anthropic", map[string]any{"type": "oauth", "access": "sk-ant-oat01-x", "refresh": "r", "expires": int64(9999999999999)}); err != nil {
		t.Fatal(err)
	}
	if _, env := readEnv(); env["CLAUDE_CODE_USE_VERTEX"] != nil || env["MY_VAR"] != "kept" {
		t.Fatalf("Use subscription left the platform: env = %v", env)
	}

	// Row: DELETE.
	cliRequest(t, ts, "PUT", "/api/claude-code/platform", map[string]any{"kind": "foundry", "auth": "apiKey", "resource": "my-res", "apiKey": "fk"}, 200)
	cliRequest(t, ts, "DELETE", "/api/claude-code/platform", nil, 200)
	if _, env := readEnv(); env["CLAUDE_CODE_USE_FOUNDRY"] != nil || env["ANTHROPIC_FOUNDRY_API_KEY"] != nil || env["MY_VAR"] != "kept" {
		t.Fatalf("DELETE: env = %v", env)
	}

	// Row: refusals.
	for _, bad := range []map[string]any{
		{"kind": "bedrock", "auth": "bearer", "region": "", "bearerToken": "x"},
		{"kind": "vertex", "auth": "serviceAccount", "project": "p", "region": "global", "keyFile": "relative.json"},
		{"kind": "foundry", "auth": "apiKey", "resource": "bad name!", "apiKey": "k"},
		{"kind": "mars"},
	} {
		if st := cliRequestFull(t, ts, "PUT", "/api/claude-code/platform", bad); st["status"] != "400" {
			t.Fatalf("PUT %v = %v, want 400", bad, st)
		}
	}
	if err := os.WriteFile(settings, []byte(`{not json`), 0o600); err != nil {
		t.Fatal(err)
	}
	if st := cliRequestFull(t, ts, "PUT", "/api/claude-code/platform", map[string]any{"kind": "bedrock", "auth": "environment", "region": "us-west-2"}); st["status"] != "500" {
		t.Fatalf("unreadable settings: %v", st)
	}
	if raw, _ := os.ReadFile(settings); string(raw) != `{not json` {
		t.Fatalf("an unreadable settings file was overwritten: %s", raw)
	}
}
