package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/cfpperche/picode/internal/store"
)

// Root pins the owner's checkout before either history read. It never
// supplies another repository, including when the requested owner is gone.
func TestGitHistoryRootPreconditionAcrossOwners(t *testing.T) {
	repo := gitRepo(t)
	st := testStore(t)
	wk, agent, err := storeWorkspaceWithAgent(st, "History", repo)
	if err != nil {
		t.Fatal(err)
	}
	term, err := st.CreateTerminalIn(wk.ID, "History", repo)
	if err != nil {
		t.Fatal(err)
	}
	ts := graphServer(t, st)
	head := getGraph(t, ts, "/api/agents/"+agent.ID+"/git").Head
	owners := []struct{ kind, id string }{
		{"agents", agent.ID}, {"terminals", term.ID}, {"workspaces", wk.ID},
	}
	for _, owner := range owners {
		for _, state := range []struct {
			name, root string
			missing    bool
			code       int
		}{
			{"legacy", "", false, http.StatusOK},
			{"pinned", canonDir(repo), false, http.StatusOK},
			{"moved", canonDir(t.TempDir()), false, http.StatusConflict},
			{"missing", "", true, http.StatusNotFound},
			{"missing_pinned", canonDir(repo), true, http.StatusNotFound},
		} {
			for _, endpoint := range []string{"git", "git/commit"} {
				t.Run(owner.kind+"/"+state.name+"/"+endpoint, func(t *testing.T) {
					id := owner.id
					if state.missing {
						id = "missing-owner"
					}
					query := url.Values{"hash": {head}}
					if state.root != "" {
						query.Set("root", state.root)
					}
					code, page := historyPage(t, ts, "/api/"+owner.kind+"/"+id+"/"+endpoint+"?"+query.Encode())
					if code != state.code {
						t.Fatalf("status = %d, want %d: %+v", code, state.code, page)
					}
					if code == http.StatusOK {
						field := "head"
						if endpoint == "git/commit" {
							field = "hash"
							if len(page["files"].([]any)) != 1 {
								t.Fatalf("commit lost its file diff: %+v", page)
							}
						}
						if page[field] != head {
							t.Fatalf("%s = %v, want %s", field, page[field], head)
						}
					} else if page["head"] != nil || page["hash"] != nil {
						t.Fatalf("refused owner returned history: %+v", page)
					}
				})
			}
		}
	}
}

func TestWorkspaceHistoryWithoutAgents(t *testing.T) {
	repo := gitRepo(t)
	side := filepath.Join(t.TempDir(), "side")
	gitRun(t, repo, "worktree", "add", "-b", "side", side)
	if err := os.WriteFile(filepath.Join(side, "side.txt"), []byte("side\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, side, "add", "side.txt")
	gitRun(t, side, "commit", "-m", "side change")
	st := testStore(t)
	wk, err := st.AddWorkspace("History only", repo)
	if err != nil {
		t.Fatal(err)
	}
	ts := graphServer(t, st)
	base := "/api/workspaces/" + wk.ID
	root := url.QueryEscape(canonDir(repo))
	graph := getGraph(t, ts, base+"/git?root="+root)
	if len(graph.Commits) != 2 || len(graph.Worktrees) != 2 || graph.Key == "" {
		t.Fatalf("workspace graph lost history or sibling: %+v", graph)
	}
	for _, wt := range graph.Worktrees {
		if len(wt.Agents) != 0 {
			t.Fatalf("agentless workspace gained occupants: %+v", wt)
		}
	}
	for _, branch := range []struct {
		name, subject string
	}{
		{"main", "first"}, {"side", "side change"},
	} {
		t.Run(branch.name, func(t *testing.T) {
			g := getGraph(t, ts, base+"/git?root="+root+"&limit=1&branches="+branch.name+"&remotes=0")
			if len(g.Commits) != 1 || g.Commits[0].Subject != branch.subject || !g.More {
				t.Fatalf("branch/window options were not applied: %+v", g)
			}
		})
	}
	for _, hash := range []struct {
		value string
		code  int
	}{
		{graph.Head, http.StatusOK}, {"", http.StatusBadRequest},
		{"--all", http.StatusNotFound}, {"HEAD", http.StatusNotFound},
	} {
		code, page := historyPage(t, ts, base+"/git/commit?root="+root+"&hash="+url.QueryEscape(hash.value))
		if code != hash.code {
			t.Fatalf("hash %q = %d, want %d: %+v", hash.value, code, hash.code, page)
		}
	}
	plain, err := st.AddWorkspace("Plain folder", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{plain.ID, store.FreeWorkspaceID} {
		for _, endpoint := range []string{"git", "git/commit?hash=" + graph.Head} {
			code, page := historyPage(t, ts, "/api/workspaces/"+id+"/"+endpoint)
			if code != http.StatusNotFound {
				t.Fatalf("%s/%s = %d, want 404: %+v", id, endpoint, code, page)
			}
		}
	}
}

func historyPage(t *testing.T, ts *httptest.Server, path string) (int, map[string]any) {
	t.Helper()
	res, err := ts.Client().Get(ts.URL + path)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var page map[string]any
	if err := json.NewDecoder(res.Body).Decode(&page); err != nil {
		t.Fatal(err)
	}
	return res.StatusCode, page
}
