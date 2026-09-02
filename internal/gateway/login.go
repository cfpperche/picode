package gateway

import (
	"fmt"
	"html"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sort"
	"strings"
	"time"
)

// The gateway's own routes live under /-/ (unused by the SPA and the
// daemon): the login page, the provider round trip, logout, health.
func (s *Server) own(w http.ResponseWriter, r *http.Request, peer string) {
	switch {
	case r.URL.Path == "/-/healthz":
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"ok":true,"providers":%d,"members":%d}`, len(s.providers), len(s.Config.Users))
	case r.URL.Path == "/-/login":
		s.loginPage(w, r)
	case strings.HasPrefix(r.URL.Path, "/-/auth/start/"):
		if !s.authLimit.allow(peer) {
			s.page(w, http.StatusTooManyRequests, "Too many attempts", "Wait ten minutes, then try again.")
			return
		}
		s.authStart(w, r, strings.TrimPrefix(r.URL.Path, "/-/auth/start/"))
	case strings.HasPrefix(r.URL.Path, "/-/auth/callback/"):
		if !s.authLimit.allow(peer) {
			s.page(w, http.StatusTooManyRequests, "Too many attempts", "Wait ten minutes, then try again.")
			return
		}
		s.authCallback(w, r, strings.TrimPrefix(r.URL.Path, "/-/auth/callback/"))
	case r.URL.Path == "/-/auth/logout":
		clearSessionCookie(w)
		http.Redirect(w, r, "/-/login", http.StatusSeeOther)
	case strings.HasPrefix(r.URL.Path, "/-/hook/"):
		s.hook(w, r, peer)
	default:
		s.page(w, http.StatusNotFound, "Not here", "There is nothing at "+html.EscapeString(r.URL.Path)+".")
	}
}

func (s *Server) providerNames() []string {
	out := make([]string, 0, len(s.providers))
	for n := range s.providers {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

func (s *Server) loginPage(w http.ResponseWriter, r *http.Request) {
	if len(s.providers) == 0 {
		s.page(w, http.StatusNotFound, "No login here", "This gateway only admits devices on its tailnet.")
		return
	}
	next := safeNext(r.URL.Query().Get("next"))
	var b strings.Builder
	for _, n := range s.providerNames() {
		label := map[string]string{"google": "Continue with Google", "github": "Continue with GitHub"}[n]
		b.WriteString(`<a class="btn" href="/-/auth/start/` + n + `?next=` + url.QueryEscape(next) + `">` + label + `</a>`)
	}
	s.pageExtra(w, http.StatusOK, "Sign in", "This PiCode is on "+html.EscapeString(s.Config.Hostname)+". Sign in with the account its admin added; your device pairs after that.", b.String())
}

// authStart redirects to the provider with a fresh state (+ PKCE and
// nonce for OIDC). The state is single-use and lives ten minutes.
func (s *Server) authStart(w http.ResponseWriter, r *http.Request, name string) {
	p, ok := s.providers[name]
	if !ok {
		s.page(w, http.StatusNotFound, "Unknown provider", html.EscapeString(name)+" is not configured here.")
		return
	}
	state, err1 := randomToken()
	nonce, err2 := randomToken()
	verifier, challenge, err3 := pkce()
	if err1 != nil || err2 != nil || err3 != nil {
		s.page(w, http.StatusInternalServerError, "Cannot start a login", "No randomness available.")
		return
	}
	s.pendMu.Lock()
	for k, v := range s.pend {
		if time.Now().After(v.until) {
			delete(s.pend, k)
		}
	}
	s.pend[state] = pending{provider: name, verifier: verifier, nonce: nonce, next: safeNext(r.URL.Query().Get("next")), until: time.Now().Add(10 * time.Minute)}
	s.pendMu.Unlock()
	to, err := p.authorizeURL(r.Context(), s.redirectURI(name), state, nonce, challenge)
	if err != nil {
		s.page(w, http.StatusBadGateway, "Provider unavailable", html.EscapeString(err.Error()))
		return
	}
	http.Redirect(w, r, to, http.StatusSeeOther)
}

func (s *Server) redirectURI(name string) string {
	return strings.TrimRight(s.Config.PublicURL, "/") + "/-/auth/callback/" + name
}

// authCallback finishes the round trip: state must be ours and unused,
// the code must exchange, and the login must be on the members list —
// only then is the gateway session issued.
func (s *Server) authCallback(w http.ResponseWriter, r *http.Request, name string) {
	p, ok := s.providers[name]
	if !ok {
		s.page(w, http.StatusNotFound, "Unknown provider", html.EscapeString(name)+" is not configured here.")
		return
	}
	q := r.URL.Query()
	if e := q.Get("error"); e != "" {
		s.page(w, http.StatusBadRequest, "Sign-in cancelled", html.EscapeString(e)+". <a href=\"/-/login\">Try again</a>.")
		return
	}
	state, code := q.Get("state"), q.Get("code")
	s.pendMu.Lock()
	pend, ok := s.pend[state]
	delete(s.pend, state)
	s.pendMu.Unlock()
	if !ok || pend.provider != name || time.Now().After(pend.until) || code == "" {
		s.page(w, http.StatusBadRequest, "This sign-in link is stale", "Start again from <a href=\"/-/login\">the sign-in page</a>.")
		return
	}
	login, err := p.exchange(r.Context(), code, s.redirectURI(name), pend.verifier, pend.nonce)
	if err != nil {
		s.Log.Printf("gateway: %s login failed: %v", name, err)
		s.page(w, http.StatusBadGateway, "Sign-in failed", "The provider did not confirm who you are. <a href=\"/-/login\">Try again</a>.")
		return
	}
	if _, ok := s.Config.UserFor(login); !ok {
		s.page(w, http.StatusForbidden, "Not on this server", html.EscapeString(login)+" has no PiCode here. Ask the admin: <code>picode users add "+html.EscapeString(login)+" &lt;user&gt;</code>.")
		return
	}
	setSessionCookie(w, s.signer.issue(login, time.Now().Add(SessionTTL)), s.Secure)
	http.Redirect(w, r, pend.next, http.StatusSeeOther)
}

// safeNext keeps redirects on this origin.
func safeNext(next string) string {
	if next == "" || !strings.HasPrefix(next, "/") || strings.HasPrefix(next, "//") || strings.HasPrefix(next, "/-/") {
		return "/"
	}
	return next
}

// pageExtra is page with a block of actions under the text.
func (s *Server) pageExtra(w http.ResponseWriter, code int, heading, body, extra string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Security-Policy", PageCSP)
	w.WriteHeader(code)
	_, _ = fmt.Fprintf(w, pageTemplate, html.EscapeString(heading), html.EscapeString(heading), body+`<div class="actions">`+extra+`</div>`)
}

// hook is a member's automation webhook from outside (ADR-0045 through
// ADR-0051/52): POST /-/hook/<linux user>/<automation id>. No identity —
// the automation's own secret is the credential, checked by the daemon —
// so the gateway only routes: the user must be a member, the method must
// be POST, and a peer gets 60 tries a minute. Authorization is passed
// through untouched (it carries the secret), unlike everywhere else.
func (s *Server) hook(w http.ResponseWriter, r *http.Request, peer string) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		s.page(w, http.StatusMethodNotAllowed, "POST only", "Automation webhooks are POST requests.")
		return
	}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/-/hook/"), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		s.page(w, http.StatusNotFound, "Not here", "A webhook looks like /-/hook/&lt;user&gt;/&lt;automation&gt;.")
		return
	}
	if !s.hookLimit.allow(peer) {
		w.Header().Set("Retry-After", "60")
		s.page(w, http.StatusTooManyRequests, "Too many calls", "Sixty a minute per caller.")
		return
	}
	linux, id := parts[0], parts[1]
	if !s.Config.IsMember(linux) {
		// The same answer as a wrong secret from the daemon: a probe learns nothing.
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = fmt.Fprint(w, `{"error":"not found"}`)
		return
	}
	be, err := s.Resolve(linux)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = fmt.Fprint(w, `{"error":"PiCode is not running for this member"}`)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 80<<10) // the daemon caps at 64 KB; a little headroom for headers/encoding
	proxy := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(be.URL)
			pr.Out.URL.Path = "/api/automations/" + url.PathEscape(id) + "/fire"
			pr.Out.URL.RawPath = ""
			pr.Out.Host = s.Config.Hostname
			for _, h := range []string{"Cookie", "Forwarded", "X-Forwarded-For", "X-Forwarded-Host", "X-Forwarded-Proto", "X-Real-IP"} {
				pr.Out.Header.Del(h)
			}
			pr.Out.Header.Set("X-Forwarded-For", peer)
			pr.Out.Header.Set("X-Forwarded-Proto", "https")
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadGateway)
			_, _ = fmt.Fprint(w, `{"error":"PiCode did not answer"}`)
		},
	}
	proxy.ServeHTTP(w, r)
}
