package server

import (
	"encoding/json"
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

// ADR-0039 decision table — two agents sharing one cwd (the precondition
// behind the bug), plus a session written by hand with no owner at all
// (what a Terminal running bare `pi` looks like from pi's cwd-bucket):
//
//	session                    | owner  | A's picker | B's picker | manage view
//	---------------------------+--------+------------+------------+-------------
//	resumed onto A             | A      | yes        | no         | inUseBy A
//	resumed onto B             | B      | no         | yes        | inUseBy B
//	written by hand, no resume | nobody | no         | no         | no inUseBy
func TestListSessionsScopedPerAgent(t *testing.T) {
	ts, _, home := cleanupServer(t)
	proj := filepath.Join(home, "p")
	if err := os.MkdirAll(proj, 0o755); err != nil {
		t.Fatal(err)
	}
	wsv := addWorkspaceWithAgent(t, ts, "App", proj)
	agentA := wsv.Agent.ID

	bRes := postJSON(t, ts, "/api/workspaces/"+wsv.ID+"/agents", map[string]string{"name": "second"})
	if bRes.StatusCode != http.StatusCreated {
		t.Fatalf("add second agent = %d", bRes.StatusCode)
	}
	var agentBView agentView
	if err := json.NewDecoder(bRes.Body).Decode(&agentBView); err != nil {
		t.Fatalf("decode second agent: %v", err)
	}
	agentB := agentBView.ID

	pathA := writeManageSession(t, proj, "a.jsonl", 0)
	pathB := writeManageSession(t, proj, "b.jsonl", 0)
	pathNoOwner := writeManageSession(t, proj, "noowner.jsonl", 0)

	if res := postJSON(t, ts, "/api/workspaces/"+wsv.ID+"/sessions/resume", map[string]string{"path": pathA}); res.StatusCode != http.StatusOK {
		t.Fatalf("resume A = %d", res.StatusCode)
	}
	if res := postJSON(t, ts, "/api/workspaces/"+wsv.ID+"/sessions/resume?agent="+agentB, map[string]string{"path": pathB}); res.StatusCode != http.StatusOK {
		t.Fatalf("resume B = %d", res.StatusCode)
	}

	listFor := func(agentID string) []string {
		res := do(t, ts.Client(), mustGet(t, ts.URL+"/api/workspaces/"+wsv.ID+"/sessions?agent="+agentID))
		if res.StatusCode != http.StatusOK {
			t.Fatalf("list %s = %d", agentID, res.StatusCode)
		}
		var body struct {
			Sessions []struct {
				Path string `json:"path"`
			} `json:"sessions"`
		}
		if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
			t.Fatalf("decode list: %v", err)
		}
		paths := make([]string, len(body.Sessions))
		for i, s := range body.Sessions {
			paths[i] = s.Path
		}
		return paths
	}
	assertOnly := func(t *testing.T, got []string, want string) {
		t.Helper()
		if len(got) != 1 || got[0] != want {
			t.Fatalf("got %v, want only [%v]", got, want)
		}
	}

	assertOnly(t, listFor(agentA), pathA)
	assertOnly(t, listFor(agentB), pathB)

	// The machine-wide housekeeping view is unaffected by the ownership
	// filter — it still surfaces every session, including the unowned
	// one, for cleanup purposes (session_manage.go, sessionUseBy).
	res := do(t, ts.Client(), mustGet(t, ts.URL+"/api/workspaces/"+wsv.ID+"/sessions/manage"))
	if res.StatusCode != http.StatusOK {
		t.Fatalf("manage = %d", res.StatusCode)
	}
	var manage struct {
		Sessions []struct {
			Path    string `json:"path"`
			InUseBy any    `json:"inUseBy"`
		} `json:"sessions"`
	}
	if err := json.NewDecoder(res.Body).Decode(&manage); err != nil {
		t.Fatalf("decode manage: %v", err)
	}
	if len(manage.Sessions) != 3 {
		t.Fatalf("manage sessions = %d, want 3", len(manage.Sessions))
	}
	byPath := map[string]any{}
	for _, s := range manage.Sessions {
		byPath[s.Path] = s.InUseBy
	}
	if byPath[pathA] == nil {
		t.Error("A's session missing inUseBy in manage view")
	}
	if byPath[pathB] == nil {
		t.Error("B's session missing inUseBy in manage view")
	}
	if v, ok := byPath[pathNoOwner]; !ok {
		t.Error("unowned session missing entirely from manage view")
	} else if v != nil {
		t.Errorf("unowned session reports inUseBy = %v", v)
	}
}

// A brand-new agent that has never chatted has no session_path and no
// history yet — the picker must come back empty, not error, and not
// fall back to showing the newest file in the shared cwd bucket (the
// pre-ADR-0039 bug).
func TestListSessionsEmptyForFreshAgent(t *testing.T) {
	ts, _, home := cleanupServer(t)
	proj := filepath.Join(home, "p")
	if err := os.MkdirAll(proj, 0o755); err != nil {
		t.Fatal(err)
	}
	wsv := addWorkspaceWithAgent(t, ts, "App", proj)

	// A session nobody has resumed onto this (or any) agent yet.
	_ = writeManageSession(t, proj, "stray.jsonl", 0)

	res := do(t, ts.Client(), mustGet(t, ts.URL+"/api/workspaces/"+wsv.ID+"/sessions?agent="+wsv.Agent.ID))
	if res.StatusCode != http.StatusOK {
		t.Fatalf("list = %d", res.StatusCode)
	}
	var body struct {
		Sessions []any  `json:"sessions"`
		Current  string `json:"current"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Sessions) != 0 || body.Current != "" {
		t.Fatalf("fresh agent picker = %+v, want empty", body)
	}
}

// TestListSessionsResolvesFreshSessionPath exercises the lazy backfill:
// a session historized only by id — exactly what NewPendingAgentSession
// writes at spawn time, before pi has created any file — gets its path
// resolved, and agents.session_path itself backfilled (not just the
// agent_sessions row), the first time the picker is read. Without this,
// an actively-used fresh session's agents.session_path stays NULL
// forever (nothing else ever sets it for an agent that's never been
// explicitly resumed), so the manage view's inUseBy delete-guard would
// treat it as an orphan. Caught live against a real pi spawn during
// dogfood verification (docs/handoff.md), not from reading the code.
func TestListSessionsResolvesFreshSessionPath(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	st, err := store.Open(filepath.Join(root, "data", "picode.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	ts := httptest.NewServer(New("127.0.0.1:0", Deps{
		Store: st, Tmux: tmux.New(), Runtime: rpc.NewRuntime("cat", st, nil), AgentCmd: "cat",
	}).Handler)
	t.Cleanup(ts.Close)

	proj := filepath.Join(home, "p")
	if err := os.MkdirAll(proj, 0o755); err != nil {
		t.Fatal(err)
	}
	wsv := addWorkspaceWithAgent(t, ts, "App", proj)
	agentID := wsv.Agent.ID

	// Simulate what rpc.Runtime.Start / Deps.spawnFlags do at spawn time
	// for a fresh agent: mint a session id before any file exists.
	sid := st.NewPendingAgentSession(agentID)
	if sid == "" {
		t.Fatal("NewPendingAgentSession returned empty id")
	}
	dir := session.Dir(proj)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "fresh.jsonl")
	body := `{"type":"session","id":"` + sid + `","cwd":"` + proj + `"}` + "\n" +
		`{"type":"message","message":{"role":"user","content":[{"type":"text","text":"hi"}]}}` + "\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	res := do(t, ts.Client(), mustGet(t, ts.URL+"/api/workspaces/"+wsv.ID+"/sessions?agent="+agentID))
	if res.StatusCode != http.StatusOK {
		t.Fatalf("list = %d", res.StatusCode)
	}
	var listBody struct {
		Current  string `json:"current"`
		Sessions []struct {
			Path string `json:"path"`
		} `json:"sessions"`
	}
	if err := json.NewDecoder(res.Body).Decode(&listBody); err != nil {
		t.Fatal(err)
	}
	if listBody.Current != path || len(listBody.Sessions) != 1 {
		t.Fatalf("list = %+v, want current=%s and 1 session", listBody, path)
	}

	// The backfill must reach agents.session_path, not just the history
	// table — that's what the manage view's inUseBy guard reads.
	mres := do(t, ts.Client(), mustGet(t, ts.URL+"/api/workspaces/"+wsv.ID+"/sessions/manage"))
	if mres.StatusCode != http.StatusOK {
		t.Fatalf("manage = %d", mres.StatusCode)
	}
	var manage struct {
		Sessions []struct {
			Path    string `json:"path"`
			InUseBy any    `json:"inUseBy"`
		} `json:"sessions"`
	}
	if err := json.NewDecoder(mres.Body).Decode(&manage); err != nil {
		t.Fatal(err)
	}
	if len(manage.Sessions) != 1 || manage.Sessions[0].InUseBy == nil {
		t.Fatalf("manage = %+v, want 1 session with inUseBy set", manage.Sessions)
	}
}
