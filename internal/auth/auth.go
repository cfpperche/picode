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
	// CertNames answers the DNS names the served certificates cover
	// (tlsutil.CertNames); each is an allowed Host (ADR-0215). Nil means
	// none.
	CertNames func() []string
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
	// The deploy guard runs on this machine before a restart (ADR-0086);
	// from anywhere else the route is an ordinary guarded API.
	case p == "/api/deploy/readiness" && r.Method == http.MethodGet && Loopback(r):
		return true
	case p == "/mcp/communication":
		return true // ADR-0104: mandatory connection credential in its own handler.
	case r.Method == http.MethodPost && fireRoute.MatchString(p):
		return true
	}
	return false
}

// guarded is what Wrap inspects at all. Everything else — the UI, its
// assets, /pair and the ticketed /preview routes — passes straight
// through, so the host and origin checks below never see them either.
//
// /pair used to carry a line in exempt() as well, which read as a
// deliberate pass and was in fact unreachable: it is not guarded, so Wrap
// returns before exempt is ever asked. The line is gone; whether the
// pairing form should instead be guarded-and-exempt, and so inherit the
// Host and Origin checks, is an open question in
// docs/handoff/open/process.md — it changes who can reach pairing, which
// is not a tidy-up.
func guarded(r *http.Request) bool {
	p := r.URL.Path
	return strings.HasPrefix(p, "/api/") || strings.HasPrefix(p, "/ws/") || p == "/mcp/communication"
}

func isUpgrade(r *http.Request) bool {
	return strings.EqualFold(r.Header.Get("Upgrade"), "websocket")
}

// crossSite: the browser says the request comes from another site.
func crossSite(r *http.Request) bool {
	return strings.EqualFold(r.Header.Get("Sec-Fetch-Site"), "cross-site")
}

// firstParty: the request comes from PiCode's own page, a typed address or
// bookmark ("none"), or a client that sends no fetch metadata at all
// (curl, scripts, older browsers). Only these may be handed a loopback
// session (ADR-0215): a same-site page — another dev server on
// localhost:3000 — is someone else's code.
func firstParty(r *http.Request) bool {
	switch strings.ToLower(r.Header.Get("Sec-Fetch-Site")) {
	case "", "same-origin", "none":
		return true
	}
	return false
}

// pairableHost: over plain HTTP (PICODE_INSECURE=1) browsers send no fetch
// metadata to a name that is not "potentially trustworthy", and a LAN peer
// can answer mDNS for picode.local or box.local — a rebinding page would
// then look first-party on a loopback connection. So without TLS only a
// loopback name is auto-paired: localhost, *.localhost, a loopback IP.
// With TLS the certificate refuses a rebound name before it gets here.
func (s *Service) pairableHost(hostport string) bool {
	if !s.cfg.Insecure {
		return true
	}
	host := strings.ToLower(hostport)
	if h, _, err := net.SplitHostPort(hostport); err == nil {
		host = strings.ToLower(h)
	}
	host = strings.TrimSuffix(strings.Trim(host, "[]"), ".")
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
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

// HostAllowed rejects names that only a DNS-rebinding page would carry
// (ADR-0049, narrowed by ADR-0215). Allowed: an IP literal, localhost and
// *.localhost (browsers pin those to loopback, so no page can rebind them),
// picode.local, this machine's hostname bare or under a private suffix
// (box.local, box.lan, box-1.tailxxxx.ts.net — Tailscale numbers a
// repeated machine name), every DNS name the served certificates cover,
// and the public URL. Before ADR-0215 any *.local, any *.ts.net and the
// hostname followed by *any* domain passed: box.attacker.example was
// allowed, and in plain-HTTP mode nothing else stood in the way.
func (s *Service) HostAllowed(hostport string) bool {
	host := strings.ToLower(hostport)
	if h, _, err := net.SplitHostPort(hostport); err == nil {
		host = strings.ToLower(h)
	}
	host = strings.TrimSuffix(strings.Trim(host, "[]"), ".")
	if host == "" {
		return false
	}
	if net.ParseIP(host) != nil {
		return true
	}
	if host == "localhost" || host == "picode.local" || strings.HasSuffix(host, ".localhost") {
		return true
	}
	if machineName(host, strings.ToLower(s.cfg.Hostname)) {
		return true
	}
	if s.cfg.CertNames != nil {
		for _, n := range s.cfg.CertNames() {
			if n == host || (strings.HasPrefix(n, "*.") && wildcardCovers(n[2:], host)) {
				return true
			}
		}
	}
	if s.cfg.PublicURL != nil {
		if u, err := url.Parse(s.cfg.PublicURL()); err == nil && u.Host != "" && strings.EqualFold(u.Hostname(), host) {
			return true
		}
	}
	return false
}

// privateSuffixes are the zones a home or office network, mDNS or a
// tailnet puts a machine name under. None of them can be registered on the
// public internet, except ts.net, whose names Tailscale assigns per tailnet.
var privateSuffixes = []string{"local", "lan", "home", "home.arpa", "localdomain", "internal"}

// machineName: the hostname itself, or its first label (a Tailscale twin
// "box-1" counts) under one private suffix or a tailnet's ts.net zone.
func machineName(host, hostname string) bool {
	if hostname == "" {
		return false
	}
	if host == hostname {
		return true
	}
	// os.Hostname can be a full name (macOS: "Name-MacBook-Pro.local",
	// some Linux setups: "box.example.com"); the network names use its
	// first label.
	hostname, _, _ = strings.Cut(hostname, ".")
	first, rest, ok := strings.Cut(host, ".")
	if !ok || !(first == hostname || tailscaleTwin(first, hostname)) {
		return false
	}
	for _, suf := range privateSuffixes {
		if rest == suf {
			return true
		}
	}
	// box-1.tailxxxx.ts.net: exactly one tailnet label before ts.net.
	if tailnet, ok := strings.CutSuffix(rest, ".ts.net"); ok && tailnet != "" && !strings.Contains(tailnet, ".") {
		return true
	}
	return false
}

// tailscaleTwin: "box-2" for "box" — the number Tailscale appends when a
// machine name is already taken in the tailnet.
func tailscaleTwin(label, hostname string) bool {
	n, ok := strings.CutPrefix(label, hostname+"-")
	if !ok || n == "" {
		return false
	}
	for _, c := range n {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// wildcardCovers: a certificate's *.zone covers exactly one label under zone.
func wildcardCovers(zone, host string) bool {
	label, rest, ok := strings.Cut(host, ".")
	return ok && label != "" && rest == zone
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
//	POST /pair with a foreign Origin             → 403 origin
//	/api/health GET, /pair, /fire POST          → pass, no principal needed
//	not /api or /ws (the UI)                     → pass
//	Host not allowed                             → 403 unknown host
//	mutating or upgrade with foreign Origin      → 403 origin
//	Sec-Fetch-Site: cross-site (not exempt)      → 403 origin (ADR-0215)
//	principal from cookie / bearer               → pass with principal
//	mode off                                     → pass, anonymous
//	mode remote + loopback + no principal
//	  + first party (no/same-origin/none)        → mint a browser session, set cookie, pass
//	  + plain HTTP and a non-loopback Host name  → 401 pairing required (ADR-0215)
//	  + same-site                                → 401 pairing required (ADR-0215)
//	otherwise                                    → 401 pairing required
func (s *Service) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Pairing is the door you use when you cannot get in yet, so it
		// stays outside guarded(): the Host check belongs to routes you
		// reach *after* pairing, and applying it here would refuse the very
		// address someone is trying to pair a new device from — locked out
		// of the one page that exists to let you in.
		//
		// Submitting a code is still a submission, though, and a page open
		// in another tab must not be able to post codes at this daemon. The
		// cross-site check alone covers that, and it cannot lock anyone out:
		// a browser posting the form from PiCode's own page sends a
		// matching Origin, and a script or `picode pair` sends none, which
		// stays allowed.
		if r.URL.Path == "/pair" && mutating(r) && !s.originAllowed(r) {
			denied(w, http.StatusForbidden, "cross-site request refused")
			return
		}
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
		// ADR-0215: a read from another site is refused too. The response
		// would be opaque to that page (no CORS), but the request itself had
		// effects: with no cookie (SameSite=Strict) on a loopback address it
		// minted a session or rotated the secret of an idle one — any tab,
		// or a previewed HTML file on <label>.localhost, could do that with
		// an <img>. /api/health stays readable: it carries CORS on purpose.
		if crossSite(r) && !exempt(r) {
			denied(w, http.StatusForbidden, "cross-site request refused")
			return
		}
		if exempt(r) {
			next.ServeHTTP(w, r)
			return
		}
		// A communication capability never inherits anonymous loopback/admin access.
		if strings.HasPrefix(bearer(r), store.PeerTokenPrefix) {
			denied(w, http.StatusUnauthorized, "communication credential cannot access owner API")
			return
		}
		p := s.resolve(r)
		mode := s.Mode()
		if p == nil {
			switch {
			case mode == ModeOff:
			case mode == ModeRemote && Loopback(r) && firstParty(r) && s.pairableHost(r.Host):
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
