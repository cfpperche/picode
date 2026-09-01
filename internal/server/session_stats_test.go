package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/session"
)

func TestNormalizeRange(t *testing.T) {
	for _, c := range []struct{ raw, want string }{
		{"", "7d"},
		{"nope", "7d"},
		{"today", "today"},
		{"7d", "7d"},
		{"30d", "30d"},
		{"all", "all"},
		{"  30d  ", "30d"},
	} {
		if got := normalizeRange(c.raw); got != c.want {
			t.Fatalf("normalizeRange(%q) = %q, want %q", c.raw, got, c.want)
		}
	}
}

func TestStatsWindow(t *testing.T) {
	now := time.Date(2026, 9, 1, 15, 30, 0, 0, time.UTC)
	today := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	tomorrow := today.AddDate(0, 0, 1)
	for _, c := range []struct {
		rng           string
		wantFrom      time.Time
		wantPriorFrom time.Time
		wantNoPrior   bool
	}{
		{"today", today, today.AddDate(0, 0, -1), false},
		{"7d", today.AddDate(0, 0, -6), today.AddDate(0, 0, -13), false},
		{"30d", today.AddDate(0, 0, -29), today.AddDate(0, 0, -59), false},
		{"all", time.Time{}, time.Time{}, true},
	} {
		from, to, priorFrom := statsWindow(c.rng, now, time.UTC)
		if !to.Equal(tomorrow) {
			t.Fatalf("%s: to = %v, want %v", c.rng, to, tomorrow)
		}
		if !from.Equal(c.wantFrom) {
			t.Fatalf("%s: from = %v, want %v", c.rng, from, c.wantFrom)
		}
		if c.wantNoPrior {
			if !priorFrom.IsZero() {
				t.Fatalf("%s: priorFrom = %v, want zero", c.rng, priorFrom)
			}
			continue
		}
		if !priorFrom.Equal(c.wantPriorFrom) {
			t.Fatalf("%s: priorFrom = %v, want %v", c.rng, priorFrom, c.wantPriorFrom)
		}
		// The prior window must be exactly as wide as the current one.
		if to.Sub(from) != from.Sub(priorFrom) {
			t.Fatalf("%s: prior window width %v != current window width %v", c.rng, from.Sub(priorFrom), to.Sub(from))
		}
	}
}

func withTestSessionRoot(t *testing.T) string {
	t.Helper()
	old := session.TestRoot
	dir := t.TempDir()
	session.TestRoot = dir
	t.Cleanup(func() { session.TestRoot = old })
	return dir
}

func TestHandleSessionStatsEmptyState(t *testing.T) {
	withTestSessionRoot(t)
	r := httptest.NewRequest(http.MethodGet, "/api/sessions/stats?range=7d", nil)
	w := httptest.NewRecorder()
	handleSessionStats(Deps{})(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	var got sessionStatsView
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v, body = %s", err, w.Body.String())
	}
	if got.Range != "7d" {
		t.Fatalf("range = %q", got.Range)
	}
	if got.Current.Cost != 0 || got.Current.Messages != 0 || got.Current.Sessions != 0 {
		t.Fatalf("current = %+v", got.Current)
	}
	if got.Prior == nil {
		t.Fatal("expected a prior window for range=7d")
	}
	if len(got.ByProvider) != 0 {
		t.Fatalf("byProvider = %+v", got.ByProvider)
	}
}

// TestHandleSessionStatsNoPreviewLeak is a sabotage-style regression test:
// it seeds a session whose message content would show up as a Preview
// string on session.Summary, then asserts the /api/sessions/stats response
// never carries a "preview" key or the message text anywhere in the body.
// This is a privacy property of the endpoint (aggregate numbers only, no
// per-session content), not just a shape check — it must keep failing if
// someone later widens WindowStats to embed a raw session list.
func TestHandleSessionStatsNoPreviewLeak(t *testing.T) {
	root := withTestSessionRoot(t)
	dir := filepath.Join(root, "--tmp--")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	secret := "this-should-never-reach-the-dashboard-response"
	body := `{"type":"model_change","provider":"anthropic","modelId":"claude"}` + "\n" +
		`{"type":"message","message":{"role":"user","content":[{"type":"text","text":"` + secret + `"}]}}` + "\n" +
		`{"type":"message","message":{"role":"assistant","usage":{"cost":{"total":0.5}}}}` + "\n"
	if err := os.WriteFile(filepath.Join(dir, "s.jsonl"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest(http.MethodGet, "/api/sessions/stats?range=all", nil)
	w := httptest.NewRecorder()
	handleSessionStats(Deps{})(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	got := w.Body.String()
	if strings.Contains(strings.ToLower(got), "preview") {
		t.Fatalf("response leaks a preview field: %s", got)
	}
	if strings.Contains(got, secret) {
		t.Fatalf("response leaks raw message content: %s", got)
	}
}
