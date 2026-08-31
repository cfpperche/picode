package server

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

// Decision-table row 5: no file and broken JSON both answer {"state": null},
// 200 — the chip hides instead of erroring; a real v1 file comes through and
// a future contract version is hidden rather than guessed at.
func TestAgentRoleState(t *testing.T) {
	ts := newTestServer(t, "cat")
	defer ts.Close()
	wk := addWorkspaceWithAgent(t, ts, "RoleState", t.TempDir())
	agentID := wk.Agents[0].ID

	rolesStateRoot = t.TempDir()
	t.Cleanup(func() { rolesStateRoot = "" })

	get := func() map[string]any {
		t.Helper()
		res, err := http.Get(ts.URL + "/api/agents/" + agentID + "/role-state")
		if err != nil {
			t.Fatalf("get: %v", err)
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK {
			t.Fatalf("status %d", res.StatusCode)
		}
		var out map[string]any
		if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
			t.Fatalf("json: %v", err)
		}
		return out
	}

	if out := get(); out["state"] != nil {
		t.Fatalf("missing file should be null state, got %v", out)
	}

	path := filepath.Join(rolesStateRoot, agentID+".json")
	if err := os.WriteFile(path, []byte("{broken"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out := get(); out["state"] != nil {
		t.Fatalf("broken JSON should be null state, got %v", out)
	}

	body := `{"v":1,"mode":"lock","role":"vision","model":"xai/grok-4.5","thinking":"medium","roles":[{"name":"vision","model":"xai/grok-4.5"}]}`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	out := get()
	state, _ := out["state"].(map[string]any)
	if state == nil || state["mode"] != "lock" || state["role"] != "vision" {
		t.Fatalf("state not served: %v", out)
	}

	if err := os.WriteFile(path, []byte(`{"v":2,"mode":"lock","roles":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if out := get(); out["state"] != nil {
		t.Fatalf("future version should be null state, got %v", out)
	}

	res, err := http.Get(ts.URL + "/api/agents/nope/role-state")
	if err != nil {
		t.Fatalf("get unknown: %v", err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown agent: want 404, got %d", res.StatusCode)
	}
}
