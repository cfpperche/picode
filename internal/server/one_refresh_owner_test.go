package server

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/cfpperche/picode/internal/credentials"
)

// One rotating refresh token, one consumer (ADR-0166 amendment 2026-09-25):
//
//	row's refresh held by another CLI → Use refused (both routes), 409 names the holder
//	row's refresh held by nobody      → Use proceeds
//	two CLIs already hold one refresh → the roster names them on the row
func TestUseRefusesALoginAnotherCLIHolds(t *testing.T) {
	ts, _, home, _ := credentialsServer(t)
	vault := credentials.Default()
	held, err := vault.Import("anthropic", json.RawMessage(`{"type":"oauth","access":"a-cc","refresh":"r-cc","expires":4102444800000}`), "", "vault", "claude@x")
	if err != nil {
		t.Fatal(err)
	}
	free, err := vault.Import("anthropic", json.RawMessage(`{"type":"oauth","access":"a-free","refresh":"r-free","expires":4102444800000}`), "", "vault", "pi@x")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(home, ".claude"), 0o700); err != nil {
		t.Fatal(err)
	}
	cc := `{"claudeAiOauth":{"accessToken":"a-cc","refreshToken":"r-cc","expiresAt":4102444800000,"scopes":["user:inference"],"subscriptionType":"max"}}`
	if err := os.WriteFile(filepath.Join(home, ".claude", ".credentials.json"), []byte(cc), 0o600); err != nil {
		t.Fatal(err)
	}

	res := cliRequest(t, ts, "POST", "/api/credentials/anthropic/"+held.ID+"/activate", map[string]any{"cli": "pi"}, 409)
	if res["heldBy"] != "Claude Code" || res["signInHere"] != true {
		t.Fatalf("guest route refusal = %v", res)
	}
	res = cliRequest(t, ts, "POST", "/api/providers/anthropic/accounts/"+held.ID+"/activate", nil, 409)
	if res["heldBy"] != "Claude Code" {
		t.Fatalf("pi route refusal = %v", res)
	}
	// Use on the CLI that already holds it is not a share.
	cliRequest(t, ts, "POST", "/api/credentials/anthropic/"+held.ID+"/activate", map[string]any{"cli": "claude-code"}, 200)
	// A login nobody holds goes through.
	cliRequest(t, ts, "POST", "/api/credentials/anthropic/"+free.ID+"/activate", map[string]any{"cli": "pi"}, 200)

	// A share made before this rule: pi's auth.json holds Claude Code's token.
	pi := `{"anthropic":{"type":"oauth","access":"a-cc","refresh":"r-cc","expires":4102444800000}}`
	if err := os.WriteFile(filepath.Join(home, ".pi", "agent", "auth.json"), []byte(pi), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, a := range rosterFor(t, ts, "claude-code", "anthropic")["accounts"].([]any) {
		row := a.(map[string]any)
		shared, _ := row["sharedWith"].([]any)
		if row["id"] == held.ID && len(shared) != 2 {
			t.Fatalf("shared row = %v", row)
		}
		if row["id"] == free.ID && len(shared) != 0 {
			t.Fatalf("unshared row marked: %v", row)
		}
	}
}
