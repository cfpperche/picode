package apps

import (
	"context"
	"errors"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

// The tmux app's reads and its one action, driven through View/Action with a
// scripted TmuxServer. No test here creates a tmux session: the seam exists so
// the decision table is exercised without a single subprocess, which is also
// what keeps this suite from stranding sessions on the developer's own server
// (the leak the tmux-isolation branch exists for).

type fakeTmuxServer struct {
	available bool
	info      tmux.ServerInfo
	sessions  []tmux.ServerSession
	receipts  map[string]tmux.SessionReceipt
	sockets   []tmux.MachineSocket
	instance  string
	killErr   error
	killed    []string
}

func (f *fakeTmuxServer) Available() bool { return f.available }

func (f *fakeTmuxServer) Instance() string { return f.instance }

func (f *fakeTmuxServer) MachineSockets(context.Context) []tmux.MachineSocket {
	return f.sockets
}
func (f *fakeTmuxServer) ServerInfo(context.Context) tmux.ServerInfo {
	return f.info
}
func (f *fakeTmuxServer) ServerSessions(context.Context) ([]tmux.ServerSession, error) {
	return f.sessions, nil
}
func (f *fakeTmuxServer) SessionReceipt(_ context.Context, name string) (tmux.SessionReceipt, error) {
	r, ok := f.receipts[name]
	if !ok {
		return tmux.SessionReceipt{}, errors.New("can't find session")
	}
	return r, nil
}
func (f *fakeTmuxServer) KillSession(_ context.Context, name string) error {
	if f.killErr != nil {
		return f.killErr
	}
	f.killed = append(f.killed, name)
	return nil
}

func tmuxTestStore(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "picode.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
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

func listByTitle(v View, title string) (Block, bool) {
	for _, b := range v.Blocks {
		if b.Type == "list" && b.Title == title {
			return b, true
		}
	}
	return Block{}, false
}

// The marker names are the wire between three packages: the daemon injects
// them (internal/server/terminals.go, internal/store/agents.go), the session
// environment carries them, and this app reads them as the ownership receipt.
// A rename in one place without the others would silently turn every session
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
	// ADR-0140: the instance stamp and the loopback URL are read here and
	// written by internal/server — pinned so a rename cannot silently turn
	// every session into "someone else's" (or reopen the reap to another
	// instance's work).
	if tmux.MarkerInstanceEnv != "PICODE_INSTANCE" {
		t.Fatalf("tmux.MarkerInstanceEnv = %q, want PICODE_INSTANCE", tmux.MarkerInstanceEnv)
	}
	if tmux.MarkerURLEnv != "PICODE_TERM_URL" {
		t.Fatalf("tmux.MarkerURLEnv = %q, want PICODE_TERM_URL", tmux.MarkerURLEnv)
	}
}

// The inventory read: three groups, the counts, and the tab badge that names
// the leftovers. Everything the Docker-mold list shows is asserted here.
func TestTmuxAppViewAttributesEverything(t *testing.T) {
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
	termSession := tmux.ShellSessionName(term.ID)
	agentSession := tmux.SessionName(agent.ID)

	src := &fakeTmuxServer{
		available: true,
		info:      tmux.ServerInfo{Running: true, SocketPath: "/tmp/tmux-1000/default", Clients: 2, Version: "3.6", ExtendedKeysFormat: "xterm"},
		sessions: []tmux.ServerSession{
			sessionRow("picode-sh-gone", 0), // leftover: name resolves, store does not
			sessionRow("sentinel-boot", 0),  // the user's own tmux
			sessionRow(agentSession, 1),     // this daemon's agent
			sessionRow(termSession, 0),      // this daemon's terminal
		},
	}
	app := tmuxApp{}
	v, err := app.View(context.Background(), Host{Store: st, Tmux: src}, "")
	if err != nil {
		t.Fatalf("View: %v", err)
	}
	if len(v.Tabs) != 3 || v.Tabs[0].Label != "Sessions" || v.Tabs[1].Label != "Server" || v.Tabs[2].Label != "Sockets" {
		t.Fatalf("tabs = %+v", v.Tabs)
	}
	// The tab badge is one number (primitives.Tab: "short count"): the
	// leftovers are named by their own group and row badges, and the count
	// chip beside an underline label has no room for a sentence.
	if v.Tabs[0].Badge != "4" {
		t.Fatalf("sessions tab badge = %q", v.Tabs[0].Badge)
	}

	yours, ok := listByTitle(v, "PiCode")
	if !ok || len(yours.Items) != 2 {
		t.Fatalf("PiCode group = %+v", yours)
	}
	byTitle := map[string]ListItem{}
	for _, i := range yours.Items {
		byTitle[i.Title] = i
	}
	agentRow, termRow := byTitle["ag-1"], byTitle["build"]
	if agentRow.Path != "item/"+agentSession || agentRow.Tone != "ok" {
		t.Fatalf("agent row = %+v", agentRow)
	}
	if termRow.Path != "item/"+termSession || termRow.ID != termSession {
		t.Fatalf("terminal row = %+v", termRow)
	}

	leftovers, ok := listByTitle(v, "Not in PiCode's records")
	if !ok || len(leftovers.Items) != 1 || leftovers.Items[0].Title != "picode-sh-gone" {
		t.Fatalf("leftovers = %+v", leftovers)
	}
	if leftovers.Items[0].Badge != "no record" || leftovers.Items[0].Path == "" {
		t.Fatalf("leftover row = %+v (must open its detail, where the removal lives)", leftovers.Items[0])
	}

	foreign, ok := listByTitle(v, "Not PiCode's")
	if !ok || len(foreign.Items) != 1 || foreign.Items[0].Path != "" {
		t.Fatalf("foreign = %+v (the user's own session has no door)", foreign)
	}
}

// The absence list: terminals with no session, and the flight recorder's
// lost-at-restart verdict on top.
// The Sockets screen lists every tmux server this instance can see — the
// operator view ADR-0139 created a need for. One tab, one block, one row per
// socket; running/dead and PiCode ownership are stated, never inferred by the
// reader.
func TestTmuxAppSocketsViewListsServers(t *testing.T) {
	src := &fakeTmuxServer{
		available: true,
		sockets: []tmux.MachineSocket{
			{Path: "/home/goat/.picode/tmux.sock", Name: "tmux.sock", Ours: true, Running: false},
			{Path: "/tmp/tmux-1000/default", Name: "default", Running: true, Sessions: 12, PicodeSessions: 10},
			{Path: "/tmp/tmux-1000/tachyon", Name: "tachyon", Running: true, Sessions: 1, PicodeSessions: 0},
			{Path: "/tmp/tmux-1000/stale", Name: "stale", Running: false},
		},
	}
	app := tmuxApp{}
	v, err := app.View(context.Background(), Host{Tmux: src}, "sockets")
	if err != nil {
		t.Fatalf("view: %v", err)
	}
	if len(v.Tabs) != 3 || v.Tabs[2].ID != "sockets" || v.Tabs[2].Badge != "2" {
		t.Fatalf("tabs = %+v, want a sockets tab with badge 2 (running)", v.Tabs)
	}
	if len(v.Blocks) != 1 || len(v.Blocks[0].Items) != 4 {
		t.Fatalf("blocks = %+v", v.Blocks)
	}
	byTitle := map[string]ListItem{}
	for _, it := range v.Blocks[0].Items {
		byTitle[it.Title] = it
	}
	ours, ok := byTitle["tmux.sock · this instance"]
	if !ok {
		t.Fatalf("our socket row missing: %+v", byTitle)
	}
	if ours.Badge != "idle" || ours.Tone != "" {
		t.Fatalf("our not-yet-running socket = %+v, want a neutral 'idle' row (it is not a leftover)", ours)
	}
	if !strings.Contains(ours.Subtitle, "next terminal") {
		t.Fatalf("our idle row subtitle = %q, want it to say what starts the server", ours.Subtitle)
	}
	def, ok := byTitle["default"]
	if !ok || def.Subtitle != "12 session(s) · 10 PiCode" || def.Tone != "ok" {
		t.Fatalf("default row = %+v", def)
	}
	if got := byTitle["tachyon"].Subtitle; got != "1 session(s) · 0 PiCode" {
		t.Fatalf("tachyon row = %q", got)
	}
	if stale := byTitle["stale"]; stale.Badge != "no server" || stale.Subtitle == "" {
		t.Fatalf("stale row = %+v", stale)
	}
}

func TestTmuxAppSocketsViewWithoutTmux(t *testing.T) {
	app := tmuxApp{}
	v, err := app.View(context.Background(), Host{Tmux: &fakeTmuxServer{available: false}}, "sockets")
	if err != nil || v.Empty == "" {
		t.Fatalf("view = %+v, %v; want the tmux-missing empty state", v, err)
	}
}

func TestTmuxAppServerViewCarriesAbsence(t *testing.T) {
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

	app := tmuxApp{}
	v, err := app.View(context.Background(), Host{
		Store: st, Tmux: &fakeTmuxServer{available: true, info: tmux.ServerInfo{Running: true}},
		LostSessions: map[string]bool{session: true},
	}, "server")
	if err != nil {
		t.Fatalf("View: %v", err)
	}
	var absent Block
	found := false
	for _, b := range v.Blocks {
		if b.Title == "In PiCode's records, not on this server" {
			absent, found = b, true
		}
	}
	if !found || len(absent.Items) != 1 {
		t.Fatalf("absence list = %+v", v.Blocks)
	}
	if absent.Items[0].Badge != "lost at restart" || absent.Items[0].ID != session {
		t.Fatalf("absent row = %+v, want %s marked lost at restart", absent.Items[0], session)
	}
	if !strings.Contains(absent.Items[0].Subtitle, session) {
		t.Fatalf("absent row = %+v, want the session name on it", absent.Items[0])
	}
}

// No tmux: the honest blankslate, not an error.
// The detail screen is where the verdict is read and the only place a removal
// can be asked for: another instance's session gets the sentence and no door.
func TestTmuxAppSessionViewRefusesAnotherInstancesSession(t *testing.T) {
	const name = "picode-sh-elsewhere"
	src := &fakeTmuxServer{
		available: true, instance: "/home/goat/.picode",
		sessions: []tmux.ServerSession{sessionRow(name, 1)},
		receipts: map[string]tmux.SessionReceipt{
			name: {Name: name, SessionID: "$9", Created: time.Unix(1789311380, 0).UTC(), PaneID: "%4", PanePID: 77,
				Instance: "/home/goat/.worktrees/other/var/qa/data", URL: "https://localhost:8475"},
		},
	}
	v, err := tmuxApp{}.View(context.Background(), Host{Tmux: src}, "item/"+name)
	if err != nil {
		t.Fatalf("View: %v", err)
	}
	for _, b := range v.Blocks {
		if b.Type == "actions" {
			t.Fatalf("blocks = %+v, want no actions block for another instance's session", v.Blocks)
		}
	}
	text := ""
	for _, b := range v.Blocks {
		text += b.Markdown
	}
	if !strings.Contains(text, "another PiCode instance") || !strings.Contains(text, "8475") {
		t.Fatalf("detail = %q, want the other instance named", text)
	}

	// The same screen for this instance's own leftover keeps the door.
	src.receipts[name] = tmux.SessionReceipt{Name: name, SessionID: "$9", Created: time.Unix(1789311380, 0).UTC(), PaneID: "%4", PanePID: 77, Instance: "/home/goat/.picode"}
	v, err = tmuxApp{}.View(context.Background(), Host{Tmux: src}, "item/"+name)
	if err != nil {
		t.Fatalf("View: %v", err)
	}
	found := false
	for _, b := range v.Blocks {
		for _, a := range b.Actions {
			if a.ID == "reap" {
				found = true
			}
		}
	}
	if !found {
		t.Fatalf("blocks = %+v, want the reap door for the instance's own leftover", v.Blocks)
	}
}

func TestTmuxAppWithoutTmux(t *testing.T) {
	app := tmuxApp{}
	for _, src := range []TmuxServer{nil, &fakeTmuxServer{available: false}} {
		v, err := app.View(context.Background(), Host{Tmux: src}, "")
		if err != nil {
			t.Fatalf("View error: %v", err)
		}
		if v.Empty == "" || len(v.Blocks) != 0 {
			t.Fatalf("view = %+v, want the honest empty screen", v)
		}
	}
}

// The removal, decision row by decision row. The untested rows are the ones
// that would kill live work.
func TestTmuxReapDecisionTable(t *testing.T) {
	const name = "picode-sh-gone"
	goodReceipt := tmux.SessionReceipt{
		Name: name, SessionID: "$7", Created: time.Unix(1789311380, 0).UTC(), PaneID: "%3", PanePID: 4242,
	}
	goodArgs := func(sessionID string, created int64, pid int) map[string]string {
		return map[string]string{"name": name, "sessionId": sessionID, "created": strconv.FormatInt(created, 10), "panePid": strconv.Itoa(pid)}
	}
	good := goodArgs("$7", 1789311380, 4242)

	t.Run("a name outside PiCode's namespace is refused", func(t *testing.T) {
		src := &fakeTmuxServer{available: true, receipts: map[string]tmux.SessionReceipt{"my-work": goodReceipt}}
		_, err := tmuxApp{}.Action(context.Background(), Host{Tmux: src}, ActionRequest{Action: "reap", Args: map[string]string{"name": "my-work", "sessionId": "$7", "created": "1789311380", "panePid": "4242"}})
		if err == nil || len(src.killed) != 0 {
			t.Fatalf("err = %v killed = %v, want refusal and nothing killed", err, src.killed)
		}
	})

	t.Run("a session that is already gone answers so", func(t *testing.T) {
		src := &fakeTmuxServer{available: true, receipts: map[string]tmux.SessionReceipt{}}
		_, err := tmuxApp{}.Action(context.Background(), Host{Tmux: src}, ActionRequest{Action: "reap", Args: goodArgs("$7", 1789311380, 4242)})
		if err == nil || len(src.killed) != 0 {
			t.Fatalf("err = %v killed = %v, want refusal and nothing killed", err, src.killed)
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
				src := &fakeTmuxServer{available: true, receipts: map[string]tmux.SessionReceipt{name: goodReceipt}}
				_, err := tmuxApp{}.Action(context.Background(), Host{Tmux: src}, ActionRequest{Action: "reap", Args: goodArgs(tc.sessionID, tc.created, int(tc.panePID))})
				if err == nil || len(src.killed) != 0 {
					t.Fatalf("err = %v killed = %v, want refusal and nothing killed", err, src.killed)
				}
			})
		}
	})

	t.Run("a marker naming a live terminal refuses and names the door", func(t *testing.T) {
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
		session := tmux.ShellSessionName(term.ID)
		src := &fakeTmuxServer{available: true, receipts: map[string]tmux.SessionReceipt{
			session: {Name: session, SessionID: "$7", Created: time.Unix(1789311380, 0).UTC(), PaneID: "%3", PanePID: 4242, TermID: term.ID},
		}}
		_, err = tmuxApp{}.Action(context.Background(), Host{Store: st, Tmux: src}, ActionRequest{
			Action: "reap", Args: map[string]string{"name": session, "sessionId": "$7", "created": "1789311380", "panePid": "4242"},
		})
		if err == nil || len(src.killed) != 0 {
			t.Fatalf("err = %v killed = %v, want refusal and nothing killed", err, src.killed)
		}
		if msg := err.Error(); !strings.Contains(msg, "build") || !strings.Contains(msg, "terminal") {
			t.Fatalf("message = %q, want it to name the terminal and its door", msg)
		}
	})

	t.Run("an unclaimed session with a matching receipt is killed once, and audited", func(t *testing.T) {
		src := &fakeTmuxServer{available: true, receipts: map[string]tmux.SessionReceipt{name: goodReceipt}}
		st := tmuxTestStore(t)
		res, err := tmuxApp{}.Action(context.Background(), Host{Store: st, Tmux: src}, ActionRequest{Action: "reap", Args: good})
		if err != nil {
			t.Fatalf("Action: %v", err)
		}
		if len(src.killed) != 1 || src.killed[0] != name {
			t.Fatalf("killed = %v, want exactly [%s]", src.killed, name)
		}
		if !strings.Contains(res.Toast, "removed") || res.View == nil {
			t.Fatalf("result = %+v, want a toast and the refreshed inventory", res)
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

	t.Run("a failed kill reports it", func(t *testing.T) {
		src := &fakeTmuxServer{available: true, killErr: context.Canceled, receipts: map[string]tmux.SessionReceipt{name: goodReceipt}}
		_, err := tmuxApp{}.Action(context.Background(), Host{Tmux: src}, ActionRequest{Action: "reap", Args: good})
		if err == nil {
			t.Fatalf("err = nil, want the kill's failure")
		}
	})

	t.Run("a session stamped by another instance is refused and named", func(t *testing.T) {
		src := &fakeTmuxServer{available: true, instance: "/home/goat/.picode", receipts: map[string]tmux.SessionReceipt{
			name: {Name: name, SessionID: "$7", Created: time.Unix(1789311380, 0).UTC(), PaneID: "%3", PanePID: 4242,
				Instance: "/home/goat/.worktrees/other/var/qa/data", URL: "https://localhost:8475"},
		}}
		_, err := tmuxApp{}.Action(context.Background(), Host{Tmux: src}, ActionRequest{Action: "reap", Args: good})
		if err == nil || len(src.killed) != 0 {
			t.Fatalf("err = %v killed = %v, want refusal and nothing killed", err, src.killed)
		}
		if msg := err.Error(); !strings.Contains(msg, "another PiCode instance") || !strings.Contains(msg, "8475") {
			t.Fatalf("message = %q, want it to name the other instance", msg)
		}
	})

	t.Run("a session stamped by this instance is a leftover and is reaped", func(t *testing.T) {
		src := &fakeTmuxServer{available: true, instance: "/home/goat/.picode", receipts: map[string]tmux.SessionReceipt{
			name: {Name: name, SessionID: "$7", Created: time.Unix(1789311380, 0).UTC(), PaneID: "%3", PanePID: 4242,
				Instance: "/home/goat/.picode"},
		}}
		if _, err := (tmuxApp{}).Action(context.Background(), Host{Tmux: src}, ActionRequest{Action: "reap", Args: good}); err != nil {
			t.Fatalf("Action: %v", err)
		}
		if len(src.killed) != 1 {
			t.Fatalf("killed = %v, want the instance's own leftover removed", src.killed)
		}
	})

	t.Run("a pre-stamp session follows its port, not its name", func(t *testing.T) {
		// No PICODE_INSTANCE: the sessions created before ADR-0140. The port
		// in the session's own PICODE_TERM_URL is the only identity left.
		receipt := func(u string) map[string]tmux.SessionReceipt {
			return map[string]tmux.SessionReceipt{name: {Name: name, SessionID: "$7", Created: time.Unix(1789311380, 0).UTC(), PaneID: "%3", PanePID: 4242, URL: u}}
		}
		other := &fakeTmuxServer{available: true, instance: "/mine", receipts: receipt("https://localhost:8445")}
		_, err := tmuxApp{}.Action(context.Background(), Host{Tmux: other, LoopbackURL: "https://localhost:8475"}, ActionRequest{Action: "reap", Args: good})
		if err == nil || len(other.killed) != 0 {
			t.Fatalf("err = %v killed = %v, want the other port refused", err, other.killed)
		}
		mine := &fakeTmuxServer{available: true, instance: "/mine", receipts: receipt("https://localhost:8475")}
		if _, err := (tmuxApp{}).Action(context.Background(), Host{Tmux: mine, LoopbackURL: "https://localhost:8475"}, ActionRequest{Action: "reap", Args: good}); err != nil {
			t.Fatalf("own port: %v", err)
		}
		if len(mine.killed) != 1 {
			t.Fatalf("killed = %v, want our own pre-stamp leftover removed", mine.killed)
		}
	})

	t.Run("no tmux at all refuses", func(t *testing.T) {
		src := &fakeTmuxServer{available: false, receipts: map[string]tmux.SessionReceipt{name: goodReceipt}}
		_, err := tmuxApp{}.Action(context.Background(), Host{Tmux: src}, ActionRequest{Action: "reap", Args: good})
		if err == nil {
			t.Fatalf("err = nil, want refusal")
		}
	})
}
