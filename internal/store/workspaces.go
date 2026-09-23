package store

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
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

// AgentCwd is the directory pi should start in. Unbound agents never use $HOME.
func AgentCwd(w Workspace, a Agent) string {
	if a.WorkPath != nil && strings.TrimSpace(*a.WorkPath) != "" {
		return *a.WorkPath
	}
	if IsFree(w) {
		h, err := os.UserHomeDir()
		if err != nil {
			return "."
		}
		return filepath.Join(h, ".picode", "work", a.ID)
	}
	return w.Path
}

// AddWorkspace registers a folder (idempotent by absolute path). The
// workspace starts empty (ADR-0027): agents are added explicitly, and
// re-adding a registered path does not resurrect a deleted agent.
func (s *Store) AddWorkspace(name, path string) (Workspace, error) {
	name = stringsTrimSpace(name)
	if name == "" {
		return Workspace{}, fmt.Errorf("store: name is required")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return Workspace{}, fmt.Errorf("store: path: %w", err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return Workspace{}, fmt.Errorf("store: path %s: %w", abs, err)
	}
	if !info.IsDir() {
		return Workspace{}, fmt.Errorf("store: path %s: not a directory", abs)
	}

	tx, err := s.db.Begin()
	if err != nil {
		return Workspace{}, err
	}
	defer func() { s.rollback(tx) }()

	// Idempotent by path.
	var existing Workspace
	err = tx.QueryRow(`SELECT id, name, path, created_at FROM workspaces WHERE path = ?`, abs).
		Scan(&existing.ID, &existing.Name, &existing.Path, &existing.CreatedAt)
	if err == nil {
		if err := s.commit(tx); err != nil {
			return Workspace{}, err
		}
		return existing, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return Workspace{}, fmt.Errorf("store: lookup: %w", err)
	}

	w := Workspace{ID: newID(name, "workspace"), Name: name, Path: abs, CreatedAt: nowUTC()}
	if _, err := tx.Exec(`INSERT INTO workspaces (id, name, path, created_at, position) VALUES (?, ?, ?, ?, `+nextPositionExpr("workspaces", "id != ?")+`)`,
		w.ID, w.Name, w.Path, w.CreatedAt, FreeWorkspaceID); err != nil {
		return Workspace{}, fmt.Errorf("store: insert workspace: %w", err)
	}
	if err := s.AppendEventTx(tx, "workspace.added", nil, &w.ID, w); err != nil {
		return Workspace{}, err
	}
	if err := s.commit(tx); err != nil {
		return Workspace{}, err
	}
	return w, nil
}

// ListWorkspaces returns every workspace except the free one, in sidebar
// order (ADR-0173). New workspaces append; a reorder rewrites the positions.
func (s *Store) ListWorkspaces() ([]Workspace, error) {
	rows, err := s.db.Query(`SELECT id, name, path, created_at FROM workspaces WHERE id != ? ORDER BY position, id`, FreeWorkspaceID)
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

// RenameWorkspace changes the name the sidebar shows; the folder stays where
// it is. The free workspace has no name to change.
func (s *Store) RenameWorkspace(id, name string) (Workspace, error) {
	name = stringsTrimSpace(name)
	if name == "" {
		return Workspace{}, fmt.Errorf("store: name is required")
	}
	if len(name) > 120 {
		return Workspace{}, fmt.Errorf("store: name is too long")
	}
	if id == FreeWorkspaceID {
		return Workspace{}, ErrNotFound
	}
	tx, err := s.db.Begin()
	if err != nil {
		return Workspace{}, err
	}
	defer s.rollback(tx)
	res, err := tx.Exec(`UPDATE workspaces SET name = ? WHERE id = ?`, name, id)
	if err != nil {
		return Workspace{}, fmt.Errorf("store: rename workspace: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return Workspace{}, ErrNotFound
	}
	var w Workspace
	if err := tx.QueryRow(`SELECT id, name, path, created_at FROM workspaces WHERE id = ?`, id).
		Scan(&w.ID, &w.Name, &w.Path, &w.CreatedAt); err != nil {
		return Workspace{}, err
	}
	if err := s.AppendEventTx(tx, "workspace.updated", nil, &w.ID, w); err != nil {
		return Workspace{}, err
	}
	if err := s.commit(tx); err != nil {
		return Workspace{}, err
	}
	return w, nil
}

// RemoveWorkspace deletes a workspace, (via cascade) its agents, tasks and
// events, and its terminals with their settings overrides — the terminals
// table has no FK (ADR-0026), so that cascade is spelled out here. The
// project folder on disk is untouched; killing the tmux sessions is the
// server's job before this runs.
func (s *Store) RemoveWorkspace(id string) (removed bool, err error) {
	removed, _, err = s.removeWorkspace(id, nil)
	return removed, err
}

// RemoveWorkspaceWithExits is a person's removal of a workspace: each of
// its agents ends too, and each gets an exit record in the same
// transaction (ADR-0194). The dialog asked about the workspace, not each
// agent, so the exits are unasked, with the skip "workspace".
func (s *Store) RemoveWorkspaceWithExits(id string, in ExitInput) (removed bool, exits []AgentExit, err error) {
	in.Asked, in.AskSkip, in.Label = false, ExitSkipWorkspace, ExitLabel{}
	return s.removeWorkspace(id, &in)
}

func (s *Store) removeWorkspace(id string, in *ExitInput) (removed bool, exits []AgentExit, err error) {
	if in != nil {
		// Read before the transaction: the store has one connection.
		agents, err := s.ListAgents(id)
		if err != nil {
			return false, nil, err
		}
		now := time.Now()
		for _, a := range agents {
			ex, err := s.buildExit(a, *in, now)
			if err != nil {
				return false, nil, err
			}
			exits = append(exits, ex)
		}
	}
	tx, err := s.db.Begin()
	if err != nil {
		return false, nil, fmt.Errorf("store: remove workspace: %w", err)
	}
	defer s.rollback(tx)
	for _, ex := range exits {
		if err := insertExitTx(tx, ex); err != nil {
			return false, nil, err
		}
		if err := s.AppendEventTx(tx, "agent_exit.recorded", nil, nil, ex); err != nil {
			return false, nil, err
		}
	}
	if _, err := tx.Exec(`DELETE FROM terminal_settings WHERE scope IN (SELECT id FROM terminals WHERE workspace_id = ?)`, id); err != nil {
		return false, nil, fmt.Errorf("store: remove workspace terminal settings: %w", err)
	}
	if _, err := tx.Exec(`DELETE FROM terminals WHERE workspace_id = ?`, id); err != nil {
		return false, nil, fmt.Errorf("store: remove workspace terminals: %w", err)
	}
	// Its integration declaration (ADR-0182) is keyed by the id with no FK;
	// left behind it would outlive the workspace it describes.
	if _, err := tx.Exec(`DELETE FROM delivery_integration WHERE scope = ?`, id); err != nil {
		return false, nil, fmt.Errorf("store: remove workspace integration: %w", err)
	}
	res, err := tx.Exec(`DELETE FROM workspaces WHERE id = ?`, id)
	if err != nil {
		return false, nil, fmt.Errorf("store: remove workspace: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, nil, err
	}
	if n == 0 && len(exits) > 0 {
		// Another removal won the race: exits for a workspace that was no
		// longer there must not land (the deferred rollback drops them).
		return false, nil, nil
	}
	if n > 0 {
		if err := s.AppendEventTx(tx, "workspace.deleted", nil, nil, idData(id)); err != nil {
			return false, nil, err
		}
	}
	if err := s.commit(tx); err != nil {
		return false, nil, fmt.Errorf("store: remove workspace: %w", err)
	}
	if n == 0 {
		return false, nil, nil
	}
	return true, exits, nil
}
