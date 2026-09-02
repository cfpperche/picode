package gateway

import (
	"context"
	"errors"
	"fmt"
	"html"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Server is the gateway handler. One handler serves both front doors:
// a peer on the tailnet is identified by whois; anyone else (through the
// plain listener behind a TLS proxy) by the gateway's signed session
// cookie, or is sent to the login page (ADR-0052).
type Server struct {
	Config   Config
	Identity *Identity
	// Resolve finds a person's daemon; tests point it at httptest servers.
	Resolve func(linux string) (Backend, error)
	// Fake is a test/scratch identity map (peer ip → login), only honoured
	// when the listener is loopback; see cmd/picode/gateway.go.
	Fake map[string]string
	Log  *log.Logger
	// Secure says the browser reached us over https (true unless a plain
	// scratch listener with no proxy): it sets the cookie's Secure flag.
	Secure bool

	signer    signer
	providers map[string]*provider
	trusted   []*net.IPNet
	pairLimit *limiter
	authLimit *limiter
	hookLimit *limiter

	pairMu   sync.Mutex
	pairLast map[string]time.Time
	pendMu   sync.Mutex
	pend     map[string]pending
}

// New wires a server with the real identity and backend resolution. sec
// may be empty when no provider is configured (tailnet-only box).
func New(cfg Config, sec Secrets) (*Server, error) {
	s := &Server{Config: cfg, Identity: NewIdentity(60 * time.Second), Resolve: Resolve, Log: log.Default(), Secure: true,
		providers: map[string]*provider{}, pairLast: map[string]time.Time{}, pend: map[string]pending{},
		pairLimit: newLimiter(5, time.Minute, 10*time.Minute), authLimit: newLimiter(5, time.Minute, 10*time.Minute),
		hookLimit: newLimiter(60, time.Minute, time.Minute)}
	nets, err := cfg.TrustedProxyNets()
	if err != nil {
		return nil, err
	}
	s.trusted = nets
	if len(cfg.OIDC) > 0 {
		if s.signer, err = newSigner(sec.CookieKey); err != nil {
			return nil, err
		}
		for name, pc := range cfg.OIDC {
			p, err := newProvider(name, pc, sec.Providers[name])
			if err != nil {
				return nil, err
			}
			s.providers[name] = p
		}
		if cfg.PublicURL == "" {
			return nil, fmt.Errorf("gateway: publicUrl is required when a login provider is configured")
		}
	}
	return s, nil
}

const cookieName = "picode_session"

// ServeHTTP: identify → map → resolve → (auto-pair | proxy).
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Strict-Transport-Security", "max-age=31536000")
	w.Header().Set("Referrer-Policy", "same-origin")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")

	peer := s.peer(r)
	if strings.HasPrefix(r.URL.Path, "/-/") {
		s.own(w, r, peer)
		return
	}
	if r.URL.Path == "/pair" && r.Method == http.MethodPost && !s.pairLimit.allow(peer) {
		s.page(w, http.StatusTooManyRequests, "Too many attempts", "Wait ten minutes and open a fresh link.")
		return
	}

	login, err := s.identify(r, peer)
	if err != nil {
		if errors.Is(err, errNeedsLogin) {
			s.needLogin(w, r)
			return
		}
		s.page(w, http.StatusServiceUnavailable, "Who are you?", "Tailscale did not tell this gateway who is connecting. "+html.EscapeString(err.Error()))
		return
	}
	linux, ok := s.Config.UserFor(login)
	if !ok {
		s.page(w, http.StatusForbidden, "Not on this server", html.EscapeString(login)+" has no PiCode here. Ask the admin: <code>picode users add "+html.EscapeString(login)+" &lt;user&gt;</code>.")
		return
	}
	be, err := s.Resolve(linux)
	if err != nil {
		var nr ErrNotRunning
		if errors.As(err, &nr) {
			s.page(w, http.StatusServiceUnavailable, "PiCode is not running for "+html.EscapeString(linux), "Ask the admin to run <code>picode provision --user "+html.EscapeString(linux)+" --shared</code>.")
			return
		}
		s.page(w, http.StatusInternalServerError, "Cannot reach PiCode", html.EscapeString(err.Error()))
		return
	}

	// First visit from a browser with no session: mint a pairing code
	// with the daemon's own token and send the person to the confirm
	// page. Only when there is no cookie at all — a stale one is the
	// SPA's business (its pairing screen), never a redirect loop.
	if navigates(r) && r.URL.Path != "/pair" && !hasCookie(r) {
		if s.allowPair(peer) {
			if code, err := s.mintPairing(r.Context(), be); err == nil {
				http.Redirect(w, r, "/pair?code="+code, http.StatusSeeOther)
				return
			} else {
				s.Log.Printf("gateway: pairing for %s: %v", linux, err)
			}
		}
	}

	proxy := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(be.URL)
			pr.Out.Host = s.Config.Hostname // what the daemon's Host/Origin checks expect (the public URL)
			// Nothing the client claims about itself survives.
			for _, h := range []string{"Authorization", "Forwarded", "X-Forwarded-For", "X-Forwarded-Host", "X-Forwarded-Proto", "X-Real-IP"} {
				pr.Out.Header.Del(h)
			}
			for h := range pr.Out.Header {
				if strings.HasPrefix(strings.ToLower(h), "x-picode-") {
					pr.Out.Header.Del(h)
				}
			}
			pr.Out.Header.Set("X-Forwarded-For", peer)
			pr.Out.Header.Set("X-Forwarded-Proto", "https")
			pr.Out.Header.Set("X-Forwarded-Host", s.Config.Hostname)
		},
		FlushInterval: -1, // SSE (/api/events) and streaming replies
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			s.page(w, http.StatusBadGateway, "PiCode did not answer", html.EscapeString(err.Error()))
		},
	}
	proxy.ServeHTTP(w, r)
}

var errNeedsLogin = errors.New("login required")

// identify: a tailnet peer is who Tailscale says; anyone else is who the
// gateway's own session says, or nobody yet.
func (s *Server) identify(r *http.Request, peer string) (string, error) {
	if s.Fake != nil {
		if login, ok := s.Fake[peer]; ok {
			return strings.ToLower(login), nil
		}
	} else if IsTailnet(net.ParseIP(peer)) {
		login, _, err := s.Identity.Whois(r.Context(), peer)
		return login, err
	}
	if len(s.providers) == 0 {
		if s.Fake != nil {
			return "", fmt.Errorf("%s is not in the fake identity map", peer)
		}
		return "", fmt.Errorf("%s is not on the tailnet, and no login provider is configured", peer)
	}
	if c, err := r.Cookie(SessionCookie); err == nil && c.Value != "" {
		if login, ok := s.signer.verify(c.Value, time.Now()); ok {
			return login, nil
		}
	}
	return "", errNeedsLogin
}

// peer is the connecting address, or the client the trusted proxy in
// front of us reports (the last X-Forwarded-For hop is the proxy's own
// observation; earlier hops are the client's claim and are ignored).
func (s *Server) peer(r *http.Request) string {
	remote := peerIP(r.RemoteAddr)
	if len(s.trusted) == 0 {
		return remote
	}
	ip := net.ParseIP(remote)
	for _, n := range s.trusted {
		if n.Contains(ip) {
			xff := r.Header.Get("X-Forwarded-For")
			parts := strings.Split(xff, ",")
			if last := strings.TrimSpace(parts[len(parts)-1]); last != "" && net.ParseIP(last) != nil {
				return last
			}
			return remote
		}
	}
	return remote
}

// needLogin sends a navigation to the login page and an API call a 401
// the SPA understands.
func (s *Server) needLogin(w http.ResponseWriter, r *http.Request) {
	if navigates(r) {
		next := r.URL.RequestURI()
		http.Redirect(w, r, "/-/login?next="+url.QueryEscape(next), http.StatusSeeOther)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = fmt.Fprint(w, `{"error":"login required","login":"/-/login"}`)
}

func (s *Server) allowPair(peer string) bool {
	s.pairMu.Lock()
	defer s.pairMu.Unlock()
	if s.pairLast == nil {
		s.pairLast = map[string]time.Time{}
	}
	if t, ok := s.pairLast[peer]; ok && time.Since(t) < 10*time.Second {
		return false
	}
	s.pairLast[peer] = time.Now()
	return true
}

// mintPairing asks the person's daemon for a one-time code, as the
// daemon's own install token. No Origin header: this is not a browser.
func (s *Server) mintPairing(ctx context.Context, be Backend) (string, error) {
	if be.Token == "" {
		return "", fmt.Errorf("no install token for %s", be.User)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, be.URL.String()+"/api/auth/pairings", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+be.Token)
	req.Header.Set("User-Agent", "picode-gateway")
	res, err := (&http.Client{Timeout: 5 * time.Second}).Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("daemon answered %s", res.Status)
	}
	var body struct {
		Code string `json:"code"`
	}
	if err := decodeJSON(res, &body); err != nil || body.Code == "" {
		return "", fmt.Errorf("no code in the daemon's answer")
	}
	return body.Code, nil
}

func navigates(r *http.Request) bool {
	if r.Method != http.MethodGet {
		return false
	}
	if m := r.Header.Get("Sec-Fetch-Mode"); m != "" {
		return m == "navigate"
	}
	return strings.Contains(r.Header.Get("Accept"), "text/html")
}

func hasCookie(r *http.Request) bool {
	c, err := r.Cookie(cookieName)
	return err == nil && c.Value != ""
}

func peerIP(remote string) string {
	host, _, err := net.SplitHostPort(remote)
	if err != nil {
		return remote
	}
	return strings.Trim(host, "[]")
}

// page is the gateway's own answer — the same look as the daemon's
// pairing page, so a person sees one product.
func (s *Server) page(w http.ResponseWriter, code int, heading, body string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(code)
	_, _ = fmt.Fprintf(w, pageTemplate, html.EscapeString(heading), html.EscapeString(heading), body)
}
