package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/store"
)

func TestNotifyOutPostsRetriesAndRecords(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "picode.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	old := notifyRetryAfter
	notifyRetryAfter = 10 * time.Millisecond
	t.Cleanup(func() { notifyRetryAfter = old })

	var calls atomic.Int32
	var got atomic.Value
	hook := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		b, _ := io.ReadAll(r.Body)
		got.Store(string(b))
		if n == 1 {
			http.Error(w, "busy", http.StatusBadGateway) // first try fails, the retry lands
			return
		}
		w.WriteHeader(200)
	}))
	t.Cleanup(hook.Close)

	r := &automationRunner{deps: Deps{Store: st, PortSnapshot: func() PortSnapshot { return PortSnapshot{PublicURL: "https://box.tail.ts.net"} }}}
	url := hook.URL
	a := store.Automation{ID: "a1", Name: "Nightly", NotifyURL: &url}
	done := make(chan error, 1)
	r.notifyOut(a, store.Run{ID: "r1", CostUSD: 0.5}, store.RunDone, "", "All green.", done)
	if err := <-done; err != nil {
		t.Fatalf("delivery: %v", err)
	}
	if calls.Load() != 2 {
		t.Fatalf("calls = %d, want a retry", calls.Load())
	}
	var body map[string]any
	_ = json.Unmarshal([]byte(got.Load().(string)), &body)
	text, _ := body["text"].(string)
	if !strings.Contains(text, "✅ *Nightly* ran · $0.50 · <https://box.tail.ts.net/#/automations/a1|Open in PiCode>") || !strings.HasSuffix(text, "All green.") {
		t.Fatalf("text %q", text)
	}
	evs, _ := st.ListEventsSince(0, 50)
	found := false
	for _, e := range evs {
		if e.Type == "automation.notify" {
			found = true
		}
	}
	if !found {
		t.Fatal("no automation.notify event")
	}

	// A 4xx is not retried; the error is recorded.
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); http.Error(w, "no", 404) }))
	t.Cleanup(bad.Close)
	calls.Store(0)
	burl := bad.URL
	a.NotifyURL = &burl
	r.notifyOut(a, store.Run{ID: "r2"}, store.RunFailed, "cost cap", "", done)
	if err := <-done; err == nil || calls.Load() != 1 {
		t.Fatalf("4xx: err=%v calls=%d", err, calls.Load())
	}

	// No URL: nothing happens, no event.
	a.NotifyURL = nil
	r.notifyOut(a, store.Run{ID: "r3"}, store.RunDone, "", "", done)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}
