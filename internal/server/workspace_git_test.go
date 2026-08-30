package server

import (
	"encoding/json"
	"net/http"
	"os/exec"
	"testing"
)

func gitInit(t *testing.T, dir string) {
	t.Helper()
	for _, args := range [][]string{
		{"init", "-q", "-b", "trunk"},
		{"-c", "user.email=t@t", "-c", "user.name=t", "commit", "-q", "--allow-empty", "-m", "seed"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
}

// Every workspace agent carries git facts for its EFFECTIVE directory. An
// agent with a workPath may live in a different repository than its
// workspace; before this, agent views shipped no git at all and the sidebar
// fell back to the workspace's — the wrong repo and branch for exactly the
// agents that need the line most.
func TestWorkspaceAgentsCarryTheirOwnGit(t *testing.T) {
	ts, _, _ := cleanupServer(t)
	wsDir := t.TempDir()
	gitInit(t, wsDir)
	otherRepo := t.TempDir()
	gitInit(t, otherRepo)
	cmd := exec.Command("git", "checkout", "-q", "-b", "feature-x")
	cmd.Dir = otherRepo
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("branch: %v: %s", err, out)
	}

	res := postJSON(t, ts, "/api/workspaces", map[string]string{"name": "App", "path": wsDir})
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("workspace = %d", res.StatusCode)
	}
	var wk struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(res.Body).Decode(&wk)
	res.Body.Close()

	// A second agent whose workPath is the other repository — the case the
	// per-agent git exists for.
	add := postJSON(t, ts, "/api/workspaces/"+wk.ID+"/agents", map[string]string{"name": "roamer", "workPath": otherRepo})
	if add.StatusCode != http.StatusCreated {
		t.Fatalf("add agent = %d", add.StatusCode)
	}
	add.Body.Close()

	list, err := ts.Client().Get(ts.URL + "/api/workspaces")
	if err != nil {
		t.Fatal(err)
	}
	defer list.Body.Close()
	var out []struct {
		Git    *struct{ Branch string } `json:"git"`
		Agents []struct {
			ID  string                   `json:"id"`
			Git *struct{ Branch string } `json:"git"`
		} `json:"agents"`
	}
	_ = json.NewDecoder(list.Body).Decode(&out)
	if len(out) != 1 || len(out[0].Agents) == 0 {
		t.Fatalf("unexpected shape: %+v", out)
	}
	ws := out[0]
	if ws.Git == nil || ws.Git.Branch != "trunk" {
		t.Fatalf("workspace git = %+v, want branch trunk", ws.Git)
	}
	var sawOwn, sawInherited bool
	for _, ag := range ws.Agents {
		if ag.Git == nil {
			t.Fatal("agent view carries no git — the sidebar can only fall back to the workspace's")
		}
		switch ag.Git.Branch {
		case "feature-x":
			sawOwn = true // the workPath agent speaks for ITS repo
		case "trunk":
			sawInherited = true // the default agent sits on the workspace path
		}
	}
	if !sawOwn || !sawInherited {
		t.Fatalf("agents = %+v; want one on feature-x (own workPath repo) and one on trunk", ws.Agents)
	}
}
