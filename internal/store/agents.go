package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/cfpperche/picode/internal/clilaunch"
	"github.com/cfpperche/picode/internal/session"
)

func stringsTrimSpace(s string) string { return strings.TrimSpace(s) }

// CLIPi is the catalog id for the Pi runtime (ADR-0160).
const CLIPi = "pi"

// Agent is a configured runtime in a workspace (ADR-0160): Pi by default,
// or any launchable catalog CLI. Managed RPC (`Runtime.Start`) is Pi-only.
type Agent struct {
	ID               string   `json:"id"`
	WorkspaceID      string   `json:"workspaceId"`
	Name             string   `json:"name"`
	CLI              string   `json:"cli"`
	CreatedAt        string   `json:"createdAt"`
	Provider         *string  `json:"provider"`
	Model            *string  `json:"model"`
	Thinking         *string  `json:"thinking"`
	OpMode           *string  `json:"opMode"`
	Checklist        string   `json:"checklist"` // obligation level (ADR-0055): changes | always | never
	SessionPath      *string  `json:"sessionPath"`
	ExtraPrompt      *string  `json:"extraPrompt"`
	LastStartedAt    *string  `json:"lastStartedAt"`
	LastStatus       string   `json:"lastStatus"`
	LastStatusAt     *string  `json:"lastStatusAt"`
	WorkPath         *string  `json:"workPath"`
	Packages         []string `json:"packages"`
	PackagesIsolated bool     `json:"packagesIsolated"`
	// Skills are this agent's own (ADR-0196 slice 4), passed at launch.
	Skills     []AgentSkill `json:"skills"`
	TerminalID *string      `json:"terminalId,omitempty"`
}

// AgentSkill is one skill the agent loads alone: a copy in PiCode's
// digest-addressed cache, named by the folder the CLI receives.
type AgentSkill struct {
	Name   string `json:"name"`
	Digest string `json:"digest"`
	Source string `json:"source,omitempty"` // where it was added from, as the person typed it
	Dir    string `json:"dir"`              // the cached folder
	// Missing is read, never stored: the cached folder is gone, so the
	// launch leaves the skill out and the agent's row says so.
	Missing bool `json:"missing,omitempty"`
}

const agentCols = `id, workspace_id, name, created_at, provider, model, thinking, extra_prompt, op_mode, session_path, last_started_at, last_status, last_status_at, work_path, packages, packages_isolated, checklist, cli, terminal_id, skills`

func scanAgent(row interface{ Scan(...any) error }, a *Agent) error {
	var pkgs, skills string
	var isolated int
	var termID sql.NullString
	err := row.Scan(&a.ID, &a.WorkspaceID, &a.Name, &a.CreatedAt, &a.Provider, &a.Model,
		&a.Thinking, &a.ExtraPrompt, &a.OpMode, &a.SessionPath, &a.LastStartedAt, &a.LastStatus, &a.LastStatusAt, &a.WorkPath, &pkgs, &isolated, &a.Checklist, &a.CLI, &termID, &skills)
	if err != nil {
		return err
	}
	a.Packages = decodePackages(pkgs)
	a.PackagesIsolated = isolated != 0
	a.Skills = decodeAgentSkills(skills)
	for i := range a.Skills {
		if _, err := os.Stat(filepath.Join(a.Skills[i].Dir, "SKILL.md")); err != nil {
			a.Skills[i].Missing = true
		}
	}
	if strings.TrimSpace(a.CLI) == "" {
		a.CLI = CLIPi
	}
	if termID.Valid && strings.TrimSpace(termID.String) != "" {
		id := strings.TrimSpace(termID.String)
		a.TerminalID = &id
	} else {
		a.TerminalID = nil
	}
	return nil
}

// IsPi is true when managed RPC (`pi --mode rpc`) may start this agent.
func (a Agent) IsPi() bool {
	c := strings.TrimSpace(a.CLI)
	return c == "" || strings.EqualFold(c, CLIPi)
}

func normalizeAgentCLI(cli string) (string, error) {
	cli = strings.TrimSpace(cli)
	if cli == "" {
		cli = CLIPi
	}
	entry, ok := clilaunch.Find(cli)
	if !ok {
		return "", invalidError{"Unknown CLI."}
	}
	if !entry.Launchable() {
		return "", invalidError{"That CLI cannot run as an agent."}
	}
	return entry.ID, nil
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

func decodeAgentSkills(raw string) []AgentSkill {
	out := []AgentSkill{}
	if json.Unmarshal([]byte(strings.TrimSpace(raw)), &out) != nil || out == nil {
		return []AgentSkill{}
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
	if _, err := tx.Exec(`INSERT INTO agents (id, workspace_id, name, created_at, last_status, position) VALUES (?, ?, ?, ?, 'never_started', `+nextPositionExpr("agents", "workspace_id = ?")+`)`,
		a.ID, a.WorkspaceID, a.Name, a.CreatedAt, a.WorkspaceID); err != nil {
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

// AgentByTerminal returns the agent bound to this terminal, if any
// (ADR-0160 Fatia C: the CLI agent's TUI lives on agents.terminal_id).
func (s *Store) AgentByTerminal(terminalID string) (Agent, error) {
	terminalID = strings.TrimSpace(terminalID)
	if terminalID == "" {
		return Agent{}, ErrNotFound
	}
	var a Agent
	row := s.db.QueryRow(`SELECT `+agentCols+` FROM agents WHERE terminal_id = ?`, terminalID)
	if err := scanAgentInto(row, &a); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Agent{}, ErrNotFound
		}
		return Agent{}, fmt.Errorf("store: agent by terminal: %w", err)
	}
	return a, nil
}

// AgentStatus values (cached view of runtime truth; see ADR-0005).
const (
	StatusNeverStarted = "never_started"
	StatusRunning      = "running"
	StatusStopped      = "stopped"
)

// SetAgentRuntime updates the cached runtime status of an agent. The
// announced agent.status carries mode "stopped" for a stop and "" for a
// start whose mode the caller did not say (see SetAgentRuntimeMode).
func (s *Store) SetAgentRuntime(id, status string) error {
	mode := ""
	if status != StatusRunning {
		mode = "stopped"
	}
	return s.SetAgentRuntimeMode(id, status, mode)
}

// SetAgentRuntimeMode is SetAgentRuntime with the run mode the server
// knows (managed | interactive | stopped), so the change feed can patch a
// start without a refetch (ADR-0048).
func (s *Store) SetAgentRuntimeMode(id, status, mode string) error {
	var startedAt any
	if status == StatusRunning {
		startedAt = nowUTC()
	}
	at := nowUTC()
	res, err := s.db.Exec(`UPDATE agents SET last_status = ?, last_status_at = ?, last_started_at = COALESCE(?, last_started_at) WHERE id = ?`,
		status, at, startedAt, id)
	if err != nil {
		return fmt.Errorf("store: agent runtime: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	// lastStatusAt rides the event so a start/stop patches the row's state
	// age in place — the pill reads it the moment the state begins.
	s.note("agent.status", &id, nil, map[string]string{"id": id, "lastStatus": status, "mode": mode, "lastStatusAt": at})
	return nil
}

// SetAgentTurnSettled stamps when a managed agent's turn finishes, so the
// sidebar can say how long the agent has been ready. The runtime stays
// running — only the timestamp moves — and the agent.settled row patches
// every open fleet view without a refetch (ADR-0048). The exit path writes
// its own stop stamp after the settle (pumpEvents reads both on one loop,
// in order), so there is no status guard here: a settle is the last fact
// about last_status_at while the process lives.
func (s *Store) SetAgentTurnSettled(id string) error {
	at := nowUTC()
	res, err := s.db.Exec(`UPDATE agents SET last_status_at = ? WHERE id = ?`, at, id)
	if err != nil {
		return fmt.Errorf("store: agent turn settled: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	s.note("agent.settled", &id, nil, map[string]string{"id": id, "lastStatusAt": at})
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
	Checklist        *string
	SessionPath      *string
	ExtraPrompt      *string
	PackagesIsolated *bool
	TerminalID       *string
}

// UpdateAgent applies a patch. Returns the row after the write.
func (s *Store) UpdateAgent(id string, p AgentPatch) (Agent, error) {
	a, err := s.GetAgent(id)
	if err != nil {
		return Agent{}, err
	}
	previousName := a.Name
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
	if p.Checklist != nil {
		lvl, err := NormalizeChecklist(*p.Checklist)
		if err != nil {
			return Agent{}, err
		}
		a.Checklist = lvl
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
	if p.TerminalID != nil {
		a.TerminalID = emptyToNil(*p.TerminalID)
	}
	iso := 0
	if a.PackagesIsolated {
		iso = 1
	}
	_, err = s.db.Exec(`UPDATE agents SET name=?, provider=?, model=?, thinking=?, extra_prompt=?, op_mode=?, session_path=?, packages_isolated=?, checklist=?, terminal_id=? WHERE id=?`,
		a.Name, a.Provider, a.Model, a.Thinking, a.ExtraPrompt, a.OpMode, a.SessionPath, iso, a.Checklist, a.TerminalID, id)
	if err != nil {
		if isUniqueConstraint(err) {
			return Agent{}, conflictError{"That terminal already belongs to an agent."}
		}
		return Agent{}, fmt.Errorf("store: update agent: %w", err)
	}
	if p.SessionPath != nil && a.SessionPath != nil {
		// Best-effort: every session_path this agent is ever pointed at
		// (resume/fork/clone/adopt/import) gets historized here, in one
		// place, so no call site can forget (ADR-0039).
		_ = s.RecordAgentSessionPath(id, *a.SessionPath)
	}
	if p.Name != nil && previousName != a.Name && a.TerminalID != nil {
		// A bound terminal borrows its agent's name (attachAgentTerminal,
		// EnsureAgentTerminal). The rename keeps the loan current, or the
		// tab strip — which labels a t:<id> tab with term.name — keeps the
		// name the agent had at bind time. Best-effort: a terminal that
		// vanished mid-flight does not fail the agent rename.
		if _, err := s.db.Exec(`UPDATE terminals SET name = ? WHERE id = ?`, normalizeTerminalName(a.Name), *a.TerminalID); err == nil {
			if t, terr := s.GetTerminal(*a.TerminalID); terr == nil {
				s.note("terminal.updated", nil, nil, t)
			}
		}
	}
	return s.agentChanged(id)
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
	// Measured on pi 0.7x: an explicit --skill loads even with --no-skills,
	// and a folder skill of the same name wins over it.
	for _, dir := range a.SkillDirs() {
		args = append(args, "--skill", dir)
	}
	if id := strings.TrimSpace(a.ID); id != "" {
		// ADR-0040: every spawn gets a private lookup/storage root, keyed
		// by agent id rather than cwd — this is what makes pi's OWN
		// in-TUI "Resume Session" picker (not just PiCode's chat picker,
		// ADR-0039) show only this agent's own sessions. An explicit
		// --session above still wins for a resume (confirmed live: pi
		// resolves an out-of-dir --session path and leaves
		// --session-dir's directory untouched), so this is safe on every
		// spawn, not just fresh ones.
		args = append(args, "--session-dir", session.AgentDir(id))
	}
	return args
}

// CLIFlagsForSpawn is CLIFlags() plus a pre-assigned --session-id for a
// fresh start (ADR-0039), so pi's auto-created session is attributable to
// this agent from the moment it exists. sessionID is ignored when empty
// (a store hiccup in NewPendingAgentSession — degrade to plain
// CLIFlags()) or when the agent already has a SessionPath (resuming:
// CLIFlags() already passes --session for that path; --session-id must
// not also be passed).
func (a Agent) CLIFlagsForSpawn(sessionID string) []string {
	args := a.CLIFlags()
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" || (a.SessionPath != nil && strings.TrimSpace(*a.SessionPath) != "") {
		return args
	}
	return append(args, "--session-id", sessionID)
}

// RolesAgentEnv is the process env pi-roles reads for a per-agent overlay (ADR-0033).
const RolesAgentEnv = "PI_ROLES_AGENT"

// CompactAgentEnv is the process env pi-compact reads for a per-agent overlay (ADR-0061).
const CompactAgentEnv = "PI_COMPACT_AGENT"

// AgentIDEnv is the neutral agent identity for packages that talk back
// to PiCode (pi-inbox, ADR-0037). Same value as RolesAgentEnv, package-
// agnostic name.
const AgentIDEnv = "PICODE_AGENT_ID"

// SpawnEnv is extra process env for this agent. Empty ID yields nothing.
func (a Agent) SpawnEnv() []string {
	id := strings.TrimSpace(a.ID)
	if id == "" {
		return nil
	}
	// ADR-0055: the checklist obligation, read by pi-checklist when installed.
	// ADR-0061: compaction overlay, same slug as roles.
	return []string{
		RolesAgentEnv + "=" + id,
		CompactAgentEnv + "=" + id,
		AgentIDEnv + "=" + id,
		ChecklistEnv + "=" + a.ChecklistLevel(),
	}
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
	return s.agentChanged(id)
}

// SkillDirs are the agent's cached skill folders that still exist, in its
// order; a folder gone from the cache is left out of the launch (the Skills
// tab names it as missing).
func (a Agent) SkillDirs() []string {
	var out []string
	for _, sk := range a.Skills {
		if sk.Dir == "" {
			continue
		}
		if _, err := os.Stat(filepath.Join(sk.Dir, "SKILL.md")); err == nil {
			out = append(out, sk.Dir)
		}
	}
	return out
}

var agentSkillName = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,62}[a-z0-9])?$`)

// SetAgentSkills replaces the agent's own skills (ADR-0196 slice 4). A name
// appears once: a later entry of the same name replaces the earlier one.
func (s *Store) SetAgentSkills(id string, list []AgentSkill) (Agent, error) {
	if _, err := s.GetAgent(id); err != nil {
		return Agent{}, err
	}
	out := make([]AgentSkill, 0, len(list))
	at := map[string]int{}
	for _, sk := range list {
		sk.Name = strings.TrimSpace(sk.Name)
		if !agentSkillName.MatchString(sk.Name) || strings.Contains(sk.Name, "--") {
			return Agent{}, invalidError{"A skill name is lowercase letters, digits and single hyphens."}
		}
		sk.Missing = false
		if sk.Dir == "" || !filepath.IsAbs(sk.Dir) {
			return Agent{}, invalidError{"A skill needs its cached folder."}
		}
		if i, ok := at[sk.Name]; ok {
			out[i] = sk
			continue
		}
		at[sk.Name] = len(out)
		out = append(out, sk)
	}
	if len(out) > 32 {
		return Agent{}, invalidError{"An agent carries at most 32 skills of its own."}
	}
	raw, err := json.Marshal(out)
	if err != nil {
		return Agent{}, err
	}
	if _, err := s.db.Exec(`UPDATE agents SET skills=? WHERE id=?`, string(raw), id); err != nil {
		return Agent{}, fmt.Errorf("store: agent skills: %w", err)
	}
	return s.agentChanged(id)
}

// agentChanged reloads the row and announces agent.updated.
func (s *Store) agentChanged(id string) (Agent, error) {
	a, err := s.GetAgent(id)
	if err != nil {
		return Agent{}, err
	}
	s.note("agent.updated", &id, &a.WorkspaceID, a)
	return a, nil
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

// AddAgent creates a Pi agent in a workspace (use FreeWorkspaceID for unbound).
func (s *Store) AddAgent(workspaceID, name, workPath string) (Agent, error) {
	if stringsTrimSpace(name) == "" {
		return Agent{}, fmt.Errorf("store: name is required")
	}
	return s.AddAgentWithCLI(workspaceID, CLIPi, name, workPath)
}

// AddAgentWithCLI creates an agent for a launchable catalog CLI (ADR-0160).
// Empty cli is Pi. Name may be empty: the catalog name is used.
func (s *Store) AddAgentWithCLI(workspaceID, cli, name, workPath string) (Agent, error) {
	return s.addAgent("", workspaceID, cli, name, workPath)
}

// AddAgentAs creates an agent under a chosen id: a removed agent brought
// back keeps the id it had (ADR-0205), so what still names it — its Pi
// session folder, automations, pins — points at it again. The id must be
// free: ErrConflict when a living agent holds it.
func (s *Store) AddAgentAs(id, workspaceID, cli, name, workPath string) (Agent, error) {
	id = stringsTrimSpace(id)
	if id == "" {
		return Agent{}, invalid("an agent id is required")
	}
	if _, err := s.GetAgent(id); err == nil {
		return Agent{}, fmt.Errorf("%w: agent %s exists", ErrConflict, id)
	} else if !errors.Is(err, ErrNotFound) {
		return Agent{}, err
	}
	return s.addAgent(id, workspaceID, cli, name, workPath)
}

func (s *Store) addAgent(id, workspaceID, cli, name, workPath string) (Agent, error) {
	cli, err := normalizeAgentCLI(cli)
	if err != nil {
		return Agent{}, err
	}
	name = stringsTrimSpace(name)
	if name == "" {
		if e, ok := clilaunch.Find(cli); ok {
			name = e.Name
		}
	}
	if name == "" {
		return Agent{}, fmt.Errorf("store: name is required")
	}
	if workspaceID == "" {
		workspaceID = FreeWorkspaceID
	}
	if _, err := s.GetWorkspace(workspaceID); err != nil {
		return Agent{}, err
	}
	if id == "" {
		id = newID(name, "agent")
	}
	a := Agent{
		ID:          id,
		WorkspaceID: workspaceID,
		Name:        name,
		CLI:         cli,
		CreatedAt:   nowUTC(),
		LastStatus:  StatusNeverStarted,
		WorkPath:    emptyToNil(workPath),
	}
	if _, err := s.db.Exec(`INSERT INTO agents (id, workspace_id, name, created_at, last_status, work_path, cli, position) VALUES (?, ?, ?, ?, ?, ?, ?, `+nextPositionExpr("agents", "workspace_id = ?")+`)`,
		a.ID, a.WorkspaceID, a.Name, a.CreatedAt, a.LastStatus, a.WorkPath, a.CLI, a.WorkspaceID); err != nil {
		return Agent{}, fmt.Errorf("store: insert agent: %w", err)
	}
	a, err = s.GetAgent(a.ID)
	if err != nil {
		return Agent{}, err
	}
	s.note("agent.added", &a.ID, &a.WorkspaceID, a)
	return a, nil
}

// ListAgents returns agents in a workspace in sidebar order (ADR-0173).
// DefaultAgent stays the oldest; this list is the one the sidebar shows.
func (s *Store) ListAgents(workspaceID string) ([]Agent, error) {
	rows, err := s.db.Query(`SELECT `+agentCols+` FROM agents WHERE workspace_id = ? ORDER BY position, id`, workspaceID)
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

// DeleteAgent removes one agent without an exit record: the rollbacks of a
// launch, a handoff or an adoption, where nobody decided anything. A
// person's removal goes through RemoveAgentWithExit (ADR-0194). Workspace
// is kept.
func (s *Store) DeleteAgent(id string) error {
	_, err := s.removeAgent(id, nil)
	return err
}

func isUniqueConstraint(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique constraint") || strings.Contains(msg, "constraint failed")
}
