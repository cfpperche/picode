package auth

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/store"
)

func newService(t *testing.T, mode string) (*Service, *store.Store) {
	t.Helper()
	dir := t.TempDir()
	st, err := store.Open(filepath.Join(dir, "picode.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if mode != "" {
		_ = st.SetSetting(ModeSettingKey, mode)
	}
	s, err := New(Config{Store: st, DataDir: dir, Insecure: true, Hostname: "box", PublicURL: func() string { return "https://box.tail.ts.net:8445" }})
	if err != nil {
		t.Fatal(err)
	}
	return s, st
}

func echo() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if p := From(r); p != nil {
			w.Header().Set("X-Principal", p.Kind)
		}
		w.WriteHeader(200)
	})
}

type call struct {
	method, path, remote, origin, host, bearer, cookie, fetchSite string
	upgrade, proxied                                              bool
	ua                                                            string
}

func do(s *Service, c call) *httptest.ResponseRecorder {
	req := httptest.NewRequest(c.method, "http://"+firstNonEmpty(c.host, "localhost:8445")+c.path, nil)
	req.RemoteAddr = firstNonEmpty(c.remote, "10.0.0.9:5555")
	if c.host != "" {
		req.Host = c.host
	}
	if c.origin != "" {
		req.Header.Set("Origin", c.origin)
	}
	if c.fetchSite != "" {
		req.Header.Set("Sec-Fetch-Site", c.fetchSite)
	}
	if c.bearer != "" {
		req.Header.Set("Authorization", "Bearer "+c.bearer)
	}
	if c.cookie != "" {
		req.AddCookie(&http.Cookie{Name: CookieName, Value: c.cookie})
	}
	if c.ua != "" {
		req.Header.Set("User-Agent", c.ua)
	}
	if c.proxied {
		req.Header.Set("X-Forwarded-For", "203.0.113.5")
	}
	if c.upgrade {
		req.Header.Set("Upgrade", "websocket")
		req.Header.Set("Connection", "Upgrade")
	}
	rec := httptest.NewRecorder()
	s.Wrap(echo()).ServeHTTP(rec, req)
	return rec
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func TestDecisionTable(t *testing.T) {
	s, st := newService(t, ModeRemote)
	tok := strings.TrimSpace(readFile(t, s.TokenPath()))
	sess, secret, _ := st.CreateSession(store.SessionBrowser, "d1", "phone", "10.0.0.9", BrowserTTL)
	revoked, revokedSecret, _ := st.CreateSession(store.SessionBrowser, "d2", "old", "", BrowserTTL)
	_ = st.RevokeSession(revoked.ID)
	_ = sess

	cases := []struct {
		name string
		c    call
		code int
		who  string
	}{
		{"health is free", call{method: "GET", path: "/api/health"}, 200, ""},
		{"ui is free", call{method: "GET", path: "/assets/app.js"}, 200, ""},
		{"pair is free", call{method: "GET", path: "/pair?code=x"}, 200, ""},
		{"fire is free", call{method: "POST", path: "/api/automations/a1/fire", origin: "http://localhost:8445"}, 200, ""},
		{"remote without session", call{method: "GET", path: "/api/workspaces"}, 401, ""},
		{"loopback browser auto-pairs", call{method: "GET", path: "/api/workspaces", remote: "127.0.0.1:1", ua: "Mozilla/5.0"}, 200, "browser"},
		{"loopback script passes without a row", call{method: "GET", path: "/api/workspaces", remote: "127.0.0.1:1", ua: "curl/8"}, 200, "loopback"},
		{"loopback behind a proxy is not loopback", call{method: "GET", path: "/api/workspaces", remote: "127.0.0.1:1", proxied: true}, 401, ""},
		{"cookie session", call{method: "GET", path: "/api/workspaces", cookie: secret}, 200, "browser"},
		{"revoked cookie", call{method: "GET", path: "/api/workspaces", cookie: revokedSecret}, 401, ""},
		{"install token", call{method: "GET", path: "/api/workspaces", bearer: tok}, 200, "install"},
		{"wrong token", call{method: "GET", path: "/api/workspaces", bearer: "nope"}, 401, ""},
		{"unknown host", call{method: "GET", path: "/api/workspaces", host: "evil.example:8445", cookie: secret}, 403, ""},
		{"public url host", call{method: "GET", path: "/api/workspaces", host: "box.tail.ts.net:8445", cookie: secret}, 200, "browser"},
		{"foreign origin on POST", call{method: "POST", path: "/api/inbox", origin: "https://evil.example", cookie: secret}, 403, ""},
		{"same origin on POST", call{method: "POST", path: "/api/inbox", origin: "http://localhost:8445", cookie: secret}, 200, "browser"},
		{"cross-site fetch metadata", call{method: "POST", path: "/api/inbox", fetchSite: "cross-site", cookie: secret}, 403, ""},
		{"ws with foreign origin", call{method: "GET", path: "/ws/agent", origin: "https://evil.example", cookie: secret, upgrade: true}, 403, ""},
		{"ws same origin", call{method: "GET", path: "/ws/agent", origin: "http://localhost:8445", cookie: secret, upgrade: true}, 200, "browser"},
		{"events with foreign origin", call{method: "GET", path: "/api/events", origin: "https://evil.example", cookie: secret}, 403, ""},
		{"curl POST without origin", call{method: "POST", path: "/api/inbox", bearer: tok}, 200, "install"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := do(s, c.c)
			if rec.Code != c.code || rec.Header().Get("X-Principal") != c.who {
				t.Fatalf("got %d %q, want %d %q (body %s)", rec.Code, rec.Header().Get("X-Principal"), c.code, c.who, rec.Body.String())
			}
		})
	}

	// The auto-paired loopback session is a real cookie.
	rec := do(s, call{method: "GET", path: "/api/workspaces", remote: "127.0.0.1:1", ua: "Mozilla/5.0"})
	if !strings.Contains(rec.Header().Get("Set-Cookie"), CookieName+"=") {
		t.Fatalf("loopback did not receive a cookie: %q", rec.Header().Get("Set-Cookie"))
	}
}

func TestModes(t *testing.T) {
	off, _ := newService(t, ModeOff)
	if rec := do(off, call{method: "GET", path: "/api/workspaces"}); rec.Code != 200 || rec.Header().Get("X-Principal") != "" {
		t.Fatalf("mode off = %d %q", rec.Code, rec.Header().Get("X-Principal"))
	}
	if rec := do(off, call{method: "POST", path: "/api/inbox", origin: "https://evil.example"}); rec.Code != 403 {
		t.Fatalf("mode off must still refuse foreign origins: %d", rec.Code)
	}
	all, _ := newService(t, ModeAll)
	if rec := do(all, call{method: "GET", path: "/api/workspaces", remote: "127.0.0.1:1", ua: "Mozilla/5.0"}); rec.Code != 401 {
		t.Fatalf("mode all loopback = %d", rec.Code)
	}
	unset, _ := newService(t, "")
	if unset.Mode() != ModeRemote {
		t.Fatalf("default mode = %q", unset.Mode())
	}
	if err := unset.SetMode("weird"); err == nil {
		t.Fatal("bad mode accepted")
	}
}

func TestPairingFlowAndLockout(t *testing.T) {
	s, st := newService(t, ModeRemote)
	code, _, _ := st.CreatePairing("sess", PairingTTL)
	req := httptest.NewRequest("GET", "http://localhost:8445/pair?code="+code, nil)
	req.RemoteAddr = "10.0.0.9:1"
	req.Header.Set("User-Agent", "Mozilla/5.0 (iPhone)")
	rec := httptest.NewRecorder()
	sess, err := s.Pair(rec, req, code, "dev-9")
	if err != nil || sess.DeviceID != "dev-9" || !strings.Contains(rec.Header().Get("Set-Cookie"), CookieName+"=") {
		t.Fatalf("pair: %+v %v %q", sess, err, rec.Header().Get("Set-Cookie"))
	}
	if _, err := s.Pair(httptest.NewRecorder(), req, code, ""); err != store.ErrPairingUsed {
		t.Fatalf("reuse = %v", err)
	}
	for i := 0; i < 4; i++ {
		if _, err := s.Pair(httptest.NewRecorder(), req, "bad", ""); !errorsIs(err, store.ErrPairingInvalid) {
			t.Fatalf("attempt %d = %v", i, err)
		}
	}
	if _, err := s.Pair(httptest.NewRecorder(), req, "bad", ""); !IsTooMany(err) {
		t.Fatalf("6th attempt = %v, want lockout", err)
	}
	fresh, _, _ := st.CreatePairing("sess", PairingTTL)
	if _, err := s.Pair(httptest.NewRecorder(), req, fresh, ""); !IsTooMany(err) {
		t.Fatal("lockout must hold even for a valid code")
	}
	if !strings.Contains(s.PairURL(req, "abc"), "https://box.tail.ts.net:8445/pair?code=abc") {
		t.Fatalf("pair url = %q", s.PairURL(req, "abc"))
	}
}

// The install token is a real token session: it lists, a restart keeps
// it, and rotation revokes the old row instead of just rewriting a file.
func TestInstallTokenIsASession(t *testing.T) {
	s, st := newService(t, ModeRemote)
	tok := strings.TrimSpace(readFile(t, s.TokenPath()))
	sess, err := st.LookupSession(tok)
	if err != nil || sess.Kind != store.SessionToken || sess.Label != InstallTokenLabel {
		t.Fatalf("token row: %+v %v", sess, err)
	}
	again, err := New(s.cfg)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(readFile(t, again.TokenPath())); got != tok {
		t.Fatal("a restart minted a new token")
	}
	if _, err := s.RotateToken(); err != nil {
		t.Fatal(err)
	}
	if _, err := st.LookupSession(tok); err == nil {
		t.Fatal("old token row still live after rotation")
	}
	list, _ := st.ListSessions()
	n := 0
	for _, x := range list {
		if x.Label == InstallTokenLabel {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("install token rows = %d, want 1", n)
	}
}

func TestTokenRotation(t *testing.T) {
	s, _ := newService(t, ModeRemote)
	old := strings.TrimSpace(readFile(t, s.TokenPath()))
	if rec := do(s, call{method: "GET", path: "/api/workspaces", bearer: old}); rec.Code != 200 {
		t.Fatal("old token rejected before rotation")
	}
	fresh, err := s.RotateToken()
	if err != nil || fresh == old {
		t.Fatal(err)
	}
	if rec := do(s, call{method: "GET", path: "/api/workspaces", bearer: old}); rec.Code != 401 {
		t.Fatal("old token still accepted")
	}
	if rec := do(s, call{method: "GET", path: "/api/workspaces", bearer: fresh}); rec.Code != 200 {
		t.Fatal("new token rejected")
	}
}

func TestModeEnvFallback(t *testing.T) {
	t.Setenv("PICODE_AUTH_MODE", "all")
	s, st := newService(t, "")
	if s.Mode() != ModeAll {
		t.Fatal("env should set the mode when the setting is absent")
	}
	_ = st.SetSetting(ModeSettingKey, ModeOff)
	if s.Mode() != ModeOff {
		t.Fatal("the setting wins over the env")
	}
	t.Setenv("PICODE_AUTH_MODE", "bogus")
	_ = st.SetSetting(ModeSettingKey, "")
	if s.Mode() != ModeRemote {
		t.Fatal("unknown values fall back to remote")
	}
}
