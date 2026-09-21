package store

import (
	"database/sql"
	"fmt"
)

// OrderError is a rejected reorder. The ids are not exactly the rows of
// that container, so nothing is written (ADR-0173).
type OrderError struct{ Reason string }

func (e *OrderError) Error() string { return e.Reason }

// nextPositionExpr is a scalar subquery placed in an INSERT. table and
// where are fixed fragments from this package, never request data.
func nextPositionExpr(table, where string) string {
	return `(SELECT COALESCE(MAX(position), -1) + 1 FROM ` + table + ` WHERE ` + where + `)`
}

// ReorderWorkspaces sets the sidebar order of every non-free workspace.
// ids must be a permutation of that set. The same order writes nothing.
func (s *Store) ReorderWorkspaces(ids []string) error {
	if err := validateIDList(ids); err != nil {
		return err
	}
	for _, id := range ids {
		if id == FreeWorkspaceID {
			return &OrderError{Reason: "the free workspace is not in this list"}
		}
	}
	return s.reorder(
		`SELECT id FROM workspaces WHERE id != ? ORDER BY position, id`,
		[]any{FreeWorkspaceID},
		`UPDATE workspaces SET position = ? WHERE id = ?`,
		"workspace.reordered",
		nil,
		map[string]any{"ids": ids},
		ids,
		"ids must list exactly the current workspaces",
	)
}

// ReorderAgents sets the sidebar order of the agents in one workspace,
// including the free workspace. ids must be a permutation of that set.
func (s *Store) ReorderAgents(workspaceID string, ids []string) error {
	if err := validateIDList(ids); err != nil {
		return err
	}
	if workspaceID == "" {
		workspaceID = FreeWorkspaceID
	}
	if _, err := s.GetWorkspace(workspaceID); err != nil {
		return err
	}
	ws := workspaceID
	return s.reorder(
		`SELECT id FROM agents WHERE workspace_id = ? ORDER BY position, id`,
		[]any{workspaceID},
		`UPDATE agents SET position = ? WHERE id = ?`,
		"agent.reordered",
		&ws,
		map[string]any{"workspaceId": workspaceID, "ids": ids},
		ids,
		"ids must list exactly the agents in this workspace",
	)
}

// ReorderTerminals sets the sidebar order of the shell terminals in one
// workspace. A terminal bound to an agent is not a sidebar row (the agent
// row stands for it) and is rejected.
func (s *Store) ReorderTerminals(workspaceID string, ids []string) error {
	if err := validateIDList(ids); err != nil {
		return err
	}
	if workspaceID == "" {
		workspaceID = FreeWorkspaceID
	}
	if _, err := s.GetWorkspace(workspaceID); err != nil {
		return err
	}
	if err := s.rejectOwnedTerminals(ids); err != nil {
		return err
	}
	ws := workspaceID
	return s.reorder(
		`SELECT id FROM terminals WHERE workspace_id = ? AND id NOT IN (SELECT terminal_id FROM agents WHERE terminal_id IS NOT NULL) ORDER BY position, id`,
		[]any{workspaceID},
		`UPDATE terminals SET position = ? WHERE id = ?`,
		"terminal.reordered",
		&ws,
		map[string]any{"workspaceId": workspaceID, "ids": ids},
		ids,
		"ids must list exactly the shell terminals in this workspace",
	)
}

func (s *Store) rejectOwnedTerminals(ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	q := `SELECT COUNT(1) FROM agents WHERE terminal_id IN (` + placeholders(len(ids)) + `)`
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	var n int
	if err := s.db.QueryRow(q, args...).Scan(&n); err != nil {
		return fmt.Errorf("store: reorder: %w", err)
	}
	if n > 0 {
		return &OrderError{Reason: "that terminal belongs to an agent"}
	}
	return nil
}

func placeholders(n int) string {
	s := "?"
	for i := 1; i < n; i++ {
		s += ",?"
	}
	return s
}

func validateIDList(ids []string) error {
	if len(ids) == 0 {
		return &OrderError{Reason: "ids are required"}
	}
	seen := make(map[string]bool, len(ids))
	for _, id := range ids {
		if id == "" || seen[id] {
			return &OrderError{Reason: "ids must not repeat"}
		}
		seen[id] = true
	}
	return nil
}

func (s *Store) reorder(selectSQL string, selectArgs []any, updateSQL, eventType string, workspaceID *string, payload map[string]any, ids []string, mismatch string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer s.rollback(tx)
	current, err := queryIDs(tx, selectSQL, selectArgs...)
	if err != nil {
		return err
	}
	if sameIDs(current, ids) {
		return s.commit(tx)
	}
	if !sameSet(current, ids) {
		return &OrderError{Reason: mismatch}
	}
	for i, id := range ids {
		if _, err := tx.Exec(updateSQL, i, id); err != nil {
			return fmt.Errorf("store: reorder: %w", err)
		}
	}
	if err := s.AppendEventTx(tx, eventType, nil, workspaceID, payload); err != nil {
		return err
	}
	return s.commit(tx)
}

func queryIDs(tx *sql.Tx, q string, args ...any) ([]string, error) {
	rows, err := tx.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("store: reorder: %w", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func sameIDs(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func sameSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	n := make(map[string]int, len(a))
	for _, id := range a {
		n[id]++
	}
	for _, id := range b {
		n[id]--
		if n[id] < 0 {
			return false
		}
	}
	return true
}
