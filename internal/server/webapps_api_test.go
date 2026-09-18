package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

func newWebappServer(t *testing.T) *httptest.Server {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "picode.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	ts := httptest.NewServer(New("127.0.0.1:0", Deps{Store: st, Tmux: tmux.New(), AgentCmd: "cat"}).Handler)
	t.Cleanup(ts.Close)
	return ts
}

func webappCall(t *testing.T, ts *httptest.Server, method, path, body string) (int, map[string]any, http.Header) {
	t.Helper()
	url := ts.URL + path
	var req *http.Request
	var err error
	if body == "" {
		req, err = http.NewRequest(method, url, nil)
	} else {
		req, err = http.NewRequest(method, url, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	out := map[string]any{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &out)
	}
	return res.StatusCode, out, res.Header
}

func webappOrigin(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	origin := httptest.NewServer(handler)
	t.Cleanup(origin.Close)
	return origin
}

const webappPage = `<!doctype html><html><head><title>Example App</title>` +
	`<link rel="icon" href="/icons/logo.png">` +
	`<link rel="icon" href="https://cdn.other.example/track.png"></head>` +
	`<body>hello</body></html>`

func webappDemoOrigin(t *testing.T) *httptest.Server {
	return webappOrigin(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/icons/logo.png":
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write([]byte("PNGPNG"))
		case "/favicon.ico":
			w.WriteHeader(http.StatusNotFound)
		default:
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(webappPage))
		}
	}))
}

func TestWebappInstallResolvesTitleAndSameOriginIcon(t *testing.T) {
	origin := webappDemoOrigin(t)
	ts := newWebappServer(t)
	code, body, _ := webappCall(t, ts, http.MethodPost, "/api/webapps", fmt.Sprintf(`{"url":%q}`, origin.URL))
	if code != http.StatusOK {
		t.Fatalf("install = %d %v", code, body)
	}
	if body["name"] != "Example App" || body["hasIcon"] != true {
		t.Fatalf("app = %v", body)
	}
	id, _ := body["id"].(string)
	if id == "" || body["createdAt"] == "" || body["url"] == "" {
		t.Fatalf("entity incomplete: %v", body)
	}

	code, _, headers := webappCall(t, ts, http.MethodGet, "/api/webapps/"+id+"/icon", "")
	if code != http.StatusOK || headers.Get("Content-Type") != "image/png" {
		t.Fatalf("icon = %d %v", code, headers)
	}
}

func TestWebappInstallWithoutFaviconStillInstalls(t *testing.T) {
	origin := webappOrigin(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<!doctype html><html><head><title>Iconless</title></head><body></body></html>`))
	}))
	ts := newWebappServer(t)
	code, body, _ := webappCall(t, ts, http.MethodPost, "/api/webapps", fmt.Sprintf(`{"url":%q}`, origin.URL))
	if code != http.StatusOK || body["hasIcon"] != false {
		t.Fatalf("install = %d %v", code, body)
	}
}

func TestWebappInstallKeepsOriginalURLNotRedirect(t *testing.T) {
	origin := webappOrigin(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/login" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(`<!doctype html><title>Sign in</title>`))
			return
		}
		http.Redirect(w, r, "/login", http.StatusFound)
	}))
	ts := newWebappServer(t)
	code, body, _ := webappCall(t, ts, http.MethodPost, "/api/webapps", fmt.Sprintf(`{"url":%q}`, origin.URL))
	if code != http.StatusOK {
		t.Fatalf("install = %d %v", code, body)
	}
	if body["url"] != origin.URL {
		t.Fatalf("installed url = %v, want the address the user typed (%s)", body["url"], origin.URL)
	}
	if body["name"] != "Sign in" {
		t.Fatalf("name should come from the final page, got %v", body["name"])
	}
}

func TestWebappManifestMetadataWins(t *testing.T) {
	png := []byte("PNGPNG")
	origin := webappOrigin(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/manifest.webmanifest":
			w.Header().Set("Content-Type", "application/manifest+json")
			_, _ = w.Write([]byte(`{"name":"Zulip","start_url":"/app/?source=pwa","scope":"/app/","display":"standalone","theme_color":"#1D2B53","icons":[{"src":"icons/app-192.png","sizes":"192x192","type":"image/png"}]}`))
		case "/icons/app-192.png":
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(png)
		default:
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(`<!doctype html><title>Wrong Title</title><link rel="manifest" href="/manifest.webmanifest">`))
		}
	}))
	ts := newWebappServer(t)
	code, body, _ := webappCall(t, ts, http.MethodPost, "/api/webapps/resolve", fmt.Sprintf(`{"url":%q}`, origin.URL))
	if code != http.StatusOK {
		t.Fatalf("resolve = %d %v", code, body)
	}
	if body["name"] != "Zulip" || body["iconAvailable"] != true || body["pwa"] != true {
		t.Fatalf("resolve should use the manifest identity, got %v", body)
	}
	if body["startUrl"] != origin.URL+"/app/?source=pwa" || body["scope"] != origin.URL+"/app/" || body["display"] != "standalone" || body["themeColor"] != "#1d2b53" {
		t.Fatalf("manifest fields = %v", body)
	}
	if body["url"] != origin.URL {
		t.Fatalf("resolve url = %v, want %s", body["url"], origin.URL)
	}
	code, body, _ = webappCall(t, ts, http.MethodPost, "/api/webapps", fmt.Sprintf(`{"url":%q}`, origin.URL))
	if code != http.StatusOK || body["name"] != "Zulip" || body["hasIcon"] != true {
		t.Fatalf("install = %d %v", code, body)
	}
	if body["startUrl"] != origin.URL+"/app/?source=pwa" || body["display"] != "standalone" {
		t.Fatalf("stored manifest fields = %v", body)
	}
}

func TestWebappManifestCrossOriginStartURLRefused(t *testing.T) {
	origin := webappOrigin(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/manifest.webmanifest":
			w.Header().Set("Content-Type", "application/manifest+json")
			_, _ = w.Write([]byte(`{"name":"Hijack","start_url":"https://evil.example/grab","display":"Crazy","theme_color":"not-a-color","icons":[]}`))
		default:
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(`<!doctype html><title>Hijack</title><link rel="manifest" href="/manifest.webmanifest">`))
		}
	}))
	ts := newWebappServer(t)
	code, body, _ := webappCall(t, ts, http.MethodPost, "/api/webapps", fmt.Sprintf(`{"url":%q}`, origin.URL))
	if code != http.StatusOK || body["name"] != "Hijack" {
		t.Fatalf("install = %d %v", code, body)
	}
	if body["startUrl"] != nil || body["display"] != nil || body["themeColor"] != nil {
		t.Fatalf("cross-origin start_url, unknown display and bad color must be dropped, got %v", body)
	}
}

func TestWebappManifestCrossOriginIconRefused(t *testing.T) {
	origin := webappOrigin(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/manifest.webmanifest":
			w.Header().Set("Content-Type", "application/manifest+json")
			_, _ = w.Write([]byte(`{"name":"Evil Icons","icons":[{"src":"https://evil.example/steal.png","sizes":"512x512","type":"image/png"}]}`))
		case "/favicon.ico":
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write([]byte("PNGPNG"))
		default:
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(`<!doctype html><title>Evil Icons</title><link rel="manifest" href="/manifest.webmanifest">`))
		}
	}))
	ts := newWebappServer(t)
	code, body, _ := webappCall(t, ts, http.MethodPost, "/api/webapps", fmt.Sprintf(`{"url":%q}`, origin.URL))
	if code != http.StatusOK || body["name"] != "Evil Icons" {
		t.Fatalf("install = %d %v", code, body)
	}
	if body["hasIcon"] != true {
		t.Fatalf("should fall back to the same-origin favicon, got %v", body)
	}
}

func TestWebappIconServedWithHardHeaders(t *testing.T) {
	origin := webappDemoOrigin(t)
	ts := newWebappServer(t)
	_, body, _ := webappCall(t, ts, http.MethodPost, "/api/webapps", fmt.Sprintf(`{"url":%q}`, origin.URL))
	id, _ := body["id"].(string)
	_, _, headers := webappCall(t, ts, http.MethodGet, "/api/webapps/"+id+"/icon", "")
	if headers.Get("Content-Security-Policy") != "default-src 'none'" || headers.Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("icon headers = %v", headers)
	}
}

func TestWebappOversizedPageStillResolvesFromHead(t *testing.T) {
	pad := strings.Repeat("<!-- filler -->", 40*1024)
	origin := webappOrigin(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, "<!doctype html><html><head><title>Big App</title><link rel=\"icon\" href=\"/f.png\"></head><body>%s</body></html>", pad)
	}))
	ts := newWebappServer(t)
	code, body, _ := webappCall(t, ts, http.MethodPost, "/api/webapps/resolve", fmt.Sprintf(`{"url":%q}`, origin.URL))
	if code != http.StatusOK {
		t.Fatalf("resolve on an oversized page = %d %v", code, body)
	}
	if body["name"] != "Big App" {
		t.Fatalf("name from the head of a truncated body = %v", body)
	}
	code, body, _ = webappCall(t, ts, http.MethodPost, "/api/webapps", fmt.Sprintf(`{"url":%q}`, origin.URL))
	if code != http.StatusOK || body["name"] != "Big App" {
		t.Fatalf("install on an oversized page = %d %v", code, body)
	}
}

func TestWebappRefreshReplacesManifestIdentityKeepsNameAndURL(t *testing.T) {
	png := []byte("PNGNEW")
	manifest := atomic.Value{}
	manifest.Store([]byte(`{"name":"Ignored Name","display":"browser"}`))
	origin := webappOrigin(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/manifest.webmanifest":
			w.Header().Set("Content-Type", "application/manifest+json")
			_, _ = w.Write(manifest.Load().([]byte))
		case "/icons/app.png":
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(png)
		default:
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(`<!doctype html><title>Old Title</title><link rel="manifest" href="/manifest.webmanifest">`))
		}
	}))
	ts := newWebappServer(t)
	code, first, _ := webappCall(t, ts, http.MethodPost, "/api/webapps", fmt.Sprintf(`{"url":%q,"name":"My Zulip"}`, origin.URL))
	if code != http.StatusOK {
		t.Fatalf("install = %d %v", code, first)
	}
	manifest.Store([]byte(`{"name":"Ignored Again","start_url":"/app/","scope":"/app/","display":"standalone","theme_color":"#123456","icons":[{"src":"/icons/app.png","sizes":"192x192","type":"image/png"}]}`))
	code, refreshed, _ := webappCall(t, ts, http.MethodPost, "/api/webapps/"+first["id"].(string)+"/refresh", "")
	if code != http.StatusOK {
		t.Fatalf("refresh = %d %v", code, refreshed)
	}
	if refreshed["name"] != "My Zulip" || refreshed["url"] != first["url"] {
		t.Fatalf("refresh must keep the user's name and url: %v", refreshed)
	}
	if refreshed["display"] != "standalone" || refreshed["startUrl"] != origin.URL+"/app/" || refreshed["themeColor"] != "#123456" || refreshed["hasIcon"] != true {
		t.Fatalf("refreshed identity = %v", refreshed)
	}
}

func TestWebappRefreshSiteDownLeavesTileUntouched(t *testing.T) {
	origin := webappDemoOrigin(t)
	url := origin.URL
	ts := newWebappServer(t)
	code, first, _ := webappCall(t, ts, http.MethodPost, "/api/webapps", fmt.Sprintf(`{"url":%q}`, url))
	if code != http.StatusOK {
		t.Fatalf("install = %d %v", code, first)
	}
	origin.Close()
	code, body, _ := webappCall(t, ts, http.MethodPost, "/api/webapps/"+first["id"].(string)+"/refresh", "")
	if code != http.StatusBadGateway || !strings.Contains(body["error"].(string), "stays as it was") {
		t.Fatalf("refresh while down = %d %v", code, body)
	}
	res, err := http.Get(ts.URL + "/api/webapps")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	defer res.Body.Close()
	var list []map[string]any
	_ = json.NewDecoder(res.Body).Decode(&list)
	if len(list) != 1 || list[0]["name"] != first["name"] || list[0]["hasIcon"] != first["hasIcon"] {
		t.Fatalf("row changed after refused refresh: %v", list)
	}
}

func TestWebappRefreshUnknownIDIs404(t *testing.T) {
	ts := newWebappServer(t)
	code, _, _ := webappCall(t, ts, http.MethodPost, "/api/webapps/nope/refresh", "")
	if code != http.StatusNotFound {
		t.Fatalf("unknown id refresh = %d", code)
	}
}

func TestWebappInstallRefusesUnreachableSite(t *testing.T) {
	ts := newWebappServer(t)
	code, body, _ := webappCall(t, ts, http.MethodPost, "/api/webapps", `{"url":"http://127.0.0.1:1/"}`)
	if code != http.StatusBadGateway {
		t.Fatalf("unreachable install = %d %v", code, body)
	}
	code, list, _ := webappCall(t, ts, http.MethodGet, "/api/webapps", "")
	if code != http.StatusOK || len(list) != 0 {
		t.Fatalf("list after refused install = %d %v", code, list)
	}
}

func TestWebappInstallEnforcesReachabilityItself(t *testing.T) {
	origin := webappOrigin(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	ts := newWebappServer(t)
	code, body, _ := webappCall(t, ts, http.MethodPost, "/api/webapps", fmt.Sprintf(`{"url":%q,"name":"Pretend"}`, origin.URL))
	if code != http.StatusBadGateway {
		t.Fatalf("5xx install = %d %v", code, body)
	}
}

func TestWebappResolveContract(t *testing.T) {
	origin := webappDemoOrigin(t)
	ts := newWebappServer(t)
	code, body, _ := webappCall(t, ts, http.MethodPost, "/api/webapps/resolve", fmt.Sprintf(`{"url":%q}`, origin.URL))
	if code != http.StatusOK {
		t.Fatalf("resolve = %d %v", code, body)
	}
	if body["name"] != "Example App" || body["iconAvailable"] != true || body["url"] == "" {
		t.Fatalf("resolve body = %v", body)
	}
	if _, ok := body["id"]; ok {
		t.Fatalf("resolve must not create a row: %v", body)
	}
}

func TestWebappDuplicateURLConflictsWithExistingID(t *testing.T) {
	origin := webappOrigin(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<!doctype html><title>Dup</title>`))
	}))
	ts := newWebappServer(t)
	url := fmt.Sprintf(`{"url":%q}`, origin.URL)
	code, first, _ := webappCall(t, ts, http.MethodPost, "/api/webapps", url)
	if code != http.StatusOK {
		t.Fatalf("first install = %d %v", code, first)
	}
	code, second, _ := webappCall(t, ts, http.MethodPost, "/api/webapps", url)
	if code != http.StatusConflict {
		t.Fatalf("second install = %d %v", code, second)
	}
	existing, _ := second["existing"].(map[string]any)
	if existing == nil || existing["id"] != first["id"] {
		t.Fatalf("conflict payload = %v", second)
	}
}

func TestWebappSchemeRefused(t *testing.T) {
	ts := newWebappServer(t)
	for _, raw := range []string{`{"url":"ftp://example.com"}`, `{"url":"javascript:alert(1)"}`, `{"url":"file:///etc/passwd"}`} {
		code, body, _ := webappCall(t, ts, http.MethodPost, "/api/webapps", raw)
		if code != http.StatusBadRequest {
			t.Fatalf("%s install = %d %v", raw, code, body)
		}
	}
}

func TestWebappPatchNameOnly(t *testing.T) {
	origin := webappOrigin(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<!doctype html><title>Patch</title>`))
	}))
	ts := newWebappServer(t)
	_, body, _ := webappCall(t, ts, http.MethodPost, "/api/webapps", fmt.Sprintf(`{"url":%q}`, origin.URL))
	id, _ := body["id"].(string)
	code, patched, _ := webappCall(t, ts, http.MethodPatch, "/api/webapps/"+id, `{"name":"Renamed"}`)
	if code != http.StatusOK || patched["name"] != "Renamed" || patched["url"] != body["url"] {
		t.Fatalf("patch = %d %v", code, patched)
	}
}

func TestWebappDeleteAndMissingRow(t *testing.T) {
	origin := webappOrigin(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<!doctype html><title>Gone</title>`))
	}))
	ts := newWebappServer(t)
	_, body, _ := webappCall(t, ts, http.MethodPost, "/api/webapps", fmt.Sprintf(`{"url":%q}`, origin.URL))
	id, _ := body["id"].(string)
	code, _, _ := webappCall(t, ts, http.MethodDelete, "/api/webapps/"+id, "")
	if code != http.StatusNoContent {
		t.Fatalf("delete = %d", code)
	}
	code, _, _ = webappCall(t, ts, http.MethodDelete, "/api/webapps/"+id, "")
	if code != http.StatusNotFound {
		t.Fatalf("second delete = %d", code)
	}
}

func TestWebappIconWithoutStoredIconIs404(t *testing.T) {
	origin := webappOrigin(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<!doctype html><title>Bare</title>`))
	}))
	ts := newWebappServer(t)
	_, body, _ := webappCall(t, ts, http.MethodPost, "/api/webapps", fmt.Sprintf(`{"url":%q}`, origin.URL))
	id, _ := body["id"].(string)
	code, _, _ := webappCall(t, ts, http.MethodGet, "/api/webapps/"+id+"/icon", "")
	if code != http.StatusNotFound {
		t.Fatalf("icon fallback = %d", code)
	}
}
