package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/cfpperche/picode/internal/llamaservice"
	"github.com/cfpperche/picode/internal/store"
)

func TestOwnedServiceHTTP(t *testing.T) {
	dir := t.TempDir()
	st, err := store.Open(filepath.Join(dir, "db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	s, err := llamaservice.New(st, dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	mux := http.NewServeMux()
	registerLlamaService(mux, Deps{Store: st, LlamaService: s})
	cases := []struct {
		method, path, body string
		code               int
	}{
		{"GET", "/api/llama/service", "", 200},
		{"PUT", "/api/llama/service", `{"config":{"port":80,"threads":0,"context":10},"revision":0}`, 409},
		{"PUT", "/api/llama/service", `{"unknown":true}`, 400},
		{"POST", "/api/llama/service/execute", `{"token":"invented"}`, 409},
		{"GET", "/api/llama/service/diagnostics", "", 200},
		{"GET", "/api/llama/service/cache", "", 200},
	}
	if runtime.GOOS == "linux" {
		cases = append(cases, struct {
			method, path, body string
			code               int
		}{"PUT", "/api/llama/service", `{"config":{"port":18080,"threads":2,"context":4096,"jinja":true},"revision":0}`, 200})
	}
	for _, c := range cases {
		t.Run(c.method+c.path+c.body, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(c.method, c.path, bytes.NewBufferString(c.body))
			mux.ServeHTTP(w, r)
			if w.Code != c.code {
				t.Fatalf("%d: %s", w.Code, w.Body.String())
			}
		})
	}
}
