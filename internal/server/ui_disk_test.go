//go:build !embedui

package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/web"
)

// withUI gives a disk build something to serve. In an embedded build the UI is
// in the binary and this file does not compile in at all.
func withUI(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html>picode"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv(web.DirEnv, dir)
	return dir
}

// ADR-0023: a disk build can legitimately have no UI yet. A wall of 404s does
// not tell anyone that, so the server says it in words — and starts serving
// the moment the files appear, without a restart.
func TestUIHandlerSaysWhenTheUIIsNotBuilt(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(web.DirEnv, dir)

	ts := httptest.NewServer(uiHandler())
	t.Cleanup(ts.Close)

	res, err := ts.Client().Get(ts.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", res.StatusCode)
	}
	if !strings.Contains(string(body), "make web") {
		t.Fatalf("the message must name the command that fixes it: %q", body)
	}

	// Now build it, without restarting the server.
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html>hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err = ts.Client().Get(ts.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	body, _ = io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != http.StatusOK || !strings.Contains(string(body), "hi") {
		t.Fatalf("status=%d body=%q — the page should work as soon as make web finishes", res.StatusCode, body)
	}
}
