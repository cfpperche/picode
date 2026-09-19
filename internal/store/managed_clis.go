package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/cfpperche/picode/internal/clilaunch"
	"github.com/cfpperche/picode/internal/grant"
)

// ManagedCLI binds an Agent CLI terminal to a workspace as a principal
// (ADR-0159). It is not an agents row: Runtime.Start never sees it.
type ManagedCLI struct {
	ID          string `json:"id"`
	WorkspaceID string `json:"workspaceId"`
	CLI         string `json:"cli"`
	TerminalID  string `json:"terminalId"`
	Name        string `json:"name"`
	CreatedAt   string `json:"createdAt"`
}

// Principal is the grant identity for this binding: always a terminal.
func (m ManagedCLI) Principal() grant.Principal {
	return grant.Principal{Kind: grant.KindTerminal, ID: m.TerminalID}
}

const managedCLICols = `id, workspace_id, cli, terminal_id, name, created_at`

func scanManagedCLI(row interface{ Scan(...any) error }, m *ManagedCLI) error {
	return row.Scan(&m.ID, &m.WorkspaceID, &m.CLI, &m.TerminalID, &m.Name, &m.CreatedAt)
}

var errManagedCLIBound = conflictError{"This terminal is already a managed principal."}

func managedCLIInvalid(msg string) error { return invalidError{msg} }

// AddManagedCLI binds an existing terminal in workspaceID to catalog CLI.
// It does not create an agent, a process, or a launch row.
func (s *Store) AddManagedCLI(workspaceID, cli, name, terminalID string) (ManagedCLI, error) {
	workspaceID = strings.TrimSpace(workspaceID)
	cli = strings.TrimSpace(cli)
	name = strings.TrimSpace(name)
	terminalID = strings.TrimSpace(terminalID)
	if workspaceID == "" || workspaceID == FreeWorkspaceID {
		return ManagedCLI{}, managedCLIInvalid("A managed CLI belongs to a workspace.")
	}
	if terminalID == "" {
		return ManagedCLI{}, managedCLIInvalid("A terminal is required.")
	}
	entry, ok := clilaunch.Find(cli)
	if !ok {
		return ManagedCLI{}, managedCLIInvalid("Unknown CLI.")
	}
	if !entry.Launchable() {
		return ManagedCLI{}, managedCLIInvalid("That CLI cannot run in a terminal.")
	}
	if name == "" {
		name = entry.Name
	}
	if _, err := s.GetWorkspace(workspaceID); err != nil {
		return ManagedCLI{}, err
	}
	tm, err := s.GetTerminal(terminalID)
	if err != nil {
		return ManagedCLI{}, err
	}
	if tm.WorkspaceID != workspaceID {
		return ManagedCLI{}, managedCLIInvalid("That terminal belongs to another workspace.")
	}
	if launch, err := s.TerminalLaunch(terminalID); err != nil {
		return ManagedCLI{}, err
	} else if launch != nil && launch.CLI != cli {
		return ManagedCLI{}, managedCLIInvalid("That terminal is launched as a different CLI.")
	}
	if existing, err := s.ManagedCLIByTerminal(terminalID); err == nil {
		_ = existing
		return ManagedCLI{}, errManagedCLIBound
	} else if err != nil && !errors.Is(err, ErrNotFound) {
		return ManagedCLI{}, err
	}

	row := ManagedCLI{
		ID:          newID(name, "mcli"),
		WorkspaceID: workspaceID,
		CLI:         cli,
		TerminalID:  terminalID,
		Name:        name,
		CreatedAt:   nowUTC(),
	}
	tx, err := s.db.Begin()
	if err != nil {
		return ManagedCLI{}, err
	}
	defer s.rollback(tx)
	if _, err = tx.Exec(`INSERT INTO managed_clis (id, workspace_id, cli, terminal_id, name, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		row.ID, row.WorkspaceID, row.CLI, row.TerminalID, row.Name, row.CreatedAt); err != nil {
		if isUniqueConstraint(err) {
			return ManagedCLI{}, errManagedCLIBound
		}
		return ManagedCLI{}, fmt.Errorf("store: insert managed CLI: %w", err)
	}
	if err = s.AppendEventTx(tx, "managed_cli.added", nil, &row.WorkspaceID, row); err != nil {
		return ManagedCLI{}, err
	}
	if err = s.commit(tx); err != nil {
		return ManagedCLI{}, err
	}
	return s.GetManagedCLI(row.ID)
}

// GetManagedCLI fetches a binding by id.
func (s *Store) GetManagedCLI(id string) (ManagedCLI, error) {
	var m ManagedCLI
	row := s.db.QueryRow(`SELECT `+managedCLICols+` FROM managed_clis WHERE id = ?`, id)
	if err := scanManagedCLI(row, &m); err != nil {
		if err == sql.ErrNoRows {
			return ManagedCLI{}, ErrNotFound
		}
		return ManagedCLI{}, fmt.Errorf("store: get managed CLI: %w", err)
	}
	return m, nil
}

// ManagedCLIByTerminal returns the binding for this terminal, if any.
func (s *Store) ManagedCLIByTerminal(terminalID string) (ManagedCLI, error) {
	var m ManagedCLI
	row := s.db.QueryRow(`SELECT `+managedCLICols+` FROM managed_clis WHERE terminal_id = ?`, terminalID)
	if err := scanManagedCLI(row, &m); err != nil {
		if err == sql.ErrNoRows {
			return ManagedCLI{}, ErrNotFound
		}
		return ManagedCLI{}, fmt.Errorf("store: managed CLI by terminal: %w", err)
	}
	return m, nil
}

// ListManagedCLIs returns bindings in a workspace, oldest first.
func (s *Store) ListManagedCLIs(workspaceID string) ([]ManagedCLI, error) {
	rows, err := s.db.Query(`SELECT `+managedCLICols+` FROM managed_clis WHERE workspace_id = ? ORDER BY created_at`, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("store: list managed CLIs: %w", err)
	}
	defer rows.Close()
	out := []ManagedCLI{}
	for rows.Next() {
		var m ManagedCLI
		if err := scanManagedCLI(rows, &m); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// RemoveManagedCLI deletes the binding only. The terminal and vendor files stay.
func (s *Store) RemoveManagedCLI(id string) error {
	m, err := s.GetManagedCLI(id)
	if err != nil {
		return err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer s.rollback(tx)
	res, err := tx.Exec(`DELETE FROM managed_clis WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("store: delete managed CLI: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	if err = s.AppendEventTx(tx, "managed_cli.removed", nil, &m.WorkspaceID, map[string]string{
		"id": m.ID, "terminalId": m.TerminalID, "workspaceId": m.WorkspaceID,
	}); err != nil {
		return err
	}
	return s.commit(tx)
}

func isUniqueConstraint(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique constraint") || strings.Contains(msg, "constraint failed")
}
