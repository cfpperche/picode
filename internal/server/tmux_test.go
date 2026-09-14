package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

// The tmux app's reads and its one action, driven through the route handlers
// with a scripted tmux source. No test here creates a tmux session: the seam
// exists so the decision table can be exercised without a single subprocess,
// which is also what keeps this suite from stranding sessions on the
// developer's own server (the leak ADR-0085's neighborhood was written for).

// fakeTmux is a tmuxServerSource whose every answer is scripted.
type fakeTmux struct {
	available bool
	info      tmux.ServerInfo
	sessions  []tmux.ServerSession
	receipts  map[string]tmux.SessionReceipt
	killErr   error
	killed    []string
}

func (f *fakeTmux) Available() bool { return f.available }
func (f *fakeTmux) ServerInfo(context.Context) tmux.ServerInfo {
	return f.info
}
func (f *fakeTmux) ServerSessions(context.Context) ([]tmux.ServerSession, error) {
	return f.sessions, nil
}
func (f *fakeTmux) SessionReceipt(_ context.Context, name string) (tmux.SessionReceipt, error) {
	r, ok := f.receipts[name]
	if !ok {
		return tmux.SessionReceipt{}, context.DeadlineExceeded // stands for "no such session"
	}
	return r, nil
}
func (f *fakeTmux) KillSession(_ context.Context, name string) error {
	if f.killErr != nil {
		return f.killErr
	}
	f.killed = append(f.killed, name)
	return nil
}

// tmuxTestStore opens a temp store.
func tmuxTestStore(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "picode.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

// tmuxTestServerWith registers ONLY the tmux routes over this store, so the
// handlers under test are the ones the real binary registers and nothing else
// has to exist for them.
func tmuxTestServerWith(t *testing.T, st *store.Store, src tmuxServerSource) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	registerTmuxRoutesWith(mux, Deps{Store: st, LostSessions: map[string]bool{}}, src)
	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)
	return ts
}

// tmuxTestServer is the common case: a fresh store and the routes over it.
func tmuxTestServer(t *testing.T, src tmuxServerSource) (*httptest.Server, *store.Store) {
	t.Helper()
	st := tmuxTestStore(t)
	return tmuxTestServerWith(t, st, src), st
}

func getTmuxJSON(t *testing.T, ts *httptest.Server, path string) (int, tmuxReadView) {
	t.Helper()
	r, err := ts.Client().Get(ts.URL + path)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	defer r.Body.Close()
	var view tmuxReadView
	if r.StatusCode == 200 {
		if err := json.NewDecoder(r.Body).Decode(&view); err != nil {
			t.Fatalf("decode %s: %v", path, err)
		}
	}
	return r.StatusCode, view
}

func postReap(t *testing.T, ts *httptest.Server, name, body string) (int, map[string]any) {
	t.Helper()
	res, err := ts.Client().Post(ts.URL+"/api/tmux/sessions/"+name+"/reap", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST reap %s: %v", name, err)
	}
	defer res.Body.Close()
	out := map[string]any{}
	_ = json.NewDecoder(res.Body).Decode(&out)
	return res.StatusCode, out
}

// sessionRow builds the tmux facts for one name; the tests only care about
// identity and attachment, so the rest is a plausible constant.
func sessionRow(name string, attached int) tmux.ServerSession {
	return tmux.ServerSession{
		Name: name, SessionID: "$7", Created: time.Unix(1789311380, 0).UTC(),
		Windows: 1, Panes: 1, Attached: attached, PaneID: "%3", PanePID: 4242,
		CurrentCommand: "bash", CurrentPath: "/home/goat/picode",
	}
}

// The marker names are the wire between three packages: the daemon injects
// them (internal/server/terminals.go, internal/store/agents.go), the session
// environment carries them, and this app reads them as the ownership receipt.
// A rename in one place without the other would silently turn every session
// into an "unclaimed" one, and the kill would then be offered for live work.
func TestTmuxMarkerNamesMatchTheirWriters(t *testing.T) {
	if tmux.MarkerAgentEnv != store.AgentIDEnv {
		t.Fatalf("tmux.MarkerAgentEnv = %q, store.AgentIDEnv = %q", tmux.MarkerAgentEnv, store.AgentIDEnv)
	}
	// PICODE_TERM_ID has no store constant (it is injected by the server), so
	// this is the one place its literal is pinned.
	if tmux.MarkerTermEnv != "PICODE_TERM_ID" {
		t.Fatalf("tmux.MarkerTermEnv = %q, want PICODE_TERM_ID", tmux.MarkerTermEnv)
	}
}

// The read: attribution, counts, order, and the absence list — the whole
// first half of the app in one request. Session names are DERIVED from the
// rows this test creates (tmux.ShellSessionName / tmux.SessionName), never
// written as literals: a literal fixture here would silently describe a
// session the store does not know and quietly turn a live terminal into a
// leftover (this test shipped that way once, and the skip made it look green).
func TestTmuxServerReadAttributesEverything(t *testing.T) {
	st := tmuxTestStore(t)
	dir := t.TempDir()
	ws, err := st.AddWorkspace("proj", dir)
	if err != nil {
		t.Fatalf("workspace: %v", err)
	}
	term, err := st.CreateTerminalIn(ws.ID, "build", dir)
	if err != nil {
		t.Fatalf("terminal: %v", err)
	}
	agent, err := st.AddAgent(ws.ID, "ag-1", "")
	if err != nil {
		t.Fatalf("agent: %v", err)
	}
	live, err := st.CreateTerminalIn(ws.ID, "live", dir)
	if err != nil {
		t.Fatalf("terminal: %v", err)
	}
	termSession := tmux.ShellSessionName(term.ID)
	agentSession := tmux.SessionName(agent.ID)
	liveSession := tmux.ShellSessionName(live.ID)
	if termSession == liveSession {
		t.Fatalf("fixture collision: both terminals derive %q", termSession)
	}

	// The server: all three rows above are on it, plus one PiCode-shaped
	// session no store row claims, plus one that is not PiCode's at all.
	ts := tmuxTestServerWith(t, st, &fakeTmux{
		available: true,
		info:      tmux.ServerInfo{Running: true, SocketPath: "/tmp/tmux-1000/default", Clients: 2, Version: "3.6", ExtendedKeysFormat: "xterm"},
		sessions: []tmux.ServerSession{
			sessionRow("picode-sh-gone", 0),
			sessionRow("sentinel-boot", 0),
			sessionRow(agentSession, 1),
			sessionRow(termSession, 0),
			sessionRow(liveSession, 0),
		},
	})

	code, view := getTmuxJSON(t, ts, "/api/tmux/server")
	if code != 200 {
		t.Fatalf("status = %d, want 200", code)
	}
	if !view.Server.Installed || !view.Server.Running || view.Server.Version != "3.6" || view.Server.Clients != 2 {
		t.Fatalf("server facts = %+v", view.Server)
	}
	if view.Server.Sessions != 5 || view.Server.PiCodeSessions != 3 || view.Server.UnclaimedSessions != 1 || view.Server.ForeignSessions != 1 {
		t.Fatalf("counts = %+v (want 5 total: 3 yours, 1 unclaimed, 1 foreign)", view.Server)
	}
	if view.Server.AttachedSessions != 1 {
		t.Fatalf("attachedSessions = %d, want 1", view.Server.AttachedSessions)
	}

	byName := map[string]tmuxSessionView{}
	for _, s := range view.Sessions {
		byName[s.Name] = s
	}
	for _, tc := range []struct {
		name, kind, scope, owner, ownerName string
	}{
		{agentSession, "agent", "yours", agent.ID, "ag-1"},
		{termSession, "terminal", "yours", term.ID, "build"},
		{liveSession, "terminal", "yours", live.ID, "live"},
		{"picode-sh-gone", "terminal", "unclaimed", "", ""},
		{"sentinel-boot", "other", "notPiCode", "", ""},
	} {
		got := byName[tc.name]
		if got.Kind != tc.kind || got.Scope != tc.scope || got.OwnerID != tc.owner || got.OwnerName != tc.ownerName {
			t.Errorf("%s = kind %q scope %q owner %q/%q, want %q %q %q %q",
				tc.name, got.Kind, got.Scope, got.OwnerID, got.OwnerName, tc.kind, tc.scope, tc.owner, tc.ownerName)
		}
	}

	// Order, asserted as the invariant rather than as a fixture list: an
	// agent before a terminal before a leftover before the user's own. The
	// exact sequence inside a rank depends on generated ids, which is not
	// what a reader of this app depends on.
	rank := map[string]int{"agent": 0, "terminal": 1, "other": 3}
	scopeRank := map[string]int{"yours": 0, "unclaimed": 2, "notPiCode": 3}
	last := -1
	for _, s := range view.Sessions {
		r := scopeRank[s.Scope]
		if s.Scope == "yours" {
			r = rank[s.Kind]
		}
		if r < last {
			t.Fatalf("session %q (scope %q kind %q) broke the rank order at rank %d after %d: %+v",
				s.Name, s.Scope, s.Kind, r, last, view.Sessions)
		}
		last = r
	}

	// Absence: the two terminals WITHOUT a session on the server are listed,
	// the one with a session is not. This is what makes the list evidence.
	if len(view.Absent) != 0 {
		t.Fatalf("absent = %+v, want none: every terminal row has a session on the server", view.Absent)
	}
}

// A terminal row whose session is missing is the absence list's one job, and
// ADR-0085's flight recorder is what tells "gone now" from "died in the
// restart window".
func TestTmuxAbsenceCarriesLostAtRestart(t *testing.T) {
	st := tmuxTestStore(t)
	dir := t.TempDir()
	ws, err := st.AddWorkspace("proj", dir)
	if err != nil {
		t.Fatalf("workspace: %v", err)
	}
	gone, err := st.CreateTerminalIn(ws.ID, "gone", dir)
	if err != nil {
		t.Fatalf("terminal: %v", err)
	}
	session := tmux.ShellSessionName(gone.ID)

	mux := http.NewServeMux()
	registerTmuxRoutesWith(mux, Deps{Store: st, LostSessions: map[string]bool{session: true}},
		&fakeTmux{available: true, info: tmux.ServerInfo{Running: true}})
	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)

	code, view := getTmuxJSON(t, ts, "/api/tmux/server")
	if code != 200 {
		t.Fatalf("status = %d, want 200", code)
	}
	if len(view.Absent) != 1 {
		t.Fatalf("absent = %+v, want the one terminal the server has no session for", view.Absent)
	}
	row := view.Absent[0]
	if row.Name != "gone" || row.Session != session || !row.LostAtRestart {
		t.Fatalf("absent row = %+v, want gone/%s with lostAtRestart", row, session)
	}
	if view.Server.UnclaimedSessions != 0 || len(view.Sessions) != 0 {
		t.Fatalf("a missing terminal must not appear as a session: %+v", view.Server)
	}
}

// A machine with no tmux binary is an empty screen with a reason, not a 503
// the app would have to invent copy for.
func TestTmuxServerReadWithoutTmux(t *testing.T) {
	ts, _ := tmuxTestServer(t, nil)
	code, view := getTmuxJSON(t, ts, "/api/tmux/server")
	if code != 200 {
		t.Fatalf("status = %d, want 200", code)
	}
	if view.Server.Installed || view.Server.Running || len(view.Sessions) != 0 || len(view.Absent) != 0 {
		t.Fatalf("view = %+v, want an honest empty one", view)
	}
}

// The decision table, row by row: what a kill request does under every
// combination of name, confirmation, receipt and ownership. The untested rows
// are the ones that would kill live work.
func TestTmuxReapDecisionTable(t *testing.T) {
	const name = "picode-sh-gone"
	goodReceipt := tmux.SessionReceipt{
		Name: name, SessionID: "$7", Created: time.Unix(1789311380, 0).UTC(), PaneID: "%3", PanePID: 4242,
	}
	body := func(confirm bool, sessionID string, created int64, pid int) string {
		raw, _ := json.Marshal(map[string]any{"confirm": confirm, "sessionId": sessionID, "created": created, "panePid": pid})
		return string(raw)
	}
	goodBody := body(true, "$7", 1789311380, 4242)

	t.Run("a name outside PiCode's namespace is refused", func(t *testing.T) {
		src := &fakeTmux{available: true, receipts: map[string]tmux.SessionReceipt{"my-work": goodReceipt}}
		ts, _ := tmuxTestServer(t, src)
		code, _ := postReap(t, ts, "my-work", goodBody)
		if code != 400 || len(src.killed) != 0 {
			t.Fatalf("status = %d killed = %v, want 400 and nothing killed", code, src.killed)
		}
	})

	t.Run("without confirmation nothing happens", func(t *testing.T) {
		src := &fakeTmux{available: true, receipts: map[string]tmux.SessionReceipt{name: goodReceipt}}
		ts, _ := tmuxTestServer(t, src)
		code, _ := postReap(t, ts, name, body(false, "$7", 1789311380, 4242))
		if code != 400 || len(src.killed) != 0 {
			t.Fatalf("status = %d killed = %v, want 400 and nothing killed", code, src.killed)
		}
	})

	t.Run("a session that is already gone answers 404", func(t *testing.T) {
		src := &fakeTmux{available: true, receipts: map[string]tmux.SessionReceipt{}}
		ts, _ := tmuxTestServer(t, src)
		code, _ := postReap(t, ts, name, goodBody)
		if code != 404 || len(src.killed) != 0 {
			t.Fatalf("status = %d killed = %v, want 404 and nothing killed", code, src.killed)
		}
	})

	t.Run("a receipt that no longer matches is refused", func(t *testing.T) {
		for _, tc := range []struct {
			what             string
			sessionID        string
			created, panePID int64
		}{
			{"session id changed", "$8", 1789311380, 4242},
			{"session was recreated", "$7", 1789311399, 4242},
			{"pane was respawned", "$7", 1789311380, 9999},
		} {
			t.Run(tc.what, func(t *testing.T) {
				src := &fakeTmux{available: true, receipts: map[string]tmux.SessionReceipt{name: goodReceipt}}
				ts, _ := tmuxTestServer(t, src)
				code, _ := postReap(t, ts, name, body(true, tc.sessionID, tc.created, int(tc.panePID)))
				if code != 409 || len(src.killed) != 0 {
					t.Fatalf("status = %d killed = %v, want 409 and nothing killed", code, src.killed)
				}
			})
		}
	})

	t.Run("a marker naming a live terminal refuses and names the door", func(t *testing.T) {
		_, st := tmuxTestServer(t, nil)
		dir := t.TempDir()
		ws, err := st.AddWorkspace("proj", dir)
		if err != nil {
			t.Fatalf("workspace: %v", err)
		}
		term, err := st.CreateTerminalIn(ws.ID, "build", dir)
		if err != nil {
			t.Fatalf("terminal: %v", err)
		}
		session := tmux.ShellSessionName(term.ID)
		src := &fakeTmux{available: true, receipts: map[string]tmux.SessionReceipt{
			session: {Name: session, SessionID: "$7", Created: time.Unix(1789311380, 0).UTC(), PaneID: "%3", PanePID: 4242, TermID: term.ID},
		}}
		mux := http.NewServeMux()
		registerTmuxRoutesWith(mux, Deps{Store: st, LostSessions: map[string]bool{}}, src)
		ts2 := httptest.NewServer(mux)
		t.Cleanup(ts2.Close)

		code, out := postReap(t, ts2, session, body(true, "$7", 1789311380, 4242))
		if code != 409 || len(src.killed) != 0 {
			t.Fatalf("status = %d killed = %v, want 409 and nothing killed", code, src.killed)
		}
		msg, _ := out["error"].(string)
		if !strings.Contains(msg, "build") || !strings.Contains(msg, "terminal") {
			t.Fatalf("message = %q, want it to name the terminal and its door", msg)
		}
	})

	t.Run("an unclaimed session with a matching receipt is killed once, and audited", func(t *testing.T) {
		src := &fakeTmux{available: true, receipts: map[string]tmux.SessionReceipt{name: goodReceipt}}
		ts, st := tmuxTestServer(t, src)
		code, out := postReap(t, ts, name, goodBody)
		if code != 200 {
			t.Fatalf("status = %d, want 200", code)
		}
		if len(src.killed) != 1 || src.killed[0] != name {
			t.Fatalf("killed = %v, want exactly [%s]", src.killed, name)
		}
		if out["audit"] != "recorded" {
			t.Fatalf("audit = %v, want recorded", out["audit"])
		}
		events, err := st.RecentEvents(50)
		if err != nil {
			t.Fatalf("events: %v", err)
		}
		found := false
		for _, ev := range events {
			if ev.Type == "tmux.session.reaped" {
				found = true
			}
		}
		if !found {
			t.Fatalf("no tmux.session.reaped event in %d recent events", len(events))
		}
	})

	t.Run("a failed kill answers 500 and reports it", func(t *testing.T) {
		src := &fakeTmux{available: true, killErr: context.Canceled, receipts: map[string]tmux.SessionReceipt{name: goodReceipt}}
		ts, _ := tmuxTestServer(t, src)
		code, _ := postReap(t, ts, name, goodBody)
		if code != 500 {
			t.Fatalf("status = %d, want 500", code)
		}
	})

	t.Run("no tmux at all answers 503", func(t *testing.T) {
		src := &fakeTmux{available: false, receipts: map[string]tmux.SessionReceipt{name: goodReceipt}}
		ts, _ := tmuxTestServer(t, src)
		code, _ := postReap(t, ts, name, goodBody)
		if code != 503 {
			t.Fatalf("status = %d, want 503", code)
		}
	})

	t.Run("an unknown field is rejected rather than ignored", func(t *testing.T) {
		src := &fakeTmux{available: true, receipts: map[string]tmux.SessionReceipt{name: goodReceipt}}
		ts, _ := tmuxTestServer(t, src)
		code, _ := postReap(t, ts, name, `{"confirm":true,"sessionId":"$7","created":1789311380,"panePid":4242,"force":true}`)
		if code != 400 || len(src.killed) != 0 {
			t.Fatalf("status = %d killed = %v, want 400 and nothing killed", code, src.killed)
		}
	})
}
