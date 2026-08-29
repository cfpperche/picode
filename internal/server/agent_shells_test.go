package server

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/cfpperche/picode/internal/tmux"
)

func TestShellsNeedAgent(t *testing.T) {
	ts, _, _ := cleanupServer(t)
	got := do(t, ts.Client(), mustGet(t, ts.URL+"/api/agents/nope/shells"))
	if got.StatusCode != http.StatusNotFound {
		t.Fatalf("GET missing = %d", got.StatusCode)
	}
	res := postJSON(t, ts, "/api/agents/nope/shells", map[string]any{})
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("POST missing = %d", res.StatusCode)
	}
}

func TestShellsCreateListKill(t *testing.T) {
	if !tmux.New().Available() {
		t.Skip("tmux not installed — integration test skipped (accepted, see docs/handoff.md)")
	}
	ts, _, _ := cleanupServer(t)
	proj := t.TempDir()
	res := postJSON(t, ts, "/api/workspaces", map[string]string{"name": "App", "path": proj})
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("add = %d", res.StatusCode)
	}
	var wk workspaceView
	if err := json.NewDecoder(res.Body).Decode(&wk); err != nil {
		t.Fatal(err)
	}
	id := wk.Agent.ID
	name := tmux.ShellSessionName(id)
	t.Cleanup(func() { _ = tmux.New().KillSession(context.Background(), name) })

	created := postJSON(t, ts, "/api/agents/"+id+"/shells", map[string]any{})
	if created.StatusCode != http.StatusCreated {
		t.Fatalf("create = %d", created.StatusCode)
	}
	var page map[string]any
	if err := json.NewDecoder(created.Body).Decode(&page); err != nil {
		t.Fatal(err)
	}
	if page["session"] != name {
		t.Fatalf("session=%v want %s", page["session"], name)
	}

	again := postJSON(t, ts, "/api/agents/"+id+"/shells", map[string]any{})
	if again.StatusCode != http.StatusOK {
		t.Fatalf("idempotent = %d", again.StatusCode)
	}

	listed := do(t, ts.Client(), mustGet(t, ts.URL+"/api/agents/"+id+"/shells"))
	if listed.StatusCode != http.StatusOK {
		t.Fatalf("list = %d", listed.StatusCode)
	}
	var bag struct {
		Shells []struct {
			Session string `json:"session"`
		} `json:"shells"`
	}
	if err := json.NewDecoder(listed.Body).Decode(&bag); err != nil {
		t.Fatal(err)
	}
	if len(bag.Shells) != 1 || bag.Shells[0].Session != name {
		t.Fatalf("list=%+v", bag.Shells)
	}

	req, _ := http.NewRequest(http.MethodDelete, ts.URL+"/api/agents/"+id+"/shells", nil)
	killed := do(t, ts.Client(), req)
	if killed.StatusCode != http.StatusNoContent {
		t.Fatalf("kill = %d", killed.StatusCode)
	}
	listed = do(t, ts.Client(), mustGet(t, ts.URL+"/api/agents/"+id+"/shells"))
	_ = json.NewDecoder(listed.Body).Decode(&bag)
	if len(bag.Shells) != 0 {
		t.Fatalf("after kill list=%+v", bag.Shells)
	}
}
