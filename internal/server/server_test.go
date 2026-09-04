package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/cfpperche/picode/internal/pipkg"
	"github.com/cfpperche/picode/internal/rpc"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
	"github.com/cfpperche/picode/internal/version"
)

// fakeBlockingAgentCmd is a tiny script that ignores argv and blocks
// reading stdin, like bare `cat` — but unlike `cat`, tolerant of whatever
// flags PiCode passes (e.g. --session-id, ADR-0039), so tmux/interactive
// lifecycle tests don't depend on CLIFlags() happening to be empty. (tmux
// execs the command directly with no shell, so a bare env var set on the
// test process would not reach it — this sidesteps that entirely.)
func fakeBlockingAgentCmd(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "fake-pi")
	if err := os.WriteFile(p, []byte("#!/bin/sh\nexec cat\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

// newTestServer builds a server with a temp registry. agentCmd defaults to
// a harmless long-running process for spawn tests ("cat").
func newTestServer(t *testing.T, agentCmd string) *httptest.Server {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "picode.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if agentCmd == "" {
		agentCmd = "cat"
	}
	ts := httptest.NewServer(New("127.0.0.1:0", Deps{
		Store:    st,
		Tmux:     tmux.New(),
		Runtime:  rpc.NewRuntime(agentCmd, st, nil),
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
	oldVersion, oldStamped := version.Version, version.Stamped
	version.Version = "9.8.7"
	version.Stamped = "release"
	t.Cleanup(func() { version.Version, version.Stamped = oldVersion, oldStamped })
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
	if body["semver"] != "9.8.7" {
		t.Errorf("semver field = %v, want 9.8.7", body["semver"])
	}
	if body["release"] != true {
		t.Errorf("release field = %v, want true for stamped builds", body["release"])
	}

	version.Stamped = ""
	res2, err := ts.Client().Get(ts.URL + "/api/version")
	if err != nil {
		t.Fatalf("GET /api/version source build: %v", err)
	}
	defer res2.Body.Close()
	var source map[string]any
	if err := json.NewDecoder(res2.Body).Decode(&source); err != nil {
		t.Fatalf("decode source build: %v", err)
	}
	if source["release"] != false {
		t.Errorf("release field = %v, want false for source builds", source["release"])
	}
}

func TestPackagesEndpoint(t *testing.T) {
	old := pipkg.UserDir
	pipkg.UserDir = func() string { return t.TempDir() }
	t.Cleanup(func() { pipkg.UserDir = old })
	ts := newTestServer(t, "cat")

	res, err := ts.Client().Get(ts.URL + "/api/packages")
	if err != nil {
		t.Fatalf("GET /api/packages: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", res.StatusCode)
	}
	var body map[string]any
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if _, ok := body["packages"]; !ok {
		t.Fatalf("missing packages: %+v", body)
	}

	bad, _ := json.Marshal(map[string]string{"source": "npm:foo; rm"})
	pres, err := ts.Client().Post(ts.URL+"/api/packages", "application/json", bytes.NewReader(bad))
	if err != nil {
		t.Fatal(err)
	}
	defer pres.Body.Close()
	if pres.StatusCode != http.StatusBadRequest {
		t.Fatalf("inject status = %d", pres.StatusCode)
	}

	proj, _ := json.Marshal(map[string]string{"source": "npm:foo", "scope": "project"})
	pres2, err := ts.Client().Post(ts.URL+"/api/packages", "application/json", bytes.NewReader(proj))
	if err != nil {
		t.Fatal(err)
	}
	defer pres2.Body.Close()
	if pres2.StatusCode != http.StatusBadRequest {
		t.Fatalf("project without workspace status = %d", pres2.StatusCode)
	}

	ures, err := ts.Client().Get(ts.URL + "/api/packages/updates")
	if err != nil {
		t.Fatalf("GET /api/packages/updates: %v", err)
	}
	defer ures.Body.Close()
	if ures.StatusCode != http.StatusOK {
		t.Fatalf("updates status = %d", ures.StatusCode)
	}
	var ubody map[string]any
	if err := json.NewDecoder(ures.Body).Decode(&ubody); err != nil {
		t.Fatal(err)
	}
	if _, ok := ubody["updates"]; !ok {
		t.Fatalf("missing updates: %+v", ubody)
	}

	agentUp, _ := json.Marshal(map[string]string{"source": "npm:foo", "scope": "agent"})
	up, err := ts.Client().Post(ts.URL+"/api/packages/update", "application/json", bytes.NewReader(agentUp))
	if err != nil {
		t.Fatal(err)
	}
	defer up.Body.Close()
	if up.StatusCode != http.StatusBadRequest {
		t.Fatalf("agent update status = %d", up.StatusCode)
	}

	badUp, _ := json.Marshal(map[string]string{"source": "npm:foo; rm", "scope": "user"})
	up2, err := ts.Client().Post(ts.URL+"/api/packages/update", "application/json", bytes.NewReader(badUp))
	if err != nil {
		t.Fatal(err)
	}
	defer up2.Body.Close()
	if up2.StatusCode != http.StatusBadRequest {
		t.Fatalf("inject update status = %d", up2.StatusCode)
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
	if _, err := exec.LookPath("tailscale"); err == nil && !body.Tailscale.Installed {
		t.Error("tailscale installed but report says otherwise")
	}
	if _, err := exec.LookPath("mkcert"); err == nil && !body.Mkcert.Installed {
		t.Error("mkcert installed but report says otherwise")
	}
	if body.Host.OS == "" || body.Host.Arch == "" {
		t.Fatalf("host os/arch empty: %+v", body.Host)
	}
	if body.Network.LAN == nil {
		t.Fatal("network.lan should be an array")
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
	var wk store.Workspace
	if err := json.NewDecoder(res.Body).Decode(&wk); err != nil {
		t.Fatalf("decode add: %v", err)
	}

	// Duplicate add is idempotent.
	req2, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/workspaces", bytes.NewReader(addBody))
	req2.Header.Set("Content-Type", "application/json")
	res2 := do(t, client, req2)
	var wk2 store.Workspace
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
	if list[0].HasFavicon {
		t.Fatal("workspace without an icon should advertise hasFavicon=false")
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
	ts := newTestServer(t, fakeBlockingAgentCmd(t))
	client := ts.Client()
	proj := t.TempDir()

	wk := addWorkspaceWithAgent(t, ts, "Lifecycle", proj)

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
	if len(list) != 1 || !list[0].Agent.Running {
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
	if len(list3) != 1 || list3[0].Agent.Running {
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

func TestTaskEndpoints(t *testing.T) {
	ts := newTestServer(t, "cat")
	client := ts.Client()
	proj := t.TempDir()

	wsv := addWorkspaceWithAgent(t, ts, "Tasks", proj)
	agentID := wsv.Agent.ID
	if agentID == "" {
		t.Fatal("default agent missing from workspace view")
	}

	// Enqueue (kind defaults to prompt).
	body, _ := json.Marshal(map[string]string{"payload": "refactor the tests"})
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/agents/"+agentID+"/tasks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resT := do(t, client, req)
	if resT.StatusCode != http.StatusCreated {
		t.Fatalf("enqueue status = %d", resT.StatusCode)
	}
	var task store.Task
	if err := json.NewDecoder(resT.Body).Decode(&task); err != nil {
		t.Fatalf("decode task: %v", err)
	}
	if task.Kind != "prompt" || task.Status != "queued" || task.Source != "user" {
		t.Errorf("task = %+v", task)
	}

	// Follow-up kind passes through.
	body2, _ := json.Marshal(map[string]string{"kind": "follow_up", "payload": "then run tests"})
	req2, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/agents/"+agentID+"/tasks", bytes.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")
	_ = do(t, client, req2)

	// List shows newest first.
	resL := do(t, client, mustGet(t, ts.URL+"/api/agents/"+agentID+"/tasks"))
	var tasks []store.Task
	if err := json.NewDecoder(resL.Body).Decode(&tasks); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(tasks) != 2 || tasks[0].Kind != "follow_up" {
		t.Fatalf("tasks = %+v", tasks)
	}

	// Invalid kind.
	body3, _ := json.Marshal(map[string]string{"kind": "bogus", "payload": "x"})
	req3, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/agents/"+agentID+"/tasks", bytes.NewReader(body3))
	req3.Header.Set("Content-Type", "application/json")
	resE := do(t, client, req3)
	if resE.StatusCode != http.StatusBadRequest {
		t.Errorf("invalid kind status = %d, want 400", resE.StatusCode)
	}

	// Missing agent.
	res404 := do(t, client, mustPost(t, ts.URL+"/api/agents/missing/tasks"))
	if res404.StatusCode != http.StatusNotFound {
		t.Errorf("missing agent status = %d, want 404", res404.StatusCode)
	}
}

func TestManagedModeFlow(t *testing.T) {
	t.Setenv("PICODE_FAKE_RPC", "1") // AgentCmd = test binary acting as fake pi rpc
	ts := newTestServer(t, os.Args[0])
	client := ts.Client()
	proj := t.TempDir()

	wsv := addWorkspaceWithAgent(t, ts, "Managed", proj)
	agentID := wsv.Agent.ID

	// Start managed.
	resStart := do(t, client, mustPost(t, ts.URL+"/api/agents/"+agentID+"/managed/start"))
	if resStart.StatusCode != http.StatusCreated {
		t.Fatalf("managed start = %d, want 201", resStart.StatusCode)
	}
	// Idempotent.
	resStart2 := do(t, client, mustPost(t, ts.URL+"/api/agents/"+agentID+"/managed/start"))
	if resStart2.StatusCode != http.StatusOK {
		t.Errorf("managed re-start = %d, want 200", resStart2.StatusCode)
	}

	// WS snapshot.
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws/agent?agent=" + agentID
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer ws.Close()
	_ = ws.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, first, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("ws read: %v", err)
	}
	var snap struct {
		Event struct {
			Type string `json:"type"`
			Mode string `json:"mode"`
		} `json:"event"`
	}
	if err := json.Unmarshal(first, &snap); err != nil || snap.Event.Type != "snapshot" || snap.Event.Mode != "managed" {
		t.Fatalf("snapshot = %s (err %v)", first, err)
	}

	// Stop.
	resStop := do(t, client, mustPost(t, ts.URL+"/api/agents/"+agentID+"/managed/stop"))
	if resStop.StatusCode != http.StatusOK {
		t.Fatalf("managed stop = %d", resStop.StatusCode)
	}
}

func TestIndexServed(t *testing.T) {
	// A disk build (ADR-0023) reads the UI from a directory that does not exist
	// under the package dir during tests. Point it at one; the assertion is that
	// the server serves the index, not where the bytes live.
	withUI(t)
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

func TestAddWorkspaceStartsEmpty(t *testing.T) {
	ts := newTestServer(t, "cat")
	proj := t.TempDir()
	res := postJSON(t, ts, "/api/workspaces", map[string]string{"name": "Empty", "path": proj})
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("add = %d", res.StatusCode)
	}
	// The contract is the JSON itself: no "agent" key at all (omitempty),
	// and agents is an empty array, not null.
	var raw map[string]json.RawMessage
	if err := json.NewDecoder(res.Body).Decode(&raw); err != nil {
		t.Fatal(err)
	}
	if _, ok := raw["agent"]; ok {
		t.Fatalf("new workspace carries an agent key: %s", raw["agent"])
	}
	var agents []json.RawMessage
	if err := json.Unmarshal(raw["agents"], &agents); err != nil || len(agents) != 0 {
		t.Fatalf("agents = %s (%v), want []", raw["agents"], err)
	}
	var wk store.Workspace
	_ = json.Unmarshal(raw["id"], new(string))
	if err := json.Unmarshal(raw["id"], &wk.ID); err != nil || wk.ID == "" {
		t.Fatalf("id missing: %s", raw["id"])
	}

	// Idempotent re-add: same workspace, still no agent created.
	again := postJSON(t, ts, "/api/workspaces", map[string]string{"name": "Empty again", "path": proj})
	if again.StatusCode != http.StatusCreated {
		t.Fatalf("re-add = %d", again.StatusCode)
	}
	var wk2 workspaceView
	_ = json.NewDecoder(again.Body).Decode(&wk2)
	if wk2.ID != wk.ID || len(wk2.Agents) != 0 || wk2.Agent != nil {
		t.Fatalf("re-add = %+v", wk2)
	}
}

func TestEmptyWorkspaceNeedsAnAgent(t *testing.T) {
	ts := newTestServer(t, "cat")
	wk := postJSON(t, ts, "/api/workspaces", map[string]string{"name": "Bare", "path": t.TempDir()})
	var wsv workspaceView
	_ = json.NewDecoder(wk.Body).Decode(&wsv)

	for _, path := range []string{
		"/api/workspaces/" + wsv.ID + "/open",
		"/api/workspaces/" + wsv.ID + "/close",
		"/api/workspaces/" + wsv.ID + "/sessions/new",
	} {
		res := postJSON(t, ts, path, map[string]string{})
		if res.StatusCode != http.StatusConflict {
			t.Fatalf("%s = %d, want 409", path, res.StatusCode)
		}
	}
	list := do(t, ts.Client(), mustGet(t, ts.URL+"/api/workspaces/"+wsv.ID+"/sessions"))
	if list.StatusCode != http.StatusConflict {
		t.Fatalf("sessions = %d, want 409", list.StatusCode)
	}
}
