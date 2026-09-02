package server

import (
	"bufio"
	"bytes"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/feed"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

type sseFrame struct {
	ID    string
	Event string
	Data  string
}

// readFrames reads SSE frames until n arrive or the deadline passes.
func readFrames(t *testing.T, body *bufio.Reader, n int, d time.Duration) []sseFrame {
	t.Helper()
	var out []sseFrame
	cur := sseFrame{}
	done := make(chan struct{})
	go func() {
		defer close(done)
		for len(out) < n {
			line, err := body.ReadString('\n')
			if err != nil {
				return
			}
			line = strings.TrimRight(line, "\r\n")
			switch {
			case line == "":
				if cur.Event != "" || cur.Data != "" {
					out = append(out, cur)
				}
				cur = sseFrame{}
			case strings.HasPrefix(line, "id: "):
				cur.ID = line[4:]
			case strings.HasPrefix(line, "event: "):
				cur.Event = line[7:]
			case strings.HasPrefix(line, "data: "):
				cur.Data = line[6:]
			}
		}
	}()
	select {
	case <-done:
	case <-time.After(d):
		t.Fatalf("got %d frames, want %d: %+v", len(out), n, out)
	}
	return out
}

func newFeedServer(t *testing.T) (*httptest.Server, *store.Store, *feed.Feed) {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "picode.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	f := &feed.Feed{Store: st}
	st.OnEvent = f.Publish
	ts := httptest.NewServer(New("127.0.0.1:0", Deps{Store: st, Tmux: tmux.New(), AgentCmd: "cat", Feed: f}).Handler)
	t.Cleanup(ts.Close)
	return ts, st, f
}

func openStream(t *testing.T, ts *httptest.Server, lastID string) (*bufio.Reader, func()) {
	t.Helper()
	req, _ := http.NewRequest("GET", ts.URL+"/api/events", nil)
	if lastID != "" {
		req.Header.Set("Last-Event-ID", lastID)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 200 || !strings.HasPrefix(res.Header.Get("Content-Type"), "text/event-stream") {
		t.Fatalf("status %d type %q", res.StatusCode, res.Header.Get("Content-Type"))
	}
	return bufio.NewReader(res.Body), func() { res.Body.Close() }
}

func TestEventsHelloLiveReplayReset(t *testing.T) {
	ts, st, _ := newFeedServer(t)
	body, closeStream := openStream(t, ts, "")
	hello := readFrames(t, body, 1, 3*time.Second)[0]
	if hello.Event != "hello" || !strings.Contains(hello.Data, bootID) {
		t.Fatalf("hello = %+v", hello)
	}

	// A mutation over the API arrives live, with an id.
	res, err := http.Post(ts.URL+"/api/inbox", "application/json", bytes.NewBufferString(`{"kind":"fyi","sourceKind":"system","reason":"qa","title":"note"}`))
	if err != nil || res.StatusCode != http.StatusCreated {
		t.Fatalf("create inbox: %v %v", err, res)
	}
	res.Body.Close()
	live := readFrames(t, body, 1, 3*time.Second)[0]
	if live.Event != "change" || live.ID == "" || !strings.Contains(live.Data, `"type":"inbox.created"`) || !strings.Contains(live.Data, `"title":"note"`) {
		t.Fatalf("live = %+v", live)
	}
	ev, err := eventFrom(live.Data)
	if err != nil || ev.Type != "inbox.created" {
		t.Fatalf("decode = %+v %v", ev, err)
	}
	closeStream()

	// Two events while disconnected are replayed from Last-Event-ID.
	_ = st.SetSetting("a", "1")
	_ = st.SetSetting("b", "2")
	body, closeStream = openStream(t, ts, live.ID)
	frames := readFrames(t, body, 3, 3*time.Second)
	if frames[0].Event != "hello" || frames[1].Event != "change" || !strings.Contains(frames[2].Data, `"type":"setting.updated"`) {
		t.Fatalf("replay = %+v", frames)
	}
	closeStream()

	// A cursor older than retention gets a reset after hello.
	if _, err := st.PruneEvents(time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	_ = st.SetSetting("c", "3")
	body, closeStream = openStream(t, ts, "1")
	frames = readFrames(t, body, 2, 3*time.Second)
	if frames[0].Event != "hello" || frames[1].Event != "reset" {
		t.Fatalf("reset = %+v", frames)
	}
	closeStream()
}

func TestEventsEphemeralHasNoID(t *testing.T) {
	ts, _, f := newFeedServer(t)
	body, closeStream := openStream(t, ts, "")
	defer closeStream()
	readFrames(t, body, 1, 3*time.Second)
	f.Ephemeral("device.online", map[string]string{"id": "d1"})
	fr := readFrames(t, body, 1, 3*time.Second)[0]
	if fr.ID != "" || fr.Event != "change" || !strings.Contains(fr.Data, `"type":"device.online"`) {
		t.Fatalf("ephemeral = %+v", fr)
	}
}

func TestEventsWithoutFeedIs503(t *testing.T) {
	ts := newTestServer(t, "cat")
	res, err := http.Get(ts.URL + "/api/events")
	if err != nil || res.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("no feed = %v %v", err, res)
	}
}
