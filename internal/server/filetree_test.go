package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/cfpperche/picode/internal/tmux"
)

// The workspace is a file-reading owner of its own (ADR-0030): browsing and
// reading confine to the registered folder, with no agent required.
func TestWorkspaceBrowseIsConfinedToItsFolder(t *testing.T) {
	ts, _, _ := cleanupServer(t)
	proj := t.TempDir()
	if err := os.WriteFile(filepath.Join(proj, "a.go"), []byte("pkg\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	res := postJSON(t, ts, "/api/workspaces", map[string]string{"name": "App", "path": proj})
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("add workspace = %d", res.StatusCode)
	}
	var wk workspaceView
	_ = json.NewDecoder(res.Body).Decode(&wk)

	got := do(t, ts.Client(), mustGet(t, ts.URL+"/api/workspaces/"+wk.ID+"/browse"))
	if got.StatusCode != http.StatusOK {
		t.Fatalf("browse = %d", got.StatusCode)
	}
	var page map[string]any
	_ = json.NewDecoder(got.Body).Decode(&page)
	if page["cwdOk"] != true || page["root"] == "" {
		t.Fatalf("browse page = %+v", page)
	}

	for path, want := range map[string]int{
		"/api/workspaces/" + wk.ID + "/browse?dir=../x":       http.StatusBadRequest,
		"/api/workspaces/" + wk.ID + "/text?path=/etc/passwd": http.StatusBadRequest,
		"/api/workspaces/nope/browse":                         http.StatusNotFound,
		"/api/workspaces/ws_free/browse":                      http.StatusNotFound,
	} {
		if got := do(t, ts.Client(), mustGet(t, ts.URL+path)); got.StatusCode != want {
			t.Errorf("%s = %d, want %d", path, got.StatusCode, want)
		}
	}

	text := do(t, ts.Client(), mustGet(t, ts.URL+"/api/workspaces/"+wk.ID+"/text?path=a.go"))
	var body map[string]any
	_ = json.NewDecoder(text.Body).Decode(&body)
	if text.StatusCode != http.StatusOK || body["text"] != "pkg\n" {
		t.Fatalf("text = %d %+v", text.StatusCode, body)
	}
}

// A workspace whose folder is gone answers the browse's soft shape, not an
// error — the tree renders "that folder is gone", it does not crash.
func TestWorkspaceBrowseOnADeletedFolder(t *testing.T) {
	ts, _, _ := cleanupServer(t)
	proj := t.TempDir()
	res := postJSON(t, ts, "/api/workspaces", map[string]string{"name": "App", "path": proj})
	var wk workspaceView
	_ = json.NewDecoder(res.Body).Decode(&wk)
	if err := os.RemoveAll(proj); err != nil {
		t.Fatal(err)
	}
	got := do(t, ts.Client(), mustGet(t, ts.URL+"/api/workspaces/"+wk.ID+"/browse"))
	var page map[string]any
	_ = json.NewDecoder(got.Body).Decode(&page)
	if got.StatusCode != http.StatusOK || page["cwdOk"] != false {
		t.Fatalf("deleted folder browse = %d %+v", got.StatusCode, page)
	}
}

// The terminal mirror browses at the live pane cwd, like its text route.
func TestTerminalBrowse(t *testing.T) {
	ts, _, _ := cleanupServer(t)
	miss := do(t, ts.Client(), mustGet(t, ts.URL+"/api/terminals/nope/browse"))
	if miss.StatusCode != http.StatusNotFound {
		t.Fatalf("gone term = %d", miss.StatusCode)
	}
	if !tmux.New().Available() {
		t.Skip("tmux not installed")
	}
	proj := t.TempDir()
	if err := os.WriteFile(filepath.Join(proj, "a.go"), []byte("pkg\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	created := postJSON(t, ts, "/api/terminals", map[string]any{"cwd": proj})
	var page map[string]any
	_ = json.NewDecoder(created.Body).Decode(&page)
	id, _ := page["id"].(string)
	sess, _ := page["session"].(string)
	t.Cleanup(func() { _ = tmux.New().KillSession(t.Context(), sess) })

	got := do(t, ts.Client(), mustGet(t, ts.URL+"/api/terminals/"+id+"/browse"))
	var out map[string]any
	_ = json.NewDecoder(got.Body).Decode(&out)
	if got.StatusCode != http.StatusOK || out["cwdOk"] != true || out["root"] == "" {
		t.Fatalf("browse = %d %+v", got.StatusCode, out)
	}
	bad := do(t, ts.Client(), mustGet(t, ts.URL+"/api/terminals/"+id+"/browse?dir=../x"))
	if bad.StatusCode != http.StatusBadRequest {
		t.Fatalf("escape = %d", bad.StatusCode)
	}
}

type gitStatusPage struct {
	Git      bool                          `json:"git"`
	RepoRoot string                        `json:"repoRoot"`
	Changes  []struct{ Path, Kind string } `json:"changes"`
}

func getGitStatus(t *testing.T, ts *httptest.Server, path string) (int, gitStatusPage) {
	t.Helper()
	res := do(t, ts.Client(), mustGet(t, ts.URL+path))
	var page gitStatusPage
	_ = json.NewDecoder(res.Body).Decode(&page)
	return res.StatusCode, page
}

// gitstatus decorates the tree: no repo is a state (git:false), and change
// paths are re-anchored from the repo toplevel to the owner's cwd, with
// changes outside the cwd dropped.
func TestGitStatusReAnchorsToTheOwnerCwd(t *testing.T) {
	repo := gitRepo(t)
	sub := filepath.Join(repo, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "a"), []byte("changed"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "fresh"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	st := testStore(t)
	ts := graphServer(t, st)

	// Owner at the toplevel sees both changes, top-relative.
	wk, err := st.AddWorkspace("Repo", repo)
	if err != nil {
		t.Fatal(err)
	}
	code, page := getGitStatus(t, ts, "/api/workspaces/"+wk.ID+"/gitstatus")
	if code != http.StatusOK || !page.Git {
		t.Fatalf("workspace gitstatus = %d %+v", code, page)
	}
	kinds := map[string]string{}
	for _, c := range page.Changes {
		kinds[c.Path] = c.Kind
	}
	if kinds["a"] != "modified" || kinds["sub/fresh"] != "untracked" {
		t.Fatalf("toplevel changes = %v", kinds)
	}

	// An agent confined to the subdir sees only its slice, re-anchored.
	if _, err := st.AddAgent(wk.ID, "deep", sub); err != nil {
		t.Fatal(err)
	}
	agents, err := st.ListAgents(wk.ID)
	if err != nil || len(agents) == 0 {
		t.Fatalf("agents: %v %v", agents, err)
	}
	code, page = getGitStatus(t, ts, "/api/agents/"+agents[0].ID+"/gitstatus")
	if code != http.StatusOK || !page.Git {
		t.Fatalf("agent gitstatus = %d %+v", code, page)
	}
	kinds = map[string]string{}
	for _, c := range page.Changes {
		kinds[c.Path] = c.Kind
	}
	if kinds["fresh"] != "untracked" {
		t.Fatalf("subdir changes = %v, want fresh (re-anchored)", kinds)
	}
	if _, leaked := kinds["../a"]; leaked || len(kinds) != 1 {
		t.Fatalf("changes outside the cwd leaked: %v", kinds)
	}
}

func TestGitStatusOnAPlainFolderIsAState(t *testing.T) {
	st := testStore(t)
	ts := graphServer(t, st)
	wk, err := st.AddWorkspace("Plain", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	code, page := getGitStatus(t, ts, "/api/workspaces/"+wk.ID+"/gitstatus")
	if code != http.StatusOK || page.Git || len(page.Changes) != 0 {
		t.Fatalf("plain folder = %d %+v, want 200 git:false", code, page)
	}
}
