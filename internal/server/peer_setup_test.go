package server

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPeerAutomaticSetupAPI(t *testing.T) {
	ts, data, home := cleanupServer(t)
	wk := cliRequest(t, ts, "POST", "/api/workspaces", map[string]any{"name": "Peer fixture", "path": home}, 201)
	a := cliRequest(t, ts, "POST", "/api/workspaces/"+wk["id"].(string)+"/agents", map[string]any{"name": "Peer fixture"}, 201)
	id := a["id"].(string)
	session := filepath.Join(home, "conversation.jsonl")
	cliRequest(t, ts, "PATCH", "/api/agents/"+id, map[string]any{"sessionPath": session}, 200)
	body := map[string]any{"kind": "agent", "ownerId": id, "sessionKey": session, "automatic": true}
	// No installed adapter: fail before minting/replacing any connection.
	cliRequest(t, ts, "POST", "/api/communication", body, 400)
	list := cliRequest(t, ts, "GET", "/api/communication", nil, 200)
	if len(list["connections"].([]any)) != 0 {
		t.Fatal("created connection without adapter")
	}
	adapter := filepath.Join(home, ".pi", "agent", "npm", "node_modules", "pi-mcp-adapter", "index.ts")
	if err := os.MkdirAll(filepath.Dir(adapter), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(adapter, []byte("fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	result := cliRequest(t, ts, "POST", "/api/communication", body, 201)
	if result["token"] != nil || result["automatic"] != true {
		t.Fatal("automatic response exposed credential")
	}
	peer := result["connection"].(map[string]any)["id"].(string)
	list = cliRequest(t, ts, "GET", "/api/communication", nil, 200)
	if list["launches"].(map[string]any)[peer] != true {
		t.Fatal("saved setup unavailable")
	}
	// Simulate a storage failure: new connection must not remain active.
	if err := os.Rename(filepath.Join(data, "communication"), filepath.Join(data, "saved-communication")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(data, "communication"), []byte("block directory"), 0600); err != nil {
		t.Fatal(err)
	}
	cliRequest(t, ts, "POST", "/api/communication", body, 500)
	list = cliRequest(t, ts, "GET", "/api/communication", nil, 200)
	for _, p := range list["connections"].([]any) {
		if p.(map[string]any)["active"] == true {
			t.Fatal("failed setup remained active")
		}
	}
}
