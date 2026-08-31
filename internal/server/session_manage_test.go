package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/session"
)

func postJSONMethod(t *testing.T, ts *httptest.Server, method, path string, body any) *http.Response {
	t.Helper()
	raw, _ := json.Marshal(body)
	req, _ := http.NewRequest(method, ts.URL+path, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	return do(t, ts.Client(), req)
}

func writeManageSession(t *testing.T, cwd, name string, age time.Duration) string {
	t.Helper()
	dir := session.Dir(cwd)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, name)
	body := `{"type":"session","id":"t","cwd":` + strconv.Quote(cwd) + `}` + "\n" +
		`{"type":"message","message":{"role":"user","content":[{"type":"text","text":"hello"}]}}` + "\n"
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if age > 0 {
		then := time.Now().Add(-age)
		if err := os.Chtimes(p, then, then); err != nil {
			t.Fatal(err)
		}
	}
	return p
}

// Decision table for the manage view (A) and the orphan sweep (B):
//
//	session state         | list shows     | delete        | sweep (30d)
//	----------------------+----------------+---------------+------------
//	orphan, fresh         | no inUseBy     | 200           | kept
//	orphan, 40d old       | no inUseBy     | 200           | removed
//	current of an agent   | inUseBy agent  | 409           | kept
//	outside workspace dir | —              | 400           | n/a
func TestSessionManageAndSweep(t *testing.T) {
	ts, _, home := cleanupServer(t)
	proj := filepath.Join(home, "p")
	if err := os.MkdirAll(proj, 0o755); err != nil {
		t.Fatal(err)
	}
	wsv := addWorkspaceWithAgent(t, ts, "App", proj)
	base := "/api/workspaces/" + wsv.ID

	orphan := writeManageSession(t, proj, "orphan.jsonl", 0)
	bound := writeManageSession(t, proj, "bound.jsonl", 0)
	old := writeManageSession(t, proj, "old.jsonl", 40*24*time.Hour)
	_ = postJSON(t, ts, base+"/sessions/resume", map[string]string{"path": bound})

	get := func(path string) (int, map[string]any) {
		res := do(t, ts.Client(), mustGet(t, ts.URL+path))
		var body map[string]any
		_ = json.NewDecoder(res.Body).Decode(&body)
		return res.StatusCode, body
	}
	del := func(url string, payload any) int {
		res := postJSONMethod(t, ts, http.MethodDelete, url, payload)
		return res.StatusCode
	}

	// List: three sessions, one bound, size present, cleanup off.
	code, view := get(base + "/sessions/manage")
	if code != http.StatusOK {
		t.Fatalf("manage = %d", code)
	}
	sessions := view["sessions"].([]any)
	if len(sessions) != 3 {
		t.Fatalf("sessions = %d, want 3", len(sessions))
	}
	byPath := map[string]map[string]any{}
	var totalSize float64
	for _, s := range sessions {
		m := s.(map[string]any)
		byPath[m["path"].(string)] = m
		totalSize += m["size"].(float64)
	}
	if totalSize == 0 || view["totalBytes"].(float64) != totalSize {
		t.Fatalf("sizes not reported: %v / %v", totalSize, view["totalBytes"])
	}
	if v, _ := view["cleanupDays"].(float64); v != 0 {
		t.Fatalf("cleanupDays default = %v, want 0 (off)", v)
	}
	if byPath[bound]["inUseBy"] == nil {
		t.Fatal("bound session has no inUseBy")
	}
	if byPath[orphan]["inUseBy"] != nil {
		t.Fatal("orphan reports inUseBy")
	}

	// Delete: orphan ok, bound 409, outside 400/404, twice 404.
	if c := del(base+"/sessions/manage", map[string]string{"path": orphan}); c != http.StatusOK {
		t.Fatalf("delete orphan = %d", c)
	}
	if _, err := os.Stat(orphan); !os.IsNotExist(err) {
		t.Fatal("orphan file still on disk")
	}
	if c := del(base+"/sessions/manage", map[string]string{"path": bound}); c != http.StatusConflict {
		t.Fatalf("delete bound = %d, want 409", c)
	}
	outside := filepath.Join(home, "elsewhere.jsonl")
	_ = os.WriteFile(outside, []byte("{}\n"), 0o644)
	if c := del(base+"/sessions/manage", map[string]string{"path": outside}); c != http.StatusBadRequest {
		t.Fatalf("delete outside = %d, want 400", c)
	}
	if c := del(base+"/sessions/manage", map[string]string{"path": orphan}); c != http.StatusNotFound {
		t.Fatalf("delete twice = %d, want 404", c)
	}

	// Sweep: 30 days removes only the old orphan; in-use stays.
	res := postJSONMethod(t, ts, http.MethodPut, "/api/session-cleanup", map[string]int{"days": 30})
	var put map[string]any
	_ = json.NewDecoder(res.Body).Decode(&put)
	if res.StatusCode != http.StatusOK || put["removed"].(float64) != 1 {
		t.Fatalf("sweep = %d %v, want removed 1", res.StatusCode, put)
	}
	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Fatal("old orphan survived sweep")
	}
	if _, err := os.Stat(bound); err != nil {
		t.Fatal("sweep deleted an in-use session")
	}
	if c, body := get("/api/session-cleanup"); c != http.StatusOK || body["days"].(float64) != 30 {
		t.Fatalf("cleanup setting read = %d %v", c, body)
	}
	if c := postJSONMethod(t, ts, http.MethodPut, "/api/session-cleanup", map[string]int{"days": -1}).StatusCode; c != http.StatusBadRequest {
		t.Fatalf("negative days = %d, want 400", c)
	}
}

// The bound session must be reported with the agent name, not just id.
func TestManageViewNamesAgent(t *testing.T) {
	ts, _, home := cleanupServer(t)
	proj := filepath.Join(home, "q")
	if err := os.MkdirAll(proj, 0o755); err != nil {
		t.Fatal(err)
	}
	wsv := addWorkspaceWithAgent(t, ts, "Q", proj)
	p := writeManageSession(t, proj, "only.jsonl", 0)
	_ = postJSON(t, ts, "/api/workspaces/"+wsv.ID+"/sessions/resume", map[string]string{"path": p})

	res := do(t, ts.Client(), mustGet(t, ts.URL+"/api/workspaces/"+wsv.ID+"/sessions/manage"))
	var view sessionManageView
	if err := json.NewDecoder(res.Body).Decode(&view); err != nil {
		t.Fatal(err)
	}
	if len(view.Sessions) != 1 || view.Sessions[0].InUseBy == nil {
		t.Fatalf("view = %+v", view.Sessions)
	}
	if strings.TrimSpace(view.Sessions[0].InUseBy.AgentName) == "" {
		t.Fatal("inUseBy has no agent name")
	}
}

// Machine-wide view: sessions from every folder, tagged with the workspace
// they belong to; delete validates against the sessions root, not a folder.
func TestAllSessionsView(t *testing.T) {
	ts, _, home := cleanupServer(t)
	proj := filepath.Join(home, "p")
	if err := os.MkdirAll(proj, 0o755); err != nil {
		t.Fatal(err)
	}
	other := filepath.Join(home, "other")
	if err := os.MkdirAll(other, 0o755); err != nil {
		t.Fatal(err)
	}
	ws := postJSON(t, ts, "/api/workspaces", map[string]string{"name": "App", "path": proj})
	var wsv workspaceView
	if err := json.NewDecoder(ws.Body).Decode(&wsv); err != nil {
		t.Fatal(err)
	}
	inWs := writeManageSession(t, proj, "in-ws.jsonl", 0)
	outside := writeManageSession(t, other, "outside.jsonl", 0)

	res := do(t, ts.Client(), mustGet(t, ts.URL+"/api/sessions/all"))
	if res.StatusCode != http.StatusOK {
		t.Fatalf("all = %d", res.StatusCode)
	}
	var body struct {
		Sessions []struct {
			Path      string `json:"path"`
			Cwd       string `json:"cwd"`
			Workspace string `json:"workspace"`
		} `json:"sessions"`
		TotalBytes int64 `json:"totalBytes"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	var sawInWs, sawOutside bool
	for _, s := range body.Sessions {
		if s.Path == inWs {
			sawInWs = true
			if s.Workspace != "App" {
				t.Fatalf("in-ws tagged %q, want App", s.Workspace)
			}
		}
		if s.Path == outside {
			sawOutside = true
			if s.Workspace != "" {
				t.Fatalf("outside tagged %q, want empty", s.Workspace)
			}
		}
	}
	if !sawInWs || !sawOutside {
		t.Fatalf("all view missed sessions: inWs=%v outside=%v", sawInWs, sawOutside)
	}

	// Machine delete: outside folder ok; path outside the root rejected.
	if c := postJSONMethod(t, ts, http.MethodDelete, "/api/sessions/all", map[string]string{"path": outside}).StatusCode; c != http.StatusOK {
		t.Fatalf("delete outside-folder = %d", c)
	}
	if _, err := os.Stat(outside); !os.IsNotExist(err) {
		t.Fatal("outside session still on disk")
	}
	notRoot := filepath.Join(home, "loose.jsonl")
	_ = os.WriteFile(notRoot, []byte("{}\n"), 0o644)
	if c := postJSONMethod(t, ts, http.MethodDelete, "/api/sessions/all", map[string]string{"path": notRoot}).StatusCode; c != http.StatusBadRequest {
		t.Fatalf("delete non-root = %d, want 400", c)
	}
}
