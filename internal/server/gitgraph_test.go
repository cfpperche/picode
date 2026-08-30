package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/cfpperche/picode/internal/rpc"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

// The payload's reason to exist: two agents in two worktrees of one repo must
// answer with the same key and be attributed to their own checkout.
func TestGraphCollapsesWorktreesAndNamesOccupants(t *testing.T) {
	repo := gitRepo(t)
	side := filepath.Join(t.TempDir(), "side")
	gitRun(t, repo, "worktree", "add", "-b", "side", side)

	st := testStore(t)
	_, main, err := st.AddWorkspace("App", repo)
	if err != nil {
		t.Fatal(err)
	}
	sideAgent, err := st.AddAgent(main.WorkspaceID, "sider", side)
	if err != nil {
		t.Fatal(err)
	}
	ts := graphServer(t, st)

	fromMain := getGraph(t, ts, "/api/agents/"+main.ID+"/git")
	fromSide := getGraph(t, ts, "/api/agents/"+sideAgent.ID+"/git")

	if fromMain.Key == "" || fromMain.Key != fromSide.Key {
		t.Fatalf("worktrees must share the key: %q vs %q", fromMain.Key, fromSide.Key)
	}
	if len(fromMain.Worktrees) != 2 {
		t.Fatalf("expected 2 worktrees, got %+v", fromMain.Worktrees)
	}
	for _, wt := range fromMain.Worktrees {
		want := main.ID
		if wt.Branch == "side" {
			want = sideAgent.ID
		}
		if len(wt.Agents) != 1 || wt.Agents[0].ID != want {
			t.Fatalf("worktree %s (%s) occupants = %+v, want %s",
				wt.Branch, wt.Path, wt.Agents, want)
		}
	}
}

// A cwd that is not a repository has no graph, and that is a 404, not an empty
// one — ADR-0022 hangs the entry point off exactly this answer.
func TestGraphOfANonRepoIsNotFound(t *testing.T) {
	st := testStore(t)
	_, agent, err := st.AddWorkspace("Scratch", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ts := graphServer(t, st)
	res, err := ts.Client().Get(ts.URL + "/api/agents/" + agent.ID + "/git")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", res.StatusCode)
	}
}

// An unrelated repository nested inside a checkout must not borrow its head.
func TestNestedRepoIsNotAnOccupant(t *testing.T) {
	repo := gitRepo(t)
	inner := filepath.Join(repo, "vendor", "other")
	if err := os.MkdirAll(inner, 0o755); err != nil {
		t.Fatal(err)
	}
	gitRun(t, inner, "init", "-b", "main")

	st := testStore(t)
	_, outer, err := st.AddWorkspace("App", repo)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.AddAgent(outer.WorkspaceID, "vendored", inner); err != nil {
		t.Fatal(err)
	}
	ts := graphServer(t, st)

	g := getGraph(t, ts, "/api/agents/"+outer.ID+"/git")
	if len(g.Worktrees) != 1 {
		t.Fatalf("expected one worktree, got %+v", g.Worktrees)
	}
	if len(g.Worktrees[0].Agents) != 1 || g.Worktrees[0].Agents[0].ID != outer.ID {
		t.Fatalf("only the outer agent belongs here: %+v", g.Worktrees[0].Agents)
	}
}

func TestDeepestRootMatchesComponents(t *testing.T) {
	roots := []string{"/a/b", "/a/b/.worktrees/x", "/a/bc"}
	for _, c := range []struct {
		cwd   string
		root  string
		exact bool
	}{
		{"/a/b", "/a/b", true},
		{"/a/bc", "/a/bc", true},
		{"/a/b/.worktrees/x", "/a/b/.worktrees/x", true},
		{"/a/b/.worktrees/x/deep", "/a/b/.worktrees/x", false},
		{"/a/b/src", "/a/b", false},
		{"/a/bcd", "", false},
		{"/elsewhere", "", false},
	} {
		root, exact := deepestRoot(roots, c.cwd)
		if root != c.root || exact != c.exact {
			t.Fatalf("deepestRoot(%q) = %q,%v want %q,%v", c.cwd, root, exact, c.root, c.exact)
		}
	}
}

func TestGraphLimitClamps(t *testing.T) {
	for _, c := range []struct {
		query string
		want  int
	}{
		{"", defaultGraphLimit},
		{"?limit=", defaultGraphLimit},
		{"?limit=abc", defaultGraphLimit},
		{"?limit=0", defaultGraphLimit},
		{"?limit=-5", defaultGraphLimit},
		{"?limit=10", 10},
		{"?limit=999999", maxGraphLimit},
	} {
		r := httptest.NewRequest(http.MethodGet, "/api/agents/x/git"+c.query, nil)
		if got := graphLimit(r); got != c.want {
			t.Fatalf("graphLimit(%q) = %d, want %d", c.query, got, c.want)
		}
	}
}

func getGraph(t *testing.T, ts *httptest.Server, path string) graphView {
	t.Helper()
	res, err := ts.Client().Get(ts.URL + path)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET %s = %d", path, res.StatusCode)
	}
	var g graphView
	if err := json.NewDecoder(res.Body).Decode(&g); err != nil {
		t.Fatal(err)
	}
	return g
}

func graphServer(t *testing.T, st *store.Store) *httptest.Server {
	t.Helper()
	ts := httptest.NewServer(New("127.0.0.1:0", Deps{
		Store: st, Tmux: tmux.New(), Runtime: rpc.NewRuntime("cat", st, nil), AgentCmd: "cat",
	}).Handler)
	t.Cleanup(ts.Close)
	return ts
}

func testStore(t *testing.T) *store.Store {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	st, err := store.Open(filepath.Join(t.TempDir(), "picode.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

func gitRepo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git missing")
	}
	dir := t.TempDir()
	gitRun(t, dir, "init", "-b", "main")
	gitRun(t, dir, "config", "user.email", "t@t")
	gitRun(t, dir, "config", "user.name", "t")
	if err := os.WriteFile(filepath.Join(dir, "a"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, dir, "add", ".")
	gitRun(t, dir, "commit", "-m", "first")
	return dir
}

func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}
