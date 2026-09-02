package store

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"strings"

	"github.com/cfpperche/picode/internal/session"
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

// AllAgentSessionPaths is every session_path ever historized in
// agent_sessions, across every agent (ADR-0039) — for callers that must
// not touch a session merely because it isn't any agent's *current*
// pointer (e.g. the age-based orphan sweep, ADR-0040).
func (s *Store) AllAgentSessionPaths() (map[string]bool, error) {
	out := map[string]bool{}
	rows, err := s.db.Query(`SELECT DISTINCT session_path FROM agent_sessions WHERE session_path IS NOT NULL AND session_path != ''`)
	if err != nil {
		return nil, fmt.Errorf("store: all agent session paths: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, fmt.Errorf("store: all agent session paths: %w", err)
		}
		out[p] = true
	}
	return out, rows.Err()
}

// SealPendingAgentSessions closes the attribution window that
// NewPendingAgentSession opened: every pending id whose file already
// exists gets its path recorded, so a later spawn's
// ResolvePendingAgentSession cannot adopt it. An explicit "new session"
// must stay new — adoption (ADR-0053) exists to heal a *lost* pointer,
// not to override the user asking for a fresh thread. Pending ids with
// no file are left alone: they can never be adopted, and their file may
// still appear (a slow pi).
func (s *Store) SealPendingAgentSessions(agentID string) {
	ids, err := s.PendingAgentSessionIDs(agentID)
	if err != nil || len(ids) == 0 {
		return
	}
	pending := make(map[string]bool, len(ids))
	for _, id := range ids {
		pending[id] = true
	}
	files, err := session.ListDirs(session.AgentDir(agentID))
	if err != nil {
		return
	}
	for _, f := range files { // newest first; order irrelevant here
		if pending[f.ID] {
			s.ResolveAgentSessionID(agentID, f.ID, f.Path)
		}
	}
}

// ResolveAgentSessionID backfills the session_path for a pending
// session_id row once its file shows up on disk. Cosmetic bookkeeping
// only — the filter already matches on session_id alone, so a failed or
// never-called resolve never hides a session.
func (s *Store) ResolveAgentSessionID(agentID, sessionID, path string) {
	_, _ = s.db.Exec(`UPDATE agent_sessions SET session_path = ? WHERE agent_id = ? AND session_id = ? AND session_path IS NULL`,
		path, agentID, sessionID)
}

// PendingAgentSessionIDs is the session ids this agent is already
// attributed (historized by NewPendingAgentSession, ADR-0039) but whose
// file path nobody has recorded yet.
func (s *Store) PendingAgentSessionIDs(agentID string) ([]string, error) {
	rows, err := s.db.Query(`SELECT session_id FROM agent_sessions
		WHERE agent_id = ? AND session_id IS NOT NULL AND session_id != ''
		AND (session_path IS NULL OR session_path = '')`, agentID)
	if err != nil {
		return nil, fmt.Errorf("store: pending agent sessions: %w", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("store: pending agent sessions: %w", err)
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// ResolvePendingAgentSession matches the agent's pending session ids
// against the files actually on disk in its private session dir
// (ADR-0040) and adopts the newest one: the row gets its path, and an
// agent with no current session gets agents.session_path backfilled.
// Returns the adopted path, or "" when nothing pending has a file — the
// fresh-agent case, where the caller mints a new id.
//
// Why spawn needs this: ADR-0039's backfill ran only inside the chat
// picker, so a run-mode switch between picker reads (open the agent's
// TUI, send from the chat) found SessionPath empty and minted a
// competing session — the chat and the TUI drifted onto different
// threads, each sure the other's work was "someone else's session".
func (s *Store) ResolvePendingAgentSession(agentID string) string {
	ids, err := s.PendingAgentSessionIDs(agentID)
	if err != nil || len(ids) == 0 {
		return ""
	}
	pending := make(map[string]bool, len(ids))
	for _, id := range ids {
		pending[id] = true
	}
	files, err := session.ListDirs(session.AgentDir(agentID))
	if err != nil || len(files) == 0 {
		return ""
	}
	newest := ""
	for _, f := range files { // newest first
		if !pending[f.ID] {
			continue
		}
		s.ResolveAgentSessionID(agentID, f.ID, f.Path)
		if newest == "" {
			newest = f.Path
		}
	}
	if newest == "" {
		return ""
	}
	if a, err := s.GetAgent(agentID); err == nil &&
		(a.SessionPath == nil || strings.TrimSpace(*a.SessionPath) == "") {
		_, _ = s.UpdateAgent(agentID, AgentPatch{SessionPath: &newest})
	}
	return newest
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
