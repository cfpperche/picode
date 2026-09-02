package store

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
)

// Terminal is a first-class shell (ADR-0017): a tmux session, not an agent.
// It lives in a workspace, or in ws_free when it belongs to nobody (ADR-0026).
type Terminal struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Cwd         string `json:"cwd"`
	WorkspaceID string `json:"workspaceId"`
	CreatedAt   string `json:"createdAt"`
}

const terminalCols = `id, name, cwd, workspace_id, created_at`

func scanTerminal(row interface{ Scan(...any) error }, t *Terminal) error {
	return row.Scan(&t.ID, &t.Name, &t.Cwd, &t.WorkspaceID, &t.CreatedAt)
}

// ListTerminals returns every terminal regardless of workspace. Settings
// scoping (scopeBySession / ownSessions) relies on that — a filtered list
// would silently stop tmux overrides from applying to workspace terminals.
func (s *Store) ListTerminals() ([]Terminal, error) {
	rows, err := s.db.Query(`SELECT ` + terminalCols + ` FROM terminals ORDER BY name COLLATE NOCASE, id`)
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

func (s *Store) ListWorkspaceTerminals(workspaceID string) ([]Terminal, error) {
	rows, err := s.db.Query(`SELECT `+terminalCols+` FROM terminals WHERE workspace_id = ? ORDER BY name COLLATE NOCASE, id`, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("store: list workspace terminals: %w", err)
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
	row := s.db.QueryRow(`SELECT `+terminalCols+` FROM terminals WHERE id = ?`, id)
	var t Terminal
	if err := scanTerminal(row, &t); err != nil {
		if err == sql.ErrNoRows {
			return Terminal{}, ErrNotFound
		}
		return Terminal{}, fmt.Errorf("store: get terminal: %w", err)
	}
	return t, nil
}

// CreateTerminal keeps the old shape: a free terminal.
func (s *Store) CreateTerminal(name, cwd string) (Terminal, error) {
	return s.CreateTerminalIn(FreeWorkspaceID, name, cwd)
}

// CreateTerminalIn creates a terminal owned by the given workspace (ADR-0026).
// A blank workspace means free. A workspace terminal with no cwd starts in
// the workspace folder; the $HOME default belongs to free terminals only.
func (s *Store) CreateTerminalIn(workspaceID, name, cwd string) (Terminal, error) {
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		workspaceID = FreeWorkspaceID
	}
	cwd = strings.TrimSpace(cwd)
	if workspaceID != FreeWorkspaceID {
		wk, err := s.GetWorkspace(workspaceID)
		if errors.Is(err, ErrNotFound) {
			return Terminal{}, fmt.Errorf("that workspace doesn't exist")
		}
		if err != nil {
			return Terminal{}, err
		}
		if cwd == "" {
			cwd = wk.Path
		}
	}
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
	if _, err := s.db.Exec(`INSERT INTO terminals (id, name, cwd, workspace_id, created_at) VALUES (?, ?, ?, ?, ?)`,
		id, name, cwd, workspaceID, now); err != nil {
		return Terminal{}, fmt.Errorf("store: insert terminal: %w", err)
	}
	t := Terminal{ID: id, Name: name, Cwd: cwd, WorkspaceID: workspaceID, CreatedAt: now}
	s.note("terminal.created", nil, nil, t) // terminals have no workspace FK (ADR-0026)
	return t, nil
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
	t, err := s.GetTerminal(id)
	if err != nil {
		return Terminal{}, err
	}
	s.note("terminal.updated", nil, nil, t)
	return t, nil
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
	s.note("terminal.deleted", nil, nil, idData(id))
	// The overrides are keyed by this id and nothing else refers to them, so
	// they go with it. Ids are never reused, so a row left behind would only
	// ever be dead weight — but dead weight that a future flag would read.
	return s.DeleteTerminalSettings(id)
}
