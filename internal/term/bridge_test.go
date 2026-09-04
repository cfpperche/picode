package term

import (
	"context"
	"net/http/httptest"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/cfpperche/picode/internal/tmux"
)

func startTestSession(t *testing.T, tm *tmux.Manager, name string) {
	t.Helper()
	ctx := context.Background()
	if err := tm.NewSession(ctx, name, t.TempDir(), "cat"); err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	t.Cleanup(func() { _ = tm.KillSession(ctx, name) })
}

// dial connects a test client to the bridge over an httptest server. resolve
// is the session's managed tmux options; nil means PiCode manages none.
func dial(t *testing.T, target string, resolve func(string) []tmux.ScopedValue, observers ...func(string)) *websocket.Conn {
	t.Helper()
	var observe func(string)
	if len(observers) > 0 {
		observe = observers[0]
	}
	ts := httptest.NewServer(Bridge(tmux.New(), resolve, observe))
	t.Cleanup(ts.Close)
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws/term?session=" + target
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { _ = ws.Close() })
	return ws
}

func readUntil(t *testing.T, ws *websocket.Conn, timeout time.Duration) []byte {
	t.Helper()
	_ = ws.SetReadDeadline(time.Now().Add(timeout))
	_, data, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	return data
}

type blockingWSConn struct {
	entered chan struct{}
	release chan struct{}
}

func (c *blockingWSConn) SetWriteDeadline(time.Time) error { return nil }

func (c *blockingWSConn) WriteMessage(int, []byte) error {
	c.entered <- struct{}{}
	<-c.release
	return nil
}

func TestWSWriterSerializesConcurrentWrites(t *testing.T) {
	conn := &blockingWSConn{
		entered: make(chan struct{}, 2),
		release: make(chan struct{}),
	}
	out := &wsWriter{conn: conn}

	firstDone := make(chan error, 1)
	go func() { firstDone <- out.write(websocket.BinaryMessage, []byte("first")) }()
	select {
	case <-conn.entered:
	case <-time.After(time.Second):
		t.Fatal("first write did not reach the connection")
	}

	secondReady := make(chan struct{})
	secondDone := make(chan error, 1)
	go func() {
		close(secondReady)
		secondDone <- out.write(websocket.PingMessage, nil)
	}()
	<-secondReady
	select {
	case <-conn.entered:
		t.Fatal("second write entered while the first write was blocked")
	case <-time.After(100 * time.Millisecond):
	}

	close(conn.release)
	for name, done := range map[string]chan error{
		"first":  firstDone,
		"second": secondDone,
	} {
		select {
		case err := <-done:
			if err != nil {
				t.Fatalf("%s write: %v", name, err)
			}
		case <-time.After(time.Second):
			t.Fatalf("%s write did not finish", name)
		}
	}
}

func TestInterruptInput(t *testing.T) {
	for _, tc := range []struct {
		name string
		data []byte
		want bool
	}{
		{name: "escape", data: []byte{0x1b}, want: true},
		{name: "ctrl-c", data: []byte{0x03}, want: true},
		{name: "coalesced ctrl-c", data: []byte{'x', 0x03}, want: true},
		{name: "up arrow", data: []byte("\x1b[A")},
		{name: "alt key", data: []byte("\x1bx")},
		{name: "text", data: []byte("hello")},
		{name: "empty"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := isInterruptInput(tc.data); got != tc.want {
				t.Fatalf("isInterruptInput(%q) = %v, want %v", tc.data, got, tc.want)
			}
		})
	}
}

func TestBridgeObservesInterrupt(t *testing.T) {
	tm := tmux.New()
	if !tm.Available() {
		t.Skip("tmux not installed")
	}
	name := tmux.SessionName("bridge-interrupt")
	startTestSession(t, tm, name)

	observed := make(chan string, 1)
	ws := dial(t, name, nil, func(session string) { observed <- session })
	_ = readUntil(t, ws, 2*time.Second) // initial pane draw
	if err := ws.WriteMessage(websocket.BinaryMessage, []byte{0x1b}); err != nil {
		t.Fatalf("write Escape: %v", err)
	}
	select {
	case got := <-observed:
		if got != name {
			t.Fatalf("observed session = %q, want %q", got, name)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("interrupt observer was not called")
	}
}

func TestBridgeEchoThroughTmux(t *testing.T) {
	tm := tmux.New()
	if !tm.Available() {
		t.Skip("tmux not installed — integration test skipped (accepted, see docs/handoff.md)")
	}
	name := tmux.SessionName("bridge-echo")
	startTestSession(t, tm, name)

	ws := dial(t, name, nil)

	// tmux draws the pane content on attach; `cat` echoes stdin back.
	// Send bytes and expect them to come back through the pty.
	if err := ws.WriteMessage(websocket.BinaryMessage, []byte("hello-picode\r")); err != nil {
		t.Fatalf("write: %v", err)
	}

	var got string
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		data := readUntil(t, ws, 2*time.Second)
		got += string(data)
		if strings.Contains(got, "hello-picode") {
			return // echo observed: round trip works
		}
	}
	t.Fatalf("echo not observed within timeout; got so far: %q", got)
}

func TestBridgeResizeControl(t *testing.T) {
	tm := tmux.New()
	if !tm.Available() {
		t.Skip("tmux not installed — integration test skipped (accepted, see docs/handoff.md)")
	}
	name := tmux.SessionName("bridge-resize")
	startTestSession(t, tm, name)

	ws := dial(t, name, nil)
	_ = readUntil(t, ws, 2*time.Second) // initial pane draw

	ctrl := `{"type":"resize","cols":120,"rows":40}`
	if err := ws.WriteMessage(websocket.TextMessage, []byte(ctrl)); err != nil {
		t.Fatalf("write control: %v", err)
	}

	// The resize itself may produce no output for a bare `cat` pane.
	// What matters: the pipeline stays alive — prove it with an echo
	// round-trip after the control message.
	if err := ws.WriteMessage(websocket.BinaryMessage, []byte("post-resize\r")); err != nil {
		t.Fatalf("write after resize: %v", err)
	}
	var got string
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		data := readUntil(t, ws, 2*time.Second)
		got += string(data)
		if strings.Contains(got, "post-resize") {
			return // pipeline alive after resize: control accepted
		}
		if strings.Contains(got, `"type":"error"`) {
			t.Fatalf("error frame after resize: %q", got)
		}
	}
	t.Fatalf("pipeline dead after resize; got so far: %q", got)
}

func TestBridgeRejectsForeignSessions(t *testing.T) {
	cases := []string{"", "other-session", "picode-", "picode-evil.name"}
	for _, c := range cases {
		if tmux.OwnedSessionName(c) {
			t.Errorf("OwnedSessionName(%q) = true, want false", c)
		}
	}
	if !tmux.OwnedSessionName("picode-ok") {
		t.Error("OwnedSessionName(picode-ok) = false, want true")
	}
}

// The options a session runs with are applied on attach, not only when it is
// created. A setting changed while the terminal was closed has to take hold
// when it is opened again, and a session that predates the setting entirely
// has to heal itself rather than staying odd until it is killed.
func TestBridgeAppliesResolvedOptionsOnAttach(t *testing.T) {
	tm := tmux.New()
	if !tm.Available() {
		t.Skip("tmux not installed")
	}
	name := tmux.SessionName("bridge-opts")
	startTestSession(t, tm, name)

	// Start from the opposite value, so a bridge that applies nothing fails
	// rather than coasting on a default that happens to match.
	if err := tm.SetOption(context.Background(), name, "mouse", "off"); err != nil {
		t.Fatalf("seed: %v", err)
	}

	dial(t, name, func(session string) []tmux.ScopedValue {
		if session != name {
			t.Errorf("resolver asked about %q, want %q", session, name)
		}
		return []tmux.ScopedValue{{Scope: tmux.ScopeSession, Key: "mouse", Value: "on"}}
	})

	deadline := time.Now().Add(3 * time.Second)
	for {
		if got := sessionOption(t, name, "mouse"); got == "on" {
			return
		} else if time.Now().After(deadline) {
			t.Fatalf("mouse = %q after attach, want on", got)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func sessionOption(t *testing.T, session, option string) string {
	t.Helper()
	out, err := exec.Command("tmux", "show-options", "-t", session+":", "-v", option).Output()
	if err != nil {
		return "" // an unset option reports an error, and unset is not "on"
	}
	return strings.TrimSpace(string(out))
}
