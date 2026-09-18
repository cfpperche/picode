package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWebappIconRejectsHTTPError(t *testing.T) {
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("image error placeholder"))
	}))
	defer site.Close()
	client := webappNewClient()
	defer client.CloseIdleConnections()
	if _, _, err := webappFetchIcon(t.Context(), client, site.URL); err == nil {
		t.Fatal("a missing icon must not be accepted as an installed icon")
	}
}

func TestWebAppRoutesRegistered(t *testing.T) {
	mux := http.NewServeMux()
	registerAll(mux, Deps{})
	for _, want := range []string{"GET /api/webapps", "POST /api/webapps", "POST /api/webapps/resolve", "PATCH /api/webapps/{id}", "DELETE /api/webapps/{id}", "GET /api/webapps/{id}/icon"} {
		found := false
		for _, route := range Routes() {
			if route.Method+" "+route.Pattern == want {
				found = true
			}
		}
		if !found {
			t.Errorf("missing route %s", want)
		}
	}
	_, pattern := mux.Handler(httptest.NewRequest(http.MethodGet, "/api/webapps", nil))
	if pattern != "GET /api/webapps" {
		t.Fatalf("webapps matched %q instead of its API", pattern)
	}
}
