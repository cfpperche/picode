package webhooks

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
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

func engineStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.Open(filepath.Join(t.TempDir(), "db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}
func add(t *testing.T, s *store.Store, url string) store.Webhook {
	t.Helper()
	w, err := s.AddWebhook(url, []string{"agent."})
	if err != nil {
		t.Fatal(err)
	}
	return w
}
func event(t *testing.T, s *store.Store, kind string) {
	t.Helper()
	if err := s.AppendEvent(kind, nil, nil, map[string]string{"message": "hello"}); err != nil {
		t.Fatal(err)
	}
}

func TestDeliveryDecisionTable(t *testing.T) {
	for _, tc := range []struct {
		name          string
		status        int
		paused, match bool
		calls         int
		state         string
	}{
		{"success", 204, false, true, 1, "delivered"},
		{"server error", 500, false, true, 1, "retrying"},
		{"rate limit", 429, false, true, 1, "retrying"},
		{"auth failure", 401, false, true, 1, "retrying"},
		{"redirect", 302, false, true, 1, "retrying"},
		{"paused", 204, true, true, 0, ""},
		{"filtered", 204, false, false, 0, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := engineStore(t)
			var calls atomic.Int32
			var secret string
			receiver := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				body, _ := io.ReadAll(r.Body)
				h := hmac.New(sha256.New, []byte(secret))
				h.Write([]byte(r.Header.Get("X-Picode-Timestamp") + "."))
				h.Write(body)
				if got := r.Header.Get("X-Picode-Signature"); got != "sha256="+hex.EncodeToString(h.Sum(nil)) {
					t.Errorf("bad signature: %s", got)
				}
				var ev store.Event
				if err := json.Unmarshal(body, &ev); err != nil || ev.Type != "agent.updated" {
					t.Errorf("body: %s %v", body, err)
				}
				if r.Header.Get("X-Picode-Event-Id") == "" || r.Header.Get("X-Picode-Event-Type") != ev.Type {
					t.Error("missing event headers")
				}
				rw.Header().Set("Location", "/should-not-follow")
				rw.WriteHeader(tc.status)
			}))
			defer receiver.Close()
			w := add(t, s, receiver.URL)
			secret = w.Secret
			if tc.paused {
				w.Enabled = false
				var err error
				w, err = s.UpdateWebhook(w)
				if err != nil {
					t.Fatal(err)
				}
			}
			kind := "agent.updated"
			if !tc.match {
				kind = "task.updated"
			}
			event(t, s, kind)
			e := New(s)
			now := time.Now().UTC()
			e.Now = func() time.Time { return now }
			if err := e.DeliverOnce(context.Background(), w.ID); err != nil {
				t.Fatal(err)
			}
			got, _ := s.GetWebhook(w.ID)
			if int(calls.Load()) != tc.calls || got.LastStatus != tc.state {
				t.Fatalf("calls %d state %#v", calls.Load(), got)
			}
			if tc.state == "retrying" {
				if got.Failures != 1 || got.NextAttemptAt != now.Add(time.Minute).Format(time.RFC3339Nano) {
					t.Fatalf("backoff %#v", got)
				}
				// A new engine simulates restart; persisted retry prevents a hot loop.
				restarted := New(s)
				restarted.Now = e.Now
				if err := restarted.DeliverOnce(context.Background(), w.ID); err != nil {
					t.Fatal(err)
				}
				if calls.Load() != 1 {
					t.Fatal("restart lost backoff")
				}
				now = now.Add(time.Minute)
				if err := restarted.DeliverOnce(context.Background(), w.ID); err != nil {
					t.Fatal(err)
				}
				got, _ = s.GetWebhook(w.ID)
				if calls.Load() != 2 || got.Failures != 2 || got.NextAttemptAt != now.Add(2*time.Minute).Format(time.RFC3339Nano) {
					t.Fatal("retry did not progress")
				}
			}
			if tc.state == "delivered" || !tc.match {
				latest, _ := s.LatestEventID()
				for i := 0; i < 3; i++ {
					if err := e.DeliverOnce(context.Background(), w.ID); err != nil {
						t.Fatal(err)
					}
				}
				end, _ := s.LatestEventID()
				if end != latest {
					t.Fatal("webhook events sustained a delivery loop")
				}
				if int(calls.Load()) != tc.calls {
					t.Fatal("success replayed")
				}
			}
		})
	}
}

func TestRetryKeepsEventOrder(t *testing.T) {
	s := engineStore(t)
	var ids []int64
	receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var ev store.Event
		_ = json.NewDecoder(r.Body).Decode(&ev)
		ids = append(ids, ev.ID)
		if len(ids) == 1 {
			w.WriteHeader(503)
		} else {
			w.WriteHeader(204)
		}
	}))
	defer receiver.Close()
	w := add(t, s, receiver.URL)
	event(t, s, "agent.one")
	event(t, s, "agent.two")
	e := New(s)
	now := time.Now().UTC()
	e.Now = func() time.Time { return now }
	for i := 0; i < 3; i++ {
		if err := e.DeliverOnce(context.Background(), w.ID); err != nil {
			t.Fatal(err)
		}
		now = now.Add(time.Minute)
	}
	if len(ids) != 3 || ids[0] != ids[1] || ids[2] <= ids[1] {
		t.Fatalf("order: %v", ids)
	}
}

func TestRetentionGapAndTestIsolation(t *testing.T) {
	s := engineStore(t)
	var calls atomic.Int32
	receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(204) }))
	defer receiver.Close()
	w := add(t, s, receiver.URL)
	event(t, s, "agent.one")
	event(t, s, "agent.two")
	if _, err := s.PruneEvents(time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	event(t, s, "agent.new")
	e := New(s)
	if err := e.DeliverOnce(context.Background(), w.ID); err != nil {
		t.Fatal(err)
	}
	got, _ := s.GetWebhook(w.ID)
	if got.LastStatus != "missed" || calls.Load() != 0 {
		t.Fatalf("gap: %#v calls %d", got, calls.Load())
	}
	rows, err := s.ListEventsSince(0, 100)
	if err != nil || len(rows) == 0 {
		t.Fatalf("gap events: %v %v", rows, err)
	}
	last := rows[len(rows)-1]
	var receipt struct {
		Status string `json:"status"`
		Error  string `json:"error"`
		Cursor int64  `json:"cursor"`
	}
	if err := json.Unmarshal(last.Data, &receipt); err != nil {
		t.Fatal(err)
	}
	if last.Type != "webhook.delivery" || receipt.Status != "missed" || receipt.Error == "" || receipt.Cursor != got.Cursor {
		t.Fatalf("gap not durably recorded: %s", last.Data)
	}
	before := got
	before.Enabled = false
	before, err = s.UpdateWebhook(before)
	if err != nil {
		t.Fatal(err)
	}
	result, err := e.Test(context.Background(), w.ID)
	if err != nil || !result.Delivered {
		t.Fatalf("test: %#v %v", result, err)
	}
	got, _ = s.GetWebhook(w.ID)
	if got.Cursor != before.Cursor || got.LastStatus != before.LastStatus || got.NextAttemptAt != before.NextAttemptAt {
		t.Fatal("test modified delivery state")
	}
}

func TestEditWhilePostingInvalidatesAcknowledgement(t *testing.T) {
	s := engineStore(t)
	entered := make(chan struct{})
	release := make(chan struct{})
	receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { close(entered); <-release; w.WriteHeader(204) }))
	defer receiver.Close()
	w := add(t, s, receiver.URL)
	event(t, s, "agent.one")
	e := New(s)
	done := make(chan error, 1)
	go func() { done <- e.DeliverOnce(context.Background(), w.ID) }()
	<-entered
	w.Enabled = false
	if _, err := s.UpdateWebhook(w); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Test(context.Background(), w.ID); !errors.Is(err, ErrBusy) {
		t.Fatalf("duplicate delivery allowed: %v", err)
	}
	close(release)
	if err := <-done; !errors.Is(err, store.ErrWebhookConflict) {
		t.Fatalf("stale ack: %v", err)
	}
	got, _ := s.GetWebhook(w.ID)
	if got.Cursor != w.Cursor {
		t.Fatal("stale acknowledgement advanced cursor")
	}
}

func TestNetworkFailureDoesNotLeakURL(t *testing.T) {
	s := engineStore(t)
	w := add(t, s, "http://127.0.0.1:1/secret-path?token=secret-token")
	event(t, s, "agent.one")
	e := New(s)
	if err := e.DeliverOnce(context.Background(), w.ID); err != nil {
		t.Fatal(err)
	}
	got, _ := s.GetWebhook(w.ID)
	if got.LastStatus != "retrying" || strings.Contains(got.LastError, "secret") {
		t.Fatalf("error leaked: %s", got.LastError)
	}
}

func TestBackoffAndFilters(t *testing.T) {
	for i, want := range []time.Duration{time.Minute, 2 * time.Minute, 4 * time.Minute, 8 * time.Minute, 16 * time.Minute, 32 * time.Minute, time.Hour, time.Hour} {
		if got := Backoff(i + 1); got != want {
			t.Fatalf("%d: %v", i, got)
		}
	}
	if Matches([]string{"web"}, "webhook.created") || Matches(nil, "agent.one") || !Matches([]string{"agent."}, "agent.one") {
		t.Fatal("prefix contract")
	}
}
