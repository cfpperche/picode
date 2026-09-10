// Browser surface proxy tests (ADR-0114): the proxy forwards engine
// frames/status to the client, pacing (config/ack) to the engine, refuses
// input in the read-only phase, and reports honest status envelopes when
// there is no browser or the engine is unreachable.

package server

import (
	context "context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
	"github.com/gorilla/websocket"
)

// fakeEngine is a stand-in for the agent-browser stream server. Frames
// pushed on `frames` fan out to every open connection.
type fakeEngine struct {
	port      int
	received  chan string
	frames    chan string
	closeConn chan struct{}
	ts        *httptest.Server

	mu    sync.Mutex
	conns map[*websocket.Conn]chan string
}

func (e *fakeEngine) broadcast(msg string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, ch := range e.conns {
		ch <- msg
	}
}

func newFakeEngine(t *testing.T) *fakeEngine {
	t.Helper()
	e := &fakeEngine{
		received:  make(chan string, 8),
		frames:    make(chan string, 8),
		closeConn: make(chan struct{}),
		conns:     map[*websocket.Conn]chan string{},
	}
	go func() {
		for {
			select {
			case msg := <-e.frames:
				e.broadcast(msg)
			case <-e.closeConn:
				return
			}
		}
	}()
	up := websocket.Upgrader{}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		inbox := make(chan string, 8)
		e.mu.Lock()
		e.conns[conn] = inbox
		e.mu.Unlock()
		defer func() {
			e.mu.Lock()
			delete(e.conns, conn)
			e.mu.Unlock()
		}()
		go func() {
			for {
				select {
				case msg := <-inbox:
					if err := conn.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
						return
					}
				case <-e.closeConn:
					return
				}
			}
		}()
		for {
			mt, data, err := conn.ReadMessage()
			if err != nil {
				return
			}
			if mt == websocket.TextMessage {
				e.received <- string(data)
			}
		}
	}))
	e.ts = ts
	t.Cleanup(func() { close(e.closeConn); ts.Close() })
	if u, err := url.Parse(ts.URL); err == nil {
		e.port, _ = strconv.Atoi(u.Port())
	}
	return e
}

func dialBrowser(t *testing.T, ts *httptest.Server, agent string) *websocket.Conn {
	t.Helper()
	ws, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(ts.URL, "http")+"/ws/browser?agent="+agent, nil)
	if err != nil {
		t.Fatalf("dial /ws/browser: %v", err)
	}
	t.Cleanup(func() { ws.Close() })
	return ws
}

func readMsg(t *testing.T, ws *websocket.Conn) string {
	t.Helper()
	ws.SetReadDeadline(time.Now().Add(3 * time.Second))
	_, data, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	return string(data)
}

func TestBrowserWSNoBrowser(t *testing.T) {
	ts := httptest.NewServer(New("127.0.0.1:0", Deps{}).Handler)
	t.Cleanup(ts.Close)
	ws := dialBrowser(t, ts, "missing-agent")
	got := readMsg(t, ws)
	if !strings.Contains(got, `"reason":"no-browser"`) {
		t.Fatalf("expected no-browser status, got %s", got)
	}
}

func TestBrowserWSEndToEnd(t *testing.T) {
	engine := newFakeEngine(t)
	deps := Deps{BrowserStream: func(ctx context.Context, agentID string) (int, bool) {
		return engine.port, engine.port > 0
	}}
	ts := httptest.NewServer(New("127.0.0.1:0", deps).Handler)
	t.Cleanup(ts.Close)

	ws := dialBrowser(t, ts, "a1")

	// Handshake: wait until the proxy is pumping to the engine, so pushed
	// frames cannot race the proxy's own engine dial.
	syncEngine := func() {
		t.Helper()
		if err := ws.WriteMessage(websocket.TextMessage, []byte(`{"type":"config","maxFps":10}`)); err != nil {
			t.Fatal(err)
		}
		select {
		case <-engine.received:
		case <-time.After(3 * time.Second):
			t.Fatal("engine handshake timed out")
		}
	}
	syncEngine()

	// Frame flows engine → client verbatim.
	engine.frames <- `{"type":"frame","seq":41,"data":"ZmFrZQ=="}`
	if got := readMsg(t, ws); !strings.Contains(got, `"seq":41`) {
		t.Fatalf("frame not forwarded: %s", got)
	}

	// Pacing flows client → engine.
	if err := ws.WriteMessage(websocket.TextMessage, []byte(`{"type":"config","maxFps":10}`)); err != nil {
		t.Fatal(err)
	}
	select {
	case got := <-engine.received:
		if !strings.Contains(got, `"maxFps":10`) {
			t.Fatalf("config not forwarded: %s", got)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("config never reached the engine")
	}

	// Input is refused while the session has no consent (ADR-0115), and
	// never reaches the engine.
	if err := ws.WriteMessage(websocket.TextMessage, []byte(`{"type":"input_mouse","eventType":"mousePressed","x":1,"y":2}`)); err != nil {
		t.Fatal(err)
	}
	got := readMsg(t, ws)
	if !strings.Contains(got, `"code":"watch-only"`) {
		t.Fatalf("expected watch-only refusal, got %s", got)
	}
	select {
	case reached := <-engine.received:
		t.Fatalf("input reached the engine without consent: %s", reached)
	case <-time.After(300 * time.Millisecond):
	}

	// Unknown types are refused as unsupported.
	if err := ws.WriteMessage(websocket.TextMessage, []byte(`{"type":"reboot","x":1}`)); err != nil {
		t.Fatal(err)
	}
	got = readMsg(t, ws)
	if !strings.Contains(got, `"code":"unsupported"`) {
		t.Fatalf("expected unsupported refusal, got %s", got)
	}

	// Unknown engine types are dropped, not fanned out. A fresh connection,
	// because an exceeded read deadline poisons a gorilla connection.
	ws2 := dialBrowser(t, ts, "a1")
	if err := ws2.WriteMessage(websocket.TextMessage, []byte(`{"type":"config","maxFps":12}`)); err != nil {
		t.Fatal(err)
	}
	select {
	case <-engine.received:
	case <-time.After(3 * time.Second):
		t.Fatal("engine handshake timed out on the second connection")
	}
	engine.frames <- `{"type":"somesecret","payload":"x"}`
	engine.frames <- `{"type":"status","connected":true}`
	got2 := readMsg(t, ws2)
	if strings.Contains(got2, "somesecret") {
		t.Fatalf("unknown engine type reached the client: %s", got2)
	}
	if !strings.Contains(got2, `"status"`) {
		t.Fatalf("status after the dropped message missing: %s", got2)
	}
}

// With consent on, input messages reach the engine verbatim (ADR-0115).
func TestBrowserWSInputWithConsent(t *testing.T) {
	engine := newFakeEngine(t)
	consents := true
	deps := Deps{
		BrowserStream: func(ctx context.Context, agentID string) (int, bool) {
			return engine.port, engine.port > 0
		},
		BrowserInputConsent: func(ctx context.Context, agentID string) bool { return consents },
	}
	ts := httptest.NewServer(New("127.0.0.1:0", deps).Handler)
	t.Cleanup(ts.Close)

	ws := dialBrowser(t, ts, "a1")
	if err := ws.WriteMessage(websocket.TextMessage, []byte(`{"type":"input_mouse","eventType":"mousePressed","x":10,"y":20,"button":"left","clickCount":1}`)); err != nil {
		t.Fatal(err)
	}
	select {
	case got := <-engine.received:
		if !strings.Contains(got, `"mousePressed"`) {
			t.Fatalf("input not forwarded verbatim: %s", got)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("consented input never reached the engine")
	}

	// Consent off again: the very next input is refused.
	consents = false
	if err := ws.WriteMessage(websocket.TextMessage, []byte(`{"type":"input_keyboard","eventType":"char","text":"a"}`)); err != nil {
		t.Fatal(err)
	}
	got := readMsg(t, ws)
	if !strings.Contains(got, `"code":"watch-only"`) {
		t.Fatalf("expected watch-only after consent off, got %s", got)
	}
}

// The engine dying must end the client connection (the surface shows
// "ended"), never leave it hanging on a dead pipe (regression).
func TestBrowserWSEngineDeathClosesClient(t *testing.T) {
	engine := newFakeEngine(t)
	deps := Deps{BrowserStream: func(ctx context.Context, agentID string) (int, bool) {
		return engine.port, engine.port > 0
	}}
	ts := httptest.NewServer(New("127.0.0.1:0", deps).Handler)
	t.Cleanup(ts.Close)

	ws := dialBrowser(t, ts, "a1")
	// Handshake so the proxy is fully pumping.
	if err := ws.WriteMessage(websocket.TextMessage, []byte(`{"type":"config","maxFps":10}`)); err != nil {
		t.Fatal(err)
	}
	select {
	case <-engine.received:
	case <-time.After(3 * time.Second):
		t.Fatal("handshake timed out")
	}

	// Kill the engine's connections (engine process death): the proxy must
	// close the client side, not leave it on a dead pipe.
	engine.ts.CloseClientConnections()
	ws.SetReadDeadline(time.Now().Add(3 * time.Second))
	if _, _, err := ws.ReadMessage(); err == nil {
		t.Fatal("client connection stayed open after the engine died")
	}
}

// The consent toggle route flips the mirror the proxy reads (ADR-0115).
func TestBrowserInputRoute(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "picode.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	ts := httptest.NewServer(New("127.0.0.1:0", Deps{Store: st, Tmux: tmux.New()}).Handler)
	t.Cleanup(ts.Close)

	// Unknown agent → 404 before anything else.
	res, err := http.Post(ts.URL+"/api/agents/nope/browser-input", "application/json", strings.NewReader(`{"on":true}`))
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown agent status = %d, want 404", res.StatusCode)
	}

	// Real but stopped agent → 409 (the mirror belongs to a live session).
	ws, err := st.AddWorkspace("w", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ag, err := st.AddAgent(ws.ID, "atlas", "")
	if err != nil {
		t.Fatal(err)
	}
	res, err = http.Post(ts.URL+"/api/agents/"+ag.ID+"/browser-input", "application/json", strings.NewReader(`{"on":true}`))
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusConflict {
		t.Fatalf("stopped agent status = %d, want 409", res.StatusCode)
	}
}
