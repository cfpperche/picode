package server

import (
	"errors"
	"net/http"
	"os"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/catalog"
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

// The catalog's llama.cpp read is kept, a failure included: a server that
// accepts and never answers cost every /api/catalog its 2 s timeout.
func TestCatalogLlamaIsKept(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeAuth := func(url string) {
		dir := filepath.Join(home, ".pi", "agent")
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
		body := `{"llama.cpp":{"type":"api_key","key":"k","env":{"LLAMA_BASE_URL":"` + url + `"}}}`
		if err := os.WriteFile(filepath.Join(dir, "auth.json"), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	writeAuth("http://127.0.0.1:1")
	calls := 0
	fail := true
	prev := listCatalogLlama
	listCatalogLlama = func() ([]llama.Model, error) {
		calls++
		if fail {
			return nil, errors.New("timeout")
		}
		return []llama.Model{{ID: "qwen", Status: "loaded"}}, nil
	}
	t.Cleanup(func() { listCatalogLlama = prev; forgetCatalogLlama() })
	forgetCatalogLlama()

	rep := catalog.Report{Providers: []catalog.Provider{{ID: "llama.cpp"}}}
	attachLlamaModels(&rep)
	attachLlamaModels(&rep)
	if calls != 1 {
		t.Fatalf("a failed read asked %d times, want 1", calls)
	}
	// A changed server is a different entry, never the old answer.
	fail = false
	writeAuth("http://127.0.0.1:2")
	attachLlamaModels(&rep)
	if calls != 2 || len(rep.Providers[0].Models) != 1 {
		t.Fatalf("calls=%d models=%v after the URL changed", calls, rep.Providers[0].Models)
	}
	// A model operation drops the kept answer.
	forgetCatalogLlama()
	attachLlamaModels(&catalog.Report{})
	rep2 := catalog.Report{Providers: []catalog.Provider{{ID: "llama.cpp"}}}
	attachLlamaModels(&rep2)
	if calls != 3 {
		t.Fatalf("calls=%d after forget, want 3", calls)
	}
}
