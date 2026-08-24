package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

func stringsTrimSpace(s string) string { return strings.TrimSpace(s) }

// Agent is a configured pi instance in a workspace.
type Agent struct {
	ID            string  `json:"id"`
	WorkspaceID   string  `json:"workspaceId"`
	Name          string  `json:"name"`
	CreatedAt     string  `json:"createdAt"`
	Provider      *string `json:"provider"`
	Model         *string `json:"model"`
	Thinking      *string `json:"thinking"`
	ExtraPrompt   *string `json:"extraPrompt"`
	LastStartedAt *string `json:"lastStartedAt"`
	LastStatus    string  `json:"lastStatus"`
	LastStatusAt  *string `json:"lastStatusAt"`
}

const agentCols = `id, workspace_id, name, created_at, provider, model, thinking, extra_prompt, last_started_at, last_status, last_status_at`

func scanAgent(row interface{ Scan(...any) error }) (Agent, error) {
	var a Agent
	err := row.Scan(&a.ID, &a.WorkspaceID, &a.Name, &a.CreatedAt, &a.Provider, &a.Model,
		&a.Thinking, &a.ExtraPrompt, &a.LastStartedAt, &a.LastStatus, &a.LastStatusAt)
	return a, err
}

// ensureDefaultAgentTx creates the workspace's default agent if missing.
// v1 invariant: every workspace has exactly one agent named "default"
// (display-named after the workspace); M3 relaxes this with the wizard.
func ensureDefaultAgentTx(tx txRunner, workspaceID, wsName, createdAt string) (Agent, error) {
	var count int
	if err := tx.QueryRow(`SELECT COUNT(1) FROM agents WHERE workspace_id = ?`, workspaceID).Scan(&count); err != nil {
		return Agent{}, fmt.Errorf("store: count agents: %w", err)
	}
	if count > 0 {
		var a Agent
		err := tx.QueryRow(`SELECT `+agentCols+` FROM agents WHERE workspace_id = ? ORDER BY created_at LIMIT 1`, workspaceID).
			Scan(&a.ID, &a.WorkspaceID, &a.Name, &a.CreatedAt, &a.Provider, &a.Model,
				&a.Thinking, &a.ExtraPrompt, &a.LastStartedAt, &a.LastStatus, &a.LastStatusAt)
		if err != nil {
			return Agent{}, fmt.Errorf("store: default agent: %w", err)
		}
		return a, nil
	}
	a := Agent{
		ID:          newID(wsName+"-agent", "agent"),
		WorkspaceID: workspaceID,
		Name:        "default",
		CreatedAt:   createdAt,
		LastStatus:  "never_started",
	}
	if _, err := tx.Exec(`INSERT INTO agents (id, workspace_id, name, created_at, last_status) VALUES (?, ?, ?, ?, 'never_started')`,
		a.ID, a.WorkspaceID, a.Name, a.CreatedAt); err != nil {
		return Agent{}, fmt.Errorf("store: insert agent: %w", err)
	}
	return a, nil
}

// txRunner is the intersection of *sql.Tx and *sql.DB used by helpers.
type txRunner interface {
	Exec(query string, args ...any) (sql.Result, error)
	QueryRow(query string, args ...any) *sql.Row
}

// DefaultAgent returns the workspace's default agent.
func (s *Store) DefaultAgent(workspaceID string) (Agent, error) {
	var a Agent
	row := s.db.QueryRow(`SELECT `+agentCols+` FROM agents WHERE workspace_id = ? ORDER BY created_at LIMIT 1`, workspaceID)
	if err := scanAgentInto(row, &a); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Agent{}, ErrNotFound
		}
		return Agent{}, fmt.Errorf("store: default agent: %w", err)
	}
	return a, nil
}

// GetAgent fetches an agent by id.
func (s *Store) GetAgent(id string) (Agent, error) {
	var a Agent
	row := s.db.QueryRow(`SELECT `+agentCols+` FROM agents WHERE id = ?`, id)
	if err := scanAgentInto(row, &a); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Agent{}, ErrNotFound
		}
		return Agent{}, fmt.Errorf("store: get agent: %w", err)
	}
	return a, nil
}

// AgentStatus values (cached view of runtime truth; see ADR-0005).
const (
	StatusNeverStarted = "never_started"
	StatusRunning      = "running"
	StatusStopped      = "stopped"
)

// SetAgentRuntime updates the cached runtime status of an agent.
func (s *Store) SetAgentRuntime(id, status string) error {
	var startedAt any
	if status == StatusRunning {
		startedAt = nowUTC()
	}
	res, err := s.db.Exec(`UPDATE agents SET last_status = ?, last_status_at = ?, last_started_at = COALESCE(?, last_started_at) WHERE id = ?`,
		status, nowUTC(), startedAt, id)
	if err != nil {
		return fmt.Errorf("store: agent runtime: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

func scanAgentInto(row *sql.Row, a *Agent) error {
	return row.Scan(&a.ID, &a.WorkspaceID, &a.Name, &a.CreatedAt, &a.Provider, &a.Model,
		&a.Thinking, &a.ExtraPrompt, &a.LastStartedAt, &a.LastStatus, &a.LastStatusAt)
}

// AgentPatch is a partial update. Empty string clears a nullable column
// (inherit pi defaults). Nil pointer means leave unchanged.
type AgentPatch struct {
	Name        *string
	Provider    *string
	Model       *string
	Thinking    *string
	ExtraPrompt *string
}

// UpdateAgent applies a patch. Returns the row after the write.
func (s *Store) UpdateAgent(id string, p AgentPatch) (Agent, error) {
	a, err := s.GetAgent(id)
	if err != nil {
		return Agent{}, err
	}
	if p.Name != nil {
		n := stringsTrimSpace(*p.Name)
		if n == "" {
			return Agent{}, fmt.Errorf("store: name is required")
		}
		a.Name = n
	}
	if p.Provider != nil {
		a.Provider = emptyToNil(*p.Provider)
	}
	if p.Model != nil {
		a.Model = emptyToNil(*p.Model)
	}
	if p.Thinking != nil {
		a.Thinking = emptyToNil(*p.Thinking)
	}
	if p.ExtraPrompt != nil {
		a.ExtraPrompt = emptyToNil(*p.ExtraPrompt)
	}
	_, err = s.db.Exec(`UPDATE agents SET name=?, provider=?, model=?, thinking=?, extra_prompt=? WHERE id=?`,
		a.Name, a.Provider, a.Model, a.Thinking, a.ExtraPrompt, id)
	if err != nil {
		return Agent{}, fmt.Errorf("store: update agent: %w", err)
	}
	return s.GetAgent(id)
}

// CLIFlags are the pi argv extras for this agent's stored config (ADR-0009).
func (a Agent) CLIFlags() []string {
	var args []string
	if a.Provider != nil && *a.Provider != "" {
		args = append(args, "--provider", *a.Provider)
	}
	if a.Model != nil && *a.Model != "" {
		args = append(args, "--model", *a.Model)
	}
	if a.Thinking != nil && *a.Thinking != "" {
		args = append(args, "--thinking", *a.Thinking)
	}
	if a.ExtraPrompt != nil && *a.ExtraPrompt != "" {
		args = append(args, "--append-system-prompt", *a.ExtraPrompt)
	}
	return args
}

func emptyToNil(s string) *string {
	s = stringsTrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}
