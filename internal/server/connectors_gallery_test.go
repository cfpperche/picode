package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cfpperche/picode/internal/mcpcatalog"
)

func galleryHits(t *testing.T, ts *httptest.Server, q string) []map[string]any {
	t.Helper()
	res, err := ts.Client().Get(ts.URL + "/api/connectors/gallery?q=" + q)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/connectors/gallery status %d", res.StatusCode)
	}
	var body struct {
		Hits []map[string]any `json:"hits"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	return body.Hits
}

func hitString(hit map[string]any, key string) string {
	s, _ := hit[key].(string)
	return s
}

func TestConnectorsGalleryServesSeed(t *testing.T) {
	ts := newTestServer(t, "cat")

	hits := galleryHits(t, ts, "")
	if len(hits) == 0 {
		t.Fatal("gallery returned no hits")
	}
	if hitString(hits[0], "source") != "picode" {
		t.Fatalf("first hit = %+v, want seed first", hits[0])
	}
	var deepwiki map[string]any
	for _, hit := range hits {
		if hitString(hit, "id") == "deepwiki" {
			deepwiki = hit
			break
		}
	}
	if deepwiki == nil {
		t.Fatal("deepwiki missing from gallery")
	}
	if hitString(deepwiki, "url") != "https://mcp.deepwiki.com/mcp" ||
		hitString(deepwiki, "kind") != "url" ||
		hitString(deepwiki, "source") != "picode" ||
		deepwiki["featured"] != true {
		t.Fatalf("deepwiki hit = %+v", deepwiki)
	}

	// Search narrows to the matching card, case-insensitively.
	hits = galleryHits(t, ts, "chrome")
	if len(hits) != 1 || hitString(hits[0], "id") != "chrome-devtools" {
		t.Fatalf("q=chrome hits = %+v", hits)
	}
	if hitString(hits[0], "kind") != "stdio" || hitString(hits[0], "command") != "npx" {
		t.Fatalf("chrome-devtools hit = %+v", hits[0])
	}
	hits = galleryHits(t, ts, "COPILOT")
	if len(hits) != 1 || hitString(hits[0], "id") != "github" || hitString(hits[0], "auth") != "oauth" {
		t.Fatalf("q=COPILOT hits = %+v", hits)
	}
	hits = galleryHits(t, ts, "no-such-connector-xyz")
	if len(hits) != 0 {
		t.Fatalf("q miss hits = %+v", hits)
	}
}

func TestConnectorsGalleryRegistryAfterRefresh(t *testing.T) {
	token := "syncedwikitest"
	pages := map[string]string{
		"": `{"servers":[{"server":{"name":"io.picodetest/synced","title":"Synced Test",` +
			`"description":"A ` + token + ` connector","remotes":[{"type":"streamable-http","url":"https://synced.example/mcp"}]},` +
			`"_meta":{"io.modelcontextprotocol.registry/official":{"status":"active"}}}],"metadata":{}}`,
	}
	var reg = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, pages[r.URL.Query().Get("cursor")])
	}))
	t.Cleanup(reg.Close)

	st := mcpcatalog.NewStore(t.TempDir())
	st.Base = reg.URL
	ts := httptest.NewServer(New("127.0.0.1:0", Deps{Connectors: st}).Handler)
	t.Cleanup(ts.Close)

	// The seed answers regardless of the registry; seeds stay first.
	hits := galleryHits(t, ts, "")
	if len(hits) == 0 || hitString(hits[0], "source") != "picode" {
		t.Fatalf("first hit = %+v, want seed first", hits[0])
	}

	if err := st.Refresh(context.Background()); err != nil {
		t.Fatalf("refresh: %v", err)
	}
	hits = galleryHits(t, ts, token)
	if len(hits) != 1 || hitString(hits[0], "id") != "io.picodetest/synced" {
		t.Fatalf("registry hits = %+v", hits)
	}
	if hitString(hits[0], "source") != "registry" || hits[0]["featured"] != false ||
		hitString(hits[0], "url") != "https://synced.example/mcp" {
		t.Fatalf("registry hit = %+v", hits[0])
	}
	// Full listing: seed cards first, registry row appended.
	hits = galleryHits(t, ts, "")
	if hitString(hits[len(hits)-1], "id") != "io.picodetest/synced" {
		t.Fatalf("registry row not after seed: last = %+v", hits[len(hits)-1])
	}
}
