package server

import (
	"encoding/json"
	"github.com/cfpperche/picode/internal/store"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

type sessionStatusPage struct {
	Git      bool   `json:"git"`
	RepoRoot string `json:"repoRoot"`
	Branch   string `json:"branch"`
	Changes  []struct {
		Path, Kind string
		Add, Del   int
	} `json:"changes"`
	Totals             map[string]int `json:"totals"`
	WorktreesTruncated int            `json:"worktreesTruncated"`
	Worktrees          []struct {
		Path, Ref, Branch, Worktree, Upstream string
		Detached                              bool
		Ahead, Behind                         int
		Changes                               []struct {
			Path, Kind string
			Add, Del   int
		} `json:"changes"`
		Totals map[string]int `json:"totals"`
	} `json:"worktrees"`
}

func getSessionStatus(t *testing.T, ts *httptest.Server, path string) (int, sessionStatusPage) {
	t.Helper()
	res := do(t, ts.Client(), mustGet(t, ts.URL+path))
	defer res.Body.Close()
	var page sessionStatusPage
	_ = json.NewDecoder(res.Body).Decode(&page)
	return res.StatusCode, page
}

// The session-following signal: an owner on a clean checkout whose sibling
// worktree is dirty must learn about it from its own gitstatus — branch,
// ref, changes and totals — while clean siblings and its own checkout stay
// out of the list. The ref reads back through ?worktree= end to end.
func TestGitStatusListsDirtyLinkedWorktrees(t *testing.T) {
	repo := gitRepo(t)
	side := filepath.Join(t.TempDir(), "side")
	gitRun(t, repo, "worktree", "add", "-b", "side", side)
	if err := os.WriteFile(filepath.Join(side, "messy.txt"), []byte("sibling work\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	clean := filepath.Join(t.TempDir(), "clean")
	gitRun(t, repo, "worktree", "add", "-b", "clean", clean)

	st := testStore(t)
	_, agent, err := storeWorkspaceWithAgent(st, "App", repo)
	if err != nil {
		t.Fatal(err)
	}
	ts := graphServer(t, st)

	code, page := getSessionStatus(t, ts, "/api/agents/"+agent.ID+"/gitstatus")
	if code != http.StatusOK || !page.Git {
		t.Fatalf("gitstatus = %d %+v", code, page)
	}
	if len(page.Changes) != 0 {
		t.Fatalf("root must stay clean: %+v", page.Changes)
	}
	if len(page.Worktrees) != 1 {
		t.Fatalf("worktrees = %+v, want exactly the dirty sibling", page.Worktrees)
	}
	wt := page.Worktrees[0]
	if wt.Branch != "side" || wt.Ref != "side" || !sameDir(wt.Path, side) {
		t.Fatalf("sibling entry = %+v", wt)
	}
	if len(wt.Changes) != 1 || wt.Changes[0].Path != "messy.txt" || wt.Totals["files"] != 1 {
		t.Fatalf("sibling changes = %+v totals = %+v", wt.Changes, wt.Totals)
	}

	// The advertised ref reads through the sibling, never the owner.
	code, scoped := getGitStatus(t, ts, "/api/agents/"+agent.ID+"/gitstatus?worktree="+url.QueryEscape(wt.Ref))
	if code != http.StatusOK || len(scoped.Changes) != 1 || scoped.Changes[0].Path != "messy.txt" {
		t.Fatalf("ref %q reads %+v, want the sibling's file", wt.Ref, scoped.Changes)
	}
}

// Worktree listing works from any checkout: an agent bound inside a linked
// worktree sees the main checkout's dirt as its sibling.
func TestGitStatusWorktreesFromInsideAWorktree(t *testing.T) {
	repo := gitRepo(t)
	side := filepath.Join(t.TempDir(), "side")
	gitRun(t, repo, "worktree", "add", "-b", "side", side)
	if err := os.WriteFile(filepath.Join(repo, "mine.txt"), []byte("owner work\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	st := testStore(t)
	ws, err := st.AddWorkspace("App", side)
	if err != nil {
		t.Fatal(err)
	}
	agent, err := st.AddAgent(ws.ID, "Side", "")
	if err != nil {
		t.Fatal(err)
	}
	ts := graphServer(t, st)

	code, page := getSessionStatus(t, ts, "/api/agents/"+agent.ID+"/gitstatus")
	if code != http.StatusOK || !page.Git {
		t.Fatalf("gitstatus = %d %+v", code, page)
	}
	if len(page.Worktrees) != 1 || page.Worktrees[0].Branch != "main" {
		t.Fatalf("worktrees = %+v, want the dirty main checkout", page.Worktrees)
	}
	if len(page.Worktrees[0].Changes) != 1 || page.Worktrees[0].Changes[0].Path != "mine.txt" {
		t.Fatalf("main changes = %+v", page.Worktrees[0].Changes)
	}
}

// A detached checkout has no branch to be named by: its ref is the full
// HEAD hash, and it reads back through ?worktree= like any other.
func TestGitStatusDetachedWorktreeRef(t *testing.T) {
	repo := gitRepo(t)
	side := filepath.Join(t.TempDir(), "side")
	gitRun(t, repo, "worktree", "add", "--detach", side)
	if err := os.WriteFile(filepath.Join(side, "messy.txt"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	st := testStore(t)
	_, agent, err := storeWorkspaceWithAgent(st, "App", repo)
	if err != nil {
		t.Fatal(err)
	}
	ts := graphServer(t, st)

	code, page := getSessionStatus(t, ts, "/api/agents/"+agent.ID+"/gitstatus")
	if code != http.StatusOK || len(page.Worktrees) != 1 {
		t.Fatalf("worktrees = %+v", page.Worktrees)
	}
	wt := page.Worktrees[0]
	if !wt.Detached || len(wt.Ref) != 40 {
		t.Fatalf("detached entry = %+v, want detached with a full-hash ref", wt)
	}
	code, scoped := getGitStatus(t, ts, "/api/agents/"+agent.ID+"/gitstatus?worktree="+wt.Ref)
	if code != http.StatusOK || len(scoped.Changes) != 1 || scoped.Changes[0].Path != "messy.txt" {
		t.Fatalf("hash ref reads %+v, want the sibling's file", scoped.Changes)
	}
}

// No linked worktrees, no field: old readers (and the phone) see the page
// they always saw.
func TestGitStatusOmitsWorktreesFieldWhenNone(t *testing.T) {
	repo := gitRepo(t)
	st := testStore(t)
	_, agent, err := storeWorkspaceWithAgent(st, "App", repo)
	if err != nil {
		t.Fatal(err)
	}
	ts := graphServer(t, st)
	res := do(t, ts.Client(), mustGet(t, ts.URL+"/api/agents/"+agent.ID+"/gitstatus"))
	defer res.Body.Close()
	var raw map[string]any
	if err := json.NewDecoder(res.Body).Decode(&raw); err != nil {
		t.Fatal(err)
	}
	if _, ok := raw["worktrees"]; ok {
		t.Fatalf("worktrees field present with no siblings: %+v", raw)
	}
	if _, ok := raw["worktreesTruncated"]; ok {
		t.Fatalf("worktreesTruncated present with no siblings: %+v", raw)
	}
}

// Past the cap the rest count as truncated instead of turning one status
// read into seconds of git.
func TestGitStatusWorktreesTruncated(t *testing.T) {
	repo := gitRepo(t)
	for i := 0; i < maxSessionWorktrees+1; i++ {
		name := "side" + string(rune('a'+i))
		side := filepath.Join(t.TempDir(), name)
		gitRun(t, repo, "worktree", "add", "-b", name, side)
		if err := os.WriteFile(filepath.Join(side, "messy.txt"), []byte("x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	st := testStore(t)
	_, agent, err := storeWorkspaceWithAgent(st, "App", repo)
	if err != nil {
		t.Fatal(err)
	}
	ts := graphServer(t, st)
	code, page := getSessionStatus(t, ts, "/api/agents/"+agent.ID+"/gitstatus")
	if code != http.StatusOK {
		t.Fatalf("gitstatus = %d", code)
	}
	if len(page.Worktrees) != maxSessionWorktrees || page.WorktreesTruncated != 1 {
		t.Fatalf("worktrees = %d truncated = %d, want %d and 1", len(page.Worktrees), page.WorktreesTruncated, maxSessionWorktrees)
	}
}

// The text routes read through the sibling checkout named by ?worktree=,
// for every owner kind — the center editor opens a followed worktree's
// file through the same door as the owner's own. Unknown refs are 404,
// and the root precondition still asserts the resolved checkout.
func TestWorktreeScopedTextReads(t *testing.T) {
	repo := gitRepo(t)
	side := filepath.Join(t.TempDir(), "side")
	gitRun(t, repo, "worktree", "add", "-b", "side", side)
	if err := os.WriteFile(filepath.Join(side, "messy.txt"), []byte("sibling work\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "mine.txt"), []byte("owner work\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	st := testStore(t)
	ws, agent, err := storeWorkspaceWithAgent(st, "App", repo)
	if err != nil {
		t.Fatal(err)
	}
	term, err := st.CreateTerminalIn(store.FreeWorkspaceID, "sh", repo)
	if err != nil {
		t.Fatal(err)
	}
	ts := graphServer(t, st)
	bases := map[string]string{
		"agent":     "/api/agents/" + agent.ID,
		"terminal":  "/api/terminals/" + term.ID,
		"workspace": "/api/workspaces/" + ws.ID,
	}
	for name, base := range bases {
		code, body := getBody(t, ts, base+"/text?worktree=side&path=messy.txt")
		if code != http.StatusOK || !strings.Contains(body, "sibling work") {
			t.Fatalf("%s scoped text = %d, want the sibling's content: %s", name, code, body)
		}
		code, _ = getBody(t, ts, base+"/text?worktree=side&path=mine.txt")
		if code != http.StatusNotFound {
			t.Fatalf("%s sibling read of an owner-only file = %d, want 404", name, code)
		}
		code, _ = getBody(t, ts, base+"/text?worktree=nosuch&path=messy.txt")
		if code != http.StatusNotFound {
			t.Fatalf("%s bad worktree = %d, want 404", name, code)
		}
		// The root precondition asserts the resolved checkout, not the owner.
		for _, tc := range []struct {
			root string
			want int
		}{{"", http.StatusOK}, {canonDir(side), http.StatusOK}, {canonDir(repo), http.StatusConflict}} {
			code, _ := getBody(t, ts, base+"/text?worktree=side&path=messy.txt&root="+url.QueryEscape(tc.root))
			if code != tc.want {
				t.Fatalf("%s scoped root %q = %d, want %d", name, tc.root, code, tc.want)
			}
		}
	}
}

// Saving through ?worktree= lands in the sibling checkout, and a bad ref
// saves nowhere.
func TestWorktreeScopedTextSave(t *testing.T) {
	repo := gitRepo(t)
	side := filepath.Join(t.TempDir(), "side")
	gitRun(t, repo, "worktree", "add", "-b", "side", side)
	if err := os.WriteFile(filepath.Join(side, "messy.txt"), []byte("before\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	st := testStore(t)
	ws, agent, err := storeWorkspaceWithAgent(st, "App", repo)
	if err != nil {
		t.Fatal(err)
	}
	term, err := st.CreateTerminalIn(store.FreeWorkspaceID, "sh", repo)
	if err != nil {
		t.Fatal(err)
	}
	ts := graphServer(t, st)
	put := func(t *testing.T, url, body string) int {
		t.Helper()
		req, _ := http.NewRequest(http.MethodPut, ts.URL+url, strings.NewReader(body))
		res := do(t, ts.Client(), req)
		defer res.Body.Close()
		return res.StatusCode
	}
	for name, base := range map[string]string{
		"agent":     "/api/agents/" + agent.ID,
		"terminal":  "/api/terminals/" + term.ID,
		"workspace": "/api/workspaces/" + ws.ID,
	} {
		fi, err := os.Stat(filepath.Join(side, "messy.txt"))
		if err != nil {
			t.Fatal(err)
		}
		saved := `{"path":"messy.txt","text":"through the sibling (` + name + `)\n","mtime":` + strconv.FormatInt(fi.ModTime().UnixMilli(), 10) + `}`
		if code := put(t, base+"/text?worktree=side", saved); code != http.StatusOK {
			t.Fatalf("%s scoped save = %d", name, code)
		}
		want := "through the sibling (" + name + ")\n"
		if got, _ := os.ReadFile(filepath.Join(side, "messy.txt")); string(got) != want {
			t.Fatalf("%s sibling file = %q", name, got)
		}
	}
	if _, err := os.Stat(filepath.Join(repo, "messy.txt")); !os.IsNotExist(err) {
		t.Fatal("save leaked into the owner's checkout")
	}
	fi, err := os.Stat(filepath.Join(side, "messy.txt"))
	if err != nil {
		t.Fatal(err)
	}
	saved := `{"path":"messy.txt","text":"nope\n","mtime":` + strconv.FormatInt(fi.ModTime().UnixMilli(), 10) + `}`
	if code := put(t, "/api/terminals/"+term.ID+"/text?worktree=nosuch", saved); code != http.StatusNotFound {
		t.Fatalf("terminal bad worktree save = %d, want 404", code)
	}
	if code := put(t, "/api/workspaces/"+ws.ID+"/text?worktree=side&root="+url.QueryEscape(canonDir(repo)), saved); code != http.StatusConflict {
		t.Fatalf("workspace stale root save = %d, want 409", code)
	}
}

// A worktree whose checkout is gone (prunable in git's list) never appears:
// the status read stays 200 with no worktrees field instead of failing or
// lying about a folder that is not there.
func TestGitStatusSkipsPrunableWorktree(t *testing.T) {
	repo := gitRepo(t)
	side := filepath.Join(t.TempDir(), "side")
	gitRun(t, repo, "worktree", "add", "-b", "side", side)
	if err := os.WriteFile(filepath.Join(side, "messy.txt"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(side); err != nil {
		t.Fatal(err)
	}

	st := testStore(t)
	_, agent, err := storeWorkspaceWithAgent(st, "App", repo)
	if err != nil {
		t.Fatal(err)
	}
	ts := graphServer(t, st)
	res := do(t, ts.Client(), mustGet(t, ts.URL+"/api/agents/"+agent.ID+"/gitstatus"))
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("gitstatus = %d", res.StatusCode)
	}
	var raw map[string]any
	if err := json.NewDecoder(res.Body).Decode(&raw); err != nil {
		t.Fatal(err)
	}
	if _, ok := raw["worktrees"]; ok {
		t.Fatalf("prunable checkout listed: %+v", raw["worktrees"])
	}
}

// A bare repository has checkouts but no working tree of its own: the page
// is git:false, with no worktrees field to misread.
func TestGitStatusBareRepository(t *testing.T) {
	repo := gitRepo(t)
	bare := filepath.Join(t.TempDir(), "bare.git")
	gitRun(t, repo, "clone", "--bare", "--quiet", repo, bare)

	st := testStore(t)
	ws, err := st.AddWorkspace("Bare", bare)
	if err != nil {
		t.Fatal(err)
	}
	agent, err := st.AddAgent(ws.ID, "Bare", "")
	if err != nil {
		t.Fatal(err)
	}
	ts := graphServer(t, st)
	res := do(t, ts.Client(), mustGet(t, ts.URL+"/api/agents/"+agent.ID+"/gitstatus"))
	defer res.Body.Close()
	var raw map[string]any
	if err := json.NewDecoder(res.Body).Decode(&raw); err != nil {
		t.Fatal(err)
	}
	if raw["git"] != false {
		t.Fatalf("bare page = %+v, want git:false", raw)
	}
	if _, ok := raw["worktrees"]; ok {
		t.Fatalf("bare page lists worktrees: %+v", raw["worktrees"])
	}
}

// The image-meta route reads through the sibling checkout too: a file that
// exists only there resolves scoped and errors unscoped.
func TestWorktreeScopedImageMeta(t *testing.T) {
	repo := gitRepo(t)
	side := filepath.Join(t.TempDir(), "side")
	gitRun(t, repo, "worktree", "add", "-b", "side", side)
	if err := os.WriteFile(filepath.Join(side, "dot.png"), pngBytes, 0o644); err != nil {
		t.Fatal(err)
	}

	st := testStore(t)
	ws, agent, err := storeWorkspaceWithAgent(st, "App", repo)
	if err != nil {
		t.Fatal(err)
	}
	ts := graphServer(t, st)
	for name, base := range map[string]string{"agent": "/api/agents/" + agent.ID, "workspace": "/api/workspaces/" + ws.ID} {
		code, body := getBody(t, ts, base+"/file?worktree=side&path=dot.png")
		if code != http.StatusOK || !strings.Contains(body, `"name":"dot.png"`) {
			t.Fatalf("%s scoped image = %d, want the sibling's meta: %s", name, code, body)
		}
		// The image route maps read errors to 400 (its own contract);
		// the point stands: unscoped, the sibling-only file is not there.
		if code, _ := getBody(t, ts, base+"/file?path=dot.png"); code != http.StatusBadRequest {
			t.Fatalf("%s unscoped read = %d, want 400", name, code)
		}
		if code, _ := getBody(t, ts, base+"/file?worktree=nosuch&path=dot.png"); code != http.StatusNotFound {
			t.Fatalf("%s bad worktree = %d, want 404", name, code)
		}
	}
}
