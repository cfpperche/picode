package server

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/cfpperche/picode/internal/store"
)

func TestManagedCLIHTTPDoesNotCreateAgent(t *testing.T) {
	ts := newTestServer(t, "cat")
	proj := t.TempDir()
	res := postJSON(t, ts, "/api/workspaces", map[string]string{"name": "App", "path": proj})
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("add workspace = %d", res.StatusCode)
	}
	var wk workspaceView
	if err := json.NewDecoder(res.Body).Decode(&wk); err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if len(wk.Agents) != 0 || len(wk.ManagedCLIs) != 0 {
		t.Fatalf("empty workspace got agents=%d managed=%d", len(wk.Agents), len(wk.ManagedCLIs))
	}

	res = postJSON(t, ts, "/api/workspaces/"+wk.ID+"/principals", map[string]string{
		"cli": "claude-code", "name": "Claude",
	})
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("bind = %d", res.StatusCode)
	}
	var created principalView
	if err := json.NewDecoder(res.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if created.Kind != "terminal" || created.CLI != "claude-code" || created.ManagedCLIID == "" || created.Key == "" {
		t.Fatalf("principal = %+v", created)
	}
	if created.AgentID != "" {
		t.Fatalf("guest principal carried an agent id: %+v", created)
	}

	got := do(t, ts.Client(), mustGet(t, ts.URL+"/api/workspaces/"+wk.ID+"/principals"))
	if got.StatusCode != http.StatusOK {
		t.Fatalf("list = %d", got.StatusCode)
	}
	var list []principalView
	if err := json.NewDecoder(got.Body).Decode(&list); err != nil {
		t.Fatal(err)
	}
	got.Body.Close()
	if len(list) != 1 || list[0].Kind != "terminal" || list[0].ID != created.ID {
		t.Fatalf("list = %+v", list)
	}

	reload := do(t, ts.Client(), mustGet(t, ts.URL+"/api/workspaces"))
	var wss []workspaceView
	if err := json.NewDecoder(reload.Body).Decode(&wss); err != nil {
		t.Fatal(err)
	}
	reload.Body.Close()
	if len(wss) != 1 {
		t.Fatalf("workspaces = %d", len(wss))
	}
	if len(wss[0].Agents) != 0 {
		t.Fatalf("binding created Pi agents: %+v", wss[0].Agents)
	}
	if len(wss[0].ManagedCLIs) != 1 || wss[0].ManagedCLIs[0].CLI != "claude-code" {
		t.Fatalf("managedClis = %+v", wss[0].ManagedCLIs)
	}

	dup := postJSON(t, ts, "/api/workspaces/"+wk.ID+"/principals", map[string]string{
		"cli": "claude-code", "terminalId": created.ID,
	})
	if dup.StatusCode != http.StatusConflict {
		t.Fatalf("duplicate = %d", dup.StatusCode)
	}
	dup.Body.Close()

	bad := postJSON(t, ts, "/api/workspaces/"+wk.ID+"/principals", map[string]string{"cli": "nope"})
	if bad.StatusCode != http.StatusBadRequest {
		t.Fatalf("unknown CLI = %d", bad.StatusCode)
	}
	bad.Body.Close()

	del, _ := http.NewRequest(http.MethodDelete, ts.URL+"/api/managed-clis/"+created.ManagedCLIID, nil)
	gone := do(t, ts.Client(), del)
	if gone.StatusCode != http.StatusNoContent {
		t.Fatalf("unbind = %d", gone.StatusCode)
	}

	reload = do(t, ts.Client(), mustGet(t, ts.URL+"/api/workspaces"))
	wss = nil
	if err := json.NewDecoder(reload.Body).Decode(&wss); err != nil {
		t.Fatal(err)
	}
	reload.Body.Close()
	if len(wss[0].ManagedCLIs) != 0 {
		t.Fatalf("unbind left managedClis = %+v", wss[0].ManagedCLIs)
	}
}

func TestManagedCLIHTTPRefusesFreeWorkspace(t *testing.T) {
	ts := newTestServer(t, "cat")
	res := postJSON(t, ts, "/api/workspaces/"+store.FreeWorkspaceID+"/principals", map[string]string{
		"cli": "claude-code",
	})
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("free workspace = %d", res.StatusCode)
	}
	res.Body.Close()
}
