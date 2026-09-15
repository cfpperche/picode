package store

// The work browser's download list (slice 3.3d): the shell reports each
// download as it starts and again when it lands or breaks, and the settings
// dialog reads, searches and prunes the list. Mutations announce
// browserdownload.updated, so open readers refetch (ADR-0048).

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// BrowserDownload is one file the built-in browser fetched.
type BrowserDownload struct {
	ID        int64  `json:"id"`
	URL       string `json:"url"`
	Name      string `json:"name"`
	Path      string `json:"path"`
	Host      string `json:"host"`
	Total     int64  `json:"total"`
	Received  int64  `json:"received"`
	Status    string `json:"status"`
	StartedAt string `json:"startedAt"`
}

// AddBrowserDownload records a download as it starts. Re-downloading onto the
// same path updates that row instead of adding a second one for the same file.
func (s *Store) AddBrowserDownload(url, path string, total int64) (BrowserDownload, error) {
	url = strings.TrimSpace(url)
	if url == "" {
		return BrowserDownload{}, fmt.Errorf("store: a download needs a URL")
	}
	name := path
	if i := strings.LastIndexAny(path, `\/`); i >= 0 {
		name = path[i+1:]
	}
	if name == "" {
		name = "download"
	}
	when := time.Now().UTC().Format(time.RFC3339Nano)
	host := browserVisitHost(url)
	if _, err := s.db.Exec(`
		INSERT INTO browser_downloads (url, name, path, host, total, received, status, started_at)
		VALUES (?, ?, ?, ?, ?, 0, 'started', ?)
		ON CONFLICT(path) WHERE path != '' DO UPDATE SET
			url = excluded.url, name = excluded.name, host = excluded.host,
			total = excluded.total, received = 0, status = 'started',
			started_at = excluded.started_at`,
		url, name, path, host, total, when); err != nil {
		return BrowserDownload{}, fmt.Errorf("store: add browser download: %w", err)
	}
	row, err := s.browserDownloadByPath(path)
	if err != nil {
		return BrowserDownload{}, err
	}
	s.note("browserdownload.updated", nil, nil, nil)
	return row, nil
}

// FinishBrowserDownload writes the outcome the shell reported: completed or
// interrupted, with how many bytes arrived. A path nobody recorded is not an
// error — the shell may report a download that started before this list did.
func (s *Store) FinishBrowserDownload(path, status string, received int64) error {
	if path == "" {
		return nil
	}
	if status != "completed" && status != "interrupted" {
		return fmt.Errorf("store: %q is not a download outcome", status)
	}
	res, err := s.db.Exec(`UPDATE browser_downloads SET status = ?, received = ? WHERE path = ?`, status, received, path)
	if err != nil {
		return fmt.Errorf("store: finish browser download: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	s.note("browserdownload.updated", nil, nil, nil)
	return nil
}

func (s *Store) browserDownloadByPath(path string) (BrowserDownload, error) {
	row := s.db.QueryRow(`
		SELECT id, url, name, path, host, total, received, status, started_at
		FROM browser_downloads WHERE path = ?`, path)
	return scanBrowserDownload(row)
}

// ListBrowserDownloads returns the newest first; a query matches the file
// name, the URL or the host.
func (s *Store) ListBrowserDownloads(limit int, query string) ([]BrowserDownload, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	q := strings.TrimSpace(query)
	rows, err := s.db.Query(`
		SELECT id, url, name, path, host, total, received, status, started_at
		FROM browser_downloads
		WHERE (? = '' OR name LIKE '%' || ? || '%' OR url LIKE '%' || ? || '%' OR host LIKE '%' || ? || '%')
		ORDER BY started_at DESC, id DESC
		LIMIT ?`, q, q, q, q, limit)
	if err != nil {
		return nil, fmt.Errorf("store: list browser downloads: %w", err)
	}
	defer rows.Close()
	out := []BrowserDownload{}
	for rows.Next() {
		d, err := scanBrowserDownload(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// DeleteBrowserDownload forgets one row (the file on disk stays).
func (s *Store) DeleteBrowserDownload(id int64) error {
	res, err := s.db.Exec(`DELETE FROM browser_downloads WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("store: delete browser download: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	s.note("browserdownload.updated", nil, nil, nil)
	return nil
}

// ClearBrowserDownloads forgets every row.
func (s *Store) ClearBrowserDownloads() error {
	if _, err := s.db.Exec(`DELETE FROM browser_downloads`); err != nil {
		return fmt.Errorf("store: clear browser downloads: %w", err)
	}
	s.note("browserdownload.updated", nil, nil, nil)
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanBrowserDownload(row rowScanner) (BrowserDownload, error) {
	var d BrowserDownload
	if err := row.Scan(&d.ID, &d.URL, &d.Name, &d.Path, &d.Host, &d.Total, &d.Received, &d.Status, &d.StartedAt); err != nil {
		return BrowserDownload{}, err
	}
	return d, nil
}
