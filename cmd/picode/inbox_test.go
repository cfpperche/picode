package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// newInboxCLITestServer is a minimal daemon for the inbox door: it files
// items and answers them the way the human's Inbox would — POST respond
// records the response and marks the item done.
func newInboxCLITestServer(t *testing.T) (*httptest.Server, *atomic.Bool) {
	t.Helper()
	t.Setenv("PICODE_DATA", t.TempDir()) // no token file: requests stay anonymous
	answered := &atomic.Bool{}
	var nextID atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/inbox", func(w http.ResponseWriter, r *http.Request) {
		var p map[string]any
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		if v, ok := p["title"]; !ok || v == "" {
			http.Error(w, "title is required", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": fmt.Sprintf("in-test-%d", nextID.Add(1))})
	})
	mux.HandleFunc("POST /api/inbox/{id}/respond", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Text string `json:"text"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"id": r.PathValue("id"), "state": "done"})
	})
	mux.HandleFunc("GET /api/inbox/{id}", func(w http.ResponseWriter, r *http.Request) {
		// The done state only exists once a respond has landed, exactly
		// like the real daemon: the asker polls until the human acts.
		state := "unread"
		var response *string
		if answered.Load() {
			s := "respond: sqlite is fine"
			state, response = "done", &s
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"id": r.PathValue("id"), "state": state, "response": response})
	})
	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)
	return ts, answered
}

// fakeClock turns the 2 s poll sleep into an instant, counted tick.
func fakeClock() (now func() time.Time, sleep func(time.Duration), ticks *atomic.Int32) {
	var t0 atomic.Int64
	ticks = &atomic.Int32{}
	return func() time.Time { return time.Unix(t0.Load(), 0) },
		func(d time.Duration) { t0.Add(int64(d)); ticks.Add(1) },
		ticks
}

func TestInboxNotifyHappyPath(t *testing.T) {
	ts, _ := newInboxCLITestServer(t)
	t.Setenv("PICODE_AGENT_ID", "ag-7") // a managed agent files under its own name
	var out, errOut bytes.Buffer
	if code := runInboxNotify([]string{"--url", ts.URL, "--title", "tests are green", "--body", "all 42", "--reason", "ci report"}, &out, &errOut); code != 0 {
		t.Fatalf("notify = %d, stderr %q", code, errOut.String())
	}
	if got := strings.TrimSpace(out.String()); got != "in-test-1" {
		t.Fatalf("stdout = %q, want the item id", got)
	}
	if errOut.Len() != 0 {
		t.Fatalf("stderr = %q, want clean", errOut.String())
	}
}

func TestInboxPayloadShapeAndIdentity(t *testing.T) {
	// Identity is ADR-0134's assertion model: PiCode names managed agents
	// and its own terminals; a bare guest files as system, honestly.
	kind, source := inboxIdentity(func(string) string { return "" })
	if kind != "system" || source == "" {
		t.Fatalf("identity without env = %q/%q, want an honest system source", kind, source)
	}
	kind, source = inboxIdentity(func(k string) string {
		if k == "PICODE_TERM_ID" {
			return "term-9"
		}
		return ""
	})
	if kind != "terminal" || source != "term-9" {
		t.Fatalf("terminal identity = %q/%q", kind, source)
	}
	kind, source = inboxIdentity(func(string) string { return "ag-1" })
	if kind != "agent" || source != "ag-1" {
		t.Fatalf("agent identity beats terminal = %q/%q", kind, source)
	}

	// The payload is the daemon's contract, shared with pi-inbox and
	// `picode mcp` — pinned field by field.
	p, err := inboxNotifyPayload("system", "goat@box", "  tests are green  ", " all 42 ", "")
	if err != nil {
		t.Fatalf("notify payload: %v", err)
	}
	want := inboxPayload{Kind: "fyi", SourceKind: "system", SourceID: "goat@box",
		Reason: "agent notification", Title: "tests are green", Body: "all 42", Blocking: false}
	if p != want {
		t.Fatalf("notify payload = %+v, want %+v (default reason, trimmed fields)", p, want)
	}
	if _, err := inboxNotifyPayload("system", "x", "   ", "", ""); err == nil {
		t.Fatal("notify without a title must be refused")
	}

	p, err = inboxAskPayload("agent", "ag-1", "postgres or sqlite?", "one service, two devs")
	if err != nil {
		t.Fatalf("ask payload: %v", err)
	}
	if p.Kind != "question" || !p.Blocking || p.Reason != "agent needs your input" ||
		p.Title != "postgres or sqlite?" || p.Body != "postgres or sqlite?\n\none service, two devs" {
		t.Fatalf("ask payload = %+v (blocking, context joins the body)", p)
	}
	p, err = inboxAskPayload("agent", "ag-1", "postgres or sqlite?", "")
	if err != nil || p.Body != "postgres or sqlite?" {
		t.Fatalf("ask without context = %+v, %v", p, err)
	}
	if long := inboxClip(strings.Repeat("x", 300), inboxMaxTitle); len([]rune(long)) != inboxMaxTitle {
		t.Fatalf("clip = %d runes, want %d", len([]rune(long)), inboxMaxTitle)
	}
	if _, err := inboxAskPayload("agent", "ag-1", "  ", ""); err == nil {
		t.Fatal("ask without a question must be refused")
	}
}

func TestInboxNotifyDaemonUnreachable(t *testing.T) {
	newInboxCLITestServer(t) // isolates PICODE_DATA
	t.Setenv("PICODE_URL", "")
	t.Setenv("PICODE_TERM_URL", "")
	var out, errOut bytes.Buffer
	// No server.json anywhere and no --url: one soft line, exit 1.
	if code := runInboxNotify([]string{"--title", "x"}, &out, &errOut); code != 1 {
		t.Fatalf("no daemon = %d", code)
	}
	if lines := strings.Count(strings.TrimSpace(errOut.String()), "\n") + 1; lines != 1 || out.Len() != 0 {
		t.Fatalf("stderr = %q (%d lines), stdout %q", errOut.String(), lines, out.String())
	}
	errOut.Reset()
	// A dial that cannot connect is the same soft one-liner. A closed
	// httptest server answers with an instant RST — a filtered port would
	// hang on the client's 90 s timeout instead.
	dead := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	target := dead.URL
	dead.Close()
	if code := runInboxNotify([]string{"--url", target, "--title", "x"}, &out, &errOut); code != 1 {
		t.Fatalf("refused = %d", code)
	}
	if !strings.Contains(errOut.String(), "picode inbox:") {
		t.Fatalf("stderr = %q", errOut.String())
	}
}

func TestInboxFlagParsing(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := inboxMain(nil, &out, &errOut); code != 2 || !strings.Contains(errOut.String(), "usage: picode inbox") {
		t.Fatalf("bare inbox = %d, stderr %q", code, errOut.String())
	}
	errOut.Reset()
	if code := inboxMain([]string{"fly"}, &out, &errOut); code != 2 || !strings.Contains(errOut.String(), "unknown command") {
		t.Fatalf("unknown subcommand = %d, stderr %q", code, errOut.String())
	}
	errOut.Reset()
	if code := inboxMain([]string{"notify"}, &out, &errOut); code != 2 {
		t.Fatalf("notify without --title = %d, stderr %q", code, errOut.String())
	}
	errOut.Reset()
	if code := inboxMain([]string{"ask"}, &out, &errOut); code != 2 {
		t.Fatalf("ask without --question = %d, stderr %q", code, errOut.String())
	}
}

func TestInboxAskWithoutWait(t *testing.T) {
	ts, _ := newInboxCLITestServer(t)
	var out, errOut bytes.Buffer
	if code := runInboxAsk([]string{"--url", ts.URL, "--question", "postgres or sqlite?"}, &out, &errOut, time.Sleep, time.Now); code != 0 {
		t.Fatalf("ask = %d, stderr %q", code, errOut.String())
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 || lines[0] != "in-test-1" || lines[1] != "Answer in the Inbox app." {
		t.Fatalf("stdout = %q", out.String())
	}
	if errOut.Len() != 0 {
		t.Fatalf("stderr = %q, want clean without --wait", errOut.String())
	}
}

func TestInboxAskWaitFlow(t *testing.T) {
	ts, answered := newInboxCLITestServer(t)
	now, clockSleep, _ := fakeClock()
	var out, errOut bytes.Buffer

	// The human, once the asker has started waiting: answer through the
	// real respond route; the daemon then reports the item done.
	kick := make(chan struct{}, 4)
	go func() {
		for range kick {
			body := strings.NewReader(`{"verb":"respond","text":"sqlite is fine"}`)
			res, err := http.Post(ts.URL+"/api/inbox/in-test-1/respond", "application/json", body)
			if err != nil {
				t.Error("human respond:", err)
				return
			}
			res.Body.Close()
			answered.Store(true)
			return
		}
	}()

	code := runInboxAsk([]string{"--url", ts.URL, "--question", "postgres or sqlite?", "--wait"},
		&out, &errOut, func(d time.Duration) {
			select {
			case kick <- struct{}{}:
			default:
			}
			clockSleep(d)
		}, now)
	if code != 0 {
		t.Fatalf("ask --wait = %d, stderr %q", code, errOut.String())
	}
	// The human's answer is the only thing on stdout; the waiting note
	// went to stderr with the item id.
	if got := strings.TrimSpace(out.String()); got != "The human answered: sqlite is fine" {
		t.Fatalf("stdout = %q", got)
	}
	if !strings.Contains(errOut.String(), "waiting for your answer in the Inbox (item in-test-1)") {
		t.Fatalf("stderr = %q", errOut.String())
	}
}

func TestInboxAskWaitTimeout(t *testing.T) {
	ts, _ := newInboxCLITestServer(t)
	now, sleep, _ := fakeClock()
	var out, errOut bytes.Buffer
	if code := runInboxAsk([]string{"--url", ts.URL, "--question", "postgres or sqlite?", "--wait", "--timeout", "5s"}, &out, &errOut, sleep, now); code != 1 {
		t.Fatalf("timed wait = %d", code)
	}
	if out.Len() != 0 {
		t.Fatalf("stdout = %q, want empty on timeout", out.String())
	}
	if !strings.Contains(errOut.String(), "no answer in 5s") || !strings.Contains(errOut.String(), "in-test-1") {
		t.Fatalf("stderr = %q", errOut.String())
	}
}

func TestInboxAskWaitItemRemoved(t *testing.T) {
	t.Setenv("PICODE_DATA", t.TempDir())
	var polls atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/inbox", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "in-gone"})
	})
	mux.HandleFunc("GET /api/inbox/in-gone", func(w http.ResponseWriter, r *http.Request) {
		// Cleared by a sweep while the asker waited.
		if polls.Add(1) > 1 {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "in-gone", "state": "unread"})
	})
	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)

	now, sleep, _ := fakeClock()
	var out, errOut bytes.Buffer
	if code := runInboxAsk([]string{"--url", ts.URL, "--question", "still there?", "--wait"}, &out, &errOut, sleep, now); code != 0 {
		t.Fatalf("removed wait = %d, stderr %q", code, errOut.String())
	}
	if !strings.Contains(out.String(), "answered/removed elsewhere") {
		t.Fatalf("stdout = %q", out.String())
	}
}
