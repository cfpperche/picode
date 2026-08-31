package server

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The change dot expands into a patch through any of the three owners; the
// path is confined the same way file reads are.
func TestWorkDiffRoutesAndConfinement(t *testing.T) {
	repo := gitRepo(t)
	if err := os.WriteFile(filepath.Join(repo, "a"), []byte("changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	st := testStore(t)
	ts := graphServer(t, st)
	wk, err := st.AddWorkspace("Repo", repo)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.AddAgent(wk.ID, "builder", ""); err != nil {
		t.Fatal(err)
	}
	agents, _ := st.ListAgents(wk.ID)

	for _, base := range []string{
		"/api/workspaces/" + wk.ID,
		"/api/agents/" + agents[0].ID,
	} {
		res := do(t, ts.Client(), mustGet(t, ts.URL+base+"/gitdiff?path=a"))
		var body map[string]any
		_ = json.NewDecoder(res.Body).Decode(&body)
		if res.StatusCode != http.StatusOK {
			t.Fatalf("%s = %d %+v", base, res.StatusCode, body)
		}
		patch, _ := body["patch"].(string)
		if !strings.Contains(patch, "+changed") {
			t.Fatalf("%s patch = %q", base, patch)
		}
		if body["truncated"] != false || body["binary"] != false {
			t.Fatalf("%s flags = %+v", base, body)
		}
	}

	for path, want := range map[string]int{
		"/api/workspaces/" + wk.ID + "/gitdiff?path=../x":     http.StatusBadRequest,
		"/api/workspaces/" + wk.ID + "/gitdiff?path=":         http.StatusBadRequest,
		"/api/workspaces/" + wk.ID + "/gitdiff?path=nope.txt": http.StatusNotFound,
		"/api/workspaces/nope/gitdiff?path=a":                 http.StatusNotFound,
	} {
		if res := do(t, ts.Client(), mustGet(t, ts.URL+path)); res.StatusCode != want {
			t.Errorf("%s = %d, want %d", path, res.StatusCode, want)
		}
	}
}

// Reveal resolves the folder server-side and hands it to the OS opener;
// the test swaps the opener for a recorder — CI has no file manager.
func TestRevealResolvesTheOwnerFolder(t *testing.T) {
	st := testStore(t)
	ts := graphServer(t, st)
	proj := t.TempDir()
	wk, err := st.AddWorkspace("App", proj)
	if err != nil {
		t.Fatal(err)
	}

	var opened string
	orig := revealFn
	revealFn = func(p string) error { opened = p; return nil }
	t.Cleanup(func() { revealFn = orig })

	res := do(t, ts.Client(), mustPostBody(t, ts.URL+"/api/workspaces/"+wk.ID+"/reveal", "{}"))
	if res.StatusCode != http.StatusOK {
		t.Fatalf("reveal = %d", res.StatusCode)
	}
	if opened != proj {
		t.Fatalf("opened %q, want %q", opened, proj)
	}

	if err := os.MkdirAll(filepath.Join(proj, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	res = do(t, ts.Client(), mustPostBody(t, ts.URL+"/api/workspaces/"+wk.ID+"/reveal", `{"path":"sub"}`))
	if res.StatusCode != http.StatusOK || opened != filepath.Join(proj, "sub") {
		t.Fatalf("subdir reveal = %d, opened %q", res.StatusCode, opened)
	}

	opened = ""
	for body, want := range map[string]int{
		`{"path":"../../etc"}`: http.StatusBadRequest,
	} {
		if res := do(t, ts.Client(), mustPostBody(t, ts.URL+"/api/workspaces/"+wk.ID+"/reveal", body)); res.StatusCode != want {
			t.Errorf("reveal %s = %d, want %d", body, res.StatusCode, want)
		}
	}
	if opened != "" {
		t.Fatalf("an escaping path reached the opener: %q", opened)
	}

	if res := do(t, ts.Client(), mustPostBody(t, ts.URL+"/api/workspaces/ws_free/reveal", "{}")); res.StatusCode != http.StatusNotFound {
		t.Fatalf("ws_free reveal = %d", res.StatusCode)
	}
}

func mustPostBody(t *testing.T, url, body string) *http.Request {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(body))
	if err != nil {
		t.Fatalf("build POST: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	return req
}
