package server

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func TestWorkspaceFavicon(t *testing.T) {
	ts := newTestServer(t, "cat")
	proj := t.TempDir()
	wk := addWorkspaceWithAgent(t, ts, "App", proj)
	url := ts.URL + "/api/workspaces/" + wk.ID + "/favicon"

	get := func() *http.Response { return do(t, ts.Client(), mustGet(t, url)) }

	if res := get(); res.StatusCode != http.StatusNotFound {
		t.Fatalf("no favicon = %d, want 404", res.StatusCode)
	}

	// public/ is found when the root has nothing.
	if err := os.MkdirAll(filepath.Join(proj, "public"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(proj, "public", "favicon.svg"), []byte("<svg/>"), 0o644); err != nil {
		t.Fatal(err)
	}
	res := get()
	if res.StatusCode != http.StatusOK || res.Header.Get("Content-Type") != "image/svg+xml" {
		t.Fatalf("public svg = %d %q", res.StatusCode, res.Header.Get("Content-Type"))
	}
	if cc := res.Header.Get("Cache-Control"); cc != "private, max-age=300" {
		t.Fatalf("cache-control = %q", cc)
	}
	if csp := res.Header.Get("Content-Security-Policy"); csp != "sandbox" {
		t.Fatalf("csp = %q", csp)
	}

	// The root outranks public/, and svg outranks png within a dir.
	if err := os.WriteFile(filepath.Join(proj, "favicon.png"), []byte("png-bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	if res := get(); res.Header.Get("Content-Type") != "image/png" {
		t.Fatalf("root png should win: %q", res.Header.Get("Content-Type"))
	}
	if err := os.WriteFile(filepath.Join(proj, "favicon.svg"), []byte("<svg/>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if res := get(); res.Header.Get("Content-Type") != "image/svg+xml" {
		t.Fatalf("root svg should win: %q", res.Header.Get("Content-Type"))
	}

	// An oversized file is skipped, falling through to the next candidate.
	if err := os.WriteFile(filepath.Join(proj, "favicon.svg"), make([]byte, maxFavicon+1), 0o644); err != nil {
		t.Fatal(err)
	}
	if res := get(); res.Header.Get("Content-Type") != "image/png" {
		t.Fatalf("oversized svg should be skipped: %q", res.Header.Get("Content-Type"))
	}

	if res := do(t, ts.Client(), mustGet(t, ts.URL+"/api/workspaces/ws-nope/favicon")); res.StatusCode != http.StatusNotFound {
		t.Fatalf("missing workspace = %d", res.StatusCode)
	}
}
