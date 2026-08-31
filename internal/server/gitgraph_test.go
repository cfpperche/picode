package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/gitgraph"
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
	_, main, err := storeWorkspaceWithAgent(st, "App", repo)
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
	_, agent, err := storeWorkspaceWithAgent(st, "Scratch", t.TempDir())
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
	_, outer, err := storeWorkspaceWithAgent(st, "App", repo)
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

func TestCommitRouteReadsThroughTheOwner(t *testing.T) {
	repo := gitRepo(t)
	st := testStore(t)
	_, agent, err := storeWorkspaceWithAgent(st, "App", repo)
	if err != nil {
		t.Fatal(err)
	}
	ts := graphServer(t, st)

	g := getGraph(t, ts, "/api/agents/"+agent.ID+"/git")
	if len(g.Commits) == 0 {
		t.Fatal("no commits to ask about")
	}
	hash := g.Commits[0].Hash

	res, err := ts.Client().Get(ts.URL + "/api/agents/" + agent.ID + "/git/commit?hash=" + hash)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", res.StatusCode)
	}
	var d struct {
		Hash    string `json:"hash"`
		Subject string `json:"subject"`
		Files   []struct {
			Path  string `json:"path"`
			Patch string `json:"patch"`
		} `json:"files"`
	}
	if err := json.NewDecoder(res.Body).Decode(&d); err != nil {
		t.Fatal(err)
	}
	if d.Hash != hash || d.Subject != "first" {
		t.Fatalf("wrong commit: %+v", d)
	}
	if len(d.Files) != 1 || d.Files[0].Path != "a" || !strings.Contains(d.Files[0].Patch, "+a") {
		t.Fatalf("files = %+v", d.Files)
	}
}

// The route reads through the owner's cwd, so a real commit that lives in a
// different repository is simply not there. That is the confinement working,
// not an accident of naming.
func TestCommitFromAnotherRepoIsNotFound(t *testing.T) {
	mine := gitRepo(t)
	theirs := gitRepo(t)
	// gitRepo builds identical repositories, and git is content-addressed: same
	// tree, same message, same second gives the *same* hash. Make theirs differ
	// or this test proves nothing.
	if err := os.WriteFile(filepath.Join(theirs, "only-theirs"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, theirs, "add", ".")
	gitRun(t, theirs, "commit", "-m", "theirs alone")
	theirHash := strings.TrimSpace(gitOut(t, theirs, "rev-parse", "HEAD"))
	mineHash := strings.TrimSpace(gitOut(t, mine, "rev-parse", "HEAD"))
	if theirHash == mineHash {
		t.Fatal("the two repositories must not share a commit for this test to mean anything")
	}

	st := testStore(t)
	_, agent, err := storeWorkspaceWithAgent(st, "App", mine)
	if err != nil {
		t.Fatal(err)
	}
	ts := graphServer(t, st)

	for _, q := range []string{
		"?hash=" + theirHash,
		"?hash=HEAD",
		"?hash=--help",
		"",
	} {
		res, err := ts.Client().Get(ts.URL + "/api/agents/" + agent.ID + "/git/commit" + q)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode == http.StatusOK {
			t.Fatalf("query %q must not succeed, got 200", q)
		}
	}
}

func gitOut(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
	return string(out)
}

// The endpoint that adds an agent to a workspace used to hardcode an empty
// work path, so the case ADR-0022 exists for — two agents in sibling worktrees
// of one repo — could only be built out of free agents. This is that case,
// through the API a workspace actually uses.
func TestWorkspaceAgentCanLiveInASiblingWorktree(t *testing.T) {
	repo := gitRepo(t)
	side := filepath.Join(t.TempDir(), "side")
	gitRun(t, repo, "worktree", "add", "-b", "side", side)

	st := testStore(t)
	_, main, err := storeWorkspaceWithAgent(st, "App", repo)
	if err != nil {
		t.Fatal(err)
	}
	ts := graphServer(t, st)

	sideID := addWorkspaceAgent(t, ts, main.WorkspaceID, map[string]any{
		"name": "sider", "workPath": side,
	})

	g := getGraph(t, ts, "/api/agents/"+main.ID+"/git")
	if len(g.Worktrees) != 2 {
		t.Fatalf("expected 2 worktrees, got %+v", g.Worktrees)
	}
	for _, wt := range g.Worktrees {
		want := main.ID
		if wt.Branch == "side" {
			want = sideID
		}
		if len(wt.Agents) != 1 || wt.Agents[0].ID != want {
			t.Fatalf("worktree %s occupants = %+v, want %s", wt.Branch, wt.Agents, want)
		}
	}
}

func TestWorkspaceAgentWorkPathRules(t *testing.T) {
	repo := gitRepo(t)
	st := testStore(t)
	_, first, err := storeWorkspaceWithAgent(st, "App", repo)
	if err != nil {
		t.Fatal(err)
	}
	ts := graphServer(t, st)

	// Absent: unchanged behaviour — the agent lives on the workspace folder.
	plainID := addWorkspaceAgent(t, ts, first.WorkspaceID, map[string]any{"name": "plain"})
	plain, err := st.GetAgent(plainID)
	if err != nil {
		t.Fatal(err)
	}
	if plain.WorkPath != nil && *plain.WorkPath != "" {
		t.Fatalf("no workPath sent, agent must stay on the workspace: %v", *plain.WorkPath)
	}

	// Blank strings are the same as absent, not a request for a work dir.
	blankID := addWorkspaceAgent(t, ts, first.WorkspaceID, map[string]any{"name": "blank", "workPath": "   "})
	blank, err := st.GetAgent(blankID)
	if err != nil {
		t.Fatal(err)
	}
	if blank.WorkPath != nil && *blank.WorkPath != "" {
		t.Fatalf("blank workPath must not create a work dir: %v", *blank.WorkPath)
	}

	// A path that exists but is not a directory is refused, not swallowed.
	file := filepath.Join(t.TempDir(), "a-file")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	res := postWorkspaceAgent(t, ts, first.WorkspaceID, map[string]any{"name": "bad", "workPath": file})
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", res.StatusCode)
	}
}

func postWorkspaceAgent(t *testing.T, ts *httptest.Server, wsID string, body map[string]any) *http.Response {
	t.Helper()
	raw, _ := json.Marshal(body)
	res, err := ts.Client().Post(ts.URL+"/api/workspaces/"+wsID+"/agents", "application/json", bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func addWorkspaceAgent(t *testing.T, ts *httptest.Server, wsID string, body map[string]any) string {
	t.Helper()
	res := postWorkspaceAgent(t, ts, wsID, body)
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create agent = %d", res.StatusCode)
	}
	var out struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out.ID == "" {
		t.Fatal("no agent id in response")
	}
	return out.ID
}

// Resolving a directory to its repository spawns git (~23ms measured), and
// agents commonly share a directory. The memo is only worth having if it
// actually collapses those calls — so count them.
func TestOccupantScanAsksGitOncePerDirectory(t *testing.T) {
	repo := gitRepo(t)
	deep := filepath.Join(repo, "deep")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}

	st := testStore(t)
	ws, root, err := storeWorkspaceWithAgent(st, "App", repo)
	if err != nil {
		t.Fatal(err)
	}
	_ = root
	// Five agents sharing one subdirectory, and three each in their own.
	for i := 0; i < 5; i++ {
		if _, err := st.AddAgent(ws.ID, fmt.Sprintf("same%d", i), deep); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 3; i++ {
		dir := filepath.Join(repo, "own", fmt.Sprintf("d%d", i))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if _, err := st.AddAgent(ws.ID, fmt.Sprintf("own%d", i), dir); err != nil {
			t.Fatal(err)
		}
	}

	g := gitgraph.Load(repo, 20)
	if g == nil {
		t.Fatal("no graph")
	}

	prev := gitKeyOf
	t.Cleanup(func() { gitKeyOf = prev })
	var asked []string
	gitKeyOf = func(dir string) string {
		asked = append(asked, dir)
		return prev(dir)
	}

	deps := Deps{Store: st}
	occ := deps.occupantsByWorktree(g)

	// Four distinct directories below the root: the shared one asked once, the
	// three private ones once each. Without the memo the shared one alone
	// would account for five.
	if len(asked) != 4 {
		t.Fatalf("git asked %d times for %d agents: %v", len(asked), 8, asked)
	}
	seen := map[string]int{}
	for _, d := range asked {
		seen[d]++
	}
	for d, n := range seen {
		if n != 1 {
			t.Fatalf("%s asked %d times", d, n)
		}
	}
	// And the agents still land where they belong.
	total := 0
	for _, list := range occ {
		total += len(list)
	}
	if total != 9 { // 8 added plus the workspace's own first agent
		t.Fatalf("occupants = %d, want 9: %+v", total, occ)
	}
}

// Agents at a worktree root are resolved by their path alone; git is not
// asked at all. That is the shape the product produces, and it must stay free.
func TestOccupantScanAsksNothingForAgentsAtTheRoot(t *testing.T) {
	repo := gitRepo(t)
	st := testStore(t)
	ws, _, err := storeWorkspaceWithAgent(st, "App", repo)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 20; i++ {
		if _, err := st.AddAgent(ws.ID, fmt.Sprintf("a%d", i), repo); err != nil {
			t.Fatal(err)
		}
	}
	g := gitgraph.Load(repo, 20)

	prev := gitKeyOf
	t.Cleanup(func() { gitKeyOf = prev })
	calls := 0
	gitKeyOf = func(dir string) string { calls++; return prev(dir) }

	deps := Deps{Store: st}
	deps.occupantsByWorktree(g)
	if calls != 0 {
		t.Fatalf("git asked %d times for agents that need no asking", calls)
	}
}
