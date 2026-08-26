package store

import (
	"database/sql"
	"fmt"
	"strings"
)

const maxPinTags = 16
const maxPinTitle = 200
const maxPinBody = 100_000

// Pin is a flat machine-scoped note (no folder tree).
type Pin struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Tags      []string  `json:"tags"`
	Body      string    `json:"body"`
	CreatedAt string    `json:"createdAt"`
	UpdatedAt string    `json:"updatedAt"`
	FileCount int       `json:"fileCount"`
	Files     []PinFile `json:"files,omitempty"`
}

func scanPin(row interface{ Scan(...any) error }, p *Pin) error {
	var tags string
	if err := row.Scan(&p.ID, &p.Title, &tags, &p.Body, &p.CreatedAt, &p.UpdatedAt); err != nil {
		return err
	}
	p.Tags = decodePackages(tags)
	return nil
}

func scanPinList(row interface{ Scan(...any) error }, p *Pin) error {
	var tags string
	if err := row.Scan(&p.ID, &p.Title, &tags, &p.Body, &p.CreatedAt, &p.UpdatedAt, &p.FileCount); err != nil {
		return err
	}
	p.Tags = decodePackages(tags)
	return nil
}

func normalizePin(title string, tags []string, body string) (string, []string, string, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return "", nil, "", fmt.Errorf("title is required")
	}
	if len(title) > maxPinTitle {
		title = title[:maxPinTitle]
	}
	if len(body) > maxPinBody {
		body = body[:maxPinBody]
	}
	out := make([]string, 0, len(tags))
	seen := map[string]bool{}
	for _, t := range tags {
		t = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(t), "#")))
		t = strings.ReplaceAll(t, " ", "-")
		if t == "" || seen[t] {
			continue
		}
		seen[t] = true
		out = append(out, t)
		if len(out) >= maxPinTags {
			break
		}
	}
	return title, out, body, nil
}

func (s *Store) ListPins() ([]Pin, error) {
	rows, err := s.db.Query(`SELECT id, title, tags, body, created_at, updated_at,
		(SELECT COUNT(1) FROM pin_files f WHERE f.pin_id = pins.id) FROM pins ORDER BY updated_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("store: list pins: %w", err)
	}
	defer rows.Close()
	out := []Pin{}
	for rows.Next() {
		var p Pin
		if err := scanPinList(rows, &p); err != nil {
			return nil, fmt.Errorf("store: scan pin: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) GetPin(id string) (Pin, error) {
	var p Pin
	err := scanPin(s.db.QueryRow(`SELECT id, title, tags, body, created_at, updated_at FROM pins WHERE id = ?`, id), &p)
	if err == sql.ErrNoRows {
		return Pin{}, ErrNotFound
	}
	if err != nil {
		return Pin{}, fmt.Errorf("store: get pin: %w", err)
	}
	files, err := s.ListPinFiles(p.ID)
	if err != nil {
		return Pin{}, err
	}
	p.Files = files
	p.FileCount = len(files)
	return p, nil
}

func (s *Store) CreatePin(title string, tags []string, body string) (Pin, error) {
	title, tags, body, err := normalizePin(title, tags, body)
	if err != nil {
		return Pin{}, err
	}
	now := nowUTC()
	p := Pin{ID: newID(title, "pin"), Title: title, Tags: tags, Body: body, CreatedAt: now, UpdatedAt: now}
	if _, err := s.db.Exec(`INSERT INTO pins (id, title, tags, body, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		p.ID, p.Title, encodePackages(p.Tags), p.Body, p.CreatedAt, p.UpdatedAt); err != nil {
		return Pin{}, fmt.Errorf("store: create pin: %w", err)
	}
	return p, nil
}

func (s *Store) UpdatePin(id, title string, tags []string, body string) (Pin, error) {
	title, tags, body, err := normalizePin(title, tags, body)
	if err != nil {
		return Pin{}, err
	}
	now := nowUTC()
	res, err := s.db.Exec(`UPDATE pins SET title = ?, tags = ?, body = ?, updated_at = ? WHERE id = ?`,
		title, encodePackages(tags), body, now, id)
	if err != nil {
		return Pin{}, fmt.Errorf("store: update pin: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return Pin{}, ErrNotFound
	}
	return s.GetPin(id)
}

func (s *Store) DeletePin(id string) error {
	res, err := s.db.Exec(`DELETE FROM pins WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("store: delete pin: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
