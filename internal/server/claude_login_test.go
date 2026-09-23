package server

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/credentials"
	"github.com/cfpperche/picode/internal/store"
)

// Claude Code's logins through the GUI (ADR-0187), one test per row of the
// decision table:
//
//	Use key row                → key in use, approved in ~/.claude.json, env at launch
//	Use subscription row       → file written, key cleared (the key would outrank it)
//	key paused / removed       → nothing injected; the subscription is back
//	person's env sets the var  → never overwritten
func TestClaudeCodeLoginInUse(t *testing.T) {
	ts, dataDir, home, _ := credentialsServer(t)
	st, err := store.Open(filepath.Join(dataDir, "picode.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	deps := Deps{Store: st, DataDir: dataDir}

	// Claude Code's own config, with a field PiCode must keep and the key's
	// tail once rejected.
	const key = "sk-ant-api03-abcdefghijklmnopqrstuvwxyz0123456789"
	tail := key[len(key)-20:]
	cfg := filepath.Join(home, ".claude.json")
	if err := os.WriteFile(cfg, []byte(`{"theme":"dark","customApiKeyResponses":{"approved":[],"rejected":["`+tail+`"]}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	row := cliRequest(t, ts, "POST", "/api/credentials", map[string]any{"provider": "anthropic", "key": key}, 201)
	keyID := row["id"].(string)
	// A second key: the vault never pauses a provider's only live account.
	cliRequest(t, ts, "POST", "/api/credentials", map[string]any{"provider": "anthropic", "key": "sk-ant-api03-second-key-zyxwvutsrqponmlkjihgf"}, 201)

	// Row: Use key row.
	res := cliRequest(t, ts, "POST", "/api/credentials/anthropic/"+keyID+"/activate", map[string]any{"cli": "claude-code"}, 200)
	if res["env"] != claudeKeyEnv {
		t.Fatalf("activate key = %v", res)
	}
	var doc struct {
		Theme     string `json:"theme"`
		Responses struct {
			Approved []string `json:"approved"`
			Rejected []string `json:"rejected"`
		} `json:"customApiKeyResponses"`
	}
	raw, _ := os.ReadFile(cfg)
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if doc.Theme != "dark" || len(doc.Responses.Approved) != 1 || doc.Responses.Approved[0] != tail || len(doc.Responses.Rejected) != 0 {
		t.Fatalf("~/.claude.json = %s", raw)
	}
	env := claudeCredentialEnv(deps, map[string]string{})
	if len(env) != 1 || env[0][0] != claudeKeyEnv || env[0][1] != key {
		t.Fatalf("launch env = %v", env)
	}
	if !rosterRowActive(t, rosterFor(t, ts, "claude-code", "anthropic"), keyID) {
		t.Fatal("the chosen key is not marked in use")
	}

	// Row: the person's own env wins.
	if env := claudeCredentialEnv(deps, map[string]string{claudeKeyEnv: "mine"}); len(env) != 0 {
		t.Fatalf("a user-set %s was overwritten: %v", claudeKeyEnv, env)
	}

	// Row: key paused → nothing injected.
	if _, err := credentials.Default().Pause("anthropic", keyID, true); err != nil {
		t.Fatal(err)
	}
	if env := claudeCredentialEnv(deps, map[string]string{}); len(env) != 0 {
		t.Fatalf("a paused key was injected: %v", env)
	}
	if _, err := credentials.Default().Pause("anthropic", keyID, false); err != nil {
		t.Fatal(err)
	}
	cliRequest(t, ts, "POST", "/api/credentials/anthropic/"+keyID+"/activate", map[string]any{"cli": "claude-code"}, 200)

	// Row: Use subscription row → file written, key cleared.
	if err := claudeVaultSink(deps)("anthropic", map[string]any{"type": "oauth", "access": "sk-ant-oat01-new", "refresh": "r", "expires": int64(9999999999999)}); err != nil {
		t.Fatal(err)
	}
	creds, err := os.ReadFile(filepath.Join(home, ".claude", ".credentials.json"))
	if err != nil {
		t.Fatalf("the browser sign-in wrote no Claude Code file: %v", err)
	}
	if !strings.Contains(string(creds), "sk-ant-oat01-new") || !strings.Contains(string(creds), "user:inference") {
		t.Fatalf(".credentials.json = %s", creds)
	}
	if env := claudeCredentialEnv(deps, map[string]string{}); len(env) != 0 {
		t.Fatalf("the key still rides after the subscription was chosen: %v", env)
	}
	if rosterRowActive(t, rosterFor(t, ts, "claude-code", "anthropic"), keyID) {
		t.Fatal("the key is still marked in use")
	}

	// Row: key removed → nothing injected even if the setting lingers.
	cliRequest(t, ts, "POST", "/api/credentials/anthropic/"+keyID+"/activate", map[string]any{"cli": "claude-code"}, 200)
	if _, err := credentials.Default().Remove("anthropic", keyID); err != nil {
		t.Fatal(err)
	}
	if env := claudeCredentialEnv(deps, map[string]string{}); len(env) != 0 {
		t.Fatalf("a removed key was injected: %v", env)
	}

	// The roster asks for Claude Code's own dialog and offers the browser.
	roster := cliRequest(t, ts, "GET", "/api/credentials?cli=claude-code", nil, 200)
	if add, _ := roster["add"].(map[string]any); add["kind"] != "claude-code" {
		t.Fatalf("add = %v", roster["add"])
	}
	if p := rosterFor(t, ts, "claude-code", "anthropic"); p["signin"] != "browser" {
		t.Fatalf("anthropic signin = %v", p["signin"])
	}
}

func rosterRowActive(t *testing.T, provider map[string]any, id string) bool {
	t.Helper()
	for _, a := range provider["accounts"].([]any) {
		row := a.(map[string]any)
		if row["id"] == id {
			active, _ := row["active"].(bool)
			return active
		}
	}
	t.Fatalf("row %s not in roster", id)
	return false
}
