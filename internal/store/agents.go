package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

func stringsTrimSpace(s string) string { return strings.TrimSpace(s) }

// Agent is a configured pi instance in a workspace.
type Agent struct {
	ID               string   `json:"id"`
	WorkspaceID      string   `json:"workspaceId"`
	Name             string   `json:"name"`
	CreatedAt        string   `json:"createdAt"`
	Provider         *string  `json:"provider"`
	Model            *string  `json:"model"`
	Thinking         *string  `json:"thinking"`
	OpMode           *string  `json:"opMode"`
	SessionPath      *string  `json:"sessionPath"`
	ExtraPrompt      *string  `json:"extraPrompt"`
	LastStartedAt    *string  `json:"lastStartedAt"`
	LastStatus       string   `json:"lastStatus"`
	LastStatusAt     *string  `json:"lastStatusAt"`
	WorkPath         *string  `json:"workPath"`
	Packages         []string `json:"packages"`
	PackagesIsolated bool     `json:"packagesIsolated"`
}

const agentCols = `id, workspace_id, name, created_at, provider, model, thinking, extra_prompt, op_mode, session_path, last_started_at, last_status, last_status_at, work_path, packages, packages_isolated`

func scanAgent(row interface{ Scan(...any) error }, a *Agent) error {
	var pkgs string
	var isolated int
	err := row.Scan(&a.ID, &a.WorkspaceID, &a.Name, &a.CreatedAt, &a.Provider, &a.Model,
		&a.Thinking, &a.ExtraPrompt, &a.OpMode, &a.SessionPath, &a.LastStartedAt, &a.LastStatus, &a.LastStatusAt, &a.WorkPath, &pkgs, &isolated)
	if err != nil {
		return err
	}
	a.Packages = decodePackages(pkgs)
	a.PackagesIsolated = isolated != 0
	return nil
}

func decodePackages(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "null" {
		return []string{}
	}
	var out []string
	if json.Unmarshal([]byte(raw), &out) != nil || out == nil {
		return []string{}
	}
	return out
}

func encodePackages(src []string) string {
	if src == nil {
		src = []string{}
	}
	b, err := json.Marshal(src)
	if err != nil {
		return "[]"
	}
	return string(b)
}

// ensureDefaultAgentTx creates the workspace's default agent if missing.
// Only the legacy JSON-registry import uses it now: pre-ADR-0027 registries
// expect a usable workspace. New workspaces start empty (AddWorkspace).
func ensureDefaultAgentTx(tx txRunner, workspaceID, wsName, createdAt string) (Agent, error) {
	var count int
	if err := tx.QueryRow(`SELECT COUNT(1) FROM agents WHERE workspace_id = ?`, workspaceID).Scan(&count); err != nil {
		return Agent{}, fmt.Errorf("store: count agents: %w", err)
	}
	if count > 0 {
		var a Agent
		row := tx.QueryRow(`SELECT `+agentCols+` FROM agents WHERE workspace_id = ? ORDER BY created_at LIMIT 1`, workspaceID)
		if err := scanAgent(row, &a); err != nil {
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
	return scanAgent(row, a)
}

// AgentPatch is a partial update. Empty string clears a nullable column
// (inherit pi defaults). Nil pointer means leave unchanged.
type AgentPatch struct {
	Name             *string
	Provider         *string
	Model            *string
	Thinking         *string
	OpMode           *string
	SessionPath      *string
	ExtraPrompt      *string
	PackagesIsolated *bool
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
	if p.OpMode != nil {
		mode, err := NormalizeOpMode(*p.OpMode)
		if err != nil {
			return Agent{}, err
		}
		a.OpMode = mode
	}
	if p.SessionPath != nil {
		a.SessionPath = emptyToNil(*p.SessionPath)
	}
	if p.ExtraPrompt != nil {
		a.ExtraPrompt = emptyToNil(*p.ExtraPrompt)
	}
	if p.PackagesIsolated != nil {
		a.PackagesIsolated = *p.PackagesIsolated
	}
	iso := 0
	if a.PackagesIsolated {
		iso = 1
	}
	_, err = s.db.Exec(`UPDATE agents SET name=?, provider=?, model=?, thinking=?, extra_prompt=?, op_mode=?, session_path=?, packages_isolated=? WHERE id=?`,
		a.Name, a.Provider, a.Model, a.Thinking, a.ExtraPrompt, a.OpMode, a.SessionPath, iso, id)
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
	if a.OpMode != nil && *a.OpMode == OpModeReadonly {
		args = append(args, "--tools", ReadonlyTools)
	}
	if a.SessionPath != nil && *a.SessionPath != "" {
		args = append(args, "--session", *a.SessionPath)
	}
	if a.ExtraPrompt != nil && *a.ExtraPrompt != "" {
		args = append(args, "--append-system-prompt", *a.ExtraPrompt)
	}
	if a.PackagesIsolated {
		args = append(args, "--no-extensions", "--no-skills", "--no-prompt-templates", "--no-themes")
	}
	for _, src := range a.Packages {
		src = strings.TrimSpace(src)
		if src != "" {
			args = append(args, "-e", src)
		}
	}
	return args
}

// RolesAgentEnv is the process env pi-roles reads for a per-agent overlay (ADR-0033).
const RolesAgentEnv = "PI_ROLES_AGENT"

// SpawnEnv is extra process env for this agent. Empty ID yields nothing.
func (a Agent) SpawnEnv() []string {
	id := strings.TrimSpace(a.ID)
	if id == "" {
		return nil
	}
	return []string{RolesAgentEnv + "=" + id}
}

// SetAgentPackages replaces the agent's extra packages (pi -e on every start).
func (s *Store) SetAgentPackages(id string, srcs []string) (Agent, error) {
	if _, err := s.GetAgent(id); err != nil {
		return Agent{}, err
	}
	out := make([]string, 0, len(srcs))
	seen := map[string]bool{}
	for _, raw := range srcs {
		s := strings.TrimSpace(raw)
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	if len(out) > 32 {
		return Agent{}, fmt.Errorf("store: too many packages")
	}
	_, err := s.db.Exec(`UPDATE agents SET packages=? WHERE id=?`, encodePackages(out), id)
	if err != nil {
		return Agent{}, fmt.Errorf("store: agent packages: %w", err)
	}
	return s.GetAgent(id)
}

const (
	OpModeFull     = "full"
	OpModeReadonly = "readonly"
	ReadonlyTools  = "read,grep,find,ls"
)

// NormalizeOpMode accepts full/readonly (and aliases). Empty or full → nil (default).
func NormalizeOpMode(raw string) (*string, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", OpModeFull:
		return nil, nil
	case OpModeReadonly, "read-only", "read_only":
		v := OpModeReadonly
		return &v, nil
	default:
		return nil, fmt.Errorf("store: unknown op mode %q", raw)
	}
}

func emptyToNil(s string) *string {
	s = stringsTrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}

// AddAgent creates an agent in a workspace (use FreeWorkspaceID for unbound).
func (s *Store) AddAgent(workspaceID, name, workPath string) (Agent, error) {
	name = stringsTrimSpace(name)
	if name == "" {
		return Agent{}, fmt.Errorf("store: name is required")
	}
	if workspaceID == "" {
		workspaceID = FreeWorkspaceID
	}
	if _, err := s.GetWorkspace(workspaceID); err != nil {
		return Agent{}, err
	}
	a := Agent{
		ID:          newID(name, "agent"),
		WorkspaceID: workspaceID,
		Name:        name,
		CreatedAt:   nowUTC(),
		LastStatus:  StatusNeverStarted,
		WorkPath:    emptyToNil(workPath),
	}
	if _, err := s.db.Exec(`INSERT INTO agents (id, workspace_id, name, created_at, last_status, work_path) VALUES (?, ?, ?, ?, ?, ?)`,
		a.ID, a.WorkspaceID, a.Name, a.CreatedAt, a.LastStatus, a.WorkPath); err != nil {
		return Agent{}, fmt.Errorf("store: insert agent: %w", err)
	}
	return s.GetAgent(a.ID)
}

// ListAgents returns agents in a workspace, oldest first.
func (s *Store) ListAgents(workspaceID string) ([]Agent, error) {
	rows, err := s.db.Query(`SELECT `+agentCols+` FROM agents WHERE workspace_id = ? ORDER BY created_at`, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("store: list agents: %w", err)
	}
	defer rows.Close()
	var out []Agent
	for rows.Next() {
		var a Agent
		if err := scanAgentIntoRows(&a, rows); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	if out == nil {
		out = []Agent{}
	}
	return out, rows.Err()
}

func scanAgentIntoRows(a *Agent, rows *sql.Rows) error {
	return scanAgent(rows, a)
}

// ListAllAgents returns every agent, including free ones.
func (s *Store) ListAllAgents() ([]Agent, error) {
	rows, err := s.db.Query(`SELECT ` + agentCols + ` FROM agents ORDER BY created_at`)
	if err != nil {
		return nil, fmt.Errorf("store: list all agents: %w", err)
	}
	defer rows.Close()
	var out []Agent
	for rows.Next() {
		var a Agent
		if err := scanAgentIntoRows(&a, rows); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	if out == nil {
		out = []Agent{}
	}
	return out, rows.Err()
}

// DeleteAgent removes one agent. Workspace is kept.
func (s *Store) DeleteAgent(id string) error {
	res, err := s.db.Exec(`DELETE FROM agents WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("store: delete agent: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}
