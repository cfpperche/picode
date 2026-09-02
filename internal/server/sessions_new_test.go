package server

// The composer's "+ New" (POST /sessions/new). An explicit new session
// must stay fresh end to end: the pointer cleared, the ADR-0053 adoption
// window sealed so the next spawn mints a new --session-id instead of
// re-adopting the thread just abandoned, and the picker listing the old
// thread without silently re-selecting it. It must also work for free
// agents, whose workspace (ws_free) is a sentinel with no real path —
// the restart used to spawn there and die with "no such file or
// directory", which the UI showed as "That folder doesn't exist".

import (
	"context"
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

func writeOwnSession(t *testing.T, agentID, sid string) string {
	t.Helper()
	dir := session.AgentDir(agentID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, sid+".jsonl")
	body := `{"type":"session","version":3,"id":"` + sid + `","cwd":"x"}` + "\n" +
		`{"type":"message","message":{"role":"user","content":[{"type":"text","text":"hi"}]}}` + "\n"
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func flagValue(flags []string, name string) string {
	for i, f := range flags {
		if f == name && i+1 < len(flags) {
			return flags[i+1]
		}
	}
	return ""
}

// Decision table for the + New flow on a workspace agent:
//
//	resume            → list current = the file (pre-condition)
//	New (stopped)     → 200; list current = "" (never resurrect); old session still listed, pointer cleared
//	next spawn        → mints a fresh --session-id, never the abandoned one
func TestNewSessionStaysFresh(t *testing.T) {
	ts, _, home := cleanupServer(t)
	proj := filepath.Join(home, "proj")
	if err := os.MkdirAll(proj, 0o755); err != nil {
		t.Fatal(err)
	}
	wsv := addWorkspaceWithAgent(t, ts, "App", proj)
	agent := wsv.Agents[0]
	wbase := "/api/workspaces/" + wsv.ID
	q := "?agent=" + agent.ID

	sid := "11111111-2222-4333-8444-555555555555"
	p := writeOwnSession(t, agent.ID, sid)

	list := func() map[string]any {
		res := do(t, ts.Client(), mustGet(t, ts.URL+wbase+"/sessions"+q))
		if res.StatusCode != http.StatusOK {
			t.Fatalf("list = %d", res.StatusCode)
		}
		var body map[string]any
		_ = json.NewDecoder(res.Body).Decode(&body)
		return body
	}

	// Resume makes the file the agent's current session.
	if res := postJSON(t, ts, wbase+"/sessions/resume"+q, map[string]string{"path": p}); res.StatusCode != http.StatusOK {
		t.Fatalf("resume = %d", res.StatusCode)
	}
	if got := list()["current"]; got != p {
		t.Fatalf("current after resume = %v, want %q", got, p)
	}

	// New: pointer cleared, the old thread listed but NOT re-selected.
	if res := postJSON(t, ts, wbase+"/sessions/new"+q, map[string]string{}); res.StatusCode != http.StatusOK {
		t.Fatalf("new = %d", res.StatusCode)
	}
	view := list()
	if got := view["current"]; got != "" {
		t.Fatalf("current after New = %v, want \"\" (the picker must not resurrect the abandoned thread)", got)
	}
	found := false
	for _, s := range view["sessions"].([]any) {
		if s.(map[string]any)["path"] == p {
			found = true
		}
	}
	if !found {
		t.Fatalf("old session vanished from the picker: %v", view["sessions"])
	}
	var wvs []workspaceView
	res := do(t, ts.Client(), mustGet(t, ts.URL+"/api/workspaces"))
	if err := json.NewDecoder(res.Body).Decode(&wvs); err != nil {
		t.Fatal(err)
	}
	for _, w := range wvs {
		if w.ID != wsv.ID {
			continue
		}
		if a := w.Agents[0]; a.SessionPath != nil && *a.SessionPath != "" {
			t.Fatalf("session_path after New = %q, want empty", *a.SessionPath)
		}
	}
}

// (helper store for the spawn assertions — kept next to its only user)
func newTestStore(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "picode.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

func mustAgent(t *testing.T, st *store.Store, id string) store.Agent {
	t.Helper()
	a, err := st.GetAgent(id)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

// Adoption vs New, side by side: with a lost pointer and no New, the
// spawn adopts the pending file (ADR-0053 — chat → TUI → chat stays on
// one thread). After the explicit-New flow (clear + seal), the same
// spawn mints a fresh --session-id instead.
func TestNewSessionNextSpawnMintsFreshID(t *testing.T) {
	old := session.TestRoot
	session.TestRoot = t.TempDir()
	t.Cleanup(func() { session.TestRoot = old })

	st := newTestStore(t)
	deps := Deps{Store: st, AgentCmd: "cat"}

	agent, err := st.AddAgent(store.FreeWorkspaceID, "a", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	sid := st.NewPendingAgentSession(agent.ID)
	if sid == "" {
		t.Fatal("mint failed")
	}
	p := writeOwnSession(t, agent.ID, sid)

	// Lost pointer, no New: adoption wins.
	flags := deps.spawnFlags(mustAgent(t, st, agent.ID))
	if got := flagValue(flags, "--session"); got != p {
		t.Fatalf("pre-seal flags = %v, want adoption of %q", flags, p)
	}

	// The explicit-New flow: clear, seal, spawn.
	empty := ""
	if _, err := st.UpdateAgent(agent.ID, store.AgentPatch{SessionPath: &empty}); err != nil {
		t.Fatal(err)
	}
	st.SealPendingAgentSessions(agent.ID)
	flags = deps.spawnFlags(mustAgent(t, st, agent.ID))
	minted := flagValue(flags, "--session-id")
	if minted == "" || minted == sid {
		t.Fatalf("post-seal flags = %v, want a fresh --session-id (not %q)", flags, sid)
	}
	if got := flagValue(flags, "--session"); got != "" {
		t.Fatalf("post-seal flags = %v, want no --session resume", got)
	}
}

// newFreeAgentServer is cleanupServer with a flag-tolerant fake pi: the
// tmux rows here spawn real sessions, and plain `cat` dies on
// "unrecognized option --session-id" the moment the session opens.
func newFreeAgentServer(t *testing.T) *httptest.Server {
	t.Helper()
	root := t.TempDir()
	home := filepath.Join(root, "home")
	dataDir := filepath.Join(root, "data")
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
	cmd := fakeBlockingAgentCmd(t)
	ts := httptest.NewServer(New("127.0.0.1:0", Deps{
		Store: st, Tmux: tmux.New(), Runtime: rpc.NewRuntime(cmd, st, nil), AgentCmd: cmd, DataDir: dataDir,
	}).Handler)
	t.Cleanup(ts.Close)
	return ts
}

// A free agent (workspace ws_free) with its TUI open: New kills the tmux
// session and respawns in the agent's own work dir. The old code spawned
// in the sentinel workspace path and failed with "no such file or
// directory" — killing the TUI and toasting "That folder doesn't exist".
func TestNewSessionFreeAgentTUIRestart(t *testing.T) {
	ts := newFreeAgentServer(t)
	work := filepath.Join(os.TempDir(), "picode-test-work", t.Name())
	if err := os.MkdirAll(work, 0o755); err != nil {
		t.Fatal(err)
	}
	res := postJSON(t, ts, "/api/agents", map[string]string{"name": "agente-auto", "path": work})
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("add free agent = %d", res.StatusCode)
	}
	var av agentView
	if err := json.NewDecoder(res.Body).Decode(&av); err != nil {
		t.Fatal(err)
	}
	tm := tmux.New()
	t.Cleanup(func() { _ = tm.KillSession(context.Background(), tmux.SessionName(av.ID)) })

	// Open the agent's TUI (interactive mode), then New.
	if r := postJSON(t, ts, "/api/agents/"+av.ID+"/open", map[string]string{}); r.StatusCode != http.StatusCreated {
		t.Fatalf("open TUI = %d", r.StatusCode)
	}
	if has, err := tm.HasSession(context.Background(), tmux.SessionName(av.ID)); err != nil || !has {
		t.Fatalf("tmux session after open: has=%v err=%v", has, err)
	}
	if r := postJSON(t, ts, "/api/workspaces/"+store.FreeWorkspaceID+"/sessions/new?agent="+av.ID, map[string]string{}); r.StatusCode != http.StatusOK {
		t.Fatalf("new on a free agent with a TUI open = %d", r.StatusCode)
	}
	if has, err := tm.HasSession(context.Background(), tmux.SessionName(av.ID)); err != nil || !has {
		t.Fatalf("tmux session after New: has=%v err=%v", has, err)
	}
}

// Same story in chat mode: the managed restart must come up in the
// agent's own work dir, not the sentinel path — cmd.Start chdirs there,
// so the old wk.Path spawn failed the POST with "no such file".
func TestNewSessionFreeAgentChatRestart(t *testing.T) {
	ts := newFreeAgentServer(t)
	work := filepath.Join(os.TempDir(), "picode-test-work", t.Name())
	if err := os.MkdirAll(work, 0o755); err != nil {
		t.Fatal(err)
	}
	res := postJSON(t, ts, "/api/agents", map[string]string{"name": "agente-auto", "path": work})
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("add free agent = %d", res.StatusCode)
	}
	var av agentView
	if err := json.NewDecoder(res.Body).Decode(&av); err != nil {
		t.Fatal(err)
	}
	if r := postJSON(t, ts, "/api/agents/"+av.ID+"/managed/start", map[string]string{}); r.StatusCode != http.StatusCreated {
		t.Fatalf("managed start = %d", r.StatusCode)
	}
	if r := postJSON(t, ts, "/api/workspaces/"+store.FreeWorkspaceID+"/sessions/new?agent="+av.ID, map[string]string{}); r.StatusCode != http.StatusOK {
		t.Fatalf("new on a running free agent = %d", r.StatusCode)
	}
}
