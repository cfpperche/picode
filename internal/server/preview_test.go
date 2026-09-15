package server

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/preview"
	"github.com/cfpperche/picode/internal/store"
)

// HTML preview decision table (ADR-0136): mint validation, the serve route's
// containment/allowlist/headers, and the ticket's expiry + session binding.

type previewFixture struct {
	ts   *httptest.Server
	st   *store.Store
	deps Deps
	root string
	ws   store.Workspace
}

func newPreviewFixture(t *testing.T, ttl time.Duration) *previewFixture {
	t.Helper()
	root := t.TempDir()
	write := func(rel, body string) {
		t.Helper()
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("site/index.html", "<!doctype html><html><body><h1>hello preview</h1><script src=\"app.js\"></script></body></html>")
	write("site/app.js", "document.title='preview';")
	write("site/mod.mjs", "export default 1;")
	write("site/data.json", `{"ok":true}`)
	write("site/notes.md", "# not a preview")
	write("site/.hidden.html", "<p>hidden</p>")
	write("site/sub/index.html", "<p>sub</p>")
	if err := os.MkdirAll(filepath.Join(root, "site", "weird.html"), 0o755); err != nil {
		t.Fatal(err)
	}
	write(".env", "SECRET=1")
	write("outside-copy.html", "<p>outside copy</p>")
	write("big.html", "")
	if err := os.Truncate(filepath.Join(root, "big.html"), maxAgentBlob+1); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret.html"), []byte("<p>secret</p>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, "secret.html"), filepath.Join(root, "site", "link.html")); err != nil {
		t.Fatal(err)
	}

	st, err := store.Open(filepath.Join(t.TempDir(), "picode.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	ws, err := st.AddWorkspace("preview", root)
	if err != nil {
		t.Fatal(err)
	}
	deps := Deps{Store: st, Previews: preview.NewStore(ttl)}
	ts := httptest.NewServer(New("127.0.0.1:0", deps).Handler)
	t.Cleanup(ts.Close)
	return &previewFixture{ts: ts, st: st, deps: deps, root: root, ws: ws}
}

func (f *previewFixture) mint(t *testing.T, query, body string) (int, map[string]any) {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, f.ts.URL+"/api/previews"+query, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := f.ts.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	var out map[string]any
	_ = json.NewDecoder(res.Body).Decode(&out)
	return res.StatusCode, out
}

func (f *previewFixture) get(t *testing.T, path string) (*http.Response, string) {
	t.Helper()
	res, err := f.ts.Client().Get(f.ts.URL + path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	b, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	return res, string(b)
}

func (f *previewFixture) mintURL(t *testing.T, rel string) string {
	t.Helper()
	code, out := f.mint(t, "", fmt.Sprintf(`{"kind":"workspace","id":%q,"path":%q}`, f.ws.ID, rel))
	if code != http.StatusOK {
		t.Fatalf("mint %s: status %d body %v", rel, code, out)
	}
	raw, _ := out["url"].(string)
	if !strings.HasPrefix(raw, "/preview/") {
		t.Fatalf("mint %s: url %q", rel, raw)
	}
	if _, ok := out["expiresAt"].(string); !ok {
		t.Fatalf("mint %s: no expiresAt in %v", rel, out)
	}
	return raw
}

func TestPreviewMintTable(t *testing.T) {
	f := newPreviewFixture(t, time.Hour)
	body := func(path string) string {
		return fmt.Sprintf(`{"kind":"workspace","id":%q,"path":%q}`, f.ws.ID, path)
	}
	rows := []struct {
		name   string
		query  string
		body   string
		status int
		msg    string
	}{
		{"document", "", body("site/index.html"), http.StatusOK, ""},
		{"htm also previews", "", body("site/index.html"), http.StatusOK, ""},
		{"unknown kind", "", `{"kind":"banana","id":"x","path":"a.html"}`, http.StatusBadRequest, "kind must be"},
		{"unknown owner", "", `{"kind":"workspace","id":"nope","path":"a.html"}`, http.StatusNotFound, "not found"},
		{"missing path", "", fmt.Sprintf(`{"kind":"workspace","id":%q}`, f.ws.ID), http.StatusBadRequest, "path is required"},
		{"not html", "", body("site/notes.md"), http.StatusBadRequest, ".html"},
		{"hidden file", "", body("site/.hidden.html"), http.StatusNotFound, "not available"},
		{"escapes root", "", body("../outside-copy.html"), http.StatusBadRequest, "escapes"},
		{"directory", "", body("site/sub"), http.StatusBadRequest, ".html"},
		{"directory named like a document", "", body("site/weird.html"), http.StatusBadRequest, "folder"},
		{"gone", "", body("site/gone.html"), http.StatusNotFound, "that file is gone"},
		{"too large", "", body("big.html"), http.StatusRequestEntityTooLarge, "too large"},
		{"root precondition (query)", "?root=/not/this/folder", body("site/index.html"), http.StatusConflict, "folder changed"},
		{"root precondition (body)", "", fmt.Sprintf(`{"kind":"workspace","id":%q,"path":"site/index.html","root":"/not/this/folder"}`, f.ws.ID), http.StatusConflict, "folder changed"},
		{"root matches", "", fmt.Sprintf(`{"kind":"workspace","id":%q,"path":"site/index.html","root":%q}`, f.ws.ID, canonDir(f.root)), http.StatusOK, ""},
		{"bad json", "", `{`, http.StatusBadRequest, "invalid JSON"},
	}
	for _, r := range rows {
		t.Run(r.name, func(t *testing.T) {
			code, out := f.mint(t, r.query, r.body)
			if code != r.status {
				t.Fatalf("status=%d want %d body=%v", code, r.status, out)
			}
			if r.msg != "" {
				got, _ := out["error"].(string)
				if !strings.Contains(got, r.msg) {
					t.Fatalf("error=%q want substring %q", got, r.msg)
				}
			}
		})
	}
}

func TestPreviewServeTable(t *testing.T) {
	f := newPreviewFixture(t, time.Hour)
	docURL := f.mintURL(t, "site/index.html")
	token := strings.Split(strings.TrimPrefix(docURL, "/preview/"), "/")[0]

	rows := []struct {
		name   string
		path   string
		status int
		mime   string
		body   string
	}{
		{"document", docURL, http.StatusOK, "text/html; charset=utf-8", "hello preview"},
		{"root path serves the document", "/preview/" + token + "/", http.StatusOK, "text/html; charset=utf-8", "hello preview"},
		{"relative script", "/preview/" + token + "/site/app.js", http.StatusOK, "text/javascript; charset=utf-8", "preview"},
		{"module", "/preview/" + token + "/site/mod.mjs", http.StatusOK, "text/javascript; charset=utf-8", "export default"},
		{"json", "/preview/" + token + "/site/data.json", http.StatusOK, "application/json; charset=utf-8", `"ok"`},
		{"directory index", "/preview/" + token + "/site/sub/", http.StatusOK, "text/html; charset=utf-8", "<p>sub</p>"},
		{"hidden file", "/preview/" + token + "/site/.hidden.html", http.StatusNotFound, "text/plain; charset=utf-8", ""},
		{"dotfile at root", "/preview/" + token + "/.env", http.StatusNotFound, "text/plain; charset=utf-8", ""},
		{"off the allowlist", "/preview/" + token + "/site/notes.md", http.StatusNotFound, "text/plain; charset=utf-8", ""},
		{"symlink escape", "/preview/" + token + "/site/link.html", http.StatusNotFound, "text/plain; charset=utf-8", ""},
		{"missing asset", "/preview/" + token + "/site/nope.js", http.StatusNotFound, "text/plain; charset=utf-8", ""},
		{"unknown token", "/preview/" + strings.Repeat("a", 43) + "/site/index.html", http.StatusNotFound, "text/plain; charset=utf-8", ""},
		{"empty token prefix", "/preview/", http.StatusNotFound, "text/plain; charset=utf-8", ""},
	}
	for _, r := range rows {
		t.Run(r.name, func(t *testing.T) {
			res, body := f.get(t, r.path)
			if res.StatusCode != r.status {
				t.Fatalf("status=%d want %d body=%q", res.StatusCode, r.status, body)
			}
			if got := res.Header.Get("Content-Type"); r.mime != "" && got != r.mime {
				t.Fatalf("content-type=%q want %q", got, r.mime)
			}
			if r.body != "" && !strings.Contains(body, r.body) {
				t.Fatalf("body=%q missing %q", body, r.body)
			}
		})
	}

	t.Run("sandbox headers on the document", func(t *testing.T) {
		res, _ := f.get(t, docURL)
		csp := res.Header.Get("Content-Security-Policy")
		if !strings.HasPrefix(csp, "sandbox allow-scripts") {
			t.Fatalf("csp=%q", csp)
		}
		if strings.Contains(csp, "allow-same-origin") {
			t.Fatalf("csp allows same-origin: %q", csp)
		}
		if !strings.Contains(csp, "frame-ancestors 'self'") {
			t.Fatalf("csp=%q", csp)
		}
		want := map[string]string{
			"Referrer-Policy":              "no-referrer",
			"X-Content-Type-Options":       "nosniff",
			"Cache-Control":                "no-store",
			"Access-Control-Allow-Origin":  "*",
			"Cross-Origin-Resource-Policy": "cross-origin",
		}
		for k, v := range want {
			if got := res.Header.Get(k); got != v {
				t.Fatalf("%s=%q want %q", k, got, v)
			}
		}
		if pp := res.Header.Get("Permissions-Policy"); !strings.Contains(pp, "camera=()") {
			t.Fatalf("permissions-policy=%q", pp)
		}
	})

	t.Run("sandbox headers on assets too", func(t *testing.T) {
		res, _ := f.get(t, "/preview/"+token+"/site/app.js")
		if res.Header.Get("Access-Control-Allow-Origin") != "*" {
			t.Fatal("assets need CORS for modules and fetch")
		}
		if !strings.HasPrefix(res.Header.Get("Content-Security-Policy"), "sandbox") {
			t.Fatal("a navigated asset must stay sandboxed")
		}
	})

	t.Run("head has no body", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodHead, f.ts.URL+docURL, nil)
		res, err := f.ts.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = res.Body.Close() }()
		if res.StatusCode != http.StatusOK {
			t.Fatalf("head status=%d", res.StatusCode)
		}
		if res.Header.Get("ETag") == "" {
			t.Fatal("no ETag for the reload check")
		}
	})

	t.Run("post is refused", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, f.ts.URL+docURL, nil)
		res, err := f.ts.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = res.Body.Close() }()
		if res.StatusCode != http.StatusMethodNotAllowed {
			body, _ := io.ReadAll(res.Body)
			t.Fatalf("post status=%d body=%q allow=%q", res.StatusCode, body, res.Header.Get("Allow"))
		}
	})
}

func TestPreviewTicketExpiry(t *testing.T) {
	f := newPreviewFixture(t, time.Millisecond)
	tk, err := f.deps.Previews.Mint("workspace", f.ws.ID, f.root, "site/index.html", "")
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(5 * time.Millisecond)
	if res, _ := f.get(t, "/preview/"+tk.Token+"/site/index.html"); res.StatusCode != http.StatusNotFound {
		t.Fatalf("expired ticket status=%d", res.StatusCode)
	}
}

func TestPreviewTicketDiesWithItsSession(t *testing.T) {
	f := newPreviewFixture(t, time.Hour)
	sess, _, err := f.st.CreateSession(store.SessionBrowser, "", "preview test", "127.0.0.1", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	tk, err := f.deps.Previews.Mint("workspace", f.ws.ID, f.root, "site/index.html", sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	url := "/preview/" + tk.Token + "/site/index.html"
	if res, _ := f.get(t, url); res.StatusCode != http.StatusOK {
		t.Fatalf("live session status=%d", res.StatusCode)
	}
	if err := f.st.RevokeSession(sess.ID); err != nil {
		t.Fatal(err)
	}
	if res, _ := f.get(t, url); res.StatusCode != http.StatusNotFound {
		t.Fatalf("revoked session status=%d", res.StatusCode)
	}
}

func TestPreviewAssetForPolicy(t *testing.T) {
	f := newPreviewFixture(t, time.Hour)
	root := canonDir(f.root)
	rows := []struct {
		rel  string
		ok   bool
		mime string
	}{
		{"site/app.js", true, "text/javascript; charset=utf-8"},
		{"site/", true, "text/html; charset=utf-8"},
		{"site", true, "text/html; charset=utf-8"},
		{"site/sub/", true, "text/html; charset=utf-8"},
		{"../outside-copy.html", false, ""},
		{"site/../../etc/passwd", false, ""},
		{"/etc/passwd", false, ""},
		{"site/.hidden.html", false, ""},
		{"site/notes.md", false, ""},
		{"site/link.html", false, ""},
		{"site/nope.js", false, ""},
	}
	for _, r := range rows {
		asset, err := previewAssetFor(root, r.rel)
		if r.ok != (err == nil) {
			t.Fatalf("%s: err=%v want ok=%v", r.rel, err, r.ok)
		}
		if r.ok && asset.mime != r.mime {
			t.Fatalf("%s: mime=%q want %q", r.rel, asset.mime, r.mime)
		}
	}
}

func TestSecurityHeadersSkipPreview(t *testing.T) {
	h := securityHeaders(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/preview/tok/index.html", nil))
	if got := rec.Header().Get("Content-Security-Policy"); got != "" {
		t.Fatalf("preview paths must carry their own policy, got %q", got)
	}
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/index.html", nil))
	if !strings.Contains(rec.Header().Get("Content-Security-Policy"), "default-src 'self'") {
		t.Fatal("the app shell still gets its policy")
	}
}

func TestPreviewServeRecordsWatchedPaths(t *testing.T) {
	f := newPreviewFixture(t, time.Hour)
	url := f.mintURL(t, "site/index.html")
	token := strings.Split(strings.TrimPrefix(url, "/preview/"), "/")[0]
	f.get(t, url)
	f.get(t, "/preview/"+token+"/site/app.js")
	root := canonDir(f.root)

	watched := f.deps.Previews.Watched(token)
	seen := map[string]bool{}
	for _, p := range watched {
		seen[p] = true
	}
	if !seen[filepath.Join(root, "site", "index.html")] {
		t.Fatalf("document not watched: %v", watched)
	}
	if !seen[filepath.Join(root, "site", "app.js")] {
		t.Fatalf("asset not watched: %v", watched)
	}
}

func TestPreviewEventsStream(t *testing.T) {
	old := previewWatchInterval
	previewWatchInterval = 10 * time.Millisecond
	t.Cleanup(func() { previewWatchInterval = old })

	f := newPreviewFixture(t, time.Hour)
	url := f.mintURL(t, "site/index.html")
	token := strings.Split(strings.TrimPrefix(url, "/preview/"), "/")[0]
	f.get(t, url)
	f.get(t, "/preview/"+token+"/site/app.js")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	timer := time.AfterFunc(5*time.Second, cancel)
	defer timer.Stop()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, f.ts.URL+"/preview/"+token+"/__events", nil)
	if err != nil {
		t.Fatal(err)
	}
	res, err := f.ts.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK || !strings.HasPrefix(res.Header.Get("Content-Type"), "text/event-stream") {
		t.Fatalf("status=%d type=%q", res.StatusCode, res.Header.Get("Content-Type"))
	}
	reader := bufio.NewReader(res.Body)
	readEvent := func() string {
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				return ""
			}
			if strings.HasPrefix(line, "event: ") {
				return strings.TrimSpace(strings.TrimPrefix(line, "event: "))
			}
		}
	}
	if ev := readEvent(); ev != "hello" {
		t.Fatalf("first event=%q (stream dead before a change)", ev)
	}

	// Change a served asset: a different size moves the signature even if
	// the clock is coarse.
	if err := os.WriteFile(filepath.Join(f.root, "site", "app.js"), []byte("// changed by the test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for {
		ev := readEvent()
		if ev == "" {
			t.Fatal("stream closed before the change event")
		}
		if ev == "change" {
			return
		}
	}
}

func TestPreviewEventsRefusals(t *testing.T) {
	f := newPreviewFixture(t, time.Hour)
	unknown := "/preview/" + strings.Repeat("a", 43) + "/__events"
	if res, _ := f.get(t, unknown); res.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown token status=%d", res.StatusCode)
	}
	url := f.mintURL(t, "site/index.html")
	token := strings.Split(strings.TrimPrefix(url, "/preview/"), "/")[0]
	req, _ := http.NewRequest(http.MethodPost, f.ts.URL+"/preview/"+token+"/__events", nil)
	res, err := f.ts.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("post status=%d", res.StatusCode)
	}
}

func TestPreviewOverlayPut(t *testing.T) {
	f := newPreviewFixture(t, time.Hour)
	url := f.mintURL(t, "site/index.html")
	token := strings.Split(strings.TrimPrefix(url, "/preview/"), "/")[0]

	put := func(t *testing.T, path, body string, raw []byte) *http.Response {
		t.Helper()
		var reader io.Reader
		if raw != nil {
			reader = bytes.NewReader(raw)
		} else {
			reader = strings.NewReader(body)
		}
		req, err := http.NewRequest(http.MethodPut, f.ts.URL+path, reader)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "text/plain; charset=utf-8")
		res, err := f.ts.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = res.Body.Close() }()
		return res
	}

	if res := put(t, url, "<!doctype html><h1>unsaved do editor</h1>", nil); res.StatusCode != http.StatusNoContent {
		t.Fatalf("put status=%d", res.StatusCode)
	}
	res, body := f.get(t, url)
	if res.StatusCode != http.StatusOK || !strings.Contains(body, "unsaved do editor") {
		t.Fatalf("overlay not served: status=%d body=%q", res.StatusCode, body)
	}
	if strings.Contains(body, "hello preview") {
		t.Fatal("the disk document leaked through the overlay")
	}
	if !strings.HasPrefix(res.Header.Get("Content-Security-Policy"), "sandbox allow-scripts") {
		t.Fatalf("overlay lost the sandbox: %q", res.Header.Get("Content-Security-Policy"))
	}
	// Assets still come from disk.
	if _, js := f.get(t, "/preview/"+token+"/site/app.js"); !strings.Contains(js, "preview") {
		t.Fatalf("asset changed under the overlay: %q", js)
	}
	// Only the ticket's own document may be overlaid.
	if res := put(t, "/preview/"+token+"/site/app.js", "nope", nil); res.StatusCode != http.StatusNotFound {
		t.Fatalf("asset overlay status=%d", res.StatusCode)
	}
	// The pane writes UTF-8 text.
	if res := put(t, url, "", []byte{0xff, 0xfe, 0x00}); res.StatusCode != http.StatusBadRequest {
		t.Fatalf("invalid utf8 status=%d", res.StatusCode)
	}
	// The editor is capped at 1 MiB; the route refuses more.
	if res := put(t, url, strings.Repeat("x", maxAgentText+1), nil); res.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized status=%d", res.StatusCode)
	}
	// Unknown token, other methods.
	if res := put(t, "/preview/"+strings.Repeat("a", 43)+"/site/index.html", "x", nil); res.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown token status=%d", res.StatusCode)
	}
	req, _ := http.NewRequest(http.MethodDelete, f.ts.URL+url, nil)
	resDel, err := f.ts.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resDel.Body.Close() }()
	if resDel.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("delete status=%d", resDel.StatusCode)
	}
}
