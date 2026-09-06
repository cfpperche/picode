package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/catalog"
)

func TestLlamaOperationFailures(t *testing.T) {
	for _, tc := range []struct {
		name         string
		unloadStatus int
		waitFails    bool
		load         bool
	}{
		{"unload rejected", 500, false, false},
		{"unload completion failed", 200, true, false},
		{"replace unload rejected", 500, false, true},
		{"replace completion failed", 200, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("HOME", t.TempDir())
			unloaded, loaded := false, false
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/models/unload":
					unloaded = true
					w.WriteHeader(tc.unloadStatus)
					fmt.Fprint(w, `{}`)
				case "/models/load":
					loaded = true
					fmt.Fprint(w, `{}`)
				case "/models":
					if unloaded && tc.waitFails {
						http.Error(w, "failed", 500)
						return
					}
					fmt.Fprint(w, `{"data":[{"id":"old","status":{"value":"loaded"}}]}`)
				}
			}))
			defer upstream.Close()
			if err := catalog.PutLlama(upstream.URL, ""); err != nil {
				t.Fatal(err)
			}
			rec := httptest.NewRecorder()
			if tc.load {
				handleLlamaLoad(rec, httptest.NewRequest("POST", "/api/llama/load", strings.NewReader(`{"id":"new","unloadOthers":true}`)))
			} else {
				handleLlamaUnload(rec, httptest.NewRequest("POST", "/api/llama/unload", strings.NewReader(`{"id":"old"}`)))
			}
			if rec.Code != http.StatusBadGateway || loaded {
				t.Fatalf("status=%d load=%v body=%s", rec.Code, loaded, rec.Body.String())
			}
		})
	}
}
