package store

import (
	"encoding/json"
	"fmt"
	"strings"
)

// SessionHandoff records one cross-CLI session handoff (ADR-0088): which
// native conversation was translated, into which CLI, how (native session
// or brief, whole conversation or since the last summary, tool shape),
// what the translation left behind (manifest), and where it continues (a
// CLI terminal or a managed agent). TargetID is empty when the target CLI
// cannot pre-assign an id (Codex in brief mode); the terminal's pinned
// session resolves it later.
type SessionHandoff struct {
	ID         string          `json:"id"`
	SourceCLI  string          `json:"sourceCli"`
	SourceID   string          `json:"sourceId"`
	SourcePath string          `json:"sourcePath,omitempty"`
	TargetCLI  string          `json:"targetCli"`
	TargetID   string          `json:"targetId,omitempty"`
	TargetPath string          `json:"targetPath,omitempty"`
	Mode       string          `json:"mode"`   // native | brief | fork
	Window     string          `json:"window"` // recent | all
	Tools      string          `json:"tools,omitempty"`
	Manifest   json.RawMessage `json:"manifest"`
	TerminalID string          `json:"terminalId,omitempty"`
	AgentID    string          `json:"agentId,omitempty"`
	CreatedAt  string          `json:"createdAt"`
}

const sessionHandoffCols = `id, source_cli, source_id, source_path, target_cli, target_id, target_path, mode, "window", tools, manifest, terminal_id, agent_id, created_at`

func scanSessionHandoff(row interface{ Scan(...any) error }) (SessionHandoff, error) {
	var h SessionHandoff
	var manifest string
	err := row.Scan(&h.ID, &h.SourceCLI, &h.SourceID, &h.SourcePath, &h.TargetCLI, &h.TargetID, &h.TargetPath, &h.Mode, &h.Window, &h.Tools, &manifest, &h.TerminalID, &h.AgentID, &h.CreatedAt)
	if err != nil {
		return h, err
	}
	if strings.TrimSpace(manifest) == "" {
		manifest = "{}"
	}
	h.Manifest = json.RawMessage(manifest)
	return h, nil
}

// AddSessionHandoff records a handoff and announces it (session.handoff)
// in the same transaction. ID and CreatedAt are assigned when empty.
func (s *Store) AddSessionHandoff(h SessionHandoff) (SessionHandoff, error) {
	h.SourceCLI, h.SourceID, h.TargetCLI = strings.TrimSpace(h.SourceCLI), strings.TrimSpace(h.SourceID), strings.TrimSpace(h.TargetCLI)
	if h.SourceCLI == "" || h.SourceID == "" || h.TargetCLI == "" {
		return h, fmt.Errorf("store: handoff needs a source cli, a source id and a target cli")
	}
	switch h.Mode {
	case "native", "brief", "fork":
	default:
		return h, fmt.Errorf("store: handoff mode must be native, brief or fork")
	}
	switch h.Window {
	case "recent", "all":
	default:
		return h, fmt.Errorf("store: handoff window must be recent or all")
	}
	if h.ID == "" {
		h.ID = newID(h.SourceCLI+"-to-"+h.TargetCLI, "handoff")
	}
	if h.CreatedAt == "" {
		h.CreatedAt = nowUTC()
	}
	if len(h.Manifest) == 0 || !json.Valid(h.Manifest) {
		h.Manifest = json.RawMessage("{}")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return h, err
	}
	defer s.rollback(tx)
	_, err = tx.Exec(`INSERT INTO session_handoffs (`+sessionHandoffCols+`) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		h.ID, h.SourceCLI, h.SourceID, h.SourcePath, h.TargetCLI, h.TargetID, h.TargetPath, h.Mode, h.Window, h.Tools, string(h.Manifest), h.TerminalID, h.AgentID, h.CreatedAt)
	if err != nil {
		return h, err
	}
	var agentID *string
	if h.AgentID != "" {
		id := h.AgentID
		agentID = &id
	}
	if err := s.AppendEventTx(tx, "session.handoff", agentID, nil, h); err != nil {
		return h, err
	}
	return h, s.commit(tx)
}

// SessionHandoffs lists handoffs, newest first, at most limit (1..1000).
func (s *Store) SessionHandoffs(limit int) ([]SessionHandoff, error) {
	if limit <= 0 || limit > 1000 {
		limit = 1000
	}
	rows, err := s.db.Query(`SELECT `+sessionHandoffCols+` FROM session_handoffs ORDER BY created_at DESC, id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []SessionHandoff{}
	for rows.Next() {
		h, err := scanSessionHandoff(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

// SessionHandoff returns one handoff by id.
func (s *Store) SessionHandoff(id string) (SessionHandoff, error) {
	h, err := scanSessionHandoff(s.db.QueryRow(`SELECT `+sessionHandoffCols+` FROM session_handoffs WHERE id = ?`, id))
	if err != nil {
		return h, fmt.Errorf("store: handoff %s: %w", id, err)
	}
	return h, nil
}
