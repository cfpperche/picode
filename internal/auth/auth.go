// Package auth is the request gate (ADR-0049): who is calling, and is the
// call allowed. Principals are browser sessions (cookie, made by pairing
// or minted for loopback) and tokens (bearer: the install token or a
// token session). Origin and Host are checked on every state-changing or
// upgrading request regardless of mode, so a page on another site cannot
// drive a local PiCode.
package auth

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/cfpperche/picode/internal/presence"
	"github.com/cfpperche/picode/internal/store"
)

// Modes.
const (
	ModeOff    = "off"    // no principal required (dev, trusted proxy); Origin/Host still checked
	ModeRemote = "remote" // loopback auto-pairs, everything else must be paired
	ModeAll    = "all"    // loopback must pair too (shared / public servers)
)

// ModeSettingKey is the store setting that selects the mode.
const ModeSettingKey = "auth.mode"

const (
	CookieName      = "picode_session"
	BrowserTTL      = 90 * 24 * time.Hour
	PairingTTL      = 10 * time.Minute
	TokenFile       = "token"
	pairFailsPerMin = 5
	pairLockout     = 10 * time.Minute
)

// Config wires the service.
type Config struct {
	Store     *store.Store
	DataDir   string
	Insecure  bool          // cookies drop Secure over plain HTTP
	PublicURL func() string // advertised origin, "" when unknown
	Hostname  string        // machine name, allowed as a Host
	// SessionLive reports whether a presence ping for that session id
	// arrived within the staleness window (ADR-0049 amendment: the
	// loopback reuse must not rotate the cookie out from under a browser
	// that is actively using it). Nil means nobody is live.
	SessionLive func(sessionID string) bool
}

// Principal is the resolved caller. Nil in mode off means anonymous.
type Principal struct {
	Session  store.Session
	Kind     string // browser | token | install
	Loopback bool
}

type ctxKey struct{}

// From returns the request's principal, if any.
func From(r *http.Request) *Principal {
	p, _ := r.Context().Value(ctxKey{}).(*Principal)
	return p
}

// InstallTokenLabel names the token session that <DataDir>/token holds.
const InstallTokenLabel = "Install token"

// Service holds the install token path and the pairing rate limiter.
type Service struct {
	cfg       Config
	tokenPath string

	mu    sync.Mutex
	fails map[string]*failWindow
}

type failWindow struct {
	count int
	start time.Time
	until time.Time
}

// New loads the install token at <DataDir>/token, or mints one. The
// token is a token session like any other (label InstallTokenLabel):
// the file only holds its secret. So it lists on Devices, presence joins
// onto it, and a bearer is resolved by one path — LookupSession.
func New(cfg Config) (*Service, error) {
	s := &Service{cfg: cfg, tokenPath: filepath.Join(cfg.DataDir, TokenFile), fails: map[string]*failWindow{}}
	if _, err := s.currentToken(); err != nil {
		if _, err := s.RotateToken(); err != nil {
			return nil, err
		}
	}
	return s, nil
}

// currentToken returns the file's secret when it resolves to a live
// token session.
func (s *Service) currentToken() (store.Session, error) {
	tok, err := os.ReadFile(s.tokenPath)
	if err != nil {
		return store.Session{}, err
	}
	sess, err := s.cfg.Store.LookupSession(strings.TrimSpace(string(tok)))
	if err != nil {
		return store.Session{}, err
	}
	if sess.Kind != store.SessionToken {
		return store.Session{}, fmt.Errorf("auth: token file holds a %s session", sess.Kind)
	}
	return sess, nil
}

// RotateToken revokes the current install-token session, creates a new
// one and writes its secret to <DataDir>/token (0600). Returns the secret.
func (s *Service) RotateToken() (string, error) {
	if old, err := s.currentToken(); err == nil {
		_ = s.cfg.Store.RevokeSession(old.ID)
	}
	_, tok, err := s.cfg.Store.CreateSession(store.SessionToken, "", InstallTokenLabel, "", 0)
	if err != nil {
		return "", fmt.Errorf("auth: token: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(s.tokenPath), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(s.tokenPath, []byte(tok+"\n"), 0o600); err != nil {
		return "", fmt.Errorf("auth: write token: %w", err)
	}
	return tok, nil
}

// TokenPath is where scripts read the install token.
func (s *Service) TokenPath() string { return s.tokenPath }

// Mode: the setting, else PICODE_AUTH_MODE (how a provisioned member
// daemon starts in mode all, ADR-0051), else remote.
func (s *Service) Mode() string {
	if s.cfg.Store != nil {
		if v, ok, err := s.cfg.Store.GetSetting(ModeSettingKey); err == nil && ok && validMode(v) {
			return v
		}
	}
	if v := os.Getenv("PICODE_AUTH_MODE"); validMode(v) {
		return v
	}
	return ModeRemote
}

func validMode(v string) bool { return v == ModeOff || v == ModeRemote || v == ModeAll }

// SetMode persists a mode.
func (s *Service) SetMode(mode string) error {
	switch mode {
	case ModeOff, ModeRemote, ModeAll:
		return s.cfg.Store.SetSetting(ModeSettingKey, mode)
	}
	return fmt.Errorf("mode must be off, remote or all")
}

// ---- request classification ----

var fireRoute = regexp.MustCompile(`^/api/automations/[^/]+/fire$`)

// exempt routes never need a principal (they carry their own gate or none).
func exempt(r *http.Request) bool {
	p := r.URL.Path
	switch {
	case p == "/api/health" && r.Method == http.MethodGet:
		return true
	case p == "/pair":
		return true
	case r.Method == http.MethodPost && fireRoute.MatchString(p):
		return true
	}
	return false
}

func guarded(r *http.Request) bool {
	p := r.URL.Path
	return strings.HasPrefix(p, "/api/") || strings.HasPrefix(p, "/ws/")
}

func isUpgrade(r *http.Request) bool {
	return strings.EqualFold(r.Header.Get("Upgrade"), "websocket")
}

func mutating(r *http.Request) bool {
	switch r.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return false
	}
	return true
}

// browserLike: only a browser gets a cookie minted; it is the client
// that will send it back.
func browserLike(r *http.Request) bool {
	ua := r.UserAgent()
	return strings.Contains(ua, "Mozilla/") || r.Header.Get("Sec-Fetch-Mode") != ""
}

// Loopback reports a direct connection from this machine (no proxy hop).
func Loopback(r *http.Request) bool {
	if r.Header.Get("X-Forwarded-For") != "" || r.Header.Get("Forwarded") != "" {
		return false
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// HostAllowed rejects names that only a DNS-rebinding page would carry:
// anything that is not loopback, an IP literal, localhost / picode.local,
// a .local or .ts.net name, this machine's hostname, or the public URL.
func (s *Service) HostAllowed(hostport string) bool {
	host := strings.ToLower(hostport)
	if h, _, err := net.SplitHostPort(hostport); err == nil {
		host = strings.ToLower(h)
	}
	host = strings.Trim(host, "[]")
	if host == "" {
		return false
	}
	if net.ParseIP(host) != nil {
		return true
	}
	switch host {
	case "localhost", "picode.local":
		return true
	}
	if strings.HasSuffix(host, ".local") || strings.HasSuffix(host, ".ts.net") || strings.HasSuffix(host, ".localhost") {
		return true
	}
	if s.cfg.Hostname != "" && (host == strings.ToLower(s.cfg.Hostname) || strings.HasPrefix(host, strings.ToLower(s.cfg.Hostname)+".")) {
		return true
	}
	if s.cfg.PublicURL != nil {
		if u, err := url.Parse(s.cfg.PublicURL()); err == nil && u.Host != "" && strings.EqualFold(u.Hostname(), host) {
			return true
		}
	}
	return false
}

// originAllowed: a browser-set Origin must be this server (same host) or
// the public URL. Sec-Fetch-Site: cross-site is refused even without one.
func (s *Service) originAllowed(r *http.Request) bool {
	if strings.EqualFold(r.Header.Get("Sec-Fetch-Site"), "cross-site") {
		return false
	}
	origin := r.Header.Get("Origin")
	if origin == "" || origin == "null" {
		return origin == "" // "null" is a sandboxed or file: page: refuse
	}
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	if strings.EqualFold(u.Host, r.Host) {
		return true
	}
	if s.cfg.PublicURL != nil {
		if pu, err := url.Parse(s.cfg.PublicURL()); err == nil && pu.Host != "" && strings.EqualFold(pu.Host, u.Host) {
			return true
		}
	}
	return false
}

// ---- principal resolution ----

func bearer(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if len(h) > 7 && strings.EqualFold(h[:7], "bearer ") {
		return strings.TrimSpace(h[7:])
	}
	return ""
}

func (s *Service) resolve(r *http.Request) *Principal {
	loop := Loopback(r)
	if c, err := r.Cookie(CookieName); err == nil && c.Value != "" {
		if sess, err := s.cfg.Store.LookupSession(c.Value); err == nil {
			return &Principal{Session: sess, Kind: sess.Kind, Loopback: loop}
		}
	}
	if tok := bearer(r); tok != "" {
		if sess, err := s.cfg.Store.LookupSession(tok); err == nil {
			kind := sess.Kind
			if sess.Label == InstallTokenLabel {
				kind = "install"
			}
			return &Principal{Session: sess, Kind: kind, Loopback: loop}
		}
	}
	return nil
}

func (s *Service) setCookie(w http.ResponseWriter, secret string, ttl time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name: CookieName, Value: secret, Path: "/", HttpOnly: true, Secure: !s.cfg.Insecure,
		SameSite: http.SameSiteStrictMode, MaxAge: int(ttl / time.Second),
	})
}

func clearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: CookieName, Value: "", Path: "/", HttpOnly: true, MaxAge: -1})
}

func denied(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_, _ = fmt.Fprintf(w, `{"error":%q,"pair":%v}`, msg, code == http.StatusUnauthorized)
}

// Wrap is the middleware. Decision table (each row tested):
//
//	/api/health GET, /pair, /fire POST          → pass, no principal needed
//	not /api or /ws (the UI)                     → pass
//	Host not allowed                             → 403 unknown host
//	mutating or upgrade with foreign Origin      → 403 origin
//	principal from cookie / bearer               → pass with principal
//	mode off                                     → pass, anonymous
//	mode remote + loopback + no principal        → mint a browser session, set cookie, pass
//	otherwise                                    → 401 pairing required
func (s *Service) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !guarded(r) {
			next.ServeHTTP(w, r)
			return
		}
		if !s.HostAllowed(r.Host) {
			denied(w, http.StatusForbidden, "unknown host")
			return
		}
		if (mutating(r) || isUpgrade(r) || r.URL.Path == "/api/events") && !s.originAllowed(r) {
			denied(w, http.StatusForbidden, "cross-site request refused")
			return
		}
		if exempt(r) {
			next.ServeHTTP(w, r)
			return
		}
		p := s.resolve(r)
		mode := s.Mode()
		if p == nil {
			switch {
			case mode == ModeOff:
			case mode == ModeRemote && Loopback(r):
				if !browserLike(r) {
					// curl and scripts on this machine pass without a row:
					// a session per request would only fill the table.
					p = &Principal{Kind: "loopback", Session: store.Session{ID: "loopback", Kind: store.SessionToken, Label: "this machine"}, Loopback: true}
					break
				}
				sess, secret, err := s.loopbackSession("This machine · " + presence.Label(r.UserAgent()))
				if err != nil {
					denied(w, http.StatusInternalServerError, err.Error())
					return
				}
				s.setCookie(w, secret, BrowserTTL)
				p = &Principal{Session: sess, Kind: sess.Kind, Loopback: true}
			default:
				if c, err := r.Cookie(CookieName); err == nil && c.Value != "" {
					clearCookie(w) // a stale cookie: let the browser forget it
				}
				denied(w, http.StatusUnauthorized, "pairing required")
				return
			}
		}
		if p != nil {
			r = r.WithContext(context.WithValue(r.Context(), ctxKey{}, p))
		}
		next.ServeHTTP(w, r)
	})
}

// ---- pairing ----

// loopbackSession answers a browser-like loopback visit with a browser
// session: reuse when the newest live session with that label already
// exists and no browser is actively using it (every headless QA profile
// used to mint its own 90-day row and the Devices list filled with
// "This machine · Linux" duplicates — ADR-0049 amendment), mint otherwise.
func (s *Service) loopbackSession(label string) (store.Session, string, error) {
	if found, err := s.cfg.Store.NewestLiveBrowserSession(label, "127.0.0.1"); err == nil && !s.protected(found) {
		if sess, secret, err := s.cfg.Store.RotateSessionSecret(found.ID, BrowserTTL); err == nil {
			return sess, secret, nil
		}
	}
	return s.cfg.Store.CreateSession(store.SessionBrowser, "", label, "127.0.0.1", BrowserTTL)
}

// reuseBurstWindow: a session younger than this is part of a first-visit
// burst — a fresh browser fires its first requests in parallel without a
// cookie, and the winner's Set-Cookie has not reached the losers yet. A
// ping from the burst itself must not make the racing request mint a
// duplicate (seen live: two "This machine · Windows" rows 50 ms apart),
// so young sessions are reused even while live.
var reuseBurstWindow = 30 * time.Second

// protected reports whether the session must not be rotated: a browser is
// actively using it (presence) and it is established — not the tail of an
// arrival burst from the very same browser.
func (s *Service) protected(sess store.Session) bool {
	if !s.live(sess.ID) {
		return false
	}
	if created, err := time.Parse(time.RFC3339Nano, sess.CreatedAt); err == nil && time.Since(created) < reuseBurstWindow {
		return false
	}
	return true
}

func (s *Service) live(id string) bool { return s.cfg.SessionLive != nil && s.cfg.SessionLive(id) }

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// allowAttempt is the pairing brute-force gate: 5 failures per minute per
// IP, then a 10-minute lockout.
func (s *Service) allowAttempt(ip string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	fw := s.fails[ip]
	if fw == nil {
		return true
	}
	if !fw.until.IsZero() && now.Before(fw.until) {
		return false
	}
	return true
}

func (s *Service) noteFailure(ip string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	fw := s.fails[ip]
	if fw == nil || now.Sub(fw.start) > time.Minute {
		fw = &failWindow{start: now}
		s.fails[ip] = fw
	}
	fw.count++
	if fw.count >= pairFailsPerMin {
		fw.until = now.Add(pairLockout)
	}
}

func (s *Service) noteSuccess(ip string) {
	s.mu.Lock()
	delete(s.fails, ip)
	s.mu.Unlock()
}

// PairURL builds the link a pairing code turns into: the public URL when
// configured, else the request's own origin. A caller that knows a
// better reachable base (the share report, when the request came in on
// loopback) passes it through PairURLFrom.
func (s *Service) PairURL(r *http.Request, code string) string {
	base := ""
	if s.cfg.PublicURL != nil {
		base = strings.TrimRight(s.cfg.PublicURL(), "/")
	}
	if base == "" {
		scheme := "https"
		if r.TLS == nil {
			scheme = "http"
		}
		base = scheme + "://" + r.Host
	}
	return PairURLFrom(base, code)
}

// PairURLFrom appends the pairing path to a base origin.
func PairURLFrom(base, code string) string {
	return strings.TrimRight(base, "/") + "/pair?code=" + url.QueryEscape(code)
}

// PublicURL is the configured public origin, "" when none.
func (s *Service) PublicURL() string {
	if s.cfg.PublicURL == nil {
		return ""
	}
	return strings.TrimRight(s.cfg.PublicURL(), "/")
}

// Pair spends a code for the visiting browser: cookie set, session
// recorded with the device's label and IP. Returns the session.
func (s *Service) Pair(w http.ResponseWriter, r *http.Request, code, deviceID string) (store.Session, error) {
	ip := clientIP(r)
	if !s.allowAttempt(ip) {
		return store.Session{}, errTooMany
	}
	if err := s.cfg.Store.ConsumePairing(code); err != nil {
		s.noteFailure(ip)
		return store.Session{}, err
	}
	s.noteSuccess(ip)
	sess, secret, err := s.cfg.Store.CreateSession(store.SessionBrowser, deviceID, presence.Label(r.UserAgent()), ip, BrowserTTL)
	if err != nil {
		return store.Session{}, err
	}
	s.setCookie(w, secret, BrowserTTL)
	return sess, nil
}

var errTooMany = errors.New("auth: too many attempts")

// IsTooMany reports the lockout error.
func IsTooMany(err error) bool { return errors.Is(err, errTooMany) }

// Logout revokes the caller's browser session and clears the cookie.
func (s *Service) Logout(w http.ResponseWriter, r *http.Request) {
	if p := From(r); p != nil && p.Kind == store.SessionBrowser {
		_ = s.cfg.Store.RevokeSession(p.Session.ID)
	}
	clearCookie(w)
}
