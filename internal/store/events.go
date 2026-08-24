package store

import (
	"database/sql"
	"fmt"
)

// Event is an orchestration audit record (ADR-0005: not a chat log).
type Event struct {
	ID          int64   `json:"id"`
	AgentID     *string `json:"agentId"`
	WorkspaceID *string `json:"workspaceId"`
	Type        string  `json:"type"`
	Data        string  `json:"data"`
	CreatedAt   string  `json:"createdAt"`
}

// AppendEvent records an orchestration event with JSON data.
func (s *Store) AppendEvent(eventType string, agentID, workspaceID *string, data any) error {
	_, err := s.appendEvent(s.db, eventType, agentID, workspaceID, data)
	return err
}

// AppendEventTx records an event inside a transaction (used by compound ops).
func (s *Store) AppendEventTx(tx *sql.Tx, eventType string, agentID, workspaceID *string, data any) error {
	_, err := s.appendEvent(tx, eventType, agentID, workspaceID, data)
	return err
}

func (s *Store) appendEvent(runner txRunner, eventType string, agentID, workspaceID *string, data any) (sql.Result, error) {
	if data == nil {
		data = map[string]any{}
	}
	body, err := marshalJSON(data)
	if err != nil {
		return nil, fmt.Errorf("store: event data: %w", err)
	}
	return runner.Exec(`INSERT INTO events (agent_id, workspace_id, type, data, created_at) VALUES (?, ?, ?, ?, ?)`,
		orNull(agentID), orNull(workspaceID), eventType, body, nowUTC())
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
		if err := rows.Scan(&e.ID, &e.AgentID, &e.WorkspaceID, &e.Type, &e.Data, &e.CreatedAt); err != nil {
			return nil, err
		}
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
	return nil
}

func orNull(p *string) any {
	if p == nil {
		return nil
	}
	return *p
}
