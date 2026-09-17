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

// BrowserPermission is one row: a decision the shell reported, or a standing
// the user saved. Only a standing is policy (the Site settings dialog's
// per-kind choice and the Ask prompt's "Always allow"); a one-off decision
// is the list's history and is never handed back to the shell.
type BrowserPermission struct {
	ID        int64  `json:"id"`
	Origin    string `json:"origin"`
	Kind      string `json:"kind"`
	Decision  string `json:"decision"`
	Standing  bool   `json:"standing"`
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

// NormalizePermissionDecision accepts the words a row can hold: the two
// standings a site can have, and the "ask" a per-kind policy offers (the
// platform's own default is the absence of a row).
func NormalizePermissionDecision(decision string) (string, error) {
	d := strings.ToLower(strings.TrimSpace(decision))
	if d != "allow" && d != "deny" && d != "ask" {
		return "", fmt.Errorf("store: %q is not a permission decision", decision)
	}
	return d, nil
}

// SetBrowserPermission records one row for a (origin, kind) pair; a repeat
// changes it in place rather than adding a second row. `standing` marks the
// row as policy — and a later one-off decision on the same pair must not
// demote it (the shell reports every decision, including ones the saved
// standing itself produced), hence the MAX in the upsert.
func (s *Store) SetBrowserPermission(origin, kind, decision string, standing bool) (BrowserPermission, error) {
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
		INSERT INTO browser_permissions (origin, kind, decision, standing, decided_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(origin, kind) DO UPDATE SET
			decision = excluded.decision,
			standing = MAX(browser_permissions.standing, excluded.standing),
			decided_at = excluded.decided_at`,
		o, k, d, standing, when); err != nil {
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
		SELECT id, origin, kind, decision, standing, decided_at
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
		SELECT id, origin, kind, decision, standing, decided_at
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

// PruneBrowserPermissions forgets the site entries whose host has not been
// visited since `since` — the dialog's cleanup for sites the human no longer
// uses. Decision table, one row per condition:
//
//	the entry                    | the site's visits          | result
//	-----------------------------|----------------------------|--------
//	the every-site policy ("*")  | anything                   | kept (it is policy, not a site)
//	a site, visited since        | inside the window          | kept
//	a site, last visit before    | only older than the window | forgotten
//	a site, never visited        | none recorded              | forgotten
//	nothing matches              | —                          | nothing forgotten, no event
//
// One mutation, one event (ADR-0048). The forgotten entries are returned so
// the caller can also clear the live shell's copy — a standing that survived
// only in the shell would come back on the next report.
func (s *Store) PruneBrowserPermissions(since time.Time) ([]BrowserPermission, error) {
	rows, err := s.db.Query(`SELECT id, origin, kind, decision, standing, decided_at
		FROM browser_permissions WHERE origin <> '*'`)
	if err != nil {
		return nil, fmt.Errorf("store: list browser permissions: %w", err)
	}
	var all []BrowserPermission
	for rows.Next() {
		p, err := scanBrowserPermission(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		all = append(all, p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	stale := make([]BrowserPermission, 0, len(all))
	for _, p := range all {
		host := browserVisitHost(p.Origin)
		if host == "" {
			continue
		}
		var last sql.NullString
		if err := s.db.QueryRow(`SELECT MAX(visited_at) FROM browser_history WHERE host = ?`, host).Scan(&last); err != nil {
			return nil, fmt.Errorf("store: last visit for %s: %w", host, err)
		}
		if !last.Valid {
			stale = append(stale, p)
			continue
		}
		when, perr := time.Parse(time.RFC3339Nano, last.String)
		if perr != nil || when.Before(since) {
			stale = append(stale, p)
		}
	}
	if len(stale) == 0 {
		return nil, nil
	}
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("store: prune browser permissions: %w", err)
	}
	for _, p := range stale {
		if _, err := tx.Exec(`DELETE FROM browser_permissions WHERE id = ?`, p.ID); err != nil {
			_ = tx.Rollback()
			return nil, fmt.Errorf("store: prune browser permissions: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("store: prune browser permissions: %w", err)
	}
	s.note("browserpermission.updated", nil, nil, nil)
	return stale, nil
}

func scanBrowserPermission(row rowScanner) (BrowserPermission, error) {
	var p BrowserPermission
	if err := row.Scan(&p.ID, &p.Origin, &p.Kind, &p.Decision, &p.Standing, &p.DecidedAt); err != nil {
		return BrowserPermission{}, err
	}
	return p, nil
}
