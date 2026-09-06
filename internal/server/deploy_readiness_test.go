package server

import (
	"net/http"
	"testing"

	"github.com/cfpperche/picode/internal/tmux"
)

// The deploy guard's question: who is working right now, across every
// workspace. Probes are the Git-actions fakes; no pi or tmux is spawned.
func TestDeployReadinessListsEveryBusyOwner(t *testing.T) {
	repo := gitRepo(t)
	st := testStore(t)
	ts := graphServer(t, st)
	ws, agent, err := storeWorkspaceWithAgent(st, "App", repo)
	if err != nil {
		t.Fatal(err)
	}
	term, err := st.CreateTerminalIn(ws.ID, "QA", repo)
	if err != nil {
		t.Fatal(err)
	}
	idle, err := st.CreateTerminalIn(ws.ID, "Idle", repo)
	if err != nil {
		t.Fatal(err)
	}

	swapProbes(t, map[string]string{}, map[string]string{})
	var out deployReadiness
	if code := getJSON(t, ts, "/api/deploy/readiness", &out); code != http.StatusOK {
		t.Fatalf("readiness = %d, want 200", code)
	}
	if !out.Ready || len(out.Busy) != 0 {
		t.Fatalf("idle fleet must be ready, got %+v", out)
	}

	swapProbes(t,
		map[string]string{tmux.ShellSessionName(term.ID): "vim"},
		map[string]string{agent.ID: "mid-turn"})
	if code := getJSON(t, ts, "/api/deploy/readiness", &out); code != http.StatusOK {
		t.Fatalf("readiness = %d, want 200", code)
	}
	if out.Ready || len(out.Busy) != 2 {
		t.Fatalf("expected two busy owners, got %+v", out)
	}
	seen := map[string]string{}
	for _, b := range out.Busy {
		seen[b.Kind+":"+b.ID] = b.Why
	}
	if seen["agent:"+agent.ID] != "mid-turn" {
		t.Fatalf("agent missing or wrong reason: %v", seen)
	}
	if seen["terminal:"+term.ID] != "running vim" {
		t.Fatalf("terminal missing or wrong reason: %v", seen)
	}
	if _, listed := seen["terminal:"+idle.ID]; listed {
		t.Fatalf("idle terminal must not be listed: %v", seen)
	}
}
