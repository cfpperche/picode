package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/cfpperche/picode/internal/rpc"
	"github.com/cfpperche/picode/internal/session"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

func cleanupServer(t *testing.T) (ts *httptest.Server, dataDir, home string) {
	t.Helper()
	root := t.TempDir()
	home = filepath.Join(root, "home")
	dataDir = filepath.Join(root, "data")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	st, err := store.Open(filepath.Join(dataDir, "picode.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	ts = httptest.NewServer(New("127.0.0.1:0", Deps{
		Store:    st,
		Tmux:     tmux.New(),
		Runtime:  rpc.NewRuntime("cat", st, nil),
		AgentCmd: "cat",
		DataDir:  dataDir,
	}).Handler)
	t.Cleanup(ts.Close)
	return ts, dataDir, home
}

func writeSession(t *testing.T, cwd string) string {
	t.Helper()
	dir := session.Dir(cwd)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, "s.jsonl")
	if err := os.WriteFile(p, []byte("{\"type\":\"session\"}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func postJSON(t *testing.T, ts *httptest.Server, path string, body any) *http.Response {
	t.Helper()
	raw, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, ts.URL+path, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	return do(t, ts.Client(), req)
}

func TestCleanupPreviewAndPurgeSessions(t *testing.T) {
	ts, _, _ := cleanupServer(t)
	proj := t.TempDir()
	res := postJSON(t, ts, "/api/workspaces", map[string]string{"name": "App", "path": proj})
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("add = %d", res.StatusCode)
	}
	var wk workspaceView
	if err := json.NewDecoder(res.Body).Decode(&wk); err != nil {
		t.Fatal(err)
	}
	sessDir := writeSession(t, proj)

	got := do(t, ts.Client(), mustGet(t, ts.URL+"/api/workspaces/"+wk.ID+"/cleanup"))
	var preview cleanupPreview
	if err := json.NewDecoder(got.Body).Decode(&preview); err != nil {
		t.Fatal(err)
	}
	if !preview.LastOccupant || preview.Sessions != 1 || preview.CanPurgeWork {
		t.Fatalf("preview = %+v", preview)
	}

	// Without flag, sessions stay and the project folder stays.
	req, _ := http.NewRequest(http.MethodDelete, ts.URL+"/api/workspaces/"+wk.ID, nil)
	del := do(t, ts.Client(), req)
	if del.StatusCode != http.StatusNoContent {
		t.Fatalf("delete = %d", del.StatusCode)
	}
	if _, err := os.Stat(sessDir); err != nil {
		t.Fatalf("sessions removed without flag: %v", err)
	}
	if _, err := os.Stat(proj); err != nil {
		t.Fatalf("project folder touched: %v", err)
	}
}

func TestCleanupPurgeSessionsOnLastOccupant(t *testing.T) {
	ts, _, _ := cleanupServer(t)
	proj := t.TempDir()
	res := postJSON(t, ts, "/api/workspaces", map[string]string{"name": "App", "path": proj})
	var wk workspaceView
	_ = json.NewDecoder(res.Body).Decode(&wk)
	sessDir := writeSession(t, proj)

	req, _ := http.NewRequest(http.MethodDelete, ts.URL+"/api/workspaces/"+wk.ID+"?sessions=1", nil)
	del := do(t, ts.Client(), req)
	if del.StatusCode != http.StatusNoContent {
		t.Fatalf("delete = %d", del.StatusCode)
	}
	if _, err := os.Stat(sessDir); !os.IsNotExist(err) {
		t.Fatalf("sessions should be gone: %v", err)
	}
	if _, err := os.Stat(proj); err != nil {
		t.Fatalf("project folder must stay: %v", err)
	}
}

func TestCleanupKeepsSharedSessions(t *testing.T) {
	ts, _, _ := cleanupServer(t)
	proj := t.TempDir()
	res := postJSON(t, ts, "/api/workspaces", map[string]string{"name": "App", "path": proj})
	var wk workspaceView
	_ = json.NewDecoder(res.Body).Decode(&wk)
	sessDir := writeSession(t, proj)

	// A free agent pointed at the same folder still occupies it.
	freeDir := proj
	res2 := postJSON(t, ts, "/api/agents", map[string]string{"name": "scratch", "path": freeDir})
	if res2.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(res2.Body)
		t.Fatalf("free add = %d %s", res2.StatusCode, body)
	}
	var free agentView
	_ = json.NewDecoder(res2.Body).Decode(&free)

	got := do(t, ts.Client(), mustGet(t, ts.URL+"/api/workspaces/"+wk.ID+"/cleanup"))
	var preview cleanupPreview
	_ = json.NewDecoder(got.Body).Decode(&preview)
	if preview.LastOccupant {
		t.Fatalf("workspace is not last occupant: %+v", preview)
	}

	req, _ := http.NewRequest(http.MethodDelete, ts.URL+"/api/workspaces/"+wk.ID+"?sessions=1", nil)
	if del := do(t, ts.Client(), req); del.StatusCode != http.StatusNoContent {
		t.Fatalf("delete = %d", del.StatusCode)
	}
	if _, err := os.Stat(sessDir); err != nil {
		t.Fatalf("shared sessions deleted: %v", err)
	}
}

func TestCleanupPurgeOwnedWorkFolder(t *testing.T) {
	ts, dataDir, _ := cleanupServer(t)
	res := postJSON(t, ts, "/api/agents", map[string]string{"name": "scratch"})
	if res.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("add free = %d %s", res.StatusCode, body)
	}
	var ag agentView
	_ = json.NewDecoder(res.Body).Decode(&ag)
	if ag.WorkPath == nil {
		t.Fatal("expected work path")
	}
	work := *ag.WorkPath
	if !ownedByWork(filepath.Join(dataDir, "work"), work) {
		t.Fatalf("work path %s not under dataDir/work", work)
	}
	if err := os.WriteFile(filepath.Join(work, "note.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	writeSession(t, work)

	got := do(t, ts.Client(), mustGet(t, ts.URL+"/api/agents/"+ag.ID+"/cleanup"))
	var preview cleanupPreview
	_ = json.NewDecoder(got.Body).Decode(&preview)
	if !preview.LastOccupant || !preview.CanPurgeWork || preview.Sessions != 1 {
		t.Fatalf("preview = %+v", preview)
	}

	req, _ := http.NewRequest(http.MethodDelete, ts.URL+"/api/agents/"+ag.ID+"?sessions=1&work=1", nil)
	if del := do(t, ts.Client(), req); del.StatusCode != http.StatusNoContent {
		t.Fatalf("delete = %d", del.StatusCode)
	}
	if _, err := os.Stat(work); !os.IsNotExist(err) {
		t.Fatalf("work folder still there: %v", err)
	}
	if _, err := os.Stat(session.Dir(work)); !os.IsNotExist(err) {
		t.Fatalf("sessions still there: %v", err)
	}
}

func TestCleanupRefusesWorkOutsideOwnedRoot(t *testing.T) {
	ts, _, _ := cleanupServer(t)
	outside := t.TempDir()
	res := postJSON(t, ts, "/api/agents", map[string]string{"name": "ext", "path": outside})
	var ag agentView
	_ = json.NewDecoder(res.Body).Decode(&ag)

	got := do(t, ts.Client(), mustGet(t, ts.URL+"/api/agents/"+ag.ID+"/cleanup"))
	var preview cleanupPreview
	_ = json.NewDecoder(got.Body).Decode(&preview)
	if preview.CanPurgeWork {
		t.Fatalf("must not offer to delete %s: %+v", outside, preview)
	}
	req, _ := http.NewRequest(http.MethodDelete, ts.URL+"/api/agents/"+ag.ID+"?work=1", nil)
	if del := do(t, ts.Client(), req); del.StatusCode != http.StatusNoContent {
		t.Fatalf("delete = %d", del.StatusCode)
	}
	if _, err := os.Stat(outside); err != nil {
		t.Fatalf("outside folder deleted: %v", err)
	}
}

func TestCleanupRejectsFreeWorkspaceDelete(t *testing.T) {
	ts, _, _ := cleanupServer(t)
	req, _ := http.NewRequest(http.MethodDelete, ts.URL+"/api/workspaces/"+store.FreeWorkspaceID, nil)
	res := do(t, ts.Client(), req)
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d", res.StatusCode)
	}
}

func decodeAgent(t *testing.T, res *http.Response) agentView {
	t.Helper()
	var a agentView
	if err := json.NewDecoder(res.Body).Decode(&a); err != nil {
		t.Fatal(err)
	}
	return a
}

func decodeWorkspace(t *testing.T, res *http.Response) workspaceView {
	t.Helper()
	var w workspaceView
	if err := json.NewDecoder(res.Body).Decode(&w); err != nil {
		t.Fatal(err)
	}
	return w
}

func getCleanup(t *testing.T, ts *httptest.Server, path string) cleanupPreview {
	t.Helper()
	res := do(t, ts.Client(), mustGet(t, ts.URL+path))
	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("GET %s = %d %s", path, res.StatusCode, body)
	}
	var p cleanupPreview
	if err := json.NewDecoder(res.Body).Decode(&p); err != nil {
		t.Fatal(err)
	}
	return p
}

func del(t *testing.T, ts *httptest.Server, path string) {
	t.Helper()
	req, _ := http.NewRequest(http.MethodDelete, ts.URL+path, nil)
	res := do(t, ts.Client(), req)
	if res.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("DELETE %s = %d %s", path, res.StatusCode, body)
	}
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func TestCleanupMatrix(t *testing.T) {
	ts, dataDir, _ := cleanupServer(t)

	t.Run("sibling workspace agent is not last occupant", func(t *testing.T) {
		proj := t.TempDir()
		wk := decodeWorkspace(t, postJSON(t, ts, "/api/workspaces", map[string]string{"name": "Sibs", "path": proj}))
		sib := decodeAgent(t, postJSON(t, ts, "/api/workspaces/"+wk.ID+"/agents", map[string]string{"name": "review"}))
		sess := writeSession(t, proj)
		p := getCleanup(t, ts, "/api/agents/"+sib.ID+"/cleanup")
		if p.LastOccupant || p.CanPurgeWork {
			t.Fatalf("preview = %+v", p)
		}
		del(t, ts, "/api/agents/"+sib.ID+"?sessions=1&work=1")
		if !exists(sess) {
			t.Fatal("sibling delete purged shared sessions")
		}
		if !exists(proj) {
			t.Fatal("project folder gone")
		}
		del(t, ts, "/api/workspaces/"+wk.ID)
	})

	t.Run("last workspace agent offers sessions not work", func(t *testing.T) {
		proj := t.TempDir()
		wk := decodeWorkspace(t, postJSON(t, ts, "/api/workspaces", map[string]string{"name": "Solo", "path": proj}))
		sess := writeSession(t, proj)
		p := getCleanup(t, ts, "/api/agents/"+wk.Agent.ID+"/cleanup")
		if !p.LastOccupant || p.CanPurgeWork || p.Sessions != 1 {
			t.Fatalf("preview = %+v", p)
		}
		del(t, ts, "/api/agents/"+wk.Agent.ID+"?sessions=1&work=1")
		if exists(sess) {
			t.Fatal("sessions should be gone")
		}
		if !exists(proj) {
			t.Fatal("project folder deleted via work=1")
		}
		del(t, ts, "/api/workspaces/"+wk.ID)
	})

	t.Run("workspace with two agents is last occupant together", func(t *testing.T) {
		proj := t.TempDir()
		wk := decodeWorkspace(t, postJSON(t, ts, "/api/workspaces", map[string]string{"name": "Both", "path": proj}))
		_ = decodeAgent(t, postJSON(t, ts, "/api/workspaces/"+wk.ID+"/agents", map[string]string{"name": "review"}))
		writeSession(t, proj)
		p := getCleanup(t, ts, "/api/workspaces/"+wk.ID+"/cleanup")
		if !p.LastOccupant || p.CanPurgeWork || p.Sessions != 1 {
			t.Fatalf("preview = %+v", p)
		}
		del(t, ts, "/api/workspaces/"+wk.ID+"?sessions=1&work=1")
		if exists(session.Dir(proj)) {
			t.Fatal("sessions should be gone")
		}
		if !exists(proj) {
			t.Fatal("project folder deleted")
		}
	})

	t.Run("free agent sessions=1 keeps work folder", func(t *testing.T) {
		ag := decodeAgent(t, postJSON(t, ts, "/api/agents", map[string]string{"name": "keep-work"}))
		work := *ag.WorkPath
		if !ownedByWork(filepath.Join(dataDir, "work"), work) {
			t.Fatalf("work = %s", work)
		}
		sess := writeSession(t, work)
		del(t, ts, "/api/agents/"+ag.ID+"?sessions=1")
		if exists(sess) {
			t.Fatal("sessions should be gone")
		}
		if !exists(work) {
			t.Fatal("work folder deleted without work=1")
		}
	})

	t.Run("free agent work=1 keeps sessions", func(t *testing.T) {
		ag := decodeAgent(t, postJSON(t, ts, "/api/agents", map[string]string{"name": "keep-sess"}))
		work := *ag.WorkPath
		sess := writeSession(t, work)
		del(t, ts, "/api/agents/"+ag.ID+"?work=1")
		if exists(work) {
			t.Fatal("work folder should be gone")
		}
		if !exists(sess) {
			t.Fatal("sessions deleted without sessions=1")
		}
	})

	t.Run("free agent no flags keeps both", func(t *testing.T) {
		ag := decodeAgent(t, postJSON(t, ts, "/api/agents", map[string]string{"name": "keep-both"}))
		work := *ag.WorkPath
		sess := writeSession(t, work)
		del(t, ts, "/api/agents/"+ag.ID)
		if !exists(work) || !exists(sess) {
			t.Fatal("default delete must not purge")
		}
	})

	t.Run("no sessions preview is empty", func(t *testing.T) {
		ag := decodeAgent(t, postJSON(t, ts, "/api/agents", map[string]string{"name": "empty"}))
		p := getCleanup(t, ts, "/api/agents/"+ag.ID+"/cleanup")
		if p.Sessions != 0 || !p.LastOccupant || !p.CanPurgeWork {
			t.Fatalf("preview = %+v", p)
		}
		del(t, ts, "/api/agents/"+ag.ID)
	})
}

func TestPinDeleteRemovesDir(t *testing.T) {
	ts, dataDir, _ := cleanupServer(t)
	res := postJSON(t, ts, "/api/pins", map[string]any{"title": "Tmp", "tags": []string{}, "body": ""})
	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("create pin = %d %s", res.StatusCode, body)
	}
	var pin store.Pin
	if err := json.NewDecoder(res.Body).Decode(&pin); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(dataDir, "pins", pin.ID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "f"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	del(t, ts, "/api/pins/"+pin.ID)
	if exists(dir) {
		t.Fatal("pin dir left behind")
	}
}

func TestOwnedByWork(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "a")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatal(err)
	}
	if !ownedByWork(root, child) {
		t.Fatal("child should be owned")
	}
	if ownedByWork(root, root) {
		t.Fatal("root itself is not a work folder")
	}
	if ownedByWork(root, t.TempDir()) {
		t.Fatal("sibling must not be owned")
	}
}
