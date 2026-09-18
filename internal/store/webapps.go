package store

import (
	"database/sql"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

const (
	maxWebAppName = 200
	maxWebAppURL  = 2048
	maxWebAppIcon = 262144
)

var webappScheme = map[string]bool{"http": true, "https": true}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// webappDefaultScheme picks the scheme for a scheme-less address: an IP
// literal or a loopback-ish name is a local service that speaks plain
// http; anything else is a public name that gets https.
func webappDefaultScheme(raw string) string {
	host := raw
	if i := strings.IndexAny(host, "/?#"); i >= 0 {
		host = host[:i]
	}
	if i := strings.LastIndex(host, ":"); i >= 0 && isDigits(host[i+1:]) {
		host = host[:i]
	}
	host = strings.Trim(host, "[]")
	h := strings.ToLower(host)
	if net.ParseIP(host) != nil || h == "localhost" || strings.HasSuffix(h, ".localhost") || strings.HasSuffix(h, ".local") {
		return "http"
	}
	return "https"
}

type DuplicateWebappError struct{ Existing Webapp }

func (e DuplicateWebappError) Error() string { return "a webapp for this URL already exists" }

type Webapp struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	URL        string `json:"url"`
	StartURL   string `json:"startUrl,omitempty"`
	Scope      string `json:"scope,omitempty"`
	Display    string `json:"display,omitempty"`
	ThemeColor string `json:"themeColor,omitempty"`
	HasIcon    bool   `json:"hasIcon"`
	CreatedAt  string `json:"createdAt"`
}

// WebappInput is what an install hands the store: the typed address plus
// whatever the PWA manifest proved (launch URL, scope, display, color).
type WebappInput struct {
	Name       string
	URL        string
	StartURL   string
	Scope      string
	Display    string
	ThemeColor string
	Icon       []byte
	IconMime   string
}

type webappRow struct {
	Webapp
	Icon     []byte
	IconMime string
}

const webappCols = `id, name, url, start_url, scope, display, theme_color, icon, icon_mime, created_at`

func scanWebappRow(row interface{ Scan(...any) error }) (webappRow, error) {
	var r webappRow
	var icon []byte
	if err := row.Scan(&r.ID, &r.Name, &r.URL, &r.StartURL, &r.Scope, &r.Display, &r.ThemeColor, &icon, &r.IconMime, &r.CreatedAt); err != nil {
		return webappRow{}, err
	}
	r.HasIcon = len(icon) > 0
	return r, nil
}

func NormalizeWebappURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", invalid("url is required")
	}
	if !strings.Contains(raw, "://") {
		raw = webappDefaultScheme(raw) + "://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", invalid("url is not a valid address")
	}
	if !webappScheme[u.Scheme] {
		return "", invalid("url must be http or https")
	}
	if u.Host == "" {
		return "", invalid("url needs a host, like example.com")
	}
	if u.User != nil {
		return "", invalid("url must not embed a username or password")
	}
	if strings.ContainsAny(u.Host, "/\\?#@") {
		return "", invalid("url host is malformed")
	}
	host := u.Hostname()
	if host == "" || strings.ContainsAny(host, "[]%") {
		return "", invalid("url host is malformed")
	}
	if strings.Contains(host, ":") {
		if net.ParseIP(host) == nil || !strings.HasPrefix(u.Host, "[") {
			return "", invalid("url host is malformed")
		}
	} else if strings.ContainsAny(u.Host, "[]") {
		return "", invalid("url host is malformed")
	}
	if strings.HasSuffix(u.Host, ":") {
		return "", invalid("url port is malformed")
	}
	if port := u.Port(); port != "" {
		n, err := strconv.Atoi(port)
		if err != nil || n < 1 || n > 65535 {
			return "", invalid("url port is malformed")
		}
	}
	u.Host = strings.ToLower(u.Host)
	if u.Path == "/" && u.RawQuery == "" && u.Fragment == "" {
		u.Path = ""
		u.RawPath = ""
	}
	canonical := u.String()
	if len(canonical) > maxWebAppURL {
		return "", invalid("url is too long (max %d bytes)", maxWebAppURL)
	}
	return canonical, nil
}

func normalizeWebappName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", invalid("name is required")
	}
	if len([]rune(name)) > maxWebAppName {
		return "", invalid("name is too long (max %d characters)", maxWebAppName)
	}
	return name, nil
}

func webappUnique(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "unique")
}

var webappDisplay = map[string]bool{"standalone": true, "fullscreen": true, "minimal-ui": true, "browser": true}

// normalizeWebappOptionalURL validates an optional manifest address; empty
// stays empty, anything malformed is refused rather than silently dropped.
func normalizeWebappOptionalURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}
	return NormalizeWebappURL(raw)
}

func normalizeWebappThemeColor(raw string) string {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if raw == "" {
		return ""
	}
	ok := false
	switch len(raw) {
	case 4, 5, 7, 9:
		_, ok = strings.CutPrefix(raw, "#")
	}
	if !ok {
		return ""
	}
	for _, r := range raw[1:] {
		if !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f') {
			return ""
		}
	}
	return raw
}

// normalizeWebappManifestFields validates the manifest-derived identity a
// refresh or an install carries; the name and the url are the user's and
// are handled by their own rules.
func normalizeWebappManifestFields(in WebappInput) (startURL, scope, display, themeColor string, err error) {
	startURL, err = normalizeWebappOptionalURL(in.StartURL)
	if err != nil {
		return "", "", "", "", err
	}
	scope, err = normalizeWebappOptionalURL(in.Scope)
	if err != nil {
		return "", "", "", "", err
	}
	display = strings.ToLower(strings.TrimSpace(in.Display))
	if display != "" && !webappDisplay[display] {
		return "", "", "", "", invalid("display must be standalone, fullscreen, minimal-ui or browser")
	}
	return startURL, scope, display, normalizeWebappThemeColor(in.ThemeColor), nil
}

func (s *Store) CreateWebapp(in WebappInput) (Webapp, error) {
	name, err := normalizeWebappName(in.Name)
	if err != nil {
		return Webapp{}, err
	}
	rawURL, err := NormalizeWebappURL(in.URL)
	if err != nil {
		return Webapp{}, err
	}
	startURL, scope, display, themeColor, err := normalizeWebappManifestFields(in)
	if err != nil {
		return Webapp{}, err
	}
	if len(in.Icon) > maxWebAppIcon {
		return Webapp{}, invalid("icon is too large (max %d KB)", maxWebAppIcon/1000)
	}
	now := nowUTC()
	id := newID(name, "webapp")
	tx, err := s.db.Begin()
	if err != nil {
		return Webapp{}, err
	}
	_, err = tx.Exec(`INSERT INTO webapps (id, name, url, start_url, scope, display, theme_color, icon, icon_mime, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, name, rawURL, startURL, scope, display, themeColor, in.Icon, in.IconMime, now)
	if webappUnique(err) {
		s.rollback(tx)
		existing, _ := s.GetWebappByURL(rawURL)
		return Webapp{}, DuplicateWebappError{Existing: existing.Webapp}
	}
	if err != nil {
		s.rollback(tx)
		return Webapp{}, fmt.Errorf("store: create webapp: %w", err)
	}
	created := Webapp{ID: id, Name: name, URL: rawURL, StartURL: startURL, Scope: scope, Display: display, ThemeColor: themeColor, HasIcon: len(in.Icon) > 0, CreatedAt: now}
	if err := s.AppendEventTx(tx, "webapp.installed", nil, nil, created); err != nil {
		s.rollback(tx)
		return Webapp{}, err
	}
	if err := s.commit(tx); err != nil {
		return Webapp{}, err
	}
	return created, nil
}

func (s *Store) UpdateWebappName(id, name string) (Webapp, error) {
	name, err := normalizeWebappName(name)
	if err != nil {
		return Webapp{}, err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return Webapp{}, err
	}
	res, err := tx.Exec(`UPDATE webapps SET name = ? WHERE id = ?`, name, id)
	if err != nil {
		s.rollback(tx)
		return Webapp{}, fmt.Errorf("store: rename webapp: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		s.rollback(tx)
		return Webapp{}, ErrNotFound
	}
	row, err := scanWebappRow(tx.QueryRow(`SELECT `+webappCols+` FROM webapps WHERE id = ?`, id))
	if err != nil {
		s.rollback(tx)
		return Webapp{}, fmt.Errorf("store: rename webapp: %w", err)
	}
	if err := s.AppendEventTx(tx, "webapp.updated", nil, nil, row.Webapp); err != nil {
		s.rollback(tx)
		return Webapp{}, err
	}
	if err := s.commit(tx); err != nil {
		return Webapp{}, err
	}
	return row.Webapp, nil
}

// UpdateWebappMetadata rewrites the manifest-derived identity and the
// icon of one webapp. The name and the url are the user's — a refresh
// never touches them. Empty icon bytes keep the stored icon (a favicon
// that failed to fetch this round is not a reason to forget the old one).
func (s *Store) UpdateWebappMetadata(id string, in WebappInput) (Webapp, error) {
	startURL, scope, display, themeColor, err := normalizeWebappManifestFields(in)
	if err != nil {
		return Webapp{}, err
	}
	if len(in.Icon) > maxWebAppIcon {
		return Webapp{}, invalid("icon is too large (max %d KB)", maxWebAppIcon/1000)
	}
	tx, err := s.db.Begin()
	if err != nil {
		return Webapp{}, err
	}
	defer s.rollback(tx)
	if len(in.Icon) == 0 {
		if _, err := tx.Exec(`UPDATE webapps SET start_url=?, scope=?, display=?, theme_color=? WHERE id=?`,
			startURL, scope, display, themeColor, id); err != nil {
			return Webapp{}, fmt.Errorf("store: refresh webapp: %w", err)
		}
	} else if _, err := tx.Exec(`UPDATE webapps SET start_url=?, scope=?, display=?, theme_color=?, icon=?, icon_mime=? WHERE id=?`,
		startURL, scope, display, themeColor, in.Icon, in.IconMime, id); err != nil {
		return Webapp{}, fmt.Errorf("store: refresh webapp: %w", err)
	}
	row, err := scanWebappRow(tx.QueryRow(`SELECT `+webappCols+` FROM webapps WHERE id = ?`, id))
	if err != nil {
		if err == sql.ErrNoRows {
			return Webapp{}, ErrNotFound
		}
		return Webapp{}, fmt.Errorf("store: refresh webapp: %w", err)
	}
	if err := s.AppendEventTx(tx, "webapp.updated", nil, nil, row.Webapp); err != nil {
		return Webapp{}, err
	}
	if err := s.commit(tx); err != nil {
		return Webapp{}, err
	}
	return row.Webapp, nil
}

func (s *Store) DeleteWebapp(id string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	res, err := tx.Exec(`DELETE FROM webapps WHERE id = ?`, id)
	if err != nil {
		s.rollback(tx)
		return fmt.Errorf("store: delete webapp: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		s.rollback(tx)
		return ErrNotFound
	}
	if err := s.AppendEventTx(tx, "webapp.removed", nil, nil, idData(id)); err != nil {
		s.rollback(tx)
		return err
	}
	return s.commit(tx)
}

func (s *Store) ListWebapps() ([]Webapp, error) {
	rows, err := s.db.Query(`SELECT ` + webappCols + ` FROM webapps ORDER BY created_at DESC, id DESC`)
	if err != nil {
		return nil, fmt.Errorf("store: list webapps: %w", err)
	}
	defer rows.Close()
	out := []Webapp{}
	for rows.Next() {
		r, err := scanWebappRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r.Webapp)
	}
	return out, rows.Err()
}

func (s *Store) GetWebapp(id string) (webappRow, error) {
	r, err := scanWebappRow(s.db.QueryRow(`SELECT `+webappCols+` FROM webapps WHERE id = ?`, id))
	if err == sql.ErrNoRows {
		return webappRow{}, ErrNotFound
	}
	if err != nil {
		return webappRow{}, fmt.Errorf("store: get webapp: %w", err)
	}
	return r, nil
}

func (s *Store) GetWebappByURL(rawURL string) (webappRow, error) {
	rawURL, err := NormalizeWebappURL(rawURL)
	if err != nil {
		return webappRow{}, err
	}
	r, err := scanWebappRow(s.db.QueryRow(`SELECT `+webappCols+` FROM webapps WHERE url = ?`, rawURL))
	if err == sql.ErrNoRows {
		return webappRow{}, ErrNotFound
	}
	if err != nil {
		return webappRow{}, fmt.Errorf("store: get webapp by url: %w", err)
	}
	return r, nil
}

func (s *Store) GetWebappIcon(id string) (icon []byte, mime string, err error) {
	err = s.db.QueryRow(`SELECT icon, icon_mime FROM webapps WHERE id = ?`, id).Scan(&icon, &mime)
	if err == sql.ErrNoRows {
		return nil, "", ErrNotFound
	}
	if err != nil {
		return nil, "", fmt.Errorf("store: get webapp icon: %w", err)
	}
	return icon, mime, nil
}
