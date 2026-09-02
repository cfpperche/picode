package gateway

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// fakeGoogle is an OIDC provider: discovery, authorize (auto-consents
// and bounces back with a code), token (signs an ID token), JWKS.
type fakeGoogle struct {
	key   *rsa.PrivateKey
	srv   *httptest.Server
	email string
	// twist lets a test break one claim.
	twist func(claims map[string]any)
	codes map[string]string // code → nonce
}

func newFakeGoogle(t *testing.T) *fakeGoogle {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	g := &fakeGoogle{key: key, email: "Alice@Example.com", codes: map[string]string{}}
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"issuer":%q,"authorization_endpoint":%q,"token_endpoint":%q,"jwks_uri":%q}`, g.srv.URL, g.srv.URL+"/auth", g.srv.URL+"/token", g.srv.URL+"/jwks")
	})
	mux.HandleFunc("/auth", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		code := "code-" + q.Get("state")
		g.codes[code] = q.Get("nonce")
		http.Redirect(w, r, q.Get("redirect_uri")+"?state="+url.QueryEscape(q.Get("state"))+"&code="+code, http.StatusFound)
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		nonce, ok := g.codes[r.Form.Get("code")]
		if !ok || r.Form.Get("client_id") != "cid" || r.Form.Get("client_secret") != "csec" || r.Form.Get("code_verifier") == "" {
			http.Error(w, `{"error":"invalid_grant"}`, 400)
			return
		}
		claims := map[string]any{"iss": g.srv.URL, "aud": "cid", "exp": time.Now().Add(time.Hour).Unix(), "nonce": nonce, "email": g.email, "email_verified": true}
		if g.twist != nil {
			g.twist(claims)
		}
		fmt.Fprintf(w, `{"access_token":"x","id_token":%q}`, g.sign(claims))
	})
	mux.HandleFunc("/jwks", func(w http.ResponseWriter, r *http.Request) {
		n := base64.RawURLEncoding.EncodeToString(key.N.Bytes())
		e := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes())
		fmt.Fprintf(w, `{"keys":[{"kty":"RSA","kid":"k1","n":%q,"e":%q}]}`, n, e)
	})
	g.srv = httptest.NewServer(mux)
	t.Cleanup(g.srv.Close)
	return g
}

func (g *fakeGoogle) sign(claims map[string]any) string {
	h, _ := json.Marshal(map[string]string{"alg": "RS256", "kid": "k1"})
	c, _ := json.Marshal(claims)
	signing := base64.RawURLEncoding.EncodeToString(h) + "." + base64.RawURLEncoding.EncodeToString(c)
	sum := sha256.Sum256([]byte(signing))
	sig, _ := rsa.SignPKCS1v15(rand.Reader, g.key, crypto.SHA256, sum[:])
	return signing + "." + base64.RawURLEncoding.EncodeToString(sig)
}

func newPublicGateway(t *testing.T, g *fakeGoogle, backend *httptest.Server) (*Server, *httptest.Server) {
	t.Helper()
	cfg := Config{Hostname: "box.tail1234.ts.net", Users: map[string]string{"alice@example.com": "alice", "octocat@github": "cat"},
		TrustedProxies: []string{"127.0.0.0/8"}, OIDC: map[string]ProviderConfig{"google": {Issuer: g.srv.URL}}}
	sec := Secrets{CookieKey: strings.Repeat("ab", 32), Providers: map[string]ProviderSecret{"google": {ClientID: "cid", ClientSecret: "csec"}}}
	gw := httptest.NewUnstartedServer(nil)
	cfg.PublicURL = "https://picode.example.com"
	s, err := New(cfg, sec)
	if err != nil {
		t.Fatal(err)
	}
	s.Secure = false
	s.Resolve = func(linux string) (Backend, error) {
		u, _ := url.Parse(backend.URL)
		return Backend{User: linux, URL: u, Token: "tok-" + linux}, nil
	}
	gw.Config.Handler = s
	gw.Start()
	t.Cleanup(gw.Close)
	return s, gw
}

// follow walks redirects by hand so the callback (registered as the
// public URL) can be pointed back at the test server.
func TestPublicLoginRoundTrip(t *testing.T) {
	_, be := newBackend(t, "alice")
	g := newFakeGoogle(t)
	s, gw := newPublicGateway(t, g, be)
	pub := "203.0.113.7" // the client behind the proxy

	// Off the tailnet, no cookie: navigation → login page; API → 401 with the login URL.
	res, body := get(t, gw, "/", map[string]string{"Sec-Fetch-Mode": "navigate", "X-Forwarded-For": pub})
	if res.StatusCode != 303 || !strings.HasPrefix(res.Header.Get("Location"), "/-/login?next=") {
		t.Fatalf("nav: %d %q", res.StatusCode, res.Header.Get("Location"))
	}
	if res, body = get(t, gw, "/api/health", map[string]string{"X-Forwarded-For": pub}); res.StatusCode != 401 || !strings.Contains(body, `"login":"/-/login"`) {
		t.Fatalf("api: %d %s", res.StatusCode, body)
	}
	if res, body = get(t, gw, "/-/login", map[string]string{"X-Forwarded-For": pub}); res.StatusCode != 200 || !strings.Contains(body, "Continue with Google") {
		t.Fatalf("login page: %d", res.StatusCode)
	}

	// Start → provider → callback → cookie → app.
	res, _ = get(t, gw, "/-/auth/start/google?next=/%23/inbox", map[string]string{"X-Forwarded-For": pub})
	loc := res.Header.Get("Location")
	if res.StatusCode != 303 || !strings.HasPrefix(loc, g.srv.URL+"/auth?") || !strings.Contains(loc, "code_challenge=") {
		t.Fatalf("start: %d %q", res.StatusCode, loc)
	}
	pres, err := (&http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}).Get(loc)
	if err != nil {
		t.Fatal(err)
	}
	back := pres.Header.Get("Location") // https://picode.example.com/-/auth/callback/google?state=..&code=..
	cb, _ := url.Parse(back)
	res, _ = get(t, gw, cb.RequestURI(), map[string]string{"X-Forwarded-For": pub})
	if res.StatusCode != 303 || res.Header.Get("Location") != "/#/inbox" {
		t.Fatalf("callback: %d %q", res.StatusCode, res.Header.Get("Location"))
	}
	var cookie string
	for _, c := range res.Cookies() {
		if c.Name == SessionCookie {
			cookie = c.Value
		}
	}
	if cookie == "" {
		t.Fatal("no gateway cookie")
	}
	// The same state cannot be replayed.
	if res, _ = get(t, gw, cb.RequestURI(), map[string]string{"X-Forwarded-For": pub}); res.StatusCode != 400 {
		t.Fatalf("replay = %d", res.StatusCode)
	}
	// With the cookie: identity is alice, request proxied (first visit auto-pairs).
	res, _ = get(t, gw, "/", map[string]string{"Sec-Fetch-Mode": "navigate", "X-Forwarded-For": pub, "Cookie": SessionCookie + "=" + cookie})
	if res.StatusCode != 303 || res.Header.Get("Location") != "/pair?code=code-alice" {
		t.Fatalf("after login: %d %q", res.StatusCode, res.Header.Get("Location"))
	}
	res, body = get(t, gw, "/api/x", map[string]string{"X-Forwarded-For": pub, "Cookie": SessionCookie + "=" + cookie + "; picode_session=z"})
	if res.StatusCode != 200 || !strings.Contains(body, "hello from alice") || !strings.Contains(body, "xff="+pub) {
		t.Fatalf("proxied: %d %s", res.StatusCode, body)
	}
	// A forged cookie is nobody.
	if res, _ = get(t, gw, "/api/x", map[string]string{"X-Forwarded-For": pub, "Cookie": SessionCookie + "=" + cookie + "x"}); res.StatusCode != 401 {
		t.Fatalf("forged = %d", res.StatusCode)
	}
	// X-Forwarded-For from an untrusted address is a claim, not a peer.
	s.trusted = nil
	if res, body = get(t, gw, "/api/x", map[string]string{"X-Forwarded-For": "100.64.0.9", "Cookie": SessionCookie + "=" + cookie}); res.StatusCode != 200 || !strings.Contains(body, "xff=127.0.0.1") {
		t.Fatalf("untrusted proxy: %d %s", res.StatusCode, body)
	}
	// Logout clears.
	if res, _ = get(t, gw, "/-/auth/logout", nil); res.StatusCode != 303 || !strings.Contains(res.Header.Get("Set-Cookie"), "Max-Age=0") {
		t.Fatalf("logout: %d %q", res.StatusCode, res.Header.Get("Set-Cookie"))
	}
}

func TestIDTokenClaimsAreChecked(t *testing.T) {
	_, be := newBackend(t, "alice")
	g := newFakeGoogle(t)
	_, gw := newPublicGateway(t, g, be)
	n := 0
	try := func(name string, twist func(map[string]any)) int {
		n++
		pub := fmt.Sprintf("203.0.113.%d", 10+n) // a peer each: the auth limiter is per address
		g.twist = twist
		res, _ := get(t, gw, "/-/auth/start/google", map[string]string{"X-Forwarded-For": pub})
		if res.StatusCode != 303 {
			t.Fatalf("%s: start = %d", name, res.StatusCode)
		}
		pres, err := (&http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}).Get(res.Header.Get("Location"))
		if err != nil {
			t.Fatal(err)
		}
		cb, _ := url.Parse(pres.Header.Get("Location"))
		res, _ = get(t, gw, cb.RequestURI(), map[string]string{"X-Forwarded-For": pub})
		return res.StatusCode
	}
	if c := try("ok", nil); c != 303 {
		t.Fatalf("clean login = %d", c)
	}
	for name, twist := range map[string]func(map[string]any){
		"aud":      func(c map[string]any) { c["aud"] = "other" },
		"exp":      func(c map[string]any) { c["exp"] = time.Now().Add(-time.Minute).Unix() },
		"nonce":    func(c map[string]any) { c["nonce"] = "wrong" },
		"iss":      func(c map[string]any) { c["iss"] = "https://evil.example" },
		"verified": func(c map[string]any) { c["email_verified"] = false },
		"unknown":  func(c map[string]any) { c["email"] = "mallory@example.com" },
	} {
		c := try(name, twist)
		if name == "unknown" {
			if c != 403 {
				t.Errorf("%s: %d, want 403", name, c)
			}
			continue
		}
		if c != 502 {
			t.Errorf("%s: %d, want 502", name, c)
		}
	}
}

func TestSessionSignerAndLimiter(t *testing.T) {
	s, err := newSigner(strings.Repeat("cd", 32))
	if err != nil {
		t.Fatal(err)
	}
	v := s.issue("alice@example.com", time.Now().Add(time.Hour))
	if login, ok := s.verify(v, time.Now()); !ok || login != "alice@example.com" {
		t.Fatal("round trip")
	}
	if _, ok := s.verify(v+"x", time.Now()); ok {
		t.Fatal("tampered accepted")
	}
	if _, ok := s.verify(v, time.Now().Add(2*time.Hour)); ok {
		t.Fatal("expired accepted")
	}
	if _, err := newSigner("short"); err == nil {
		t.Fatal("short key accepted")
	}

	l := newLimiter(2, time.Minute, 10*time.Minute)
	now := time.Now()
	l.now = func() time.Time { return now }
	if !l.allow("a") || !l.allow("a") || l.allow("a") {
		t.Fatal("limit")
	}
	now = now.Add(5 * time.Minute)
	if l.allow("a") {
		t.Fatal("locked out")
	}
	now = now.Add(6 * time.Minute)
	if !l.allow("a") {
		t.Fatal("lockout over")
	}
}

func TestGitHubLoginSpelling(t *testing.T) {
	gh := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			fmt.Fprint(w, `{"access_token":"at"}`)
		case "/user":
			if r.Header.Get("Authorization") != "Bearer at" {
				http.Error(w, "no", 401)
				return
			}
			fmt.Fprint(w, `{"login":"OctoCat"}`)
		}
	}))
	t.Cleanup(gh.Close)
	p, err := newProvider("github", ProviderConfig{AuthURL: gh.URL + "/auth", TokenURL: gh.URL + "/token", UserURL: gh.URL + "/user"}, ProviderSecret{ClientID: "i", ClientSecret: "s"})
	if err != nil {
		t.Fatal(err)
	}
	login, err := p.exchange(t.Context(), "code", "https://x/cb", "", "")
	if err != nil || login != "octocat@github" {
		t.Fatalf("%q %v", login, err)
	}
}
