package server

import (
	"crypto/sha256"
	"encoding/base64"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/web"
)

// writeUI gives the (non-embedded) build a UI to serve, so these tests run
// whatever `make web` last produced: the launcher has no inline script, each
// shell carries its theme bootstrap. In an embedded build web.DirEnv is
// ignored and the assertions run against the UI in the binary — the same
// invariant, real files.
func writeUI(t *testing.T, files map[string]string) {
	t.Helper()
	dir := t.TempDir()
	for name, body := range files {
		full := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv(web.DirEnv, dir)
}

// shellUI is the shape the build produces: a launcher page with no inline
// script, and three shells whose inline theme bootstrap must run before the
// stylesheet — the one script each policy has to name by hash.
func shellUI() map[string]string {
	return map[string]string{
		"index.html":         "<html><body><p>Opening PiCode…</p><script src=\"/assets/launcher.js\"></script></body></html>",
		"browser/index.html": "<html><head><script>theme('browser')</script></head><body><div id=\"root\"></div></body></html>",
		"desktop/index.html": "<html><head><script>theme('desktop')</script></head><body><div id=\"root\"></div></body></html>",
		"mobile/index.html":  "<html><head><script>theme('mobile')</script></head><body><div id=\"root\"></div></body></html>",
	}
}

func TestAppShellCarriesAHashedCSP(t *testing.T) {
	// The shape of the policy, and that assets stay policy-free. Which hash
	// belongs to which shell is TestEveryServedShellCarriesItsOwnHash's job.
	writeUI(t, shellUI())
	ts := newTestServer(t, "cat")
	res, err := ts.Client().Get(ts.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != 200 {
		t.Skipf("no UI in this build (status %d)", res.StatusCode)
	}
	csp := res.Header.Get("Content-Security-Policy")
	if csp == "" {
		t.Fatal("/ served without a CSP")
	}
	if strings.Contains(csp, "'unsafe-inline'") && !strings.Contains(csp, "style-src 'self' 'unsafe-inline'") {
		t.Fatalf("unsafe-inline must stay on styles only: %q", csp)
	}
	if !strings.Contains(csp, "script-src 'self' 'wasm-unsafe-eval'") || strings.Contains(csp, "'unsafe-eval'") {
		t.Fatalf("script-src: %q", csp)
	}
	if !strings.Contains(csp, "frame-src 'self' http://localhost:* http://127.0.0.1:*") {
		t.Fatalf("frame-src should allow this machine's servers, nothing else: %q", csp)
	}
	if !strings.Contains(csp, "connect-src 'self' ws://"+strings.TrimPrefix(ts.URL, "http://")) {
		t.Fatalf("connect-src should name this host's websocket: %q", csp)
	}
	// Assets carry no policy (workers keep their own context).
	req, _ := http.NewRequest("GET", ts.URL+"/assets/nope.js", nil)
	res2, _ := ts.Client().Do(req)
	res2.Body.Close()
	if res2.Header.Get("Content-Security-Policy") != "" {
		t.Fatal("assets must not carry the document policy")
	}
}

// TestEveryServedShellCarriesItsOwnHash is the test the coverage bug needed.
// The policy reached only "/", "/index.html" and "*.html", so in practice
// /browser/, /desktop/ and /mobile/ — the URLs the launcher sends a browser
// to and the Windows shell loads — carried no policy at all; and the hash was
// computed from the launcher's index.html, which has no inline script, so
// adding the coverage without the per-file hash would have blocked each
// shell's theme bootstrap. One row per served shell: the policy is there, and
// it names the inline scripts of the file that URL serves, exactly.
func TestEveryServedShellCarriesItsOwnHash(t *testing.T) {
	writeUI(t, shellUI())
	ts := newTestServer(t, "cat")
	for _, path := range []string{"/", "/browser/", "/desktop/", "/mobile/"} {
		t.Run(path, func(t *testing.T) {
			res, err := ts.Client().Get(ts.URL + path)
			if err != nil {
				t.Fatal(err)
			}
			body, _ := io.ReadAll(res.Body)
			res.Body.Close()
			if res.StatusCode != 200 {
				t.Fatalf("%s: status %d", path, res.StatusCode)
			}
			csp := res.Header.Get("Content-Security-Policy")
			if csp == "" {
				t.Fatalf("%s served without a CSP", path)
			}
			scripts := regexp.MustCompile(`(?s)<script>(.*?)</script>`).FindAllSubmatch(body, -1)
			if got := strings.Count(csp, "'sha256-"); got != len(scripts) {
				t.Fatalf("%s: %d inline script(s) in the page, %d hash(es) in the policy", path, len(scripts), got)
			}
			for _, m := range scripts {
				sum := sha256.Sum256(m[1])
				want := "'sha256-" + base64.StdEncoding.EncodeToString(sum[:]) + "'"
				if !strings.Contains(csp, want) {
					t.Fatalf("%s: the policy lacks the hash of its own inline script", path)
				}
			}
		})
	}
}

func TestHashInline(t *testing.T) {
	h := hashInline([]byte("<html><script>alert(1)</script><script src=x></script><script>b()</script>"))
	if strings.Count(h, "'sha256-") != 2 {
		t.Fatalf("two inline scripts expected: %q", h)
	}
	if hashInline([]byte("<p>none</p>")) != "" {
		t.Fatal("no inline script, no hash")
	}
}

func TestPairPageCSP(t *testing.T) {
	ts, _ := newAuthServer(t)
	res, err := ts.Client().Get(ts.URL + "/pair?code=x")
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.Header.Get("Content-Security-Policy") != PageCSP {
		t.Fatalf("pair csp %q", res.Header.Get("Content-Security-Policy"))
	}
}
