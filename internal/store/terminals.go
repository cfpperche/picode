package store

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
)

// Terminal is a first-class shell (ADR-0017): a tmux session, not an agent.
type Terminal struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Cwd       string `json:"cwd"`
	CreatedAt string `json:"createdAt"`
}

func scanTerminal(row interface{ Scan(...any) error }, t *Terminal) error {
	return row.Scan(&t.ID, &t.Name, &t.Cwd, &t.CreatedAt)
}

func (s *Store) ListTerminals() ([]Terminal, error) {
	rows, err := s.db.Query(`SELECT id, name, cwd, created_at FROM terminals ORDER BY name COLLATE NOCASE, id`)
	if err != nil {
		return nil, fmt.Errorf("store: list terminals: %w", err)
	}
	defer rows.Close()
	out := []Terminal{}
	for rows.Next() {
		var t Terminal
		if err := scanTerminal(rows, &t); err != nil {
			return nil, fmt.Errorf("store: scan terminal: %w", err)
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Store) GetTerminal(id string) (Terminal, error) {
	row := s.db.QueryRow(`SELECT id, name, cwd, created_at FROM terminals WHERE id = ?`, id)
	var t Terminal
	if err := scanTerminal(row, &t); err != nil {
		if err == sql.ErrNoRows {
			return Terminal{}, ErrNotFound
		}
		return Terminal{}, fmt.Errorf("store: get terminal: %w", err)
	}
	return t, nil
}

func (s *Store) CreateTerminal(name, cwd string) (Terminal, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		n, err := s.CountTerminals()
		if err != nil {
			return Terminal{}, err
		}
		name = "Terminal"
		if n > 0 {
			name = fmt.Sprintf("Terminal %d", n+1)
		}
	}
	cwd = strings.TrimSpace(cwd)
	if cwd == "" {
		h, err := os.UserHomeDir()
		if err != nil {
			return Terminal{}, fmt.Errorf("store: home: %w", err)
		}
		cwd = h
	}
	st, err := os.Stat(cwd)
	if err != nil || !st.IsDir() {
		return Terminal{}, fmt.Errorf("that folder doesn't exist")
	}
	id := newID(name, "term")
	now := nowUTC()
	if _, err := s.db.Exec(`INSERT INTO terminals (id, name, cwd, created_at) VALUES (?, ?, ?, ?)`,
		id, name, cwd, now); err != nil {
		return Terminal{}, fmt.Errorf("store: insert terminal: %w", err)
	}
	return Terminal{ID: id, Name: name, Cwd: cwd, CreatedAt: now}, nil
}

func (s *Store) CountTerminals() (int, error) {
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(1) FROM terminals`).Scan(&n); err != nil {
		return 0, fmt.Errorf("store: count terminals: %w", err)
	}
	return n, nil
}

func (s *Store) RenameTerminal(id, name string) (Terminal, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Terminal{}, fmt.Errorf("name is required")
	}
	if len(name) > 80 {
		name = name[:80]
	}
	res, err := s.db.Exec(`UPDATE terminals SET name = ? WHERE id = ?`, name, id)
	if err != nil {
		return Terminal{}, fmt.Errorf("store: rename terminal: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return Terminal{}, ErrNotFound
	}
	return s.GetTerminal(id)
}

func (s *Store) DeleteTerminal(id string) error {
	res, err := s.db.Exec(`DELETE FROM terminals WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("store: delete terminal: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
