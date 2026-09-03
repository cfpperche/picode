package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/feed"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

func TestTermStateRegistrySweep(t *testing.T) {
	now := time.Now()
	ts := NewTermStates()

	if _, changed := ts.Set("t1", TermWorking, "claude-code", now); !changed {
		t.Fatal("first set must report changed")
	}
	if _, changed := ts.Set("t1", TermWorking, "claude-code", now); changed {
		t.Fatal("repeat of the same report is not a change")
	}
	ts.Set("t2", TermNeedsYou, "codex", now)
	ts.Set("t3", TermIdle, "claude-code", now)

	got, ok := ts.Get("t1")
	if !ok || got.State != TermWorking || got.CLI != "claude-code" {
		t.Fatalf("get t1 = %+v ok=%v", got, ok)
	}

	// Sweep: stale working clears; needs-you and idle never decay
	// (needs-you waits for the human by definition).
	changed := ts.Sweep(now.Add(workingTTL+time.Minute), workingTTL)
	if len(changed) != 1 || changed[0] != "t1" {
		t.Fatalf("sweep = %v, want [t1]", changed)
	}
	if _, ok := ts.Get("t1"); ok {
		t.Fatal("stale working must be gone")
	}
	if _, ok := ts.Get("t2"); !ok {
		t.Fatal("needs-you must survive the sweep")
	}
	if _, ok := ts.Get("t3"); !ok {
		t.Fatal("idle must survive the sweep")
	}

	// Fresh working survives the same sweep.
	ts.Set("t4", TermWorking, "", now.Add(time.Minute))
	if c := ts.Sweep(now.Add(workingTTL+time.Minute), workingTTL); len(c) != 0 {
		t.Fatalf("fresh working swept: %v", c)
	}

	if !ts.Drop("t2") || ts.Drop("t2") {
		t.Fatal("drop must report whether anything was held")
	}
	if _, ok := ts.Get("t2"); ok {
		t.Fatal("dropped state must be gone")
	}
}

// TestTermStateRoundTrip walks the HTTP surface against a real terminal:
// 404 for unknown ids, 400 for a foreign state word, and a full
// set → feed event → list carries it → delete clears it round trip.
// Gated on tmux like the other terminal tests (creation opens a session).
func TestTermStateRoundTrip(t *testing.T) {
	if !tmux.New().Available() {
		t.Skip("tmux not installed — create/open gated (accepted)")
	}
	root := t.TempDir()
	st, err := store.Open(filepath.Join(root, "picode.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	f := &feed.Feed{Store: st}
	st.OnEvent = f.Publish
	events := make(chan store.Event, 16)
	f.Listen(func(ev store.Event) {
		select {
		case events <- ev:
		default:
		}
	})
	ts := httptest.NewServer(New("127.0.0.1:0", Deps{
		Store: st, Tmux: tmux.New(), AgentCmd: "cat", Feed: f, TermStates: NewTermStates(),
	}).Handler)
	t.Cleanup(ts.Close)

	// Unknown terminal first: no tmux work needed to reject it.
	if res := postJSON(t, ts, "/api/terminals/nope/state", map[string]any{"state": "working"}); res.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown terminal = %d, want 404", res.StatusCode)
	}

	created := postJSON(t, ts, "/api/terminals", map[string]any{})
	if created.StatusCode != http.StatusCreated {
		t.Fatalf("create = %d", created.StatusCode)
	}
	var page map[string]any
	_ = json.NewDecoder(created.Body).Decode(&page)
	id, _ := page["id"].(string)
	sess, _ := page["session"].(string)
	t.Cleanup(func() { _ = tmux.New().KillSession(t.Context(), sess) })

	bad := postJSON(t, ts, "/api/terminals/"+id+"/state", map[string]any{"state": "vibing"})
	if bad.StatusCode != http.StatusBadRequest {
		t.Fatalf("foreign state = %d, want 400", bad.StatusCode)
	}
	noop := postJSON(t, ts, "/api/terminals/"+id+"/state", map[string]any{"state": "working", "cli": "claude-code"})
	if noop.StatusCode != http.StatusOK {
		t.Fatalf("first set = %d", noop.StatusCode)
	}
	// The first report publishes terminal.state (the channel may carry
	// unrelated durable events from the create — scan past them).
	ev := waitForEventType(t, events, "terminal.state")
	var body struct {
		State string `json:"state"`
		CLI   string `json:"cli"`
	}
	if err := json.Unmarshal(ev.Data, &body); err != nil || body.State != TermWorking || body.CLI != "claude-code" {
		t.Fatalf("event data = %s (%v)", ev.Data, err)
	}
	// A repeat is not a change and must not publish.
	repeat := postJSON(t, ts, "/api/terminals/"+id+"/state", map[string]any{"state": "working", "cli": "claude-code"})
	if repeat.StatusCode != http.StatusOK {
		t.Fatalf("repeat set = %d", repeat.StatusCode)
	}
	select {
	case ev := <-events:
		if ev.Type == "terminal.state" {
			t.Fatal("repeat published terminal.state")
		}
	case <-time.After(150 * time.Millisecond):
	}

	listed := do(t, ts.Client(), mustGet(t, ts.URL+"/api/terminals"))
	if listed.StatusCode != http.StatusOK {
		t.Fatalf("list = %d", listed.StatusCode)
	}
	var page2 struct {
		Terminals []map[string]any `json:"terminals"`
	}
	_ = json.NewDecoder(listed.Body).Decode(&page2)
	var found map[string]any
	for _, term := range page2.Terminals {
		if term["id"] == id {
			found = term
		}
	}
	if found == nil || found["state"] != TermWorking || found["cli"] != "claude-code" {
		t.Fatalf("list view = %+v, want working/claude-code", found)
	}

	// Deleting the terminal drops its state: the next report targets a
	// record that is gone.
	req, _ := http.NewRequest(http.MethodDelete, ts.URL+"/api/terminals/"+id, nil)
	if res := do(t, ts.Client(), req); res.StatusCode != http.StatusNoContent {
		t.Fatalf("delete = %d", res.StatusCode)
	}
	if res := postJSON(t, ts, "/api/terminals/"+id+"/state", map[string]any{"state": "idle"}); res.StatusCode != http.StatusNotFound {
		t.Fatalf("state after delete = %d, want 404", res.StatusCode)
	}
}

// waitForEventType scans past unrelated events (every store mutation
// publishes) until one of the wanted type arrives.
func waitForEventType(t *testing.T, ch chan store.Event, want string) store.Event {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for {
		select {
		case ev := <-ch:
			if ev.Type == want {
				return ev
			}
		case <-deadline:
			t.Fatalf("no %s event arrived", want)
		}
	}
}
