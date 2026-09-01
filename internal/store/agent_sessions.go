package store

import (
	"crypto/rand"
	"database/sql"
	"fmt"
)

// RecordAgentSessionPath historizes a concrete session file this agent now
// owns (ADR-0039). Idempotent: a repeat call with the same path is a
// no-op via the (agent_id, session_path) unique index.
func (s *Store) RecordAgentSessionPath(agentID, path string) error {
	path = stringsTrimSpace(path)
	if path == "" {
		return nil
	}
	_, err := s.db.Exec(`INSERT INTO agent_sessions (id, agent_id, session_path, first_seen_at)
		VALUES (?, ?, ?, ?) ON CONFLICT (agent_id, session_path) DO NOTHING`,
		newID("asess", "asess"), agentID, path, nowUTC())
	if err != nil {
		return fmt.Errorf("store: record agent session: %w", err)
	}
	return nil
}

// NewPendingAgentSession mints and historizes a fresh pi session id ahead
// of spawn (ADR-0039), for an agent about to start with no current
// SessionPath. Returns "" on any DB error — spawning must never block on
// this; a failure just degrades to the pre-ADR-0039 unattributed session.
func (s *Store) NewPendingAgentSession(agentID string) string {
	id := newSessionID()
	if _, err := s.db.Exec(`INSERT INTO agent_sessions (id, agent_id, session_id, first_seen_at) VALUES (?, ?, ?, ?)`,
		newID("asess", "asess"), agentID, id, nowUTC()); err != nil {
		return ""
	}
	return id
}

// AgentSessionKeys is every session id/path an agent has ever owned — what
// handleListSessions filters session.List(cwd) against.
type AgentSessionKeys struct {
	IDs   map[string]bool
	Paths map[string]bool
}

// AgentSessionKeys loads the owned set for one agent.
func (s *Store) AgentSessionKeys(agentID string) (AgentSessionKeys, error) {
	out := AgentSessionKeys{IDs: map[string]bool{}, Paths: map[string]bool{}}
	rows, err := s.db.Query(`SELECT session_id, session_path FROM agent_sessions WHERE agent_id = ?`, agentID)
	if err != nil {
		return out, fmt.Errorf("store: agent session keys: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var sid, path sql.NullString
		if err := rows.Scan(&sid, &path); err != nil {
			return out, fmt.Errorf("store: agent session keys: %w", err)
		}
		if sid.Valid && sid.String != "" {
			out.IDs[sid.String] = true
		}
		if path.Valid && path.String != "" {
			out.Paths[path.String] = true
		}
	}
	return out, rows.Err()
}

// ResolveAgentSessionID backfills the session_path for a pending
// session_id row once its file shows up on disk. Cosmetic bookkeeping
// only — the filter already matches on session_id alone, so a failed or
// never-called resolve never hides a session.
func (s *Store) ResolveAgentSessionID(agentID, sessionID, path string) {
	_, _ = s.db.Exec(`UPDATE agent_sessions SET session_path = ? WHERE agent_id = ? AND session_id = ? AND session_path IS NULL`,
		path, agentID, sessionID)
}

// newSessionID mints a v4 UUID in the shape pi's --session-id expects.
// Not newID() — that's PiCode's slug+hex convention; pi's session ids are
// plain UUIDs. Mirrors the existing ad hoc uuid() in
// internal/share/trusthttp.go rather than adding a dependency for one
// six-line need.
func newSessionID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
