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
	"strings"
	"sync"
	"time"
)

// Server is the gateway handler.
type Server struct {
	Config   Config
	Identity *Identity
	// Resolve finds a person's daemon; tests point it at httptest servers.
	Resolve func(linux string) (Backend, error)
	// Fake is a test/scratch identity map (peer ip → login), only honoured
	// when the listener is loopback; see cmd/picode/gateway.go.
	Fake map[string]string
	Log  *log.Logger

	pairMu   sync.Mutex
	pairLast map[string]time.Time
}

// New wires a server with the real identity and backend resolution.
func New(cfg Config) *Server {
	return &Server{Config: cfg, Identity: NewIdentity(60 * time.Second), Resolve: Resolve, Log: log.Default(), pairLast: map[string]time.Time{}}
}

const cookieName = "picode_session"

// ServeHTTP: identify → map → resolve → (auto-pair | proxy).
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Strict-Transport-Security", "max-age=31536000")
	w.Header().Set("Referrer-Policy", "same-origin")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	peer := peerIP(r.RemoteAddr)
	login, err := s.whois(r.Context(), peer)
	if err != nil {
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

func (s *Server) whois(ctx context.Context, peer string) (string, error) {
	if s.Fake != nil {
		if login, ok := s.Fake[peer]; ok {
			return strings.ToLower(login), nil
		}
		return "", fmt.Errorf("%s is not in the fake identity map", peer)
	}
	login, _, err := s.Identity.Whois(ctx, peer)
	return login, err
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
