package server

import (
	"crypto/sha256"
	"encoding/base64"
	"io"
	"net/http"
	"regexp"
	"strings"
	"testing"
)

func TestAppShellCarriesAHashedCSP(t *testing.T) {
	ts := newTestServer(t, "cat")
	res, err := ts.Client().Get(ts.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != 200 || !strings.Contains(string(body), "<script>") {
		t.Skipf("no built UI in this test binary (status %d)", res.StatusCode)
	}
	csp := res.Header.Get("Content-Security-Policy")
	if csp == "" {
		t.Fatal("index.html without a CSP")
	}
	// The hash names exactly the inline bootstrap in the served HTML.
	m := regexp.MustCompile(`(?s)<script>(.*?)</script>`).FindSubmatch(body)
	if m == nil {
		t.Fatal("no inline script to hash")
	}
	sum := sha256.Sum256(m[1])
	want := "'sha256-" + base64.StdEncoding.EncodeToString(sum[:]) + "'"
	if !strings.Contains(csp, want) {
		t.Fatalf("csp %q lacks %s", csp, want)
	}
	if strings.Contains(csp, "'unsafe-inline'") && !strings.Contains(csp, "style-src 'self' 'unsafe-inline'") {
		t.Fatalf("unsafe-inline must stay on styles only: %q", csp)
	}
	if !strings.Contains(csp, "script-src 'self' 'wasm-unsafe-eval' 'sha256-") || strings.Contains(csp, "unsafe-eval'") {
		t.Fatalf("script-src: %q", csp)
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
