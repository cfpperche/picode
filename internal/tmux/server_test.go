package tmux

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// The server read model's parsing: what tmux prints, and what a row means.
// These tests script the tmux subprocess (the same seam tmux_test.go uses),
// so none of them needs a tmux server — or the user's own sessions.

// serverScript answers tmux invocations by operation name. Keying on the
// operation (not on call order) keeps a test readable when the code under
// test makes its calls in a different order than the last version did.
type serverScript struct {
	replies map[string]scriptedReply
	calls   []string
}

func (s *serverScript) run(_ context.Context, _ string, args ...string) ([]byte, error) {
	if len(args) == 0 {
		return nil, errors.New("no args")
	}
	s.calls = append(s.calls, args[0])
	r, ok := s.replies[args[0]]
	if !ok {
		return nil, errors.New("exit status 1")
	}
	if r.fail {
		return []byte(r.out), errors.New("exit status 1")
	}
	return []byte(r.out), nil
}

func managerOn(s *serverScript) *Manager { return &Manager{exec: s.run} }

// A pane row carries the session's facts; `pane_start_command` is last in the
// format precisely so a tab inside it cannot shift the fields before it.
func paneRow(session, sessionID, created, attached, paneID, pid, dead, status, command, cwd, active, start string) string {
	return strings.Join([]string{session, sessionID, created, "1", attached, paneID, pid, dead, status, command, cwd, active, start}, "\t")
}

func TestServerSessionsParsesAndGroups(t *testing.T) {
	s := &serverScript{replies: map[string]scriptedReply{
		"list-panes": {out: strings.Join([]string{
			// A session with two panes: the second is active, so it supplies
			// the details — and the session is still ONE row, counted twice.
			paneRow("picode-sh-a", "$1", "1789311380", "2", "%1", "100", "0", "", "sh", "/one", "0", "bash"),
			paneRow("picode-sh-a", "$1", "1789311380", "2", "%2", "200", "0", "", "node", "/two", "1", "node dev"),
			// A dead pane with an exit status, and a command holding a tab.
			paneRow("picode-ag-b", "$2", "1789311390", "0", "%3", "300", "1", "143", "bash", "/three", "1", "bash\t-t"),
			// A session on the user's own server, not PiCode's.
			paneRow("my-work", "$3", "1789311400", "1", "%4", "400", "0", "", "vim", "/home/goat", "1", "vim"),
		}, "\n")},
	}}

	rows, err := managerOn(s).ServerSessions(context.Background())
	if err != nil {
		t.Fatalf("ServerSessions: %v", err)
	}
	if len(s.calls) != 1 {
		t.Fatalf("tmux calls = %v, want ONE call for the whole server", s.calls)
	}
	if len(rows) != 3 {
		t.Fatalf("rows = %d, want 3 (one per session, never one per pane)", len(rows))
	}

	a := rows[0]
	if a.Name != "picode-sh-a" || a.SessionID != "$1" || a.Panes != 2 || a.Windows != 1 {
		t.Fatalf("multi-pane row = %+v", a)
	}
	if a.Attached != 2 {
		t.Fatalf("attached = %d, want the session's client count (2)", a.Attached)
	}
	if a.CurrentCommand != "node" || a.CurrentPath != "/two" || a.PaneID != "%2" || a.PanePID != 200 {
		t.Fatalf("active pane did not win the details: %+v", a)
	}
	if a.Created.Unix() != 1789311380 {
		t.Fatalf("created = %v", a.Created)
	}

	b := rows[1]
	if !b.PaneDead || b.DeadStatus != 143 {
		t.Fatalf("dead pane facts = %+v (want dead with status 143)", b)
	}
	if b.StartCommand != "bash\t-t" {
		t.Fatalf("startCommand = %q, want the tab inside it preserved", b.StartCommand)
	}

	if rows[2].Name != "my-work" || rows[2].PaneDead {
		t.Fatalf("foreign row = %+v", rows[2])
	}
}

// A dead pane with no status reported by tmux is -1, which the route renders
// as "no code" rather than as exit code zero.
func TestServerSessionsDeadPaneWithoutStatus(t *testing.T) {
	s := &serverScript{replies: map[string]scriptedReply{
		"list-panes": {out: paneRow("picode-sh-x", "$9", "1789311380", "0", "%1", "100", "1", "", "bash", "/x", "1", "bash")},
	}}
	rows, err := managerOn(s).ServerSessions(context.Background())
	if err != nil {
		t.Fatalf("ServerSessions: %v", err)
	}
	if len(rows) != 1 || !rows[0].PaneDead || rows[0].DeadStatus != -1 {
		t.Fatalf("rows = %+v, want one dead pane with status -1", rows)
	}
}

// A server that is not running is an empty inventory, never an error: the app
// draws "nothing is running" from this, and a stopped server is not a fault.
func TestServerSessionsWithoutAServer(t *testing.T) {
	for _, msg := range []string{"no server running on /tmp/tmux-1000/default", "error connecting to /tmp/tmux-1000/default (No such file or directory)"} {
		s := &serverScript{replies: map[string]scriptedReply{"list-panes": {out: msg, fail: true}}}
		rows, err := managerOn(s).ServerSessions(context.Background())
		if err != nil {
			t.Fatalf("ServerSessions(%q) error = %v, want nil", msg, err)
		}
		if rows != nil {
			t.Fatalf("ServerSessions(%q) = %v, want empty", msg, rows)
		}
	}
	// Anything else is a real failure and must not masquerade as "no server".
	s := &serverScript{replies: map[string]scriptedReply{"list-panes": {out: "tmux: unknown option", fail: true}}}
	if _, err := managerOn(s).ServerSessions(context.Background()); err == nil {
		t.Fatalf("a genuine tmux failure returned nil error")
	}
}

func TestServerInfoReadsTheServer(t *testing.T) {
	s := &serverScript{replies: map[string]scriptedReply{
		"display-message": {out: "/tmp/tmux-1000/default\n"},
		"list-clients":    {out: "501\n502\n503\n"},
		"show-options":    {out: "xterm\n"},
	}}
	info := managerOn(s).ServerInfo(context.Background())
	if !info.Running || info.SocketPath != "/tmp/tmux-1000/default" {
		t.Fatalf("info = %+v, want running with the socket path", info)
	}
	if info.Clients != 3 {
		t.Fatalf("clients = %d, want 3", info.Clients)
	}
	if info.ExtendedKeysFormat != "xterm" {
		t.Fatalf("extendedKeysFormat = %q, want xterm", info.ExtendedKeysFormat)
	}
}

// No server: running stays false and the read still answers (the app shows
// the honest empty screen), and no clients is not an error.
func TestServerInfoWithoutAServer(t *testing.T) {
	s := &serverScript{replies: map[string]scriptedReply{
		"display-message": {out: "no server running on /tmp/tmux-1000/default", fail: true},
	}}
	info := managerOn(s).ServerInfo(context.Background())
	if info.Running || info.SocketPath != "" {
		t.Fatalf("info = %+v, want not running and no path", info)
	}
	if info.Clients != 0 {
		t.Fatalf("clients = %d, want 0", info.Clients)
	}
}

// The receipt: identity from tmux plus the markers the session carries in its
// own environment — the two reads an irreversible action is verified against.
func TestSessionReceiptReadsIdentityAndMarkers(t *testing.T) {
	s := &serverScript{replies: map[string]scriptedReply{
		"display-message":  {out: "picode-sh-build\t$12\t1789311380\t%3\t4242\n"},
		"show-environment": {out: "PATH=/usr/bin\nPICODE_TERM_ID=build-1a2b3c\nPICODE_TERM_URL=http://127.0.0.1:8474\n"},
	}}
	r, err := managerOn(s).SessionReceipt(context.Background(), "picode-sh-build")
	if err != nil {
		t.Fatalf("SessionReceipt: %v", err)
	}
	if r.Name != "picode-sh-build" || r.SessionID != "$12" || r.PaneID != "%3" || r.PanePID != 4242 {
		t.Fatalf("receipt = %+v", r)
	}
	if r.Created.Unix() != 1789311380 {
		t.Fatalf("created = %v", r.Created)
	}
	if r.TermID != "build-1a2b3c" || r.AgentID != "" {
		t.Fatalf("markers = term %q agent %q, want only the term marker", r.TermID, r.AgentID)
	}
}

// An agent session carries the agent marker instead, and a session nobody
// marked reports no marker — which is what makes it "not PiCode's records"
// rather than a guess.
func TestSessionReceiptMarkerVariants(t *testing.T) {
	for _, tc := range []struct {
		name, env, wantTerm, wantAgent string
	}{
		{"agent session", "PICODE_AGENT_ID=ag-1\n", "", "ag-1"},
		{"unmarked session", "PATH=/usr/bin\n", "", ""},
		{"no environment at all", "", "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			replies := map[string]scriptedReply{
				"display-message": {out: "picode-x\t$4\t1789311380\t%1\t10\n"},
			}
			if tc.env == "" {
				replies["show-environment"] = scriptedReply{fail: true}
			} else {
				replies["show-environment"] = scriptedReply{out: tc.env}
			}
			r, err := managerOn(&serverScript{replies: replies}).SessionReceipt(context.Background(), "picode-x")
			if err != nil {
				t.Fatalf("SessionReceipt: %v", err)
			}
			if r.TermID != tc.wantTerm || r.AgentID != tc.wantAgent {
				t.Fatalf("markers = %q/%q, want %q/%q", r.TermID, r.AgentID, tc.wantTerm, tc.wantAgent)
			}
		})
	}
}

// A session that is gone fails the receipt: the route turns this into a 404
// and nothing is killed. An unreadable identity is a failure too — a receipt
// that cannot be trusted must never be treated as a match.
func TestSessionReceiptRefusesUnreadableIdentity(t *testing.T) {
	gone := &serverScript{replies: map[string]scriptedReply{
		"display-message": {out: "can't find session: picode-sh-gone", fail: true},
	}}
	if _, err := managerOn(gone).SessionReceipt(context.Background(), "picode-sh-gone"); err == nil {
		t.Fatalf("a missing session returned a receipt")
	}
	short := &serverScript{replies: map[string]scriptedReply{
		"display-message": {out: "picode-sh-x\t$4\n"},
	}}
	if _, err := managerOn(short).SessionReceipt(context.Background(), "picode-sh-x"); err == nil {
		t.Fatalf("a truncated identity returned a receipt")
	}
}
