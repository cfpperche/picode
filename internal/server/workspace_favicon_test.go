package server

import (
	"io"
	"net/http"
	"net/http/httptest"
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

// A frontend living in a subfolder (PiCode's own shape: web/public/) still
// gets its favicon found.
func TestWorkspaceFaviconNestedFrontend(t *testing.T) {
	ts := newTestServer(t, "cat")
	proj := t.TempDir()
	wk := addWorkspaceWithAgent(t, ts, "Mono", proj)
	if err := os.MkdirAll(filepath.Join(proj, "web", "public"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(proj, "web", "public", "favicon.svg"), []byte("<svg/>"), 0o644); err != nil {
		t.Fatal(err)
	}
	res := do(t, ts.Client(), mustGet(t, ts.URL+"/api/workspaces/"+wk.ID+"/favicon"))
	if res.StatusCode != http.StatusOK || res.Header.Get("Content-Type") != "image/svg+xml" {
		t.Fatalf("nested favicon = %d %q", res.StatusCode, res.Header.Get("Content-Type"))
	}
}

func writeFavicon(t *testing.T, root, rel string, data []byte) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func getFavicon(t *testing.T, ts *httptest.Server, wkID string) *http.Response {
	t.Helper()
	return do(t, ts.Client(), mustGet(t, ts.URL+"/api/workspaces/"+wkID+"/favicon"))
}

func faviconBody(t *testing.T, res *http.Response) string {
	t.Helper()
	b, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestWorkspaceFaviconNextAppRouter(t *testing.T) {
	ts := newTestServer(t, "cat")
	proj := t.TempDir()
	wk := addWorkspaceWithAgent(t, ts, "Next", proj)
	writeFavicon(t, proj, "app/icon.svg", []byte("<svg id='next'/>"))
	res := getFavicon(t, ts, wk.ID)
	if res.StatusCode != http.StatusOK || res.Header.Get("Content-Type") != "image/svg+xml" {
		t.Fatalf("app/icon.svg = %d %q", res.StatusCode, res.Header.Get("Content-Type"))
	}
	if got := faviconBody(t, res); got != "<svg id='next'/>" {
		t.Fatalf("body = %q", got)
	}
}

func TestWorkspaceFaviconAppsWebMonorepo(t *testing.T) {
	ts := newTestServer(t, "cat")
	proj := t.TempDir()
	wk := addWorkspaceWithAgent(t, ts, "Turbo", proj)
	writeFavicon(t, proj, "apps/web/app/icon.svg", []byte("<svg id='web'/>"))
	res := getFavicon(t, ts, wk.ID)
	if res.StatusCode != http.StatusOK || res.Header.Get("Content-Type") != "image/svg+xml" {
		t.Fatalf("apps/web/app/icon.svg = %d %q", res.StatusCode, res.Header.Get("Content-Type"))
	}
	if got := faviconBody(t, res); got != "<svg id='web'/>" {
		t.Fatalf("body = %q", got)
	}
}

func TestWorkspaceFaviconAppsOtherName(t *testing.T) {
	ts := newTestServer(t, "cat")
	proj := t.TempDir()
	wk := addWorkspaceWithAgent(t, ts, "Site", proj)
	writeFavicon(t, proj, "apps/site/app/icon.svg", []byte("<svg id='site'/>"))
	res := getFavicon(t, ts, wk.ID)
	if res.StatusCode != http.StatusOK || res.Header.Get("Content-Type") != "image/svg+xml" {
		t.Fatalf("apps/site/app/icon.svg = %d %q", res.StatusCode, res.Header.Get("Content-Type"))
	}
	if got := faviconBody(t, res); got != "<svg id='site'/>" {
		t.Fatalf("body = %q", got)
	}
}

func TestWorkspaceFaviconIconSvgBeatsIco(t *testing.T) {
	ts := newTestServer(t, "cat")
	proj := t.TempDir()
	wk := addWorkspaceWithAgent(t, ts, "Both", proj)
	writeFavicon(t, proj, "app/favicon.ico", []byte("ico-bytes"))
	writeFavicon(t, proj, "app/icon.svg", []byte("<svg id='mark'/>"))
	res := getFavicon(t, ts, wk.ID)
	if res.Header.Get("Content-Type") != "image/svg+xml" {
		t.Fatalf("icon.svg should beat favicon.ico: %q", res.Header.Get("Content-Type"))
	}
	if got := faviconBody(t, res); got != "<svg id='mark'/>" {
		t.Fatalf("body = %q", got)
	}
}

func TestWorkspaceFaviconRootBeatsApps(t *testing.T) {
	ts := newTestServer(t, "cat")
	proj := t.TempDir()
	wk := addWorkspaceWithAgent(t, ts, "Root", proj)
	writeFavicon(t, proj, "apps/web/app/icon.svg", []byte("<svg id='nested'/>"))
	writeFavicon(t, proj, "favicon.svg", []byte("<svg id='root'/>"))
	res := getFavicon(t, ts, wk.ID)
	if got := faviconBody(t, res); got != "<svg id='root'/>" {
		t.Fatalf("root should win: %q", got)
	}
}

func TestWorkspaceFaviconSkipsDotAndNodeModulesApps(t *testing.T) {
	ts := newTestServer(t, "cat")
	proj := t.TempDir()
	wk := addWorkspaceWithAgent(t, ts, "Skip", proj)
	writeFavicon(t, proj, "apps/node_modules/app/icon.svg", []byte("<svg id='nm'/>"))
	writeFavicon(t, proj, "apps/.cache/app/icon.svg", []byte("<svg id='dot'/>"))
	res := getFavicon(t, ts, wk.ID)
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("hidden/node_modules apps should be ignored, got %d", res.StatusCode)
	}
}

func TestFindWorkspaceFaviconPrefersWebApp(t *testing.T) {
	root := t.TempDir()
	writeFavicon(t, root, "apps/site/app/icon.svg", []byte("site"))
	writeFavicon(t, root, "apps/web/app/icon.svg", []byte("web"))
	abs, ok := findWorkspaceFavicon(root)
	if !ok {
		t.Fatal("expected a favicon")
	}
	got, err := os.ReadFile(abs)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "web" {
		t.Fatalf("apps/web should outrank other apps: %q", got)
	}
}
