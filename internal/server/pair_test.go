package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/auth"
	"github.com/cfpperche/picode/internal/feed"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

func newAuthServer(t *testing.T) (*httptest.Server, *store.Store) {
	t.Helper()
	dir := t.TempDir()
	st, err := store.Open(filepath.Join(dir, "picode.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	f := &feed.Feed{Store: st}
	st.OnEvent = f.Publish
	gate, err := auth.New(auth.Config{Store: st, DataDir: dir, Insecure: true, Hostname: "box"})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(New("127.0.0.1:0", Deps{Store: st, Tmux: tmux.New(), AgentCmd: "cat", Feed: f, Auth: gate}).Handler)
	t.Cleanup(ts.Close)
	return ts, st
}

func noRedirect() *http.Client {
	return &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
}

// A prefetch (GET) must never spend a pairing code; the tap (POST) does.
func TestPairGetDoesNotConsume(t *testing.T) {
	ts, st := newAuthServer(t)
	code, _, _ := st.CreatePairing("", auth.PairingTTL)
	c := noRedirect()

	res, err := c.Get(ts.URL + "/pair?code=" + code)
	if err != nil || res.StatusCode != 200 {
		t.Fatalf("GET /pair = %v %v", res, err)
	}
	body := readBody(res)
	if !strings.Contains(body, "Pair this device") || !strings.Contains(body, `name="code"`) {
		t.Fatalf("page = %s", body)
	}
	// Twice: still fine, nothing consumed.
	res, _ = c.Get(ts.URL + "/pair?code=" + code)
	if res.StatusCode != 200 {
		t.Fatalf("second GET = %d", res.StatusCode)
	}

	req, _ := http.NewRequest("POST", ts.URL+"/pair", strings.NewReader(url.Values{"code": {code}}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", ts.URL)
	res, err = c.Do(req)
	if err != nil || res.StatusCode != http.StatusSeeOther || !strings.Contains(res.Header.Get("Set-Cookie"), auth.CookieName+"=") {
		t.Fatalf("POST /pair = %v %v cookie=%q", res, err, res.Header.Get("Set-Cookie"))
	}
	cookie := res.Cookies()[0]

	// The code is spent now; the message points at Devices.
	req, _ = http.NewRequest("POST", ts.URL+"/pair", strings.NewReader(url.Values{"code": {code}}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", ts.URL)
	res, _ = c.Do(req)
	if res.StatusCode != http.StatusGone || !strings.Contains(readBody(res), "Devices → Pair a device") {
		t.Fatalf("reuse = %d", res.StatusCode)
	}

	// A paired browser opening any pairing link goes straight in.
	req, _ = http.NewRequest("GET", ts.URL+"/pair?code=whatever", nil)
	req.AddCookie(cookie)
	res, _ = c.Do(req)
	if res.StatusCode != http.StatusSeeOther || res.Header.Get("Location") != "/" {
		t.Fatalf("paired GET = %d %q", res.StatusCode, res.Header.Get("Location"))
	}
}

func readBody(res *http.Response) string {
	b, _ := io.ReadAll(res.Body)
	res.Body.Close()
	return string(b)
}
