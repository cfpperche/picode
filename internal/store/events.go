package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// Event is one row of the change log (ADR-0048). Every mutation the store
// makes appends one in the same transaction, so "it changed" and "it was
// announced" cannot disagree. ID is monotonic (AUTOINCREMENT) and doubles
// as the SSE cursor; Data is the entity's JSON view (or {id} on delete).
// It is still the ADR-0005 audit record — orchestration only, never chat.
type Event struct {
	ID          int64           `json:"id"`
	AgentID     *string         `json:"agentId"`
	WorkspaceID *string         `json:"workspaceId"`
	Type        string          `json:"type"`
	Data        json.RawMessage `json:"data"`
	CreatedAt   string          `json:"createdAt"`
}

// AppendEvent records an event and announces it once it is durable.
func (s *Store) AppendEvent(eventType string, agentID, workspaceID *string, data any) error {
	ev, err := s.appendEvent(s.db, eventType, agentID, workspaceID, data)
	if err != nil {
		return err
	}
	s.emit(ev)
	return nil
}

// AppendEventTx records an event inside a transaction; it is announced
// only when the store commits that transaction (commit below).
func (s *Store) AppendEventTx(tx *sql.Tx, eventType string, agentID, workspaceID *string, data any) error {
	ev, err := s.appendEvent(tx, eventType, agentID, workspaceID, data)
	if err != nil {
		return err
	}
	s.pendMu.Lock()
	if s.pending == nil {
		s.pending = map[*sql.Tx][]Event{}
	}
	s.pending[tx] = append(s.pending[tx], ev)
	s.pendMu.Unlock()
	return nil
}

func (s *Store) appendEvent(runner txRunner, eventType string, agentID, workspaceID *string, data any) (Event, error) {
	if data == nil {
		data = map[string]any{}
	}
	body, err := marshalJSON(data)
	if err != nil {
		return Event{}, fmt.Errorf("store: event data: %w", err)
	}
	now := nowUTC()
	res, err := runner.Exec(`INSERT INTO events (agent_id, workspace_id, type, data, created_at) VALUES (?, ?, ?, ?, ?)`,
		orNull(agentID), orNull(workspaceID), eventType, body, now)
	if err != nil {
		return Event{}, fmt.Errorf("store: append event: %w", err)
	}
	id, _ := res.LastInsertId()
	return Event{ID: id, AgentID: agentID, WorkspaceID: workspaceID, Type: eventType, Data: json.RawMessage(body), CreatedAt: now}, nil
}

// commit commits tx and announces the events it appended, in order.
// rollback drops them. Store methods use these instead of tx.Commit /
// tx.Rollback so a listener never sees an uncommitted change.
func (s *Store) commit(tx *sql.Tx) error {
	s.pendMu.Lock()
	evs := s.pending[tx]
	delete(s.pending, tx)
	s.pendMu.Unlock()
	if err := tx.Commit(); err != nil {
		return err
	}
	for _, ev := range evs {
		s.emit(ev)
	}
	return nil
}

func (s *Store) rollback(tx *sql.Tx) {
	s.pendMu.Lock()
	delete(s.pending, tx)
	s.pendMu.Unlock()
	_ = tx.Rollback()
}

func (s *Store) emit(ev Event) {
	if s.OnEvent != nil {
		s.OnEvent(ev)
	}
}

// note is AppendEvent for the common case where the mutation already
// succeeded and a failed audit write must not fail the caller.
func (s *Store) note(eventType string, agentID, workspaceID *string, data any) {
	_ = s.AppendEvent(eventType, agentID, workspaceID, data)
}

func idData(id string) map[string]string { return map[string]string{"id": id} }

// Note on the agent_id / workspace_id columns: they are foreign keys
// (001_init), so only pass ids that are real agents / workspaces; every
// payload carries its own ids for the reader anyway.

// ListEventsSince returns events with id > afterID, oldest first (replay).
func (s *Store) ListEventsSince(afterID int64, limit int) ([]Event, error) {
	if limit <= 0 {
		limit = 500
	}
	return s.queryEvents(`SELECT id, agent_id, workspace_id, type, data, created_at FROM events WHERE id > ? ORDER BY id ASC LIMIT ?`, afterID, limit)
}

// LatestEventID is the newest id (0 when the log is empty).
func (s *Store) LatestEventID() (int64, error) {
	var id sql.NullInt64
	if err := s.db.QueryRow(`SELECT MAX(id) FROM events`).Scan(&id); err != nil {
		return 0, fmt.Errorf("store: latest event: %w", err)
	}
	return id.Int64, nil
}

// OldestEventID is the smallest id still stored (0 when empty). A client
// cursor below it cannot be replayed.
func (s *Store) OldestEventID() (int64, error) {
	var id sql.NullInt64
	if err := s.db.QueryRow(`SELECT MIN(id) FROM events`).Scan(&id); err != nil {
		return 0, fmt.Errorf("store: oldest event: %w", err)
	}
	return id.Int64, nil
}

// PruneEvents drops events created before t (retention). Returns the count.
func (s *Store) PruneEvents(t time.Time) (int, error) {
	res, err := s.db.Exec(`DELETE FROM events WHERE created_at < ?`, t.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return 0, fmt.Errorf("store: prune events: %w", err)
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

// RecentEvents returns the newest events across all workspaces.
func (s *Store) RecentEvents(limit int) ([]Event, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.queryEvents(`SELECT id, agent_id, workspace_id, type, data, created_at FROM events ORDER BY id DESC LIMIT ?`, limit)
}

// AgentEvents returns the newest events for one agent.
func (s *Store) AgentEvents(agentID string, limit int) ([]Event, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.queryEvents(`SELECT id, agent_id, workspace_id, type, data, created_at FROM events WHERE agent_id = ? ORDER BY id DESC LIMIT ?`, agentID, limit)
}

func (s *Store) queryEvents(query string, args ...any) ([]Event, error) {
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("store: events: %w", err)
	}
	defer rows.Close()
	var out []Event
	for rows.Next() {
		var e Event
		var data string
		if err := rows.Scan(&e.ID, &e.AgentID, &e.WorkspaceID, &e.Type, &data, &e.CreatedAt); err != nil {
			return nil, err
		}
		if !json.Valid([]byte(data)) {
			data = "{}"
		}
		e.Data = json.RawMessage(data)
		out = append(out, e)
	}
	return out, rows.Err()
}

// ---------- settings ----------

// GetSetting returns a setting value ("", false when absent).
func (s *Store) GetSetting(key string) (string, bool, error) {
	var v string
	err := s.db.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&v)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("store: get setting: %w", err)
	}
	return v, true, nil
}

// SetSetting upserts a setting.
func (s *Store) SetSetting(key, value string) error {
	if _, err := s.db.Exec(`INSERT INTO settings (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value); err != nil {
		return fmt.Errorf("store: set setting: %w", err)
	}
	s.note("setting.updated", nil, nil, map[string]string{"key": key})
	return nil
}

func orNull(p *string) any {
	if p == nil {
		return nil
	}
	return *p
}
