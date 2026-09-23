package server

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/catalog"
	"github.com/cfpperche/picode/internal/feed"
	"github.com/cfpperche/picode/internal/llama"
	"github.com/cfpperche/picode/internal/llamajob"
	"github.com/cfpperche/picode/internal/store"
)

func TestLlamaJobHTTP(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	service, err := llamajob.New(st, func() (string, string) { return "http://127.0.0.1:1", "" })
	if err != nil {
		t.Fatal(err)
	}
	defer service.Close()
	mux := http.NewServeMux()
	registerLlama(mux, Deps{Store: st, LlamaJobs: service})
	for _, tc := range []struct {
		method, path, body string
		status             int
	}{
		{"POST", "/api/llama/load", `{"id":"a"}`, 400},
		{"POST", "/api/llama/load", `{"id":"a","requestKey":"test"}`, 202},
		{"POST", "/api/llama/load", `{"id":"a","requestKey":"test"}`, 202},
		{"POST", "/api/llama/load", `{"id":"b","requestKey":"test"}`, 409},
		{"GET", "/api/llama/jobs", "", 200},
		{"POST", "/api/llama/jobs/missing/cancel", "", 404},
		{"POST", "/api/llama/jobs/missing/reconcile", "", 404},
	} {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body)))
		if rec.Code != tc.status {
			t.Fatalf("%s got %d: %s", tc.path, rec.Code, rec.Body.String())
		}
	}
	rec := httptest.NewRecorder()
	handleLlamaOperation(Deps{}, "load")(rec, httptest.NewRequest("POST", "/api/llama/load", nil))
	if rec.Code != 503 {
		t.Fatal(rec.Code)
	}
}

// catalogLlamaFixture points the catalog at url (through a HOME auth.json)
// and swaps the probe for one the test drives. It returns the probe count.
type catalogLlamaFixture struct {
	t     *testing.T
	home  string
	mu    sync.Mutex
	calls map[string]int
	// answer and gate are per URL: gate, when set, blocks that URL's probe
	// until the test closes it.
	answer map[string][]llama.Model
	gate   map[string]chan struct{}
}

func newCatalogLlamaFixture(t *testing.T) *catalogLlamaFixture {
	f := &catalogLlamaFixture{t: t, home: t.TempDir(), calls: map[string]int{}, answer: map[string][]llama.Model{}, gate: map[string]chan struct{}{}}
	t.Setenv("HOME", f.home)
	prev := listCatalogLlama
	listCatalogLlama = func(url, _ string) ([]llama.Model, error) {
		f.mu.Lock()
		f.calls[url]++
		g := f.gate[url]
		f.mu.Unlock()
		if g != nil {
			<-g
		}
		f.mu.Lock()
		defer f.mu.Unlock()
		if m, ok := f.answer[url]; ok {
			return m, nil
		}
		return nil, errors.New("timeout")
	}
	reset := func() {
		catalogLlama.Lock()
		catalogLlama.key, catalogLlama.at, catalogLlama.stale, catalogLlama.models, catalogLlama.err = "", time.Time{}, false, nil, nil
		catalogLlama.flight = nil
		catalogLlama.Unlock()
	}
	reset()
	t.Cleanup(func() { listCatalogLlama = prev; reset() })
	return f
}

func (f *catalogLlamaFixture) point(url string) {
	dir := filepath.Join(f.home, ".pi", "agent")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		f.t.Fatal(err)
	}
	body := `{"llama.cpp":{"type":"api_key","key":"k","env":{"LLAMA_BASE_URL":"` + url + `"}}}`
	if err := os.WriteFile(filepath.Join(dir, "auth.json"), []byte(body), 0o600); err != nil {
		f.t.Fatal(err)
	}
}

func (f *catalogLlamaFixture) count(url string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls[url]
}

func (f *catalogLlamaFixture) models() []string {
	rep := catalog.Report{Providers: []catalog.Provider{{ID: "llama.cpp"}}}
	attachLlamaModels(&rep, false)
	var ids []string
	for _, m := range rep.Providers[0].Models {
		ids = append(ids, m.ID)
	}
	return ids
}

func waitFor(t *testing.T, what string, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for !ok() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(2 * time.Millisecond)
	}
}

const urlA, urlB = "http://127.0.0.1:1", "http://127.0.0.1:2"

// A failure is kept like an answer: two reads, one probe.
func TestCatalogLlamaKeepsAFailure(t *testing.T) {
	f := newCatalogLlamaFixture(t)
	f.point(urlA)
	f.models()
	f.models()
	if f.count(urlA) != 1 {
		t.Fatalf("probes = %d, want 1", f.count(urlA))
	}
}

// Concurrent reads with nothing kept share one probe.
func TestCatalogLlamaColdReadsShareOneProbe(t *testing.T) {
	f := newCatalogLlamaFixture(t)
	f.point(urlA)
	f.answer[urlA] = []llama.Model{{ID: "qwen", Status: "loaded"}}
	g := make(chan struct{})
	f.gate[urlA] = g
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); f.models() }()
	}
	waitFor(t, "the first probe", func() bool { return f.count(urlA) == 1 })
	close(g)
	wg.Wait()
	if f.count(urlA) != 1 {
		t.Fatalf("probes = %d, want 1", f.count(urlA))
	}
}

// A probe stores under the server it asked: A's late answer is never served
// for B.
func TestCatalogLlamaNeverFilesAUnderB(t *testing.T) {
	f := newCatalogLlamaFixture(t)
	f.answer[urlA] = []llama.Model{{ID: "model-on-A", Status: "loaded"}}
	f.answer[urlB] = []llama.Model{{ID: "model-on-B", Status: "loaded"}}
	f.point(urlA)
	f.models() // kept: A
	forgetCatalogLlama()
	g := make(chan struct{})
	f.mu.Lock()
	f.gate[urlA] = g
	f.mu.Unlock()
	f.models() // stale: serves A, refreshes A in the background (blocked)
	waitFor(t, "A's refresh", func() bool { return f.count(urlA) == 2 })
	f.point(urlB) // the owner switches servers mid-probe
	if got := f.models(); len(got) != 1 || got[0] != "model-on-B" {
		t.Fatalf("after the switch = %v", got)
	}
	close(g) // A's old refresh lands now
	waitFor(t, "A's refresh to land", func() bool {
		catalogLlama.Lock()
		defer catalogLlama.Unlock()
		return len(catalogLlama.flight) == 0
	})
	if got := f.models(); len(got) != 1 || got[0] != "model-on-B" {
		t.Fatalf("B served %v after A's late answer", got)
	}
}

// A forget during a probe is not lost: the probe's older answer is kept
// stale, so the next read asks again.
func TestCatalogLlamaForgetDuringAProbe(t *testing.T) {
	f := newCatalogLlamaFixture(t)
	f.point(urlA)
	f.answer[urlA] = []llama.Model{{ID: "old", Status: "loaded"}}
	f.models()
	forgetCatalogLlama()
	g := make(chan struct{})
	f.mu.Lock()
	f.gate[urlA] = g
	f.mu.Unlock()
	f.models() // background refresh starts, blocked
	waitFor(t, "the refresh", func() bool { return f.count(urlA) == 2 })
	forgetCatalogLlama() // e.g. a load finished while it was in flight
	f.mu.Lock()
	f.answer[urlA] = []llama.Model{{ID: "new", Status: "loaded"}}
	f.gate[urlA] = nil
	f.mu.Unlock()
	close(g)
	waitFor(t, "the refresh to land", func() bool {
		catalogLlama.Lock()
		defer catalogLlama.Unlock()
		return len(catalogLlama.flight) == 0
	})
	f.models() // stale again: must refresh
	waitFor(t, "a second refresh", func() bool { return f.count(urlA) == 3 })
	waitFor(t, "the new answer", func() bool { got := f.models(); return len(got) == 1 && got[0] == "new" })
}

// fresh waits for a new answer even when one is kept.
func TestCatalogLlamaFreshAsksAgain(t *testing.T) {
	f := newCatalogLlamaFixture(t)
	f.point(urlA)
	f.models()
	f.mu.Lock()
	f.answer[urlA] = []llama.Model{{ID: "started", Status: "loaded"}}
	f.mu.Unlock()
	rep := catalog.Report{Providers: []catalog.Provider{{ID: "llama.cpp"}}}
	attachLlamaModels(&rep, true)
	if f.count(urlA) != 2 || len(rep.Providers[0].Models) != 1 {
		t.Fatalf("probes=%d models=%v", f.count(urlA), rep.Providers[0].Models)
	}
}

// A model job's end and a service change forget; progress does not.
func TestCatalogLlamaListensForJobsAndTheService(t *testing.T) {
	f := newCatalogLlamaFixture(t)
	f.point(urlA)
	f.models()
	fd := &feed.Feed{}
	catalogLlamaListen(fd)
	stale := func() bool { catalogLlama.Lock(); defer catalogLlama.Unlock(); return catalogLlama.stale }
	fd.Publish(store.Event{Type: "llama.job", Data: []byte(`{"state":"running"}`)})
	if stale() {
		t.Fatal("a running job must not forget")
	}
	fd.Publish(store.Event{Type: "llama.job", Data: []byte(`{"state":"succeeded"}`)})
	if !stale() {
		t.Fatal("a finished job must forget")
	}
	f.models()
	waitFor(t, "the refresh", func() bool { return !stale() })
	fd.Publish(store.Event{Type: "llama.service", Data: []byte(`{"revision":2}`)})
	if !stale() {
		t.Fatal("a service change must forget")
	}
}
