package server

// omp's browser OAuth (ADR-0178): the signin route starts the engine for a
// provider it can sign in, the vault sink lands the minted credential as a
// row, and the launch bridge turns the vault's newest row per provider into
// the env names omp's declaration declares — never clobbering the user's.

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/credentials"
)

func TestOmpSigninStartsBrowserOauth(t *testing.T) {
	ts, _, _, _ := credentialsServer(t)

	res := cliRequest(t, ts, "POST", "/api/credentials/signin", map[string]any{"cli": "omp", "provider": "anthropic"}, 200)
	if res["oauth"] != true {
		t.Fatalf("signin = %v, want the browser flow", res)
	}
	url, _ := res["url"].(string)
	if !strings.Contains(url, "claude.ai/oauth/authorize") || !strings.Contains(url, "client_id=") {
		t.Fatalf("authorize url = %q", url)
	}
	// The engine is single-flight: release the listener so a shard sibling
	// (or the next run) can start its own.
	cliRequest(t, ts, "POST", "/api/oauth/cancel", nil, 200)
}

func TestOmpSigninUnknownProviderStaysTerminal(t *testing.T) {
	ts, _, _, _ := credentialsServer(t)
	// google carries no oauth kind for omp: the request falls through to the
	// terminal strip, which for omp means the TUI hint — and no browser url.
	res := cliRequest(t, ts, "POST", "/api/credentials/signin", map[string]any{"cli": "omp", "provider": "google"}, http.StatusCreated)
	if _, isOauth := res["oauth"]; isOauth {
		t.Fatalf("google started the browser flow: %v", res)
	}
	if _, hasTerminal := res["terminalId"]; !hasTerminal {
		t.Fatalf("google signin = %v, want the terminal strip", res)
	}
	cliRequest(t, ts, "POST", "/api/oauth/cancel", nil, 200)
}

func TestOmpVaultSinkLandsARow(t *testing.T) {
	ts, _, _, _ := credentialsServer(t)
	cred := map[string]any{"type": "oauth", "access": "sk-access", "refresh": "sk-refresh", "expires": int64(123)}
	if err := ompVaultSink("anthropic", cred); err != nil {
		t.Fatal(err)
	}
	roster := cliRequest(t, ts, "GET", "/api/credentials?cli=omp", nil, 200)
	for _, p := range roster["providers"].([]any) {
		row := p.(map[string]any)
		if row["id"] != "anthropic" {
			continue
		}
		accounts := row["accounts"].([]any)
		if len(accounts) != 1 {
			t.Fatalf("anthropic accounts = %v", accounts)
		}
		account := accounts[0].(map[string]any)
		if account["type"] != "oauth" {
			t.Fatalf("account type = %v", account["type"])
		}
		return
	}
	t.Fatal("the minted credential is not on the omp roster")
}

func TestOmpCredentialEnvBridge(t *testing.T) {
	_, _, _, _ = credentialsServer(t)
	// One oauth row per provider per guest: the vault's fingerprint is a
	// constant for oauth rows (their tokens rotate, so the bytes are not an
	// identity), so a fresh sign-in replaces the account instead of stacking.
	// Google's api key rides its own env name.
	if err := ompVaultSink("anthropic", map[string]any{"type": "oauth", "access": "old", "refresh": "r1"}); err != nil {
		t.Fatal(err)
	}
	if err := ompVaultSink("google", map[string]any{"type": "api_key", "key": "g-key"}); err != nil {
		t.Fatal(err)
	}
	if err := ompVaultSink("anthropic", map[string]any{"type": "oauth", "access": "new", "refresh": "r2"}); err != nil {
		t.Fatal(err)
	}
	file, err := credentials.Default().Load()
	if err != nil {
		t.Fatal(err)
	}
	if got := len(file.Providers["anthropic"].Accounts); got != 1 {
		t.Fatalf("anthropic accounts = %d, want one (re-login replaces)", got)
	}

	env := ompCredentialEnv(&file, map[string]string{})
	got := map[string]string{}
	for _, kv := range env {
		got[kv[0]] = kv[1]
	}
	if got["ANTHROPIC_OAUTH_TOKEN"] != "new" {
		t.Fatalf("ANTHROPIC_OAUTH_TOKEN = %q, want the replaced row", got["ANTHROPIC_OAUTH_TOKEN"])
	}
	if got["GEMINI_API_KEY"] != "g-key" {
		t.Fatalf("GEMINI_API_KEY = %q", got["GEMINI_API_KEY"])
	}

	// A user-configured env entry is never clobbered.
	env2 := ompCredentialEnv(&file, map[string]string{"ANTHROPIC_OAUTH_TOKEN": "user-set"})
	for _, kv := range env2 {
		if kv[0] == "ANTHROPIC_OAUTH_TOKEN" {
			t.Fatal("the bridge clobbered a user-configured env entry")
		}
	}
}

func TestOmpVaultSinkRoundTripsThroughJSON(t *testing.T) {
	_, _, _, _ = credentialsServer(t)
	if err := ompVaultSink("github-copilot", map[string]any{"type": "oauth", "access": "a", "refresh": "r", "expires": int64(9), "accountId": "acc"}); err != nil {
		t.Fatal(err)
	}
	file, err := credentials.Default().Load()
	if err != nil {
		t.Fatal(err)
	}
	slot := file.Providers["github-copilot"]
	if len(slot.Accounts) != 1 {
		t.Fatalf("accounts = %d", len(slot.Accounts))
	}
	var cred map[string]any
	if err := json.Unmarshal(slot.Accounts[0].Cred, &cred); err != nil {
		t.Fatal(err)
	}
	if cred["type"] != "oauth" || cred["access"] != "a" || cred["accountId"] != "acc" {
		t.Fatalf("cred = %v", cred)
	}
}
