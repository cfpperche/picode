package pipkg

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseNpmSearch(t *testing.T) {
	hits, err := parseNpmSearch([]byte(`{
	  "objects": [
	    {"package": {"name": "pi-web-search", "version": "1.3.1", "description": "Provider-native web search"}},
	    {"package": {"name": "", "version": "0"}}
	  ]
	}`))
	if err != nil || len(hits) != 1 || hits[0].Source != "npm:pi-web-search" {
		t.Fatalf("%v %+v", err, hits)
	}
}

func TestSearchGallery(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("text") == "" {
			t.Fatal("missing text")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"objects":[{"package":{"name":"pi-mcp-adapter","version":"2.0.0","description":"MCP"}}]}`))
	}))
	defer ts.Close()
	page, err := searchGallery(context.Background(), ts.Client(), ts.URL, "mcp")
	if err != nil || len(page.Hits) != 1 || page.Hits[0].Name != "pi-mcp-adapter" {
		t.Fatalf("%v %+v", err, page)
	}
}
