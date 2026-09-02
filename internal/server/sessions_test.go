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

// TestSpawnFlagsIncludesSessionDir locks in ADR-0040 at the point
// interactive/tmux spawns actually get their argv: --session-dir must be
// present whether the agent is starting fresh or resuming — this is what
// makes pi's own in-TUI "Resume Session" picker agent-scoped, not just
// PiCode's chat picker (ADR-0039).
func TestSpawnFlagsIncludesSessionDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	st, err := store.Open(filepath.Join(t.TempDir(), "picode.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	deps := Deps{Store: st}

	_, agent, err := storeWorkspaceWithAgent(st, "App", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	assertHasSessionDir := func(t *testing.T, flags []string, agentID string) {
		t.Helper()
		want := session.AgentDir(agentID)
		for i, f := range flags {
			if f == "--session-dir" {
				if i+1 >= len(flags) || flags[i+1] != want {
					t.Fatalf("--session-dir value in %v, want %q", flags, want)
				}
				return
			}
		}
		t.Fatalf("flags = %v, missing --session-dir", flags)
	}

	// Fresh: no SessionPath yet.
	assertHasSessionDir(t, deps.spawnFlags(agent), agent.ID)

	// Resuming: SessionPath already set.
	p := filepath.Join(t.TempDir(), "x.jsonl")
	resumed, err := st.UpdateAgent(agent.ID, store.AgentPatch{SessionPath: &p})
	if err != nil {
		t.Fatal(err)
	}
	assertHasSessionDir(t, deps.spawnFlags(resumed), resumed.ID)
}

// TestListSessionsSeesPrivateDirSessions extends the ADR-0039 picker
// table to where ADR-0040 actually puts fresh sessions — the agent's
// private dir. The picker kept listing only the cwd bucket, so an
// actively-used fresh agent read as "No sessions yet", its pending
// session never resolved, and every run-mode switch minted a competitor.
//
//	session (in A's private dir) | A's picker        | B's picker
//	-----------------------------+-------------------+------------
//	owned by A (pending id)      | yes, current=path | no
func TestListSessionsSeesPrivateDirSessions(t *testing.T) {
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
	agentA := wsv.Agent.ID

	bRes := postJSON(t, ts, "/api/workspaces/"+wsv.ID+"/agents", map[string]string{"name": "second"})
	if bRes.StatusCode != http.StatusCreated {
		t.Fatalf("add second agent = %d", bRes.StatusCode)
	}
	var agentBView agentView
	if err := json.NewDecoder(bRes.Body).Decode(&agentBView); err != nil {
		t.Fatalf("decode second agent: %v", err)
	}

	// Agent A's managed run: pending id minted at spawn, pi wrote the
	// file into A's private dir (--session-dir, ADR-0040).
	sid := st.NewPendingAgentSession(agentA)
	if sid == "" {
		t.Fatal("NewPendingAgentSession returned empty id")
	}
	dir := session.AgentDir(agentA)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "fresh.jsonl")
	body := `{"type":"session","id":"` + sid + `","cwd":"` + proj + `"}` + "\n" +
		`{"type":"message","message":{"role":"user","content":[{"type":"text","text":"hi"}]}}` + "\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	listFor := func(agentID string) (current string, n int) {
		t.Helper()
		res := do(t, ts.Client(), mustGet(t, ts.URL+"/api/workspaces/"+wsv.ID+"/sessions?agent="+agentID))
		if res.StatusCode != http.StatusOK {
			t.Fatalf("list %s = %d", agentID, res.StatusCode)
		}
		var out struct {
			Current  string `json:"current"`
			Sessions []any  `json:"sessions"`
		}
		if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
			t.Fatalf("decode: %v", err)
		}
		return out.Current, len(out.Sessions)
	}

	cur, n := listFor(agentA)
	if cur != path || n != 1 {
		t.Fatalf("A's picker = current %q, %d sessions; want %q, 1", cur, n, path)
	}
	// The read backfills agents.session_path — the manage view's
	// inUseBy guard and the next spawn both depend on it.
	a, err := st.GetAgent(agentA)
	if err != nil {
		t.Fatal(err)
	}
	if a.SessionPath == nil || *a.SessionPath != path {
		t.Fatalf("session_path = %v, want %q", a.SessionPath, path)
	}
	// Isolation holds across private dirs: B sees nothing of A's.
	if cur, n = listFor(agentBView.ID); cur != "" || n != 0 {
		t.Fatalf("B's picker = current %q, %d sessions; want empty", cur, n)
	}
}

// TestSpawnFlagsResumesPendingSession is the ADR-0053 decision table at
// the interactive spawn chokepoint — what a run-mode switch (open the
// TUI, /login, restart) spawns for an agent whose session_path is empty:
//
//	pending id | file in private dir | argv                       | session_path after
//	-----------+---------------------+----------------------------+---------------------
//	yes        | yes                 | --session <file>, no mint  | backfilled to file
//	yes        | no                  | --session-id <fresh mint>  | still empty
//	no         | —                   | --session-id <fresh mint>  | still empty
//	(resuming agents are unchanged: CLIFlags pins --session, covered by
//	TestSpawnFlagsIncludesSessionDir)
func TestSpawnFlagsResumesPendingSession(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	st, err := store.Open(filepath.Join(t.TempDir(), "picode.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	deps := Deps{Store: st}
	ws, _, err := storeWorkspaceWithAgent(st, "App", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	addAgent := func(name string) store.Agent {
		t.Helper()
		a, err := st.AddAgent(ws.ID, name, t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		return a
	}
	writeSessionFile := func(agentID, sid string) string {
		t.Helper()
		dir := session.AgentDir(agentID)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		p := filepath.Join(dir, "s.jsonl")
		body := `{"type":"session","id":"` + sid + `"}` + "\n"
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	hasFlag := func(flags []string, name string) (string, bool) {
		for i, f := range flags {
			if f == name && i+1 < len(flags) {
				return flags[i+1], true
			}
		}
		return "", false
	}

	// Pending id + file: the spawn resumes that file instead of minting.
	adopted := addAgent("adopter")
	sid := st.NewPendingAgentSession(adopted.ID)
	want := writeSessionFile(adopted.ID, sid)
	flags := deps.spawnFlags(adopted)
	if v, ok := hasFlag(flags, "--session"); !ok || v != want {
		t.Fatalf("flags = %v, want --session %q", flags, want)
	}
	if _, ok := hasFlag(flags, "--session-id"); ok {
		t.Fatalf("flags = %v, resuming spawn must not also mint --session-id", flags)
	}
	a, err := st.GetAgent(adopted.ID)
	if err != nil {
		t.Fatal(err)
	}
	if a.SessionPath == nil || *a.SessionPath != want {
		t.Fatalf("session_path = %v, want %q", a.SessionPath, want)
	}

	// Pending id but no file yet (pi never wrote one): mint a fresh id.
	ghost := addAgent("ghost")
	if sid := st.NewPendingAgentSession(ghost.ID); sid == "" {
		t.Fatal("mint failed")
	}
	flags = deps.spawnFlags(ghost)
	if _, ok := hasFlag(flags, "--session-id"); !ok {
		t.Fatalf("flags = %v, want a fresh --session-id mint", flags)
	}
	if _, ok := hasFlag(flags, "--session"); ok {
		t.Fatalf("flags = %v, --session must stay unset without a file", flags)
	}

	// Nothing pending at all: same as fresh.
	plain := addAgent("plain")
	flags = deps.spawnFlags(plain)
	if _, ok := hasFlag(flags, "--session-id"); !ok {
		t.Fatalf("flags = %v, want a fresh --session-id mint", flags)
	}
}
