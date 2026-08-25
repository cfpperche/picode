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
	    {"downloads": {"monthly": 1500}, "package": {"name": "pi-web-search", "version": "1.3.1", "description": "Provider-native web search", "keywords": ["pi-package", "pi-extension"], "publisher": {"username": "ttttmr"}}},
	    {"package": {"name": "", "version": "0"}}
	  ]
	}`))
	if err != nil || len(hits) != 1 || hits[0].Source != "npm:pi-web-search" || hits[0].Kind != "extension" || hits[0].Downloads != 1500 {
		t.Fatalf("%v %+v", err, hits)
	}
}

func TestAttachPreviews(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"previews":[{"name":"pi-web-search","media":{"url":"https://example.com/p.png"}},{"name":"none","media":null}]}`))
	}))
	defer ts.Close()
	hits := []Hit{{Name: "pi-web-search"}, {Name: "none"}}
	attachPreviews(context.Background(), ts.Client(), ts.URL, hits)
	if hits[0].Image != "https://example.com/p.png" || hits[1].Image != "" {
		t.Fatalf("%+v", hits)
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
	page, err := searchGallery(context.Background(), ts.Client(), ts.URL, "mcp", "")
	if err != nil || len(page.Hits) != 1 || page.Hits[0].Name != "pi-mcp-adapter" {
		t.Fatalf("%v %+v", err, page)
	}
}
