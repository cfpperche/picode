package server

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/browser"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

// browserServer is the real mux with a real hub, so the test exercises the
// stream, the result POST and the hub between them.
func browserServer(t *testing.T) (*httptest.Server, *browser.Hub) {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "picode.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	hub := browser.New()
	hub.Timeout = 3 * time.Second
	ts := httptest.NewServer(New("127.0.0.1:0", Deps{Store: st, Tmux: tmux.New(), AgentCmd: "cat", Browser: hub}).Handler)
	t.Cleanup(ts.Close)
	return ts, hub
}

func openBrowserStream(t *testing.T, ts *httptest.Server) (*bufio.Reader, func()) {
	t.Helper()
	req, _ := http.NewRequest("GET", ts.URL+"/api/browser/stream", nil)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 200 || !strings.HasPrefix(res.Header.Get("Content-Type"), "text/event-stream") {
		t.Fatalf("status %d type %q", res.StatusCode, res.Header.Get("Content-Type"))
	}
	return bufio.NewReader(res.Body), func() { res.Body.Close() }
}

func postBrowserResult(t *testing.T, ts *httptest.Server, body string) int {
	t.Helper()
	res, err := http.Post(ts.URL+"/api/browser/result", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	return res.StatusCode
}

func TestBrowserStreamRoundTrip(t *testing.T) {
	ts, hub := browserServer(t)
	body, closeStream := openBrowserStream(t, ts)
	defer closeStream()

	hello := readFrames(t, body, 1, 3*time.Second)[0]
	if hello.Event != "hello" || !strings.Contains(hello.Data, `"connected":true`) {
		t.Fatalf("hello = %+v", hello)
	}

	// A tool call (the daemon side) waits for the answer while the command
	// travels down the stream to the shell.
	type answer struct {
		out json.RawMessage
		err error
	}
	done := make(chan answer, 1)
	go func() {
		out, err := hub.Dispatch(context.Background(), browser.Command{
			Method:  "Page.captureScreenshot",
			Params:  json.RawMessage(`{"format":"png"}`),
			Tier:    "read",
			Domains: []string{"example.com"},
		})
		done <- answer{out, err}
	}()

	cmd := readFrames(t, body, 1, 3*time.Second)[0]
	if cmd.Event != "command" {
		t.Fatalf("frame = %+v", cmd)
	}
	var pushed browser.Command
	if err := json.Unmarshal([]byte(cmd.Data), &pushed); err != nil {
		t.Fatal(err)
	}
	if pushed.ID == "" || pushed.Method != "Page.captureScreenshot" || pushed.Tier != "read" {
		t.Fatalf("command = %+v", pushed)
	}
	if len(pushed.Domains) != 1 || pushed.Domains[0] != "example.com" {
		t.Fatalf("domains = %v", pushed.Domains)
	}

	if code := postBrowserResult(t, ts, `{"id":"`+pushed.ID+`","output":{"data":"cG5n"}}`); code != http.StatusNoContent {
		t.Fatalf("result status = %d", code)
	}
	got := <-done
	if got.err != nil || string(got.out) != `{"data":"cG5n"}` {
		t.Fatalf("dispatch = %q, %v", got.out, got.err)
	}
}

func TestBrowserResultErrorAndEdges(t *testing.T) {
	ts, hub := browserServer(t)
	body, closeStream := openBrowserStream(t, ts)
	defer closeStream()
	readFrames(t, body, 1, 3*time.Second) // hello

	// A shell refusal travels back as the tool call's error.
	done := make(chan error, 1)
	go func() {
		_, err := hub.Dispatch(context.Background(), browser.Command{Method: "Runtime.evaluate", Tier: "read"})
		done <- err
	}()
	cmd := readFrames(t, body, 1, 3*time.Second)[0]
	var pushed browser.Command
	if err := json.Unmarshal([]byte(cmd.Data), &pushed); err != nil {
		t.Fatal(err)
	}
	if code := postBrowserResult(t, ts, `{"id":"`+pushed.ID+`","error":"Runtime.evaluate needs the act tier; this agent has read"}`); code != http.StatusNoContent {
		t.Fatalf("result status = %d", code)
	}
	if err := <-done; err == nil || !strings.Contains(err.Error(), "needs the act tier") {
		t.Fatalf("dispatch err = %v", err)
	}

	// A result whose command is gone is refused with a 404, not swallowed.
	if code := postBrowserResult(t, ts, `{"id":"`+pushed.ID+`","output":null}`); code != http.StatusNotFound {
		t.Fatalf("late result status = %d, want 404", code)
	}
	if code := postBrowserResult(t, ts, `{"output":null}`); code != http.StatusBadRequest {
		t.Fatalf("id-less result status = %d, want 400", code)
	}
	if code := postBrowserResult(t, ts, `not json`); code != http.StatusBadRequest {
		t.Fatalf("malformed result status = %d, want 400", code)
	}
}

func TestBrowserToolResolvesAVerbAndWaits(t *testing.T) {
	ts, hub := browserServer(t)
	body, closeStream := openBrowserStream(t, ts)
	defer closeStream()
	readFrames(t, body, 1, 3*time.Second) // hello

	type answer struct {
		status int
		body   string
	}
	done := make(chan answer, 1)
	go func() {
		res, err := http.Post(ts.URL+"/api/browser/tool", "application/json",
			bytes.NewBufferString(`{"agent":"agent-1","verb":"snapshot"}`))
		if err != nil {
			done <- answer{0, err.Error()}
			return
		}
		defer res.Body.Close()
		out, _ := io.ReadAll(res.Body)
		done <- answer{res.StatusCode, string(out)}
	}()

	cmd := readFrames(t, body, 1, 3*time.Second)[0]
	var pushed browser.Command
	if err := json.Unmarshal([]byte(cmd.Data), &pushed); err != nil {
		t.Fatal(err)
	}
	// The tool names a verb; the daemon decides the method and the tier
	// (ADR-0134: read on the tab on screen).
	if pushed.Method != "Accessibility.getFullAXTree" || pushed.Tier != "read" {
		t.Fatalf("command = %+v", pushed)
	}
	if len(pushed.Domains) != 0 {
		t.Fatalf("domains = %v, want none by default", pushed.Domains)
	}
	if code := postBrowserResult(t, ts, `{"id":"`+pushed.ID+`","output":{"nodes":[{"role":"heading"}]}}`); code != http.StatusNoContent {
		t.Fatalf("result status = %d", code)
	}
	got := <-done
	if got.status != http.StatusOK || !strings.Contains(got.body, `"heading"`) || !strings.Contains(got.body, `"snapshot"`) {
		t.Fatalf("tool answer = %d %s", got.status, got.body)
	}
	_ = hub
}

func TestBrowserToolRefusesAnUnknownVerb(t *testing.T) {
	ts, _ := browserServer(t)
	res, err := http.Post(ts.URL+"/api/browser/tool", "application/json",
		bytes.NewBufferString(`{"agent":"agent-1","verb":"evaluate"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	out, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusBadRequest || !strings.Contains(string(out), "snapshot") {
		t.Fatalf("answer = %d %s", res.StatusCode, out)
	}
}
