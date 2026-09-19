package mcpcatalog

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/mcp"
)

func entry(name, title, desc, status, remotes string) string {
	meta := ""
	if status != "" {
		meta = fmt.Sprintf(`,"_meta":{%q:{"status":%q}}`, statusMetaKey, status)
	}
	return fmt.Sprintf(`{"server":{"name":%q,"title":%q,"description":%q,"remotes":%s}%s}`,
		name, title, desc, remotes, meta)
}

const httpRemote = `[{"type":"streamable-http","url":"https://acme.example/mcp"}]`

func page(cursor string, entries ...string) string {
	meta := "{}"
	if cursor != "" {
		meta = fmt.Sprintf(`{"nextCursor":%q}`, cursor)
	}
	return `{"servers":[` + strings.Join(entries, ",") + `],"metadata":` + meta + `}`
}

// registryServer serves the given pages by cursor ("" is the first page)
// and counts requests. hook observes each request before it is answered.
func registryServer(t *testing.T, pages map[string]string, hook func(*http.Request)) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var counter atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		counter.Add(1)
		if hook != nil {
			hook(r)
		}
		body, ok := pages[r.URL.Query().Get("cursor")]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, body)
	}))
	t.Cleanup(srv.Close)
	return srv, &counter
}

func waitFor(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("condition not met before timeout")
}

func findSeed(t *testing.T, id string) Item {
	t.Helper()
	for _, it := range Seed() {
		if it.ID == id {
			return it
		}
	}
	t.Fatalf("seed item %q missing", id)
	return Item{}
}

func TestSeedMapsPresets(t *testing.T) {
	presets := mcp.Presets()
	seed := Seed()
	if len(seed) != len(presets) {
		t.Fatalf("seed = %d items, want %d", len(seed), len(presets))
	}
	deepwiki := findSeed(t, "deepwiki")
	if deepwiki.Kind != "url" || deepwiki.URL != "https://mcp.deepwiki.com/mcp" {
		t.Fatalf("deepwiki = %+v", deepwiki)
	}
	if deepwiki.Command != "" || len(deepwiki.Args) != 0 {
		t.Fatalf("deepwiki carries stdio fields: %+v", deepwiki)
	}
	if !deepwiki.Featured || deepwiki.Source != "picode" {
		t.Fatalf("deepwiki featured/source = %v/%q", deepwiki.Featured, deepwiki.Source)
	}
	if it := findSeed(t, "notion"); it.Auth != "oauth" {
		t.Fatalf("notion auth = %q", it.Auth)
	}
	if it := findSeed(t, "github"); it.Auth != "oauth" || it.Kind != "url" {
		t.Fatalf("github = %+v", it)
	}
	if it := findSeed(t, "chrome-devtools"); it.Kind != "stdio" || it.Command != "npx" || len(it.Args) == 0 {
		t.Fatalf("chrome-devtools = %+v", it)
	}
	if it := findSeed(t, "picode-computer"); !strings.HasPrefix(it.ID, "picode-") || it.Kind != "stdio" {
		t.Fatalf("picode-computer = %+v", it)
	}
	seen := map[string]bool{}
	for _, it := range seed {
		if seen[it.ID] {
			t.Fatalf("duplicate seed id %q", it.ID)
		}
		seen[it.ID] = true
	}
}

func TestRefreshFiltersPaginatesAndOrders(t *testing.T) {
	pages := map[string]string{
		"": page("page-2",
			entry("io.acme/alpha", "Alpha", "First good remote connector", "active", httpRemote),
			entry("io.acme/dead", "Dead", "Deprecated connector", "deprecated", httpRemote),
			entry("io.acme/nodesc", "NoDesc", "", "active", httpRemote),
			entry("io.acme/wsonly", "WsOnly", "Only a websocket remote", "active", `[{"type":"ws","url":"wss://ws.example/mcp"}]`),
		),
		"page-2": page("",
			entry("io.acme/noremote", "NoRemote", "Active but no remotes at all", "active", `[]`),
			entry("deepwiki", "DeepWiki Clone", "Collides with a seed id", "active", httpRemote),
			entry("io.acme/beta", "Beta", "Second good remote connector", "active", httpRemote),
		),
	}
	var seenCursors []string
	srv, counter := registryServer(t, pages, func(r *http.Request) {
		seenCursors = append(seenCursors, r.URL.Query().Get("cursor"))
		if got := r.URL.Query().Get("version"); got != "latest" {
			t.Errorf("version = %q", got)
		}
		if got := r.URL.Query().Get("limit"); got != "100" {
			t.Errorf("limit = %q", got)
		}
	})
	s := NewStore(t.TempDir())
	s.Base = srv.URL
	if err := s.Refresh(context.Background()); err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if got := counter.Load(); got != 2 {
		t.Fatalf("registry requests = %d, want 2", got)
	}
	if len(seenCursors) != 2 || seenCursors[1] != "page-2" {
		t.Fatalf("cursors = %v", seenCursors)
	}

	hits := s.Search(context.Background(), "")
	nSeed := len(Seed())
	if len(hits) != nSeed+2 {
		t.Fatalf("hits = %d, want %d", len(hits), nSeed+2)
	}
	for i, hit := range hits[:nSeed] {
		if hit.Source != "picode" || !hit.Featured {
			t.Fatalf("hit %d is not seed-first: %+v", i, hit)
		}
	}
	// Registry rows follow the seed, sorted by name; all filtered rows gone.
	if hits[nSeed].ID != "io.acme/alpha" || hits[nSeed+1].ID != "io.acme/beta" {
		t.Fatalf("registry rows = %q, %q", hits[nSeed].ID, hits[nSeed+1].ID)
	}
	for _, hit := range hits[nSeed:] {
		if hit.Featured || hit.Source != "registry" || hit.Kind != "url" || hit.URL != "https://acme.example/mcp" {
			t.Fatalf("registry row = %+v", hit)
		}
	}
	// The seed-id collision kept exactly one deepwiki, the seed one.
	n := 0
	for _, hit := range hits {
		if hit.ID == "deepwiki" {
			n++
			if hit.Source != "picode" {
				t.Fatalf("deepwiki row = %+v", hit)
			}
		}
	}
	if n != 1 {
		t.Fatalf("deepwiki appears %d times", n)
	}

	// The cache file was written with the merged view.
	b, err := os.ReadFile(filepath.Join(s.dataDir, cacheFileName))
	if err != nil {
		t.Fatalf("cache file: %v", err)
	}
	var cf cacheFileV1
	if err := json.Unmarshal(b, &cf); err != nil {
		t.Fatalf("cache decode: %v", err)
	}
	if cf.RefreshedAt.IsZero() {
		t.Fatal("cache refreshedAt is zero")
	}
	if len(cf.Items) != nSeed+2 {
		t.Fatalf("cache items = %d, want %d", len(cf.Items), nSeed+2)
	}
}

func TestSearchQueryAndCap(t *testing.T) {
	const n = 300
	entries := make([]string, 0, n)
	for i := range n {
		// Titles run opposite to ids: name-sorting must win over id order.
		entries = append(entries, entry(
			fmt.Sprintf("reg.test/srv-%03d", i),
			fmt.Sprintf("Server %03d", n-1-i),
			fmt.Sprintf("Generic connector number %d", i),
			"active", httpRemote))
	}
	srv, _ := registryServer(t, map[string]string{"": page("", entries...)}, nil)
	s := NewStore("")
	s.Base = srv.URL
	if err := s.Refresh(context.Background()); err != nil {
		t.Fatalf("refresh: %v", err)
	}

	hits := s.Search(context.Background(), "")
	if len(hits) != maxSearchHits {
		t.Fatalf("hits = %d, want capped %d", len(hits), maxSearchHits)
	}
	if hits[0].ID != "picode-computer" || hits[0].Source != "picode" {
		t.Fatalf("first hit = %+v", hits[0])
	}
	// Registry portion is sorted by name: "Server 000" first, which is
	// id srv-299 in this fixture.
	if hits[len(Seed())].ID != "reg.test/srv-299" {
		t.Fatalf("first registry hit = %+v", hits[len(Seed())])
	}

	if got := s.Search(context.Background(), "SRV-299"); len(got) != 1 || got[0].ID != "reg.test/srv-299" {
		t.Fatalf("case-insensitive id search = %+v", got)
	}
	got := s.Search(context.Background(), "deepwiki")
	if len(got) == 0 || got[0].ID != "deepwiki" || got[0].Source != "picode" {
		t.Fatalf("seed-first search = %+v", got)
	}
	if got := s.Search(context.Background(), "repositories"); len(got) == 0 || got[0].ID != "deepwiki" {
		t.Fatalf("summary search = %+v", got)
	}
	if got := s.Search(context.Background(), "no-such-connector-xyz"); len(got) != 0 {
		t.Fatalf("empty search = %+v", got)
	}
}

func TestCacheRoundTripWarmRestart(t *testing.T) {
	srv, counter := registryServer(t, map[string]string{
		"": page("", entry("io.acme/alpha", "Alpha", "Persisted connector", "active", httpRemote)),
	}, nil)
	dir := t.TempDir()
	s1 := NewStore(dir)
	s1.Base = srv.URL
	if err := s1.Refresh(context.Background()); err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if got := counter.Load(); got != 1 {
		t.Fatalf("first store requests = %d", got)
	}

	// A restarted store loads the cache and answers warm — no fetch.
	s2 := NewStore(dir)
	s2.Base = srv.URL
	hits := s2.Search(context.Background(), "persisted")
	if len(hits) != 1 || hits[0].ID != "io.acme/alpha" || hits[0].Source != "registry" {
		t.Fatalf("warm hits = %+v", hits)
	}
	if got := counter.Load(); got != 1 {
		t.Fatalf("warm restart fetched %d extra times", got-1)
	}
	// Seed cards still come from the binary, first.
	hits = s2.Search(context.Background(), "")
	if len(hits) != len(Seed())+1 || hits[0].Source != "picode" {
		t.Fatalf("warm full hits = %d", len(hits))
	}
}

func TestNewStoreIgnoresBrokenCache(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, cacheFileName), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := NewStore(dir)
	hits := s.Search(context.Background(), "")
	if len(hits) != len(Seed()) {
		t.Fatalf("hits = %d, want seed only", len(hits))
	}
}

func TestStaleRefreshTriggersExactlyOnce(t *testing.T) {
	body := page("", entry("reg.io/slow", "Slow", "syncedwikitest slow connector", "active", httpRemote))
	var counter atomic.Int32
	gates := []chan struct{}{make(chan struct{}), make(chan struct{})}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := counter.Add(1)
		if n >= 1 && n <= 2 {
			select {
			case <-gates[n-1]:
			case <-time.After(5 * time.Second):
			}
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, body)
	}))
	t.Cleanup(srv.Close)

	s := NewStore("")
	s.Base = srv.URL
	now := time.Unix(1_700_000_000, 0)
	s.Now = func() time.Time { return now }
	ctx := context.Background()

	// First search answers with the seed immediately; the refresh runs in
	// the background against the slow registry.
	hits := s.Search(ctx, "")
	if len(hits) != len(Seed()) {
		t.Fatalf("first hits = %d, want seed only", len(hits))
	}
	waitFor(t, 2*time.Second, func() bool { return counter.Load() == 1 })

	// While it is in flight, further stale searches do not start another.
	if got := s.Search(ctx, "syncedwikitest"); len(got) != 0 {
		t.Fatalf("in-flight hits = %+v", got)
	}
	time.Sleep(50 * time.Millisecond)
	if got := counter.Load(); got != 1 {
		t.Fatalf("concurrent refreshes = %d, want 1", got)
	}
	close(gates[0])
	waitFor(t, 5*time.Second, func() bool { return len(s.Search(ctx, "syncedwikitest")) == 1 })
	if got := counter.Load(); got != 1 {
		t.Fatalf("total refreshes = %d, want 1", got)
	}

	// A fresh cache does not refresh...
	before := counter.Load()
	hits = s.Search(ctx, "")
	if len(hits) != len(Seed())+1 {
		t.Fatalf("warm hits = %d", len(hits))
	}
	time.Sleep(20 * time.Millisecond)
	if got := counter.Load(); got != before {
		t.Fatalf("warm cache refreshed (%d → %d)", before, got)
	}

	// ...but 25 hours later it does, exactly once more.
	now = now.Add(25 * time.Hour)
	s.Search(ctx, "")
	waitFor(t, 2*time.Second, func() bool { return counter.Load() == 2 })
	close(gates[1])
	waitFor(t, 5*time.Second, func() bool {
		s.mu.Lock()
		defer s.mu.Unlock()
		return counter.Load() == 2 && !s.refreshing
	})
}

func TestRefreshFailureKeepsPreviousState(t *testing.T) {
	srv, _ := registryServer(t, map[string]string{
		"": page("", entry("io.acme/alpha", "Alpha", "Survives an outage", "active", httpRemote)),
	}, nil)
	s := NewStore("")
	s.Base = srv.URL
	if err := s.Refresh(context.Background()); err != nil {
		t.Fatalf("refresh: %v", err)
	}

	// A closed local port refuses instantly (a fixed low port such as :1
	// can sit behind a NAT that drops the SYN instead).
	dead := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	deadURL := dead.URL
	dead.Close()

	empty := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, page(""))
	}))
	t.Cleanup(empty.Close)

	for name, base := range map[string]string{
		"unreachable": deadURL,
		"empty":       empty.URL,
	} {
		s.Base = base
		if err := s.Refresh(context.Background()); err == nil {
			t.Errorf("%s: refresh error = nil, want error", name)
		}
		hits := s.Search(context.Background(), "outage")
		if len(hits) != 1 || hits[0].ID != "io.acme/alpha" {
			t.Errorf("%s: hits = %+v, previous state lost", name, hits)
		}
	}

	s.Base = srv.URL
	broken := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(broken.Close)
	s.Base = broken.URL
	if err := s.Refresh(context.Background()); err == nil {
		t.Error("status 500: refresh error = nil, want error")
	}
	if hits := s.Search(context.Background(), "outage"); len(hits) != 1 {
		t.Errorf("status 500: hits = %+v, previous state lost", hits)
	}
}
