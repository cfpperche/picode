package server

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/cfpperche/picode/internal/store"
)

func TestAddWorkspaceAgentCLICreatesCLIAgent(t *testing.T) {
	ts := newTestServer(t, "cat")
	proj := t.TempDir()
	res := postJSON(t, ts, "/api/workspaces", map[string]string{"name": "App", "path": proj})
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("workspace = %d", res.StatusCode)
	}
	var wk workspaceView
	if err := json.NewDecoder(res.Body).Decode(&wk); err != nil {
		t.Fatal(err)
	}
	res.Body.Close()

	res = postJSON(t, ts, "/api/workspaces/"+wk.ID+"/agents", map[string]string{
		"cli": "claude-code", "name": "Claude",
	})
	if res.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(res.Body)
		res.Body.Close()
		t.Fatalf("add CLI agent = %d %s", res.StatusCode, b)
	}
	var ag agentView
	if err := json.NewDecoder(res.Body).Decode(&ag); err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if ag.CLI != "claude-code" || ag.IsPi() || ag.TerminalID == nil || *ag.TerminalID == "" {
		t.Fatalf("CLI agent = %+v", ag)
	}
	if ag.Mode != string(modeStopped) {
		t.Fatalf("mode=%s", ag.Mode)
	}

	list := do(t, ts.Client(), mustGet(t, ts.URL+"/api/workspaces"))
	var wss []workspaceView
	if err := json.NewDecoder(list.Body).Decode(&wss); err != nil {
		t.Fatal(err)
	}
	list.Body.Close()
	if len(wss) != 1 || len(wss[0].Agents) != 1 || wss[0].Agents[0].CLI != "claude-code" {
		t.Fatalf("workspaces = %+v", wss)
	}
	start := postJSON(t, ts, "/api/agents/"+ag.ID+"/managed/start", map[string]string{})
	if start.StatusCode != http.StatusBadRequest {
		t.Fatalf("managed start CLI agent = %d", start.StatusCode)
	}
	start.Body.Close()

	bad := postJSON(t, ts, "/api/workspaces/"+wk.ID+"/agents", map[string]string{"cli": "not-a-cli"})
	if bad.StatusCode != http.StatusBadRequest {
		t.Fatalf("unknown cli = %d", bad.StatusCode)
	}
	bad.Body.Close()
}

// Free agents follow the same decision table as workspace agents
// (ADR-0179 extends ADR-0160): a CLI row with a free launch terminal on its
// folder, Pi by default, unknown CLIs refused, managed start Pi-only.
func TestAddFreeAgentCLICreatesCLIAgent(t *testing.T) {
	ts := newTestServer(t, "cat")
	dir := t.TempDir()
	res := postJSON(t, ts, "/api/agents", map[string]string{"cli": "claude-code", "name": "Claude", "path": dir})
	if res.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(res.Body)
		res.Body.Close()
		t.Fatalf("add free CLI agent = %d %s", res.StatusCode, b)
	}
	var ag agentView
	if err := json.NewDecoder(res.Body).Decode(&ag); err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if ag.CLI != "claude-code" || ag.IsPi() || ag.WorkspaceID != store.FreeWorkspaceID || ag.TerminalID == nil || *ag.TerminalID == "" {
		t.Fatalf("free CLI agent = %+v", ag)
	}
	var list struct {
		Terminals []struct {
			ID          string `json:"id"`
			WorkspaceID string `json:"workspaceId"`
			Cwd         string `json:"cwd"`
		} `json:"terminals"`
	}
	tl := do(t, ts.Client(), mustGet(t, ts.URL+"/api/terminals"))
	if err := json.NewDecoder(tl.Body).Decode(&list); err != nil {
		t.Fatal(err)
	}
	tl.Body.Close()
	found := false
	for _, tm := range list.Terminals {
		if tm.ID != *ag.TerminalID {
			continue
		}
		found = true
		if tm.WorkspaceID != store.FreeWorkspaceID || tm.Cwd != dir {
			t.Fatalf("launch terminal = %+v (want free, cwd %s)", tm, dir)
		}
	}
	if !found {
		t.Fatalf("launch terminal %s not listed", *ag.TerminalID)
	}
	start := postJSON(t, ts, "/api/agents/"+ag.ID+"/managed/start", map[string]string{})
	if start.StatusCode != http.StatusBadRequest {
		t.Fatalf("managed start free CLI agent = %d", start.StatusCode)
	}
	start.Body.Close()

	pi := decodeAgent(t, postJSON(t, ts, "/api/agents", map[string]string{"name": "Atlas", "provider": "anthropic", "model": "m", "thinking": "low"}))
	if !pi.IsPi() || pi.TerminalID != nil || pi.Provider == nil || *pi.Provider != "anthropic" {
		t.Fatalf("default free agent = %+v", pi)
	}
	bad := postJSON(t, ts, "/api/agents", map[string]string{"cli": "not-a-cli", "name": "x"})
	if bad.StatusCode != http.StatusBadRequest {
		t.Fatalf("unknown cli = %d", bad.StatusCode)
	}
	bad.Body.Close()
}

func TestAddWorkspaceAgentDefaultIsPi(t *testing.T) {
	ts := newTestServer(t, "cat")
	wk := addWorkspaceWithAgent(t, ts, "App", t.TempDir())
	if wk.Agents[0].CLI != store.CLIPi || !wk.Agents[0].IsPi() {
		t.Fatalf("default agent cli=%q", wk.Agents[0].CLI)
	}
	if wk.Agents[0].TerminalID != nil {
		t.Fatalf("pi agent has terminal %v", wk.Agents[0].TerminalID)
	}
}
