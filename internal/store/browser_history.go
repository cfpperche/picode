package store

import (
	"database/sql"
	"fmt"
	"strings"
)

// The work browser's history (desktop browser plan, slice 3): the app
// records one visit per navigation it sees; the address-bar dropdown and
// Settings ▸ Browser read it. A re-reported URL updates the newest row in
// place instead of piling up rows while a page sits open. Cap: the oldest
// rows fall off past browserHistoryCap.

const (
	browserHistoryCap        = 5000
	browserHistoryMaxURL     = 2048
	browserHistoryMaxTitle   = 300
	browserHistoryMaxResults = 500
)

// BrowserVisit is one recorded page visit.
type BrowserVisit struct {
	ID        int64  `json:"id"`
	URL       string `json:"url"`
	Title     string `json:"title"`
	Host      string `json:"host"`
	Typed     bool   `json:"typed"`
	VisitedAt string `json:"visitedAt"`
}

func browserVisitHost(raw string) string {
	h := strings.TrimSpace(raw)
	h = strings.TrimPrefix(h, "http://")
	h = strings.TrimPrefix(h, "https://")
	if i := strings.IndexAny(h, "/?#"); i >= 0 {
		h = h[:i]
	}
	if i := strings.LastIndex(h, ":"); i >= 0 && !strings.Contains(h, "]") {
		h = h[:i]
	}
	return strings.ToLower(h)
}

// AddBrowserVisit records a page visit. When the newest row already names
// the same URL, it is updated in place (title, typed flag, timestamp) — a
// page that sits open must not grow the table. Every mutation announces
// browserhistory.updated.
func (s *Store) AddBrowserVisit(url, title string, typed bool) (BrowserVisit, error) {
	url = strings.TrimSpace(url)
	title = strings.TrimSpace(title)
	if url == "" || len(url) > browserHistoryMaxURL {
		return BrowserVisit{}, fmt.Errorf("store: add browser visit: url is required (max %d bytes)", browserHistoryMaxURL)
	}
	if len(title) > browserHistoryMaxTitle {
		title = title[:browserHistoryMaxTitle]
	}
	host := browserVisitHost(url)
	visited := nowUTC()

	var (
		id   int64
		last string
	)
	err := s.db.QueryRow(`SELECT id, url FROM browser_history ORDER BY id DESC LIMIT 1`).Scan(&id, &last)
	switch {
	case err == sql.ErrNoRows:
		// first row: fall through to insert
	case err != nil:
		return BrowserVisit{}, fmt.Errorf("store: add browser visit: %w", err)
	case strings.EqualFold(last, url): // same page re-reported (host case drifts): update in place
		if _, err := s.db.Exec(`UPDATE browser_history SET title = ?, typed = CASE WHEN ? THEN 1 ELSE typed END, visited_at = ? WHERE id = ?`,
			title, typed, visited, id); err != nil {
			return BrowserVisit{}, fmt.Errorf("store: update browser visit: %w", err)
		}
		s.note("browserhistory.updated", nil, nil, map[string]string{"url": url})
		return BrowserVisit{ID: id, URL: url, Title: title, Host: host, Typed: typed, VisitedAt: visited}, nil
	}

	res, err := s.db.Exec(`INSERT INTO browser_history (url, title, host, typed, visited_at) VALUES (?, ?, ?, ?, ?)`,
		url, title, host, typed, visited)
	if err != nil {
		return BrowserVisit{}, fmt.Errorf("store: add browser visit: %w", err)
	}
	id, err = res.LastInsertId()
	if err != nil {
		return BrowserVisit{}, fmt.Errorf("store: add browser visit: %w", err)
	}
	if _, err := s.db.Exec(`DELETE FROM browser_history WHERE id NOT IN (
		SELECT id FROM browser_history ORDER BY id DESC LIMIT ?)`, browserHistoryCap); err != nil {
		return BrowserVisit{}, fmt.Errorf("store: prune browser history: %w", err)
	}
	s.note("browserhistory.updated", nil, nil, map[string]string{"url": url})
	return BrowserVisit{ID: id, URL: url, Title: title, Host: host, Typed: typed, VisitedAt: visited}, nil
}

// ListBrowserHistory returns the newest visits first, optionally filtered
// by a substring of url/title/host.
func (s *Store) ListBrowserHistory(limit int, query string) ([]BrowserVisit, error) {
	if limit <= 0 || limit > browserHistoryMaxResults {
		limit = 100
	}
	q := strings.TrimSpace(query)
	rows, err := s.db.Query(`SELECT id, url, title, host, typed, visited_at FROM browser_history
		WHERE (? = '' OR url LIKE '%' || ? || '%' OR title LIKE '%' || ? || '%' OR host LIKE '%' || ? || '%')
		ORDER BY visited_at DESC, id DESC LIMIT ?`, q, q, q, q, limit)
	if err != nil {
		return nil, fmt.Errorf("store: list browser history: %w", err)
	}
	defer rows.Close()
	out := []BrowserVisit{}
	for rows.Next() {
		var v BrowserVisit
		var typed int
		if err := rows.Scan(&v.ID, &v.URL, &v.Title, &v.Host, &typed, &v.VisitedAt); err != nil {
			return nil, fmt.Errorf("store: list browser history: %w", err)
		}
		v.Typed = typed == 1
		out = append(out, v)
	}
	return out, rows.Err()
}

// DeleteBrowserVisit removes one visit.
func (s *Store) DeleteBrowserVisit(id int64) error {
	res, err := s.db.Exec(`DELETE FROM browser_history WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("store: delete browser visit: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	s.note("browserhistory.updated", nil, nil, nil)
	return nil
}

// DeleteBrowserVisits removes several visits at once — the history
// dialog's selection. Ids that are not there are not an error: the list
// the dialog saw may already be stale. Returns how many rows went.
func (s *Store) DeleteBrowserVisits(ids []int64) (int64, error) {
	marks := make([]string, 0, len(ids))
	args := make([]any, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		marks = append(marks, "?")
		args = append(args, id)
	}
	if len(marks) == 0 {
		return 0, nil
	}
	res, err := s.db.Exec(`DELETE FROM browser_history WHERE id IN (`+strings.Join(marks, ",")+`)`, args...)
	if err != nil {
		return 0, fmt.Errorf("store: delete browser visits: %w", err)
	}
	n, _ := res.RowsAffected()
	if n > 0 {
		s.note("browserhistory.updated", nil, nil, nil)
	}
	return n, nil
}

// ClearBrowserHistory removes every visit.
func (s *Store) ClearBrowserHistory() error {
	if _, err := s.db.Exec(`DELETE FROM browser_history`); err != nil {
		return fmt.Errorf("store: clear browser history: %w", err)
	}
	s.note("browserhistory.updated", nil, nil, nil)
	return nil
}

// ClearBrowserHistorySince deletes the visits at or after one instant, in
// RFC3339 — the time range the Clear browsing data dialog picked. An empty
// since clears everything, the same as ClearBrowserHistory.
func (s *Store) ClearBrowserHistorySince(since string) error {
	if since == "" {
		return s.ClearBrowserHistory()
	}
	if _, err := s.db.Exec(`DELETE FROM browser_history WHERE visited_at >= ?`, since); err != nil {
		return fmt.Errorf("store: clear browser history since: %w", err)
	}
	s.note("browserhistory.updated", nil, nil, nil)
	return nil
}
