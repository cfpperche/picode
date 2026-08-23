package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/cfpperche/picode/internal/tmux"
	"github.com/cfpperche/picode/internal/workspace"
)

// newTestServer builds a server with a temp registry. agentCmd defaults to
// a harmless long-running process for spawn tests ("cat").
func newTestServer(t *testing.T, agentCmd string) *httptest.Server {
	t.Helper()
	reg, err := workspace.Open(filepath.Join(t.TempDir(), "workspaces.json"))
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	if agentCmd == "" {
		agentCmd = "cat"
	}
	ts := httptest.NewServer(New("127.0.0.1:0", Deps{
		Registry: reg,
		Tmux:     tmux.New(),
		AgentCmd: agentCmd,
	}).Handler)
	t.Cleanup(ts.Close)
	return ts
}

// do sends the request and fails the test on transport errors.
func do(t *testing.T, client *http.Client, req *http.Request) *http.Response {
	t.Helper()
	res, err := client.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", req.Method, req.URL, err)
	}
	t.Cleanup(func() { _ = res.Body.Close() })
	return res
}

func TestHealthEndpoint(t *testing.T) {
	ts := newTestServer(t, "cat")

	res, err := ts.Client().Get(ts.URL + "/api/health")
	if err != nil {
		t.Fatalf("GET /api/health: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.StatusCode, http.StatusOK)
	}

	var body map[string]any
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("status field = %v, want ok", body["status"])
	}
}

func TestVersionEndpoint(t *testing.T) {
	ts := newTestServer(t, "cat")

	res, err := ts.Client().Get(ts.URL + "/api/version")
	if err != nil {
		t.Fatalf("GET /api/version: %v", err)
	}
	defer res.Body.Close()

	var body map[string]any
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["name"] != "picode" {
		t.Errorf("name field = %v, want picode", body["name"])
	}
}

func TestSystemEndpoint(t *testing.T) {
	ts := newTestServer(t, "cat")

	res, err := ts.Client().Get(ts.URL + "/api/system")
	if err != nil {
		t.Fatalf("GET /api/system: %v", err)
	}
	defer res.Body.Close()

	var body systemReport
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, err := exec.LookPath("tmux"); err == nil && !body.Tmux.Installed {
		t.Error("tmux installed but report says otherwise")
	}
	if _, err := exec.LookPath("cat"); err == nil && !body.Pi.Installed {
		t.Error("agent cmd (cat) installed but report says pi missing")
	}
}

func TestWorkspaceAPI(t *testing.T) {
	ts := newTestServer(t, "cat")
	client := ts.Client()
	proj := t.TempDir()

	// Add.
	addBody, _ := json.Marshal(map[string]string{"name": "Demo", "path": proj})
	res, err := client.Post(ts.URL+"/api/workspaces", "application/json", bytes.NewReader(addBody))
	if err != nil {
		t.Fatalf("POST /api/workspaces: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("add status = %d, want 201: %s", res.StatusCode, body)
	}
	var wk workspace.Workspace
	if err := json.NewDecoder(res.Body).Decode(&wk); err != nil {
		t.Fatalf("decode add: %v", err)
	}

	// Duplicate add is idempotent.
	req2, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/workspaces", bytes.NewReader(addBody))
	req2.Header.Set("Content-Type", "application/json")
	res2 := do(t, client, req2)
	var wk2 workspace.Workspace
	_ = json.NewDecoder(res2.Body).Decode(&wk2)
	if wk2.ID != wk.ID {
		t.Errorf("duplicate add id = %q, want %q", wk2.ID, wk.ID)
	}

	// Invalid path rejected.
	bad, _ := json.Marshal(map[string]string{"name": "Bad", "path": "/definitely/not/here"})
	req3, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/workspaces", bytes.NewReader(bad))
	req3.Header.Set("Content-Type", "application/json")
	res3 := do(t, client, req3)
	if res3.StatusCode != http.StatusBadRequest {
		t.Errorf("bad path status = %d, want 400", res3.StatusCode)
	}

	// List shows it (not running without tmux/gated environment).
	res4 := do(t, client, mustGet(t, ts.URL+"/api/workspaces"))
	var list []workspaceView
	if err := json.NewDecoder(res4.Body).Decode(&list); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(list) != 1 || list[0].ID != wk.ID {
		t.Fatalf("list = %+v, want 1 workspace %s", list, wk.ID)
	}

	// Open/missing → 404.
	res5 := do(t, client, mustPost(t, ts.URL+"/api/workspaces/missing/open"))
	if res5.StatusCode != http.StatusNotFound {
		t.Errorf("open missing status = %d, want 404", res5.StatusCode)
	}

	// Remove → 204, then list empty.
	req6, _ := http.NewRequest(http.MethodDelete, ts.URL+"/api/workspaces/"+wk.ID, nil)
	res6 := do(t, client, req6)
	if res6.StatusCode != http.StatusNoContent {
		t.Errorf("delete status = %d, want 204", res6.StatusCode)
	}
	res7 := do(t, client, mustGet(t, ts.URL+"/api/workspaces"))
	var list2 []workspaceView
	_ = json.NewDecoder(res7.Body).Decode(&list2)
	if len(list2) != 0 {
		t.Errorf("list after delete = %d, want 0", len(list2))
	}
}

func TestOpenCloseLifecycle(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not installed — integration test skipped (accepted, see docs/handoff.md)")
	}
	ts := newTestServer(t, "cat")
	client := ts.Client()
	proj := t.TempDir()

	addReq, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/workspaces", bytes.NewReader(mustJSONBody("Lifecycle", proj)))
	addReq.Header.Set("Content-Type", "application/json")
	res := do(t, client, addReq)
	var wk workspace.Workspace
	_ = json.NewDecoder(res.Body).Decode(&wk)

	// Open.
	resOpen := do(t, client, mustPost(t, ts.URL+"/api/workspaces/"+wk.ID+"/open"))
	if resOpen.StatusCode != http.StatusCreated {
		t.Fatalf("open status = %d, want 201", resOpen.StatusCode)
	}

	// Idempotent open.
	resOpen2 := do(t, client, mustPost(t, ts.URL+"/api/workspaces/"+wk.ID+"/open"))
	if resOpen2.StatusCode != http.StatusOK {
		t.Errorf("re-open status = %d, want 200", resOpen2.StatusCode)
	}

	// List reports running.
	resList := do(t, client, mustGet(t, ts.URL+"/api/workspaces"))
	var list []workspaceView
	_ = json.NewDecoder(resList.Body).Decode(&list)
	if len(list) != 1 || !list[0].Running {
		t.Fatalf("running flag after open: %+v", list)
	}

	// Close.
	resClose := do(t, client, mustPost(t, ts.URL+"/api/workspaces/"+wk.ID+"/close"))
	if resClose.StatusCode != http.StatusOK {
		t.Fatalf("close status = %d", resClose.StatusCode)
	}

	resList2 := do(t, client, mustGet(t, ts.URL+"/api/workspaces"))
	var list3 []workspaceView
	_ = json.NewDecoder(resList2.Body).Decode(&list3)
	if len(list3) != 1 || list3[0].Running {
		t.Fatalf("running flag after close: %+v", list3)
	}
}

func mustJSONBody(name, path string) []byte {
	b, _ := json.Marshal(map[string]string{"name": name, "path": path})
	return b
}

func mustGet(t *testing.T, url string) *http.Request {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("build GET: %v", err)
	}
	return req
}

func mustPost(t *testing.T, url string) *http.Request {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		t.Fatalf("build POST: %v", err)
	}
	return req
}

func TestIndexServed(t *testing.T) {
	ts := newTestServer(t, "cat")

	res, err := ts.Client().Get(ts.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.StatusCode, http.StatusOK)
	}
	if ct := res.Header.Get("Content-Type"); ct == "" {
		t.Error("Content-Type header missing for index")
	}
}
