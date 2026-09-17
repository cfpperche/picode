package store

// The servers the human asked the Servers panel not to show (Servers panel
// v2). A hide is keyed by the identity of the process that holds the socket —
// port + pid + start token — not by the port alone: a new process on the same
// port is a new row in the panel, never a silent hole. Mutations announce
// devserver.hidden (ADR-0048).

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// DevServerHide is one hidden listener: which port, which process, and the
// label the panel had when the human hid it (so a "hidden" list can say what
// it was without re-reading /proc).
type DevServerHide struct {
	ID       int64  `json:"id"`
	Port     int    `json:"port"`
	PID      int    `json:"pid"`
	StartKey string `json:"startKey"`
	Owner    string `json:"owner"`
	Tool     string `json:"tool"`
	HiddenAt string `json:"hiddenAt"`
}

// HideDevServer records one hide; hiding the same listener twice updates the
// stored label in place rather than adding a second row.
func (s *Store) HideDevServer(port, pid int, startKey, owner, tool string) (DevServerHide, error) {
	if port <= 0 || port > 65535 {
		return DevServerHide{}, fmt.Errorf("store: %d is not a port", port)
	}
	if pid <= 0 {
		return DevServerHide{}, fmt.Errorf("store: a hide needs the process that holds the port")
	}
	key := strings.TrimSpace(startKey)
	if key == "" {
		return DevServerHide{}, fmt.Errorf("store: a hide needs the process's start token")
	}
	when := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := s.db.Exec(`
		INSERT INTO dev_server_hides (port, pid, start_key, owner, tool, hidden_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(port, pid, start_key) DO UPDATE SET
			owner = excluded.owner,
			tool = excluded.tool,
			hidden_at = excluded.hidden_at`,
		port, pid, key, strings.TrimSpace(owner), strings.TrimSpace(tool), when); err != nil {
		return DevServerHide{}, fmt.Errorf("store: hide dev server: %w", err)
	}
	row, err := s.devServerHideFor(port, pid, key)
	if err != nil {
		return DevServerHide{}, err
	}
	s.note("devserver.hidden", nil, nil, map[string]any{"port": port, "pid": pid, "tool": row.Tool, "owner": row.Owner, "hidden": true})
	return row, nil
}

func (s *Store) devServerHideFor(port, pid int, startKey string) (DevServerHide, error) {
	row := s.db.QueryRow(`
		SELECT id, port, pid, start_key, owner, tool, hidden_at
		FROM dev_server_hides WHERE port = ? AND pid = ? AND start_key = ?`, port, pid, startKey)
	return scanDevServerHide(row)
}

// UnhideDevServer forgets one hide, which is what "Show again" asks for. A
// row that is already gone answers sql.ErrNoRows.
func (s *Store) UnhideDevServer(id int64) error {
	res, err := s.db.Exec(`DELETE FROM dev_server_hides WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("store: unhide dev server: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	s.note("devserver.hidden", nil, nil, map[string]any{"id": id, "hidden": false})
	return nil
}

// ListDevServerHides returns the hides, newest first.
func (s *Store) ListDevServerHides() ([]DevServerHide, error) {
	rows, err := s.db.Query(`
		SELECT id, port, pid, start_key, owner, tool, hidden_at
		FROM dev_server_hides ORDER BY hidden_at DESC, id DESC`)
	if err != nil {
		return nil, fmt.Errorf("store: list dev server hides: %w", err)
	}
	defer rows.Close()
	out := []DevServerHide{}
	for rows.Next() {
		h, err := scanDevServerHide(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

// PruneDevServerHides forgets every hide whose id is not in keep: the server
// passes the hides whose process is still alive, because a hide whose process
// is gone can never match a listener again — it would only make the panel's
// "N hidden" line lie. One mutation, one event, and none at all when there is
// nothing to forget.
func (s *Store) PruneDevServerHides(keep []int64) (int, error) {
	live := make(map[int64]bool, len(keep))
	for _, id := range keep {
		live[id] = true
	}
	rows, err := s.ListDevServerHides()
	if err != nil {
		return 0, err
	}
	stale := make([]int64, 0, len(rows))
	for _, h := range rows {
		if !live[h.ID] {
			stale = append(stale, h.ID)
		}
	}
	if len(stale) == 0 {
		return 0, nil
	}
	tx, err := s.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("store: prune dev server hides: %w", err)
	}
	for _, id := range stale {
		if _, err := tx.Exec(`DELETE FROM dev_server_hides WHERE id = ?`, id); err != nil {
			s.rollback(tx)
			return 0, fmt.Errorf("store: prune dev server hides: %w", err)
		}
	}
	if err := s.commit(tx); err != nil {
		return 0, fmt.Errorf("store: prune dev server hides: %w", err)
	}
	s.note("devserver.hidden", nil, nil, map[string]any{"pruned": len(stale), "hidden": false})
	return len(stale), nil
}

func scanDevServerHide(row rowScanner) (DevServerHide, error) {
	var h DevServerHide
	if err := row.Scan(&h.ID, &h.Port, &h.PID, &h.StartKey, &h.Owner, &h.Tool, &h.HiddenAt); err != nil {
		return DevServerHide{}, err
	}
	return h, nil
}
