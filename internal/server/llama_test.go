package server

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

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
