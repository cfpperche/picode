package server

import (
	"bytes"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// pngBytes is a 1×1 transparent PNG, so the git-blob route's MIME allowlist
// answers instead of 415.
var pngBytes = func() []byte {
	b, err := base64.StdEncoding.DecodeString(
		"iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg==")
	if err != nil {
		panic(err)
	}
	return b
}()

// The sibling worktree's dirty state must reach the graph loaded through the
// owner's checkout — that is ADR-0073's whole point — and the owner's own
// entry must be marked self so the UI can label it "this worktree".
func TestGraphCarriesPerWorktreeUncommitted(t *testing.T) {
	repo := gitRepo(t)
	side := filepath.Join(t.TempDir(), "side")
	gitRun(t, repo, "worktree", "add", "-b", "side", side)
	if err := os.WriteFile(filepath.Join(side, "messy.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	st := testStore(t)
	_, agent, err := storeWorkspaceWithAgent(st, "App", repo)
	if err != nil {
		t.Fatal(err)
	}
	g := getGraph(t, graphServer(t, st), "/api/agents/"+agent.ID+"/git")

	found := map[string]worktreeView{}
	for _, wt := range g.Worktrees {
		found[wt.Branch] = wt
	}
	main, sideWt := found["main"], found["side"]
	if !main.Self || main.Uncommitted != nil {
		t.Fatalf("main must be self and clean: %+v", main)
	}
	if sideWt.Self {
		t.Fatal("side must not be marked self")
	}
	if sideWt.Uncommitted == nil || sideWt.Uncommitted.Count != 1 {
		t.Fatalf("side must carry the dirty count through the graph: %+v", sideWt.Uncommitted)
	}
}

// The worktree param re-points the status/diff/blob reads at the sibling
// checkout named by branch — never at a path from the URL. Bad names are 404,
// not a silent fallback to the owner's own working tree.
func TestWorktreeScopedEndpoints(t *testing.T) {
	repo := gitRepo(t)
	side := filepath.Join(t.TempDir(), "side")
	gitRun(t, repo, "worktree", "add", "-b", "side", side)
	if err := os.WriteFile(filepath.Join(side, "messy.txt"), []byte("sibling work\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// A different dirty file in the owner's checkout, to prove the scoping.
	if err := os.WriteFile(filepath.Join(repo, "mine.txt"), []byte("owner work\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	st := testStore(t)
	_, agent, err := storeWorkspaceWithAgent(st, "App", repo)
	if err != nil {
		t.Fatal(err)
	}
	ts := graphServer(t, st)
	base := "/api/agents/" + agent.ID

	code, page := getGitStatus(t, ts, base+"/gitstatus?worktree=side")
	if code != http.StatusOK || !page.Git {
		t.Fatalf("scoped gitstatus = %d %+v", code, page)
	}
	if len(page.Changes) != 1 || page.Changes[0].Path != "messy.txt" {
		t.Fatalf("scoped status must list only the sibling's file: %+v", page.Changes)
	}

	code, page = getGitStatus(t, ts, base+"/gitstatus")
	if code != http.StatusOK || len(page.Changes) != 1 || page.Changes[0].Path != "mine.txt" {
		t.Fatalf("unscoped status must stay on the owner: %d %+v", code, page.Changes)
	}

	// Empty means absent (the owner's own tree, 200); anything that names no
	// checked-out branch or worktree head is 404.
	code, page = getGitStatus(t, ts, base+"/gitstatus?worktree=")
	if code != http.StatusOK || len(page.Changes) != 1 || page.Changes[0].Path != "mine.txt" {
		t.Fatalf("empty worktree param must mean the owner: %d %+v", code, page.Changes)
	}
	for _, bad := range []string{"nosuch", "main%2F..", "HEAD"} {
		code, _ := getGitStatus(t, ts, base+"/gitstatus?worktree="+bad)
		if code != http.StatusNotFound {
			t.Fatalf("gitstatus?worktree=%q = %d, want 404", bad, code)
		}
	}

	code, body := getBody(t, ts, base+"/gitdiff?worktree=side&path=messy.txt")
	if code != http.StatusOK {
		t.Fatalf("scoped gitdiff = %d: %s", code, body)
	}
	if want := "+sibling work"; !strings.Contains(body, want) {
		t.Fatalf("scoped diff must show the sibling's patch, want %q in:\n%s", want, body)
	}

	code, body = getBody(t, ts, base+"/gitdiff?worktree=side&path=mine.txt")
	if code != http.StatusNotFound {
		t.Fatalf("a file the sibling does not have = %d, want 404: %s", code, body)
	}

	// A committed asset in the sibling, previewable by the git-blob route.
	if err := os.WriteFile(filepath.Join(side, "dot.png"), pngBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, side, "add", "dot.png")
	gitRun(t, side, "commit", "-m", "asset")

	code, body = getBody(t, ts, base+"/git/blob?worktree=side&hash=HEAD&path=dot.png")
	if code != http.StatusOK || len(body) == 0 {
		t.Fatalf("scoped git blob = %d: %s", code, body)
	}

	code, body = getBody(t, ts, base+"/blob?worktree=side&path=dot.png")
	if code != http.StatusOK || len(body) == 0 {
		t.Fatalf("scoped work blob = %d: %s", code, body)
	}
}

func getBody(t *testing.T, ts *httptest.Server, path string) (int, string) {
	t.Helper()
	res := do(t, ts.Client(), mustGet(t, ts.URL+path))
	buf := new(bytes.Buffer)
	_, _ = buf.ReadFrom(res.Body)
	res.Body.Close()
	return res.StatusCode, buf.String()
}
