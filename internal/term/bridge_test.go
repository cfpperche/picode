package term

import (
	"context"
	"net/http/httptest"
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

// dial connects a test client to the bridge over an httptest server.
func dial(t *testing.T, target string) *websocket.Conn {
	t.Helper()
	ts := httptest.NewServer(Bridge(tmux.New()))
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

func TestBridgeEchoThroughTmux(t *testing.T) {
	tm := tmux.New()
	if !tm.Available() {
		t.Skip("tmux not installed — integration test skipped (accepted, see docs/handoff.md)")
	}
	name := tmux.SessionName("bridge-echo")
	startTestSession(t, tm, name)

	ws := dial(t, name)

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

	ws := dial(t, name)
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
