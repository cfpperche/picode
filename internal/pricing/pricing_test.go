package pricing

import (
	"context"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// doc is a slice of LiteLLM's real shape (2026-09-22).
const doc = `{
 "sample_spec": {"max_tokens": "set to max"},
 "claude-opus-5": {"input_cost_per_token": 5e-06, "output_cost_per_token": 2.5e-05,
   "cache_read_input_token_cost": 5e-07, "cache_creation_input_token_cost": 6.25e-06},
 "gpt-5.6-sol": {"input_cost_per_token": 4e-06, "output_cost_per_token": 3.2e-05,
   "cache_read_input_token_cost": 4e-07},
 "azure_ai/grok-4.6": {"input_cost_per_token": 2e-06, "output_cost_per_token": 1e-05},
 "xai/grok-4.6": {"input_cost_per_token": 2e-06, "output_cost_per_token": 1e-05},
 "a/split-model": {"input_cost_per_token": 1e-06, "output_cost_per_token": 1e-06},
 "b/split-model": {"input_cost_per_token": 2e-06, "output_cost_per_token": 1e-06},
 "chatgpt/gpt-5.3-codex-spark": {"max_tokens": 128000},
 "opus": {"input_cost_per_token": 1, "output_cost_per_token": 1}
}`

func approx(a, b float64) bool { return math.Abs(a-b) < 1e-12 }

func TestLookup(t *testing.T) {
	tab, err := Parse([]byte(doc))
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		model  string
		ok     bool
		input  float64
		cacheW float64
	}{
		{"claude-opus-5", true, 5e-06, 6.25e-06},
		{"Claude-Opus-5[1m]", true, 5e-06, 6.25e-06}, // variant and case
		{"gpt-5.6-sol", true, 4e-06, 4e-06},          // no cache-write rate: priced as input
		{"grok-4.6", true, 2e-06, 2e-06},             // bare alias: every provider agrees
		{"split-model", false, 0, 0},                 // two providers, two prices: no alias
		{"gpt-5.3-codex-spark", false, 0, 0},         // listed without rates
		{"opus", false, 0, 0},                        // a family name is never guessed
		{"<synthetic>", false, 0, 0},
		{"", false, 0, 0},
	}
	for _, tc := range cases {
		t.Run(tc.model, func(t *testing.T) {
			r, ok := tab.Lookup(tc.model)
			if ok != tc.ok {
				t.Fatalf("ok = %v, want %v", ok, tc.ok)
			}
			if ok && (!approx(r.Input, tc.input) || !approx(r.CacheWrite, tc.cacheW)) {
				t.Fatalf("rate = %+v", r)
			}
		})
	}
}

func TestCostChargesEachKindAtItsRate(t *testing.T) {
	tab, _ := Parse([]byte(doc))
	// 1000 uncached in, 100 out, 10000 cache read, 200 cache write.
	got, ok := tab.Cost("claude-opus-5", 1000, 100, 10000, 200)
	want := 1000*5e-06 + 100*2.5e-05 + 10000*5e-07 + 200*6.25e-06
	if !ok || !approx(got, want) {
		t.Fatalf("cost = %v, %v; want %v", got, ok, want)
	}
	var none *Table
	if _, ok := none.Cost("claude-opus-5", 1, 1, 1, 1); ok {
		t.Fatal("a nil table priced a turn")
	}
}

func TestLoaderKeepsTheLastGoodTable(t *testing.T) {
	Set(nil)
	t.Cleanup(func() { Set(nil) })
	status := http.StatusOK
	body := doc
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()
	path := filepath.Join(t.TempDir(), "var", "litellm-prices.json")
	l := Loader{Path: path, URL: srv.URL}

	if err := l.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	v := Current().Version()
	if v == "" {
		t.Fatal("no table installed")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("table not written to disk: %v", err)
	}

	// A failing fetch and an empty table both keep what is loaded.
	status = http.StatusBadGateway
	if err := l.Refresh(context.Background()); err == nil {
		t.Fatal("a 502 was accepted")
	}
	status, body = http.StatusOK, `{"sample_spec": {}}`
	if err := l.Refresh(context.Background()); err == nil {
		t.Fatal("a table pricing nothing was accepted")
	}
	if Current().Version() != v {
		t.Fatal("a failed refresh replaced the table")
	}

	// A fresh process loads the copy from disk without the network.
	Set(nil)
	if err := (Loader{Path: path}).LoadCached(); err != nil {
		t.Fatal(err)
	}
	if Current().Version() != v {
		t.Fatal("the cached copy is not the table that was fetched")
	}
}

func TestURLFromEnv(t *testing.T) {
	for _, tc := range []struct{ env, want string }{
		{"", DefaultURL},
		{"off", ""},
		{"OFF", ""},
		{"https://example.test/p.json", "https://example.test/p.json"},
	} {
		t.Setenv("PICODE_PRICE_TABLE_URL", tc.env)
		if got := URLFromEnv(); got != tc.want {
			t.Fatalf("%q -> %q, want %q", tc.env, got, tc.want)
		}
	}
}
