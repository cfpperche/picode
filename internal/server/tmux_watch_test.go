package server

// The tmux server watch's decision table (plan tmux-resilience phase 1):
// every row of emitTmuxServerTransitions has a case here, plus a probe test
// with a fake prober and one integration test that flips Running for real —
// on an isolated socket, with the socket_path assertion the 2026-09-15
// lesson made mandatory.

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/tmux"
)

type eventRecorder struct {
	mu   sync.Mutex
	rows []Event
}

// Event mirrors the fields the recorder captures; the store's Event type is
// not exported here, so tests keep their own shape.
type Event struct {
	Type string
	Data any
}

func (r *eventRecorder) AppendEvent(eventType string, _, _ *string, data any) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rows = append(r.rows, Event{Type: eventType, Data: data})
	return nil
}

func (r *eventRecorder) types() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, 0, len(r.rows))
	for _, e := range r.rows {
		out = append(out, e.Type)
	}
	return out
}

func TestTmuxServerTransitions(t *testing.T) {
	seen := time.Date(2026, 9, 15, 15, 0, 0, 0, time.UTC)
	cases := []struct {
		name string
		prev tmuxServerState
		cur  tmuxServerState
		want []string
	}{
		{
			name: "server lost with sessions at stake records and logs",
			prev: tmuxServerState{running: true, socket: "/tmp/tmux-1000/default", sessions: 28},
			cur:  tmuxServerState{},
			want: []string{"terminal.server_lost"},
		},
		{
			name: "server lost with zero sessions is a normal exit",
			prev: tmuxServerState{running: true, socket: "/tmp/tmux-1000/default", sessions: 0},
			cur:  tmuxServerState{},
			want: nil,
		},
		{
			name: "server back records",
			prev: tmuxServerState{},
			cur:  tmuxServerState{running: true, socket: "/tmp/tmux-1000/default", sessions: 2},
			want: []string{"terminal.server_back"},
		},
		{
			name: "alive to alive records nothing",
			prev: tmuxServerState{running: true, socket: "/tmp/tmux-1000/default", sessions: 5},
			cur:  tmuxServerState{running: true, socket: "/tmp/tmux-1000/default", sessions: 4},
			want: nil,
		},
		{
			name: "mass drop while running logs but does not record",
			prev: tmuxServerState{running: true, socket: "/tmp/tmux-1000/default", sessions: 28},
			cur:  tmuxServerState{running: true, socket: "/tmp/tmux-1000/default", sessions: 3},
			want: nil,
		},
		{
			name: "absent to absent records nothing",
			prev: tmuxServerState{},
			cur:  tmuxServerState{},
			want: nil,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := &eventRecorder{}
			emitTmuxServerTransitions(rec, tc.prev, tc.cur, seen)
			got := rec.types()
			if len(got) != len(tc.want) {
				t.Fatalf("events = %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("events = %v, want %v", got, tc.want)
				}
			}
			if len(rec.rows) == 1 && rec.rows[0].Type == "terminal.server_lost" {
				data, _ := rec.rows[0].Data.(map[string]any)
				if data["sessions"] != 28 {
					t.Fatalf("loss event sessions = %v, want 28", data["sessions"])
				}
				if data["socket"] != tc.prev.socket {
					t.Fatalf("loss event socket = %v, want %v", data["socket"], tc.prev.socket)
				}
				if data["lastSeen"] != seen.Format(time.RFC3339) {
					t.Fatalf("loss event lastSeen = %v, want %v", data["lastSeen"], seen.Format(time.RFC3339))
				}
			}
		})
	}
}

type fakeProber struct {
	available bool
	info      tmux.ServerInfo
	sessions  []tmux.ServerSession
	listErr   error
}

func (f *fakeProber) Available() bool { return f.available }
func (f *fakeProber) ServerInfo(context.Context) tmux.ServerInfo {
	return f.info
}
func (f *fakeProber) ServerSessions(context.Context) ([]tmux.ServerSession, error) {
	return f.sessions, f.listErr
}

func TestProbeTmuxServer(t *testing.T) {
	ctx := context.Background()
	t.Run("absent zeroes the count", func(t *testing.T) {
		got := probeTmuxServer(ctx, &fakeProber{available: true}, tmuxServerState{running: true, sessions: 28})
		if got.running || got.sessions != 0 {
			t.Fatalf("probe = %+v, want absent with 0", got)
		}
	})
	t.Run("running counts sessions", func(t *testing.T) {
		got := probeTmuxServer(ctx, &fakeProber{
			available: true,
			info:      tmux.ServerInfo{Running: true, SocketPath: "/tmp/x/tmux-1000/default"},
			sessions:  []tmux.ServerSession{{Name: "a"}, {Name: "b"}},
		}, tmuxServerState{})
		if !got.running || got.sessions != 2 || got.socket != "/tmp/x/tmux-1000/default" {
			t.Fatalf("probe = %+v", got)
		}
	})
	t.Run("a failed listing cannot fake a loss", func(t *testing.T) {
		got := probeTmuxServer(ctx, &fakeProber{
			available: true,
			info:      tmux.ServerInfo{Running: true},
			listErr:   os.ErrDeadlineExceeded,
		}, tmuxServerState{running: true, sessions: 28})
		if !got.running || got.sessions != 28 {
			t.Fatalf("probe = %+v, want the previous count kept", got)
		}
	})
}

// TestTmuxServerWatchIntegration flips Running for real: a private server on
// an isolated socket, probed through the real Manager. The socket_path
// assertion skips the test if TMUX_TMPDIR isolation is not in force — the
// 2026-09-15 rule is that a test must never observe or touch the live server.
func TestTmuxServerWatchIntegration(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not installed — integration test skipped")
	}
	m := tmux.New()
	if !m.Available() {
		t.Skip("tmux not available")
	}
	tmp := t.TempDir()
	// Scrub what outranks TMUX_TMPDIR ($TMUX, $TMUX_PANE) and restore after.
	for _, key := range []string{"TMUX", "TMUX_PANE", "TMUX_TMPDIR"} {
		if old, ok := os.LookupEnv(key); ok {
			_ = os.Unsetenv(key)
			t.Cleanup(func() { _ = os.Setenv(key, old) })
		}
	}
	t.Setenv("TMUX_TMPDIR", tmp)

	ctx := context.Background()
	name := tmux.SessionName("watch-" + time.Now().Format("150405-000000000"))
	// A session starts the private server; the socket must land under tmp.
	if err := m.NewSession(ctx, name, tmp, "sleep", "30"); err != nil {
		t.Fatalf("fixture session: %v", err)
	}
	info := m.ServerInfo(ctx)
	if !info.Running || !strings.HasPrefix(info.SocketPath, tmp) {
		t.Skipf("fixture socket %q did not land under %q — refusing to run against the live server", info.SocketPath, tmp)
	}

	// Alive: the probe sees the session.
	cur := probeTmuxServer(ctx, m, tmuxServerState{})
	if !cur.running || cur.sessions != 1 {
		t.Fatalf("alive probe = %+v, want running with 1 session", cur)
	}
	// The server dies (exact name, never kill-server on a shared socket).
	if err := m.KillSession(ctx, name); err != nil {
		t.Fatalf("fixture cleanup: %v", err)
	}
	// Zero sessions ends the private server (exit-empty); give it a moment.
	deadline := time.Now().Add(5 * time.Second)
	for {
		cur = probeTmuxServer(ctx, m, cur)
		if !cur.running {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("fixture server still running after its last session closed: %+v", cur)
		}
		time.Sleep(100 * time.Millisecond)
	}
	// The transition records the loss with the count it had.
	rec := &eventRecorder{}
	emitTmuxServerTransitions(rec, tmuxServerState{running: true, socket: info.SocketPath, sessions: 1}, cur, time.Now().UTC())
	if got := rec.types(); len(got) != 1 || got[0] != "terminal.server_lost" {
		t.Fatalf("events = %v, want terminal.server_lost", got)
	}
}
