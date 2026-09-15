package store

// The work browser's site permissions (slice 3, Browser permissions): the
// shell reports each allow/deny as it happens, and Settings ▸ Browser reads,
// changes and prunes the standings. Mutations announce
// browserpermission.updated (ADR-0048).

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// BrowserPermission is one standing: what one site may do with one kind.
type BrowserPermission struct {
	ID        int64  `json:"id"`
	Origin    string `json:"origin"`
	Kind      string `json:"kind"`
	Decision  string `json:"decision"`
	DecidedAt string `json:"decidedAt"`
}

// permissionKinds is the closed vocabulary the daemon stores. The shell knows
// the platform's own kinds; this list is what the dialog can show.
var permissionKinds = map[string]bool{
	"camera": true, "microphone": true, "location": true, "notifications": true,
	"clipboard": true, "autoplay": true, "sensors": true, "midi": true,
	"fonts": true, "filesystem": true,
}

// NormalizePermissionKind checks the kind against the closed list.
func NormalizePermissionKind(kind string) (string, error) {
	k := strings.ToLower(strings.TrimSpace(kind))
	if !permissionKinds[k] {
		return "", fmt.Errorf("store: %q is not a permission kind", kind)
	}
	return k, nil
}

// NormalizePermissionDecision accepts the two standings a site can hold.
func NormalizePermissionDecision(decision string) (string, error) {
	d := strings.ToLower(strings.TrimSpace(decision))
	if d != "allow" && d != "deny" {
		return "", fmt.Errorf("store: %q is not a permission decision", decision)
	}
	return d, nil
}

// SetBrowserPermission records one site's standing; a repeat changes it in
// place rather than adding a second row for the same pair.
func (s *Store) SetBrowserPermission(origin, kind, decision string) (BrowserPermission, error) {
	o := strings.ToLower(strings.TrimSpace(origin))
	if o == "" {
		return BrowserPermission{}, fmt.Errorf("store: a permission needs an origin")
	}
	k, err := NormalizePermissionKind(kind)
	if err != nil {
		return BrowserPermission{}, err
	}
	d, err := NormalizePermissionDecision(decision)
	if err != nil {
		return BrowserPermission{}, err
	}
	when := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := s.db.Exec(`
		INSERT INTO browser_permissions (origin, kind, decision, decided_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(origin, kind) DO UPDATE SET
			decision = excluded.decision, decided_at = excluded.decided_at`,
		o, k, d, when); err != nil {
		return BrowserPermission{}, fmt.Errorf("store: set browser permission: %w", err)
	}
	row, err := s.browserPermissionFor(o, k)
	if err != nil {
		return BrowserPermission{}, err
	}
	s.note("browserpermission.updated", nil, nil, nil)
	return row, nil
}

func (s *Store) browserPermissionFor(origin, kind string) (BrowserPermission, error) {
	row := s.db.QueryRow(`
		SELECT id, origin, kind, decision, decided_at
		FROM browser_permissions WHERE origin = ? AND kind = ?`, origin, kind)
	return scanBrowserPermission(row)
}

// ListBrowserPermissions returns the standings, newest first, optionally for
// one kind; a query matches the origin.
func (s *Store) ListBrowserPermissions(limit int, kind, query string) ([]BrowserPermission, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	k := strings.ToLower(strings.TrimSpace(kind))
	q := strings.TrimSpace(query)
	rows, err := s.db.Query(`
		SELECT id, origin, kind, decision, decided_at
		FROM browser_permissions
		WHERE (? = '' OR kind = ?) AND (? = '' OR origin LIKE '%' || ? || '%')
		ORDER BY decided_at DESC, id DESC
		LIMIT ?`, k, k, q, q, limit)
	if err != nil {
		return nil, fmt.Errorf("store: list browser permissions: %w", err)
	}
	defer rows.Close()
	out := []BrowserPermission{}
	for rows.Next() {
		p, err := scanBrowserPermission(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// DeleteBrowserPermission forgets one standing (the site will ask again).
func (s *Store) DeleteBrowserPermission(id int64) error {
	res, err := s.db.Exec(`DELETE FROM browser_permissions WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("store: delete browser permission: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	s.note("browserpermission.updated", nil, nil, nil)
	return nil
}

// ClearBrowserPermissions forgets every standing, or every standing of one
// kind when kind is given.
func (s *Store) ClearBrowserPermissions(kind string) error {
	k := strings.ToLower(strings.TrimSpace(kind))
	var err error
	if k == "" {
		_, err = s.db.Exec(`DELETE FROM browser_permissions`)
	} else {
		_, err = s.db.Exec(`DELETE FROM browser_permissions WHERE kind = ?`, k)
	}
	if err != nil {
		return fmt.Errorf("store: clear browser permissions: %w", err)
	}
	s.note("browserpermission.updated", nil, nil, nil)
	return nil
}

func scanBrowserPermission(row rowScanner) (BrowserPermission, error) {
	var p BrowserPermission
	if err := row.Scan(&p.ID, &p.Origin, &p.Kind, &p.Decision, &p.DecidedAt); err != nil {
		return BrowserPermission{}, err
	}
	return p, nil
}
