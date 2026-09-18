package store

// The browser annotations (v2c): one row per thing the human pointed at in the
// work browser, with the comment they wrote next to it. The row is a pointer —
// the DOM, the computed CSS and the crop are text here, the image lives as a
// file in the terminal's drop folder (never bytes in SQLite, the 2026-09-06
// attach study's rule). Mutations announce browserannotation.updated
// (ADR-0048).

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// BrowserAnnotation is one annotation: where it was made, what was pointed at,
// what the human said, and the two files staged for the agent.
type BrowserAnnotation struct {
	ID          int64  `json:"id"`
	TerminalID  string `json:"terminalId"`
	WorkspaceID string `json:"workspaceId"`
	URL         string `json:"url"`
	Title       string `json:"title"`
	Host        string `json:"host"`
	Selector    string `json:"selector"`
	Comment     string `json:"comment"`
	DOM         string `json:"dom"`
	CSS         string `json:"css"`
	Shot        string `json:"shot"` // file name inside the drop folder
	Note        string `json:"note"` // file name of the markdown note
	CreatedAt   string `json:"createdAt"`
}

const browserAnnotationCols = `id, terminal_id, workspace_id, url, title, host, selector,
	comment, dom, css, shot, note, created_at`

// CreateBrowserAnnotation records one annotation. Decision table:
//
//	url            | something to look at (selector, dom or shot) | comment | result
//	---------------|---------------------------------------------|---------|--------
//	set            | set                                         | any     | stored
//	set            | none of the three                           | any     | refused
//	empty          | set                                         | any     | refused
//	empty          | none of the three                           | any     | refused
//
// The comment is deliberately *not* required: the element is context, and the
// sentence may arrive later through the prompt door's caption (the ADR).
func (s *Store) CreateBrowserAnnotation(a BrowserAnnotation) (BrowserAnnotation, error) {
	url := strings.TrimSpace(a.URL)
	selector := strings.TrimSpace(a.Selector)
	dom := strings.TrimSpace(a.DOM)
	shot := strings.TrimSpace(a.Shot)
	if url == "" {
		return BrowserAnnotation{}, fmt.Errorf("store: annotation needs a page url")
	}
	if selector == "" && dom == "" && shot == "" {
		return BrowserAnnotation{}, fmt.Errorf("store: annotation needs an element, its html or a screenshot")
	}
	host := browserVisitHost(url)
	created := time.Now().UTC().Format(time.RFC3339Nano)
	res, err := s.db.Exec(`INSERT INTO browser_annotations
		(terminal_id, workspace_id, url, title, host, selector, comment, dom, css, shot, note, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		a.TerminalID, a.WorkspaceID, url, a.Title, host, selector, a.Comment, dom, a.CSS, shot, a.Note, created)
	if err != nil {
		return BrowserAnnotation{}, fmt.Errorf("store: create browser annotation: %w", err)
	}
	id, _ := res.LastInsertId()
	s.note("browserannotation.updated", nil, nil, map[string]string{"url": url})
	return BrowserAnnotation{
		ID: id, TerminalID: a.TerminalID, WorkspaceID: a.WorkspaceID, URL: url, Title: a.Title,
		Host: host, Selector: selector, Comment: a.Comment, DOM: dom, CSS: a.CSS,
		Shot: shot, Note: a.Note, CreatedAt: created,
	}, nil
}

// ListBrowserAnnotations returns the newest first, optionally filtered by a
// substring of the page address, title, comment or selector.
func (s *Store) ListBrowserAnnotations(limit int, query string) ([]BrowserAnnotation, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	q := strings.TrimSpace(query)
	rows, err := s.db.Query(`SELECT `+browserAnnotationCols+` FROM browser_annotations
		WHERE (? = '' OR url LIKE '%' || ? || '%' OR title LIKE '%' || ? || '%'
			OR comment LIKE '%' || ? || '%' OR selector LIKE '%' || ? || '%')
		ORDER BY id DESC LIMIT ?`, q, q, q, q, q, limit)
	if err != nil {
		return nil, fmt.Errorf("store: list browser annotations: %w", err)
	}
	defer rows.Close()
	out := []BrowserAnnotation{}
	for rows.Next() {
		var a BrowserAnnotation
		if err := rows.Scan(&a.ID, &a.TerminalID, &a.WorkspaceID, &a.URL, &a.Title, &a.Host,
			&a.Selector, &a.Comment, &a.DOM, &a.CSS, &a.Shot, &a.Note, &a.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// DeleteBrowserAnnotation forgets the row. The staged files stay on purpose:
// the agent may still be reading them, and the drop folder's own sweep is what
// removes old ones.
func (s *Store) DeleteBrowserAnnotation(id int64) error {
	res, err := s.db.Exec(`DELETE FROM browser_annotations WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("store: delete browser annotation: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	s.note("browserannotation.updated", nil, nil, nil)
	return nil
}
