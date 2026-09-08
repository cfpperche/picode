package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/store"
)

var _ = store.Terminal{}

func postCompose(t *testing.T, ts *httptest.Server, path string, body map[string]any) (int, composeView, map[string]any) {
	t.Helper()
	raw, _ := json.Marshal(body)
	res, err := http.Post(ts.URL+path, "application/json", bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	payload, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	var view composeView
	var errBody map[string]any
	_ = json.Unmarshal(payload, &view)
	_ = json.Unmarshal(payload, &errBody)
	return res.StatusCode, view, errBody
}

// composeAgent is the whole fixture these tests need: one workspace agent in
// a repository with a remote and a second branch.
func composeAgent(t *testing.T) (*httptest.Server, store.Agent, string) {
	t.Helper()
	repo := gitRepo(t)
	gitRun(t, repo, "remote", "add", "origin", "https://example.invalid/r.git")
	gitRun(t, repo, "branch", "feat/x")
	st := testStore(t)
	_, agent, err := storeWorkspaceWithAgent(st, "App", repo)
	if err != nil {
		t.Fatal(err)
	}
	return graphServer(t, st), agent, repo
}

func TestComposeReturnsTheExactCommandAndItsTier(t *testing.T) {
	ts, agent, _ := composeAgent(t)
	base := "/api/agents/" + agent.ID + "/git/compose"
	cases := []struct {
		action string
		body   map[string]any
		want   string
		tier   string
	}{
		{"fetch", map[string]any{}, "git fetch --prune", "A"},
		{"merge", map[string]any{"target": "feat/x"}, "git merge --no-ff feat/x", "B"},
		{"create-branch", map[string]any{"target": "feat/x", "name": "feat/y"}, "git switch -c feat/y feat/x", "A"},
		{"commit", map[string]any{"message": "it's fine"}, `git add -A && git commit -m 'it'\''s fine'`, "B"},
		{"reset-hard", map[string]any{"target": "feat/x"}, "git reset --hard feat/x", "C"},
	}
	for _, c := range cases {
		body := map[string]any{"action": c.action}
		for k, v := range c.body {
			body[k] = v
		}
		status, view, _ := postCompose(t, ts, base, body)
		if status != http.StatusOK {
			t.Errorf("%s: status = %d", c.action, status)
			continue
		}
		if view.Command != c.want {
			t.Errorf("%s command = %q; want %q", c.action, view.Command, c.want)
		}
		if view.Tier != c.tier {
			t.Errorf("%s tier = %q; want %q", c.action, view.Tier, c.tier)
		}
		if view.Verb == "" || view.Prompt == "" {
			t.Errorf("%s: verb %q / prompt %q", c.action, view.Verb, view.Prompt)
		}
		if strings.Contains(view.Prompt, "git ") {
			t.Errorf("%s prompt reads like a command: %q", c.action, view.Prompt)
		}
	}
}

// The branch a push publishes is read from the checkout, never taken from the
// caller: a browser that names another branch cannot make PiCode compose a
// push for it.
func TestComposeReadsTheBranchFromTheCheckoutNotTheCaller(t *testing.T) {
	ts, agent, repo := composeAgent(t)
	gitRun(t, repo, "checkout", "-q", "feat/x")
	status, view, _ := postCompose(t, ts, "/api/agents/"+agent.ID+"/git/compose",
		map[string]any{"action": "push", "target": "main", "name": "main"})
	if status != http.StatusOK {
		t.Fatalf("status = %d", status)
	}
	if view.Branch != "feat/x" {
		t.Errorf("branch = %q; want the checked-out feat/x", view.Branch)
	}
	if view.Command != "git push -u origin feat/x" {
		t.Errorf("command = %q; want the checked-out branch published", view.Command)
	}
}

func TestComposeRefusesABadTargetAndAnUnknownAction(t *testing.T) {
	ts, agent, _ := composeAgent(t)
	base := "/api/agents/" + agent.ID + "/git/compose"
	status, _, body := postCompose(t, ts, base, map[string]any{"action": "merge", "target": "--force"})
	if status != http.StatusBadRequest {
		t.Errorf("bad target: status = %d; want 400", status)
	}
	if body["error"] == nil {
		t.Error("a refusal must name what is wrong")
	}
	if status, _, _ := postCompose(t, ts, base, map[string]any{"action": "rm-rf"}); status != http.StatusNotFound {
		t.Errorf("unknown action: status = %d; want 404", status)
	}
}

// The root precondition of ADR-0074: a command composed for one folder is
// never handed to another.
func TestComposeRefusesAStaleRoot(t *testing.T) {
	ts, agent, _ := composeAgent(t)
	status, _, body := postCompose(t, ts, "/api/agents/"+agent.ID+"/git/compose",
		map[string]any{"action": "fetch", "root": "/somewhere/else"})
	if status != http.StatusConflict {
		t.Fatalf("status = %d; want 409", status)
	}
	if body["reason"] != "moved" {
		t.Errorf("reason = %v; want moved", body["reason"])
	}
}

func TestComposeOnAPlainFolderIs404(t *testing.T) {
	st := testStore(t)
	plain := t.TempDir()
	_, agent, err := storeWorkspaceWithAgent(st, "Plain", plain)
	if err != nil {
		t.Fatal(err)
	}
	ts := graphServer(t, st)
	if status, _, _ := postCompose(t, ts, "/api/agents/"+agent.ID+"/git/compose", map[string]any{"action": "fetch"}); status != http.StatusNotFound {
		t.Errorf("status = %d; want 404 on a folder that is not a repository", status)
	}
}

// A repository whose only remote is not called origin still composes for it.
func TestComposePicksTheOnlyRemoteWhenItIsNotOrigin(t *testing.T) {
	repo := gitRepo(t)
	gitRun(t, repo, "remote", "add", "upstream", "https://example.invalid/u.git")
	st := testStore(t)
	_, agent, err := storeWorkspaceWithAgent(st, "App", repo)
	if err != nil {
		t.Fatal(err)
	}
	ts := graphServer(t, st)
	_, view, _ := postCompose(t, ts, "/api/agents/"+agent.ID+"/git/compose", map[string]any{"action": "push"})
	if !strings.Contains(view.Command, "upstream") {
		t.Errorf("command = %q; want the repository's only remote", view.Command)
	}
}

// Every owner kind composes, like every other git read (ADR-0027).
func TestComposeThroughEveryOwnerKind(t *testing.T) {
	repo := gitRepo(t)
	st := testStore(t)
	ws, agent, err := storeWorkspaceWithAgent(st, "App", repo)
	if err != nil {
		t.Fatal(err)
	}
	term, err := st.CreateTerminalIn(ws.ID, "sh", repo)
	if err != nil {
		t.Fatal(err)
	}
	ts := graphServer(t, st)
	for _, path := range []string{
		"/api/agents/" + agent.ID + "/git/compose",
		"/api/terminals/" + term.ID + "/git/compose",
		"/api/workspaces/" + ws.ID + "/git/compose",
	} {
		status, view, _ := postCompose(t, ts, path, map[string]any{"action": "fetch"})
		if status != http.StatusOK || view.Command != "git fetch --prune" {
			t.Errorf("%s: status = %d, command = %q", path, status, view.Command)
		}
	}
}

// Composing is not delivering: it must leave the working tree exactly as it
// found it, whatever the action.
func TestComposeNeverTouchesTheRepository(t *testing.T) {
	ts, agent, repo := composeAgent(t)
	before := gitOut(t, repo, "log", "--all", "--format=%H")
	branches := gitOut(t, repo, "for-each-ref", "--format=%(refname)")
	for _, action := range []string{"reset-hard", "delete-branch-force", "clean", "discard", "push-force", "create-branch"} {
		postCompose(t, ts, "/api/agents/"+agent.ID+"/git/compose",
			map[string]any{"action": action, "target": "feat/x", "name": "tmp"})
	}
	if got := gitOut(t, repo, "log", "--all", "--format=%H"); got != before {
		t.Error("composing changed the history")
	}
	if got := gitOut(t, repo, "for-each-ref", "--format=%(refname)"); got != branches {
		t.Errorf("composing changed the refs:\n%s\nwas\n%s", got, branches)
	}
	if status := gitOut(t, repo, "status", "--porcelain"); strings.TrimSpace(status) != "" {
		t.Errorf("composing dirtied the working tree: %q", status)
	}
}

// The catalog the browser gates on must be reachable and complete.
func TestGitActionCatalogIsServed(t *testing.T) {
	ts := graphServer(t, testStore(t))
	res, err := http.Get(ts.URL + "/api/git/actions")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", res.StatusCode)
	}
	var page struct {
		Actions []struct {
			ID    string   `json:"id"`
			Tier  string   `json:"tier"`
			Needs []string `json:"needs"`
		} `json:"actions"`
	}
	if err := json.NewDecoder(res.Body).Decode(&page); err != nil {
		t.Fatal(err)
	}
	if len(page.Actions) < 30 {
		t.Fatalf("catalog has %d actions; the composer knows far more", len(page.Actions))
	}
	byID := map[string]string{}
	for _, a := range page.Actions {
		if a.Needs == nil {
			t.Errorf("%s: needs encoded as null", a.ID)
		}
		byID[a.ID] = a.Tier
	}
	for action, tier := range map[string]string{
		"fetch": "A", "merge": "B", "reset-hard": "C", "push": "C", "delete-branch": "B",
	} {
		if byID[action] != tier {
			t.Errorf("%s tier = %q; want %q", action, byID[action], tier)
		}
	}
}

// The token endpoint is how a delivered action reports back (ADR-0096), so
// every owner kind that can open a graph has to answer it — the workspace
// variant was missing from ADR-0038.
func TestGitHeadThroughEveryOwnerKind(t *testing.T) {
	repo := gitRepo(t)
	st := testStore(t)
	ws, agent, err := storeWorkspaceWithAgent(st, "App", repo)
	if err != nil {
		t.Fatal(err)
	}
	term, err := st.CreateTerminalIn(ws.ID, "sh", repo)
	if err != nil {
		t.Fatal(err)
	}
	ts := graphServer(t, st)
	tokens := map[string]string{}
	for _, path := range []string{
		"/api/agents/" + agent.ID + "/git/head",
		"/api/terminals/" + term.ID + "/git/head",
		"/api/workspaces/" + ws.ID + "/git/head",
	} {
		res, err := http.Get(ts.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		var page struct {
			Key   string `json:"key"`
			Token string `json:"token"`
		}
		_ = json.NewDecoder(res.Body).Decode(&page)
		res.Body.Close()
		if res.StatusCode != http.StatusOK || page.Token == "" {
			t.Errorf("%s: status = %d, token = %q", path, res.StatusCode, page.Token)
		}
		tokens[path] = page.Token
	}
	// Three owners, one repository: the token is a fact about the repository,
	// so they must agree.
	seen := map[string]bool{}
	for _, v := range tokens {
		seen[v] = true
	}
	if len(seen) != 1 {
		t.Errorf("owners of one repository disagreed about its token: %v", tokens)
	}
}

// A worktree command names its checkout under the repository root, not
// relative to wherever the terminal sits — read through a sibling worktree,
// the same action must not nest a checkout inside it (ADR-0096 review).
func TestComposeWorktreePathIsUnderTheRepositoryRoot(t *testing.T) {
	repo := gitRepo(t)
	side := filepath.Join(t.TempDir(), "side")
	gitRun(t, repo, "worktree", "add", "-b", "side", side)
	st := testStore(t)
	_, agent, err := storeWorkspaceWithAgent(st, "App", side)
	if err != nil {
		t.Fatal(err)
	}
	ts := graphServer(t, st)
	_, view, _ := postCompose(t, ts, "/api/agents/"+agent.ID+"/git/compose",
		map[string]any{"action": "create-worktree", "target": "main", "name": "x"})
	want := "git worktree add '" + canonDir(repo) + "/.worktrees/x' main"
	if view.Command != want {
		t.Errorf("command = %q; want %q", view.Command, want)
	}
	if view.Head == "" {
		t.Error("compose must report the checkout's head for the undo record")
	}
}

// A second remote's branches are that remote's, not origin's.
func TestComposeReadsTheRemoteFromTheTarget(t *testing.T) {
	ts, agent, repo := composeAgent(t)
	gitRun(t, repo, "remote", "add", "upstream", "https://example.invalid/u.git")
	_, view, _ := postCompose(t, ts, "/api/agents/"+agent.ID+"/git/compose",
		map[string]any{"action": "delete-remote-branch", "target": "upstream/feat-x"})
	if view.Command != "git push upstream --delete feat-x" {
		t.Errorf("command = %q", view.Command)
	}
}
