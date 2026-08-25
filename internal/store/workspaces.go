package store

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Workspace is one registered project folder.
type Workspace struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Path      string `json:"path"`
	CreatedAt string `json:"createdAt"`
}

// ErrNotFound is returned when a row does not exist.
var ErrNotFound = errors.New("not found")

// Free workspace (ADR-0011): unbound agents. Hidden from ListWorkspaces.
const (
	FreeWorkspaceID   = "ws_free"
	FreeWorkspacePath = "__picode_free__"
)

// IsFree reports a reserved unbound workspace.
func IsFree(w Workspace) bool {
	return w.ID == FreeWorkspaceID || w.Path == FreeWorkspacePath
}

// AgentCwd is the directory pi should start in.
func AgentCwd(w Workspace) string {
	if IsFree(w) {
		h, err := os.UserHomeDir()
		if err != nil {
			return "."
		}
		return h
	}
	return w.Path
}

// AddWorkspace registers a folder (idempotent by absolute path) and ensures
// its default agent exists. Returns the workspace and its default agent.
func (s *Store) AddWorkspace(name, path string) (Workspace, Agent, error) {
	name = stringsTrimSpace(name)
	if name == "" {
		return Workspace{}, Agent{}, fmt.Errorf("store: name is required")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return Workspace{}, Agent{}, fmt.Errorf("store: path: %w", err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return Workspace{}, Agent{}, fmt.Errorf("store: path %s: %w", abs, err)
	}
	if !info.IsDir() {
		return Workspace{}, Agent{}, fmt.Errorf("store: path %s: not a directory", abs)
	}

	tx, err := s.db.Begin()
	if err != nil {
		return Workspace{}, Agent{}, err
	}
	defer func() { _ = tx.Rollback() }()

	// Idempotent by path.
	var existing Workspace
	err = tx.QueryRow(`SELECT id, name, path, created_at FROM workspaces WHERE path = ?`, abs).
		Scan(&existing.ID, &existing.Name, &existing.Path, &existing.CreatedAt)
	if err == nil {
		agent, err := ensureDefaultAgentTx(tx, existing.ID, existing.Name, existing.CreatedAt)
		if err != nil {
			return Workspace{}, Agent{}, err
		}
		if err := tx.Commit(); err != nil {
			return Workspace{}, Agent{}, err
		}
		return existing, agent, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return Workspace{}, Agent{}, fmt.Errorf("store: lookup: %w", err)
	}

	w := Workspace{ID: newID(name, "workspace"), Name: name, Path: abs, CreatedAt: nowUTC()}
	if _, err := tx.Exec(`INSERT INTO workspaces (id, name, path, created_at) VALUES (?, ?, ?, ?)`,
		w.ID, w.Name, w.Path, w.CreatedAt); err != nil {
		return Workspace{}, Agent{}, fmt.Errorf("store: insert workspace: %w", err)
	}
	agent, err := ensureDefaultAgentTx(tx, w.ID, w.Name, w.CreatedAt)
	if err != nil {
		return Workspace{}, Agent{}, err
	}
	if err := s.AppendEventTx(tx, "workspace_added", nil, &w.ID, nil); err != nil {
		return Workspace{}, Agent{}, err
	}
	if err := tx.Commit(); err != nil {
		return Workspace{}, Agent{}, err
	}
	return w, agent, nil
}

// ListWorkspaces returns all workspaces ordered by name.
func (s *Store) ListWorkspaces() ([]Workspace, error) {
	rows, err := s.db.Query(`SELECT id, name, path, created_at FROM workspaces WHERE id != ? ORDER BY name`, FreeWorkspaceID)
	if err != nil {
		return nil, fmt.Errorf("store: list workspaces: %w", err)
	}
	defer rows.Close()
	var out []Workspace
	for rows.Next() {
		var w Workspace
		if err := rows.Scan(&w.ID, &w.Name, &w.Path, &w.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

// GetWorkspace fetches one workspace by id.
func (s *Store) GetWorkspace(id string) (Workspace, error) {
	var w Workspace
	err := s.db.QueryRow(`SELECT id, name, path, created_at FROM workspaces WHERE id = ?`, id).
		Scan(&w.ID, &w.Name, &w.Path, &w.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Workspace{}, ErrNotFound
	}
	if err != nil {
		return Workspace{}, fmt.Errorf("store: get workspace: %w", err)
	}
	return w, nil
}

// RemoveWorkspace deletes a workspace and (via cascade) its agents, tasks
// and events. The project folder on disk is untouched.
func (s *Store) RemoveWorkspace(id string) (removed bool, err error) {
	res, err := s.db.Exec(`DELETE FROM workspaces WHERE id = ?`, id)
	if err != nil {
		return false, fmt.Errorf("store: remove workspace: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}
