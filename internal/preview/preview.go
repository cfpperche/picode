// Package preview mints and resolves the short-lived capability tickets that
// let one HTML file render in a sandboxed frame (ADR-0136). A ticket binds a
// token to one owner directory, one document and the session that asked for
// it; the package also owns the read-only web-asset policy — the closed MIME
// list and the dotfile rule. It never writes a file, never touches Pi's
// session files, and knows nothing about HTTP.
package preview

import (
	"crypto/rand"
	"encoding/base32"
	"encoding/base64"
	"fmt"
	"path"
	"strings"
	"sync"
	"time"
)

// LabelBytes is the entropy in a ticket's DNS label. The label is the
// origin's name (ADR-0137), so it is a credential too.
const LabelBytes = 16

// DefaultTTL is how long a minted ticket lives. The pane mints again on
// every open and Reload, so a token that leaks (a log, a screenshot, a
// shared URL) has an hour of reach at most.
const DefaultTTL = time.Hour

// TokenBytes is the entropy in a ticket token; the token is a credential.
const TokenBytes = 32

// MaxWatch caps the paths watched per ticket, so a page that pulls in a
// thousand files cannot turn the live-reload poller into a filesystem
// crawl. The document is served first and always fits.
const MaxWatch = 200

// Ticket is one capability: a token that serves Root, and the document under
// it the pane first asked for. Label is the same capability's DNS name, the
// origin the pane may reach it at (ADR-0137). FrameOrigin is the origin that
// minted it — the only one allowed to frame or read it. SessionID is empty
// for anonymous mode.
type Ticket struct {
	Token       string
	Label       string
	OwnerKind   string // agent | term | workspace
	OwnerID     string
	Root        string // absolute, symlink-resolved directory at mint time
	Path        string // slash-relative document path under Root
	SessionID   string
	FrameOrigin string // minting UI origin; empty when not a browser
	CreatedAt   time.Time
	ExpiresAt   time.Time
}

// Request is what a mint needs. It is a struct rather than six positional
// strings because two of them are credentials and one is an origin.
type Request struct {
	OwnerKind   string
	OwnerID     string
	Root        string
	Path        string
	SessionID   string
	FrameOrigin string
}

// Store holds live tickets in memory. A daemon restart drops all of them.
type Store struct {
	mu      sync.Mutex
	ttl     time.Duration
	now     func() time.Time
	tickets map[string]*entry
	byLabel map[string]string // DNS label → token, for the origin form
}

// entry is one ticket plus the files it has served, the set a live-reload
// stream stats, and the unsaved editor buffer the pane may overlay on the
// document (ADR-0136 v1.5).
type entry struct {
	ticket  Ticket
	watch   map[string]struct{}
	overlay *string
}

// NewStore builds a store; ttl <= 0 means DefaultTTL.
func NewStore(ttl time.Duration) *Store {
	if ttl <= 0 {
		ttl = DefaultTTL
	}
	return &Store{ttl: ttl, now: time.Now, tickets: make(map[string]*entry), byLabel: make(map[string]string)}
}

// Mint creates a ticket for one document. It does not validate the path; the
// caller did that against the filesystem.
func (s *Store) Mint(req Request) (Ticket, error) {
	tok, err := newToken()
	if err != nil {
		return Ticket{}, err
	}
	now := s.now()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sweepLocked(now)
	// A label collision is a birthday problem with 2^128 options; loop a few
	// times rather than trust it, and never reuse a live label.
	t := Ticket{
		Token:       tok,
		OwnerKind:   req.OwnerKind,
		OwnerID:     req.OwnerID,
		Root:        req.Root,
		Path:        req.Path,
		SessionID:   req.SessionID,
		FrameOrigin: req.FrameOrigin,
		CreatedAt:   now,
		ExpiresAt:   now.Add(s.ttl),
	}
	for i := 0; i < 5; i++ {
		label, err := newLabel()
		if err != nil {
			return Ticket{}, err
		}
		if _, taken := s.byLabel[label]; taken {
			continue
		}
		t.Label = label
		break
	}
	if t.Label == "" {
		return Ticket{}, fmt.Errorf("preview: could not allocate a label")
	}
	s.tickets[tok] = &entry{ticket: t, watch: make(map[string]struct{})}
	s.byLabel[t.Label] = tok
	return t, nil
}

// Get resolves a token, forgetting it once expired.
func (s *Store) Get(token string) (Ticket, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.getLocked(token)
}

// ByLabel resolves a ticket's DNS label (the origin form, ADR-0137); labels
// are case-insensitive because DNS names are.
func (s *Store) ByLabel(label string) (Ticket, bool) {
	label = strings.ToLower(strings.TrimSpace(label))
	if label == "" {
		return Ticket{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	tok, ok := s.byLabel[label]
	if !ok {
		return Ticket{}, false
	}
	return s.getLocked(tok)
}

func (s *Store) getLocked(token string) (Ticket, bool) {
	e, ok := s.tickets[token]
	if !ok {
		return Ticket{}, false
	}
	if !s.now().Before(e.ticket.ExpiresAt) {
		s.removeLocked(token)
		return Ticket{}, false
	}
	return e.ticket, true
}

// Touch records a file the ticket served, among the paths a live-reload
// stream polls. Unknown and expired tokens are ignored; the per-ticket cap
// is MaxWatch.
func (s *Store) Touch(token, path string) {
	if token == "" || path == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.tickets[token]
	if !ok || !s.now().Before(e.ticket.ExpiresAt) {
		return
	}
	if _, seen := e.watch[path]; seen {
		return
	}
	if len(e.watch) >= MaxWatch {
		return
	}
	e.watch[path] = struct{}{}
}

// Watched snapshots the ticket's served files for one poll. A missing or
// expired ticket has none.
func (s *Store) Watched(token string) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.tickets[token]
	if !ok || !s.now().Before(e.ticket.ExpiresAt) {
		return nil
	}
	out := make([]string, 0, len(e.watch))
	for p := range e.watch {
		out = append(out, p)
	}
	return out
}

// SetOverlay replaces the ticket's document with the pane's unsaved editor
// buffer: the preview shows what the editor holds, disk untouched. The
// ticket is the gate (the sandbox already owns its own content); the caller
// only offers this for the document the ticket was minted for.
func (s *Store) SetOverlay(token, text string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.tickets[token]
	if !ok || !s.now().Before(e.ticket.ExpiresAt) {
		return false
	}
	t := text
	e.overlay = &t
	return true
}

// ClearOverlay drops the ticket's unsaved buffer so the file on disk serves
// again. The ticket's own origin uses this on Save: the buffer and the file
// are equal at that moment, and clearing keeps the origin (and its storage)
// alive while leaving no overlay to mask later disk changes.
func (s *Store) ClearOverlay(token string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.tickets[token]
	if !ok || !s.now().Before(e.ticket.ExpiresAt) {
		return false
	}
	e.overlay = nil
	return true
}

// Overlay answers the ticket's unsaved buffer, when the pane set one.
func (s *Store) Overlay(token string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.tickets[token]
	if !ok || !s.now().Before(e.ticket.ExpiresAt) || e.overlay == nil {
		return "", false
	}
	return *e.overlay, true
}

// Len reports live (unexpired) tickets.
func (s *Store) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sweepLocked(s.now())
	return len(s.tickets)
}

func (s *Store) sweepLocked(now time.Time) {
	for tok, e := range s.tickets {
		if !now.Before(e.ticket.ExpiresAt) {
			s.removeLocked(tok)
		}
	}
}

func (s *Store) removeLocked(token string) {
	if e, ok := s.tickets[token]; ok {
		delete(s.byLabel, e.ticket.Label)
	}
	delete(s.tickets, token)
}

func newToken() (string, error) {
	b := make([]byte, TokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("preview: token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// newLabel is the DNS-safe half of a ticket: 26 lowercase base32 characters
// (130 bits), one label, valid inside `<label>.localhost`.
func newLabel() (string, error) {
	b := make([]byte, LabelBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("preview: label: %w", err)
	}
	enc := base32.StdEncoding.WithPadding(base32.NoPadding)
	return strings.ToLower(enc.EncodeToString(b)), nil
}

// mimeByExt is the closed list of types a preview may serve: web assets, not
// the project. `.env`, key files, extension-less files and anything absent
// here are structurally unservable (ADR-0136).
var mimeByExt = map[string]string{
	".html":  "text/html; charset=utf-8",
	".htm":   "text/html; charset=utf-8",
	".css":   "text/css; charset=utf-8",
	".js":    "text/javascript; charset=utf-8",
	".mjs":   "text/javascript; charset=utf-8",
	".cjs":   "text/javascript; charset=utf-8",
	".json":  "application/json; charset=utf-8",
	".map":   "application/json; charset=utf-8",
	".svg":   "image/svg+xml",
	".png":   "image/png",
	".jpg":   "image/jpeg",
	".jpeg":  "image/jpeg",
	".gif":   "image/gif",
	".webp":  "image/webp",
	".avif":  "image/avif",
	".ico":   "image/x-icon",
	".bmp":   "image/bmp",
	".woff":  "font/woff",
	".woff2": "font/woff2",
	".ttf":   "font/ttf",
	".otf":   "font/otf",
	".wasm":  "application/wasm",
	".txt":   "text/plain; charset=utf-8",
	".csv":   "text/csv; charset=utf-8",
	".xml":   "text/xml; charset=utf-8",
	".mp3":   "audio/mpeg",
	".wav":   "audio/wav",
	".ogg":   "audio/ogg",
	".m4a":   "audio/mp4",
	".mp4":   "video/mp4",
	".webm":  "video/webm",
	".pdf":   "application/pdf",
	".glb":   "model/gltf-binary",
	".gltf":  "model/gltf+json",
}

// MIMEType answers the Content-Type for a servable asset. The second result
// is false for anything outside the list.
func MIMEType(rel string) (string, bool) {
	m, ok := mimeByExt[strings.ToLower(path.Ext(rel))]
	return m, ok
}

// IsDocument reports the one thing a ticket may be minted for.
func IsDocument(rel string) bool {
	base := strings.ToLower(path.Base(rel))
	// A file named exactly `.html` is a dotfile with no name, not a document.
	if base == ".html" || base == ".htm" {
		return false
	}
	switch path.Ext(base) {
	case ".html", ".htm":
		return true
	default:
		return false
	}
}

// Hidden reports a path with any dot-prefixed segment: the preview never
// serves `.env`, `.git/…` or a `.ssh` alias, however it is spelled.
func Hidden(rel string) bool {
	for _, part := range strings.Split(rel, "/") {
		if strings.HasPrefix(part, ".") {
			return true
		}
	}
	return false
}
