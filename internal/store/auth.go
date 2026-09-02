package store

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Authentication rows (ADR-0049). A Session is a principal: a browser
// that paired (cookie) or a token minted for scripts (bearer). Pairings
// are one-time codes that turn a visit into a browser session.

const (
	SessionBrowser = "browser"
	SessionToken   = "token"
)

var (
	ErrSessionInvalid = errors.New("store: session invalid")
	ErrPairingInvalid = errors.New("store: pairing code invalid")
	ErrPairingUsed    = errors.New("store: pairing code already used")
	ErrPairingExpired = errors.New("store: pairing code expired")
)

// Session is one principal. The secret never leaves CreateSession.
type Session struct {
	ID         string  `json:"id"`
	Kind       string  `json:"kind"`
	DeviceID   string  `json:"deviceId,omitempty"`
	Label      string  `json:"label"`
	IP         string  `json:"ip,omitempty"`
	CreatedAt  string  `json:"createdAt"`
	LastSeenAt string  `json:"lastSeenAt"`
	ExpiresAt  *string `json:"expiresAt,omitempty"`
	RevokedAt  *string `json:"revokedAt,omitempty"`
}

const sessionCols = `id, kind, device_id, label, ip, created_at, last_seen_at, expires_at, revoked_at`

func scanSession(row interface{ Scan(...any) error }, s *Session) error {
	return row.Scan(&s.ID, &s.Kind, &s.DeviceID, &s.Label, &s.IP, &s.CreatedAt, &s.LastSeenAt, &s.ExpiresAt, &s.RevokedAt)
}

func randomSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("store: random: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// HashSecret is the one-way form every secret is stored in.
func HashSecret(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

// CreateSession mints a principal and returns its secret once. ttl 0 =
// never expires (token sessions); browser sessions pass 90 days.
func (s *Store) CreateSession(kind, deviceID, label, ip string, ttl time.Duration) (Session, string, error) {
	if kind != SessionBrowser && kind != SessionToken {
		return Session{}, "", fmt.Errorf("store: session kind %q", kind)
	}
	secret, err := randomSecret()
	if err != nil {
		return Session{}, "", err
	}
	now := nowUTC()
	sess := Session{ID: newID(label, "sess"), Kind: kind, DeviceID: strings.TrimSpace(deviceID), Label: strings.TrimSpace(label),
		IP: strings.TrimSpace(ip), CreatedAt: now, LastSeenAt: now}
	if len(sess.Label) > 80 {
		sess.Label = sess.Label[:80]
	}
	if ttl > 0 {
		exp := time.Now().UTC().Add(ttl).Format(time.RFC3339Nano)
		sess.ExpiresAt = &exp
	}
	if _, err := s.db.Exec(`INSERT INTO auth_sessions (id, secret_hash, kind, device_id, label, ip, created_at, last_seen_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, sess.ID, HashSecret(secret), sess.Kind, sess.DeviceID, sess.Label, sess.IP, sess.CreatedAt, sess.LastSeenAt, sess.ExpiresAt); err != nil {
		return Session{}, "", fmt.Errorf("store: create session: %w", err)
	}
	s.note("session.created", nil, nil, sess)
	return sess, secret, nil
}

// LookupSession resolves a secret to a live session (not revoked, not
// expired). last_seen_at is touched at most once a minute; the touch is
// bookkeeping, not a change anyone watches, so it appends no event.
func (s *Store) LookupSession(secret string) (Session, error) {
	if secret == "" {
		return Session{}, ErrSessionInvalid
	}
	var sess Session
	err := scanSession(s.db.QueryRow(`SELECT `+sessionCols+` FROM auth_sessions WHERE secret_hash = ?`, HashSecret(secret)), &sess)
	if errors.Is(err, sql.ErrNoRows) {
		return Session{}, ErrSessionInvalid
	}
	if err != nil {
		return Session{}, fmt.Errorf("store: lookup session: %w", err)
	}
	if sess.RevokedAt != nil {
		return Session{}, ErrSessionInvalid
	}
	if sess.ExpiresAt != nil {
		if t, err := time.Parse(time.RFC3339Nano, *sess.ExpiresAt); err == nil && time.Now().After(t) {
			return Session{}, ErrSessionInvalid
		}
	}
	if last, err := time.Parse(time.RFC3339Nano, sess.LastSeenAt); err != nil || time.Since(last) > time.Minute {
		now := nowUTC()
		_, _ = s.db.Exec(`UPDATE auth_sessions SET last_seen_at = ? WHERE id = ?`, now, sess.ID)
		sess.LastSeenAt = now
	}
	return sess, nil
}

// ListSessions returns every session, newest first, revoked ones excluded.
func (s *Store) ListSessions() ([]Session, error) {
	rows, err := s.db.Query(`SELECT ` + sessionCols + ` FROM auth_sessions WHERE revoked_at IS NULL ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("store: list sessions: %w", err)
	}
	defer rows.Close()
	out := []Session{}
	for rows.Next() {
		var sess Session
		if err := scanSession(rows, &sess); err != nil {
			return nil, err
		}
		out = append(out, sess)
	}
	return out, rows.Err()
}

// RevokeSession ends a principal. Idempotent on an already revoked row.
func (s *Store) RevokeSession(id string) error {
	res, err := s.db.Exec(`UPDATE auth_sessions SET revoked_at = ? WHERE id = ? AND revoked_at IS NULL`, nowUTC(), id)
	if err != nil {
		return fmt.Errorf("store: revoke session: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	s.note("session.revoked", nil, nil, idData(id))
	return nil
}

// PruneSessions deletes revoked or expired rows older than t.
func (s *Store) PruneSessions(t time.Time) (int, error) {
	cut := t.UTC().Format(time.RFC3339Nano)
	res, err := s.db.Exec(`DELETE FROM auth_sessions WHERE (revoked_at IS NOT NULL AND revoked_at < ?) OR (expires_at IS NOT NULL AND expires_at < ?)`, cut, cut)
	if err != nil {
		return 0, fmt.Errorf("store: prune sessions: %w", err)
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

// CreatePairing mints a one-time code (returned once) valid for ttl.
func (s *Store) CreatePairing(createdBy string, ttl time.Duration) (string, string, error) {
	code, err := randomSecret()
	if err != nil {
		return "", "", err
	}
	exp := time.Now().UTC().Add(ttl).Format(time.RFC3339Nano)
	if _, err := s.db.Exec(`INSERT INTO auth_pairings (code_hash, created_by, created_at, expires_at) VALUES (?, ?, ?, ?)`,
		HashSecret(code), createdBy, nowUTC(), exp); err != nil {
		return "", "", fmt.Errorf("store: create pairing: %w", err)
	}
	s.note("pairing.created", nil, nil, map[string]string{"createdBy": createdBy, "expiresAt": exp})
	return code, exp, nil
}

// ConsumePairing spends a code exactly once.
func (s *Store) ConsumePairing(code string) error {
	if code == "" {
		return ErrPairingInvalid
	}
	var exp string
	var used *string
	err := s.db.QueryRow(`SELECT expires_at, used_at FROM auth_pairings WHERE code_hash = ?`, HashSecret(code)).Scan(&exp, &used)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrPairingInvalid
	}
	if err != nil {
		return fmt.Errorf("store: consume pairing: %w", err)
	}
	if used != nil {
		return ErrPairingUsed
	}
	if t, err := time.Parse(time.RFC3339Nano, exp); err == nil && time.Now().After(t) {
		return ErrPairingExpired
	}
	res, err := s.db.Exec(`UPDATE auth_pairings SET used_at = ? WHERE code_hash = ? AND used_at IS NULL`, nowUTC(), HashSecret(code))
	if err != nil {
		return fmt.Errorf("store: consume pairing: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrPairingUsed // lost the race
	}
	s.note("pairing.used", nil, nil, nil)
	return nil
}
