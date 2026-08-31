package server

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/cfpperche/picode/internal/store"
)

// addWorkspaceWithAgent posts a workspace and a first agent — the
// pre-ADR-0027 shape most tests want. Workspaces start empty now, so the
// agent is an explicit second POST.
func addWorkspaceWithAgent(t *testing.T, ts *httptest.Server, name, dir string) workspaceView {
	t.Helper()
	res := postJSON(t, ts, "/api/workspaces", map[string]string{"name": name, "path": dir})
	if res.StatusCode != 201 {
		t.Fatalf("add workspace = %d", res.StatusCode)
	}
	var wk workspaceView
	if err := json.NewDecoder(res.Body).Decode(&wk); err != nil {
		t.Fatalf("decode workspace: %v", err)
	}
	res.Body.Close()
	if len(wk.Agents) > 0 {
		return wk // idempotent re-add: the agent is already there
	}
	ares := postJSON(t, ts, "/api/workspaces/"+wk.ID+"/agents", map[string]string{"name": "default"})
	if ares.StatusCode != 201 {
		t.Fatalf("add agent = %d", ares.StatusCode)
	}
	var av agentView
	if err := json.NewDecoder(ares.Body).Decode(&av); err != nil {
		t.Fatalf("decode agent: %v", err)
	}
	ares.Body.Close()
	wk.Agent = &av
	wk.Agents = []agentView{av}
	return wk
}

// storeWorkspaceWithAgent is the store-level twin, keeping the old
// AddWorkspace(name, path) (Workspace, Agent, error) shape for tests.
func storeWorkspaceWithAgent(s *store.Store, name, path string) (store.Workspace, store.Agent, error) {
	w, err := s.AddWorkspace(name, path)
	if err != nil {
		return store.Workspace{}, store.Agent{}, err
	}
	a, err := s.AddAgent(w.ID, "default", "")
	return w, a, err
}
