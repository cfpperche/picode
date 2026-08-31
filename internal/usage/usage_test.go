package usage

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/catalog"
)

func TestFetchDecisionTable(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".pi", "agent"), 0o700); err != nil {
		t.Fatal(err)
	}

	var anthropicHits, tokenHits int
	mux := http.NewServeMux()
	mux.HandleFunc("/anthropic/usage", func(w http.ResponseWriter, r *http.Request) {
		anthropicHits++
		if got := r.Header.Get("Authorization"); !strings.HasPrefix(got, "Bearer ") {
			t.Errorf("missing bearer")
		}
		if r.Header.Get("Authorization") == "Bearer stale" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"five_hour":{"utilization":10,"resets_at":"2026-08-31T19:00:00Z"},"seven_day":{"utilization":2,"resets_at":"2026-09-05T12:00:00Z"}}`))
	})
	mux.HandleFunc("/anthropic/profile", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"plan":"Max"}`))
	})
	mux.HandleFunc("/anthropic/token", func(w http.ResponseWriter, r *http.Request) {
		tokenHits++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"fresh","refresh_token":"r2","expires_in":3600}`))
	})
	mux.HandleFunc("/anthropic/empty", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	})
	mux.HandleFunc("/anthropic/429", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	})
	mux.HandleFunc("/anthropic/500", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	mux.HandleFunc("/xai/billing", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"config":{"creditUsagePercent":42,"currentPeriod":{"end":"2026-09-05T12:00:00Z"}}}`))
	})
	mux.HandleFunc("/xai/settings", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"subscription_tier_display":"SuperGrok"}`))
	})
	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)

	client := NewClient(ts.Client())
	client.Now = func() time.Time { return time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC) }
	client.Endpoints = map[string]string{
		"anthropic.usage":   ts.URL + "/anthropic/usage",
		"anthropic.profile": ts.URL + "/anthropic/profile",
		"anthropic.token":   ts.URL + "/anthropic/token",
		"xai.billing":       ts.URL + "/xai/billing",
		"xai.settings":      ts.URL + "/xai/settings",
	}

	ctx := context.Background()

	t.Run("unsigned", func(t *testing.T) {
		rep := client.Fetch(ctx, "anthropic")
		if rep.Status != StatusUnsupported {
			t.Fatalf("status %s", rep.Status)
		}
	})

	t.Run("api_key anthropic", func(t *testing.T) {
		if err := catalog.PutAPIKey("anthropic", "sk-test"); err != nil {
			t.Fatal(err)
		}
		rep := client.Fetch(ctx, "anthropic")
		if rep.Status != StatusUnsupported {
			t.Fatalf("status %s", rep.Status)
		}
	})

	t.Run("api_key xai", func(t *testing.T) {
		if err := catalog.PutAPIKey("xai", "xai-key"); err != nil {
			t.Fatal(err)
		}
		rep := client.Fetch(ctx, "xai")
		if rep.Status != StatusUnsupported {
			t.Fatalf("status %s", rep.Status)
		}
	})

	t.Run("oauth ok", func(t *testing.T) {
		if err := catalog.PutOAuth("anthropic", map[string]any{
			"type": "oauth", "access": "live", "refresh": "r1",
			"expires": float64(time.Now().Add(time.Hour).UnixMilli()),
		}); err != nil {
			t.Fatal(err)
		}
		rep := client.Fetch(ctx, "anthropic")
		if rep.Status != StatusOK {
			t.Fatalf("%s %s", rep.Status, rep.Error)
		}
		if rep.Plan != "Max" || len(rep.Windows) != 2 {
			t.Fatalf("%+v", rep)
		}
		if *rep.Windows[0].UsedPercent != 10 {
			t.Fatalf("pct %v", *rep.Windows[0].UsedPercent)
		}
	})

	t.Run("oauth empty windows", func(t *testing.T) {
		client.Endpoints["anthropic.usage"] = ts.URL + "/anthropic/empty"
		if err := catalog.PutOAuth("anthropic", map[string]any{
			"type": "oauth", "access": "live", "refresh": "r1",
			"expires": float64(time.Now().Add(time.Hour).UnixMilli()),
		}); err != nil {
			t.Fatal(err)
		}
		rep := client.Fetch(ctx, "anthropic")
		if rep.Status != StatusOK || len(rep.Windows) != 0 {
			t.Fatalf("%+v", rep)
		}
		client.Endpoints["anthropic.usage"] = ts.URL + "/anthropic/usage"
	})

	t.Run("401 then refresh", func(t *testing.T) {
		if err := catalog.PutOAuth("anthropic", map[string]any{
			"type": "oauth", "access": "stale", "refresh": "r1",
			"expires": float64(time.Now().Add(time.Hour).UnixMilli()),
		}); err != nil {
			t.Fatal(err)
		}
		before := tokenHits
		rep := client.Fetch(ctx, "anthropic")
		if rep.Status != StatusOK {
			t.Fatalf("%s %s", rep.Status, rep.Error)
		}
		if tokenHits == before {
			t.Fatal("expected refresh")
		}
		cred, _ := catalog.ActiveOAuth("anthropic")
		if cred.Access != "fresh" {
			t.Fatalf("stored %s", cred.Access)
		}
	})

	t.Run("expired refresh works", func(t *testing.T) {
		if err := catalog.PutOAuth("anthropic", map[string]any{
			"type": "oauth", "access": "old", "refresh": "r1",
			"expires": float64(1),
		}); err != nil {
			t.Fatal(err)
		}
		rep := client.Fetch(ctx, "anthropic")
		if rep.Status != StatusOK {
			t.Fatalf("%s %s", rep.Status, rep.Error)
		}
	})

	t.Run("expired refresh fails", func(t *testing.T) {
		client.Endpoints["anthropic.token"] = ts.URL + "/anthropic/500"
		if err := catalog.PutOAuth("anthropic", map[string]any{
			"type": "oauth", "access": "old", "refresh": "r1",
			"expires": float64(1),
		}); err != nil {
			t.Fatal(err)
		}
		rep := client.Fetch(ctx, "anthropic")
		if rep.Status != StatusAuthRequired {
			t.Fatalf("status %s", rep.Status)
		}
		client.Endpoints["anthropic.token"] = ts.URL + "/anthropic/token"
	})

	t.Run("429", func(t *testing.T) {
		client.Endpoints["anthropic.usage"] = ts.URL + "/anthropic/429"
		if err := catalog.PutOAuth("anthropic", map[string]any{
			"type": "oauth", "access": "live", "refresh": "r1",
			"expires": float64(time.Now().Add(time.Hour).UnixMilli()),
		}); err != nil {
			t.Fatal(err)
		}
		rep := client.Fetch(ctx, "anthropic")
		if rep.Status != StatusError || rep.Error != "Rate limited." {
			t.Fatalf("%+v", rep)
		}
		client.Endpoints["anthropic.usage"] = ts.URL + "/anthropic/usage"
	})

	t.Run("5xx", func(t *testing.T) {
		client.Endpoints["anthropic.usage"] = ts.URL + "/anthropic/500"
		if err := catalog.PutOAuth("anthropic", map[string]any{
			"type": "oauth", "access": "live", "refresh": "r1",
			"expires": float64(time.Now().Add(time.Hour).UnixMilli()),
		}); err != nil {
			t.Fatal(err)
		}
		rep := client.Fetch(ctx, "anthropic")
		if rep.Status != StatusError || rep.Error != "Couldn't load usage." {
			t.Fatalf("%+v", rep)
		}
		client.Endpoints["anthropic.usage"] = ts.URL + "/anthropic/usage"
	})

	t.Run("xai oauth", func(t *testing.T) {
		if err := catalog.PutOAuth("xai", map[string]any{
			"type": "oauth", "access": "g1", "refresh": "gr",
			"expires": float64(time.Now().Add(time.Hour).UnixMilli()),
		}); err != nil {
			t.Fatal(err)
		}
		rep := client.Fetch(ctx, "xai")
		if rep.Status != StatusOK || rep.Plan != "SuperGrok" {
			t.Fatalf("%+v", rep)
		}
		if len(rep.Windows) == 0 || *rep.Windows[0].UsedPercent != 42 {
			t.Fatalf("windows %+v", rep.Windows)
		}
	})

	t.Run("no adapter", func(t *testing.T) {
		if err := catalog.PutAPIKey("openai", "sk"); err != nil {
			t.Fatal(err)
		}
		rep := client.Fetch(ctx, "openai")
		if rep.Status != StatusUnsupported {
			t.Fatalf("%s", rep.Status)
		}
	})

	_ = anthropicHits
}

func TestFetchAPIKeyMeters(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	mux := http.NewServeMux()
	mux.HandleFunc("/zai", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer sk-zai" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"level":"pro","limits":[{"type":"TOKENS_LIMIT","unit":3,"number":5,"percentage":10}]}}`))
	})
	mux.HandleFunc("/ocg", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"rolling":{"usagePercent":5},"weekly":{"usagePercent":8},"monthly":{"usagePercent":12}}`))
	})
	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)
	client := NewClient(ts.Client())
	client.Endpoints = map[string]string{
		"zai.quota":         ts.URL + "/zai",
		"opencode-go.usage": ts.URL + "/ocg",
	}
	ctx := context.Background()

	if err := catalog.PutAPIKey("zai", "sk-zai"); err != nil {
		t.Fatal(err)
	}
	rep := client.Fetch(ctx, "zai")
	if rep.Status != StatusOK || rep.AuthType != "api_key" || !hasWindow(rep.Windows, "5h") {
		t.Fatalf("%+v", rep)
	}

	if err := catalog.PutAPIKey("opencode-go", "sk-go"); err != nil {
		t.Fatal(err)
	}
	rep = client.Fetch(ctx, "opencode-go")
	if rep.Status != StatusOK || len(rep.Windows) != 3 {
		t.Fatalf("%+v", rep)
	}
}

func TestCodexResetsAndRedeem(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	consumed := false
	mux := http.NewServeMux()
	mux.HandleFunc("/usage", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"plan_type":"plus","five_hour":{"used_percent":90}}`))
	})
	mux.HandleFunc("/resets", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if consumed {
			_, _ = w.Write([]byte(`{"credits":[]}`))
			return
		}
		_, _ = w.Write([]byte(`{"credits":[{"id":"RateLimitResetCredit_1","status":"available","expires_at":"2099-01-01T00:00:00Z"}]}`))
	})
	mux.HandleFunc("/redeem", func(w http.ResponseWriter, r *http.Request) {
		consumed = true
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	})
	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)
	client := NewClient(ts.Client())
	client.Endpoints = map[string]string{
		"codex.usage":  ts.URL + "/usage",
		"codex.resets": ts.URL + "/resets",
		"codex.redeem": ts.URL + "/redeem",
	}
	if err := catalog.PutOAuth("openai-codex", map[string]any{
		"type": "oauth", "access": "live", "refresh": "r", "accountId": "acct",
		"expires": float64(time.Now().Add(time.Hour).UnixMilli()),
	}); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	rep := client.Fetch(ctx, "openai-codex")
	if rep.Status != StatusOK || len(rep.Resets) != 1 {
		t.Fatalf("fetch %+v", rep)
	}
	out := client.Redeem(ctx, "openai-codex", "")
	if out.Status != StatusOK {
		t.Fatalf("redeem %+v", out)
	}
	if !consumed || len(out.Resets) != 0 {
		t.Fatalf("after %+v consumed=%v", out, consumed)
	}
}

func TestRedeemNone(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	mux := http.NewServeMux()
	mux.HandleFunc("/usage", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"five_hour":{"used_percent":10}}`))
	})
	mux.HandleFunc("/resets", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"credits":[]}`))
	})
	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)
	client := NewClient(ts.Client())
	client.Endpoints = map[string]string{
		"codex.usage":  ts.URL + "/usage",
		"codex.resets": ts.URL + "/resets",
	}
	if err := catalog.PutOAuth("openai-codex", map[string]any{
		"type": "oauth", "access": "live", "refresh": "r",
		"expires": float64(time.Now().Add(time.Hour).UnixMilli()),
	}); err != nil {
		t.Fatal(err)
	}
	rep := client.Redeem(context.Background(), "openai-codex", "")
	if rep.Status != StatusError || rep.Error != "No reset available." {
		t.Fatalf("%+v", rep)
	}
}
