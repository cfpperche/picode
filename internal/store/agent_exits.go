package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/cfpperche/picode/internal/clilaunch"
)

// Exit records (ADR-0194). A person's removal of an agent writes one row in
// the same transaction as the delete: who the agent was, how it was set up,
// what PiCode observed, where its sessions were, and the person's answer.

// ExitTaxonomyVersion is stamped on every row, so a later version can add
// codes and still read these.
const ExitTaxonomyVersion = 1

// Exit outcomes (the person's answer; "" = no answer).
const (
	ExitResolved   = "resolved"
	ExitPartial    = "partial"
	ExitUnresolved = "unresolved"
	ExitTrial      = "trial"
)

// Why the question was not asked.
const (
	ExitSkipOff    = "off"    // the person turned the question off
	ExitSkipIdle   = "idle"   // the agent never worked
	ExitSkipBrief  = "brief"  // it lived less than ExitAskMinLifetime
	ExitSkipClient = "client" // the server would have asked; the caller did not
)

// ExitAskMinLifetime is the shortest life worth a question (Claude Code's
// /insights skips sessions under a minute too).
const ExitAskMinLifetime = time.Minute

// Exit origins: which face removed the agent.
const (
	ExitFromDesktop = "desktop"
	ExitFromMobile  = "mobile"
	ExitFromAPI     = "api"
)

const maxExitNote = 1000

// exitAskSetting is absent or "1" for on, "0" for off.
const exitAskSetting = "agent_exits.ask"

// ExitChoice is one code with the words the clients show for it.
type ExitChoice struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Hint  string `json:"hint,omitempty"`
}

// ExitTaxonomy is what the clients render; the server validates against it.
type ExitTaxonomy struct {
	Version  int          `json:"version"`
	Outcomes []ExitChoice `json:"outcomes"`
	Reasons  []ExitChoice `json:"reasons"`
	// ReasonsFor names the outcomes that ask for reasons.
	ReasonsFor []string `json:"reasonsFor"`
}

var exitOutcomes = []ExitChoice{
	{ID: ExitResolved, Label: "Resolved"},
	{ID: ExitPartial, Label: "Partly"},
	{ID: ExitUnresolved, Label: "Didn't resolve"},
	{ID: ExitTrial, Label: "Just trying"},
}

var exitReasons = []ExitChoice{
	{ID: "setup", Label: "Wrong setup", Hint: "Model, CLI, permissions, connectors or folder"},
	{ID: "misunderstood", Label: "Misunderstood the task", Hint: "Or took the wrong approach"},
	{ID: "stuck", Label: "Got stuck or looped"},
	{ID: "false_done", Label: "Said done when it wasn't"},
	{ID: "slow_costly", Label: "Too slow or too costly"},
	{ID: "switched", Label: "Switched to another agent"},
}

// ExitTaxonomyV1 returns a copy of the version-1 taxonomy.
func ExitTaxonomyV1() ExitTaxonomy {
	return ExitTaxonomy{
		Version:    ExitTaxonomyVersion,
		Outcomes:   append([]ExitChoice(nil), exitOutcomes...),
		Reasons:    append([]ExitChoice(nil), exitReasons...),
		ReasonsFor: []string{ExitPartial, ExitUnresolved},
	}
}

func exitHasReasons(outcome string) bool {
	return outcome == ExitPartial || outcome == ExitUnresolved
}

func exitKnown(list []ExitChoice, id string) bool {
	for _, c := range list {
		if c.ID == id {
			return true
		}
	}
	return false
}

// ExitLabel is the person's answer.
type ExitLabel struct {
	Outcome string   `json:"outcome"`
	Reasons []string `json:"reasons"`
	Note    string   `json:"note"`
}

// normalizeExitLabel validates codes against the taxonomy: reasons only
// after an outcome that asks for them, each once, in taxonomy order.
func normalizeExitLabel(l ExitLabel) (ExitLabel, error) {
	out := ExitLabel{Outcome: strings.TrimSpace(l.Outcome), Reasons: []string{}, Note: strings.TrimSpace(l.Note)}
	if out.Outcome != "" && !exitKnown(exitOutcomes, out.Outcome) {
		return ExitLabel{}, invalid("unknown outcome %q", out.Outcome)
	}
	seen := map[string]bool{}
	for _, r := range l.Reasons {
		r = strings.TrimSpace(r)
		if !exitKnown(exitReasons, r) {
			return ExitLabel{}, invalid("unknown reason %q", r)
		}
		seen[r] = true
	}
	if len(seen) > 0 && !exitHasReasons(out.Outcome) {
		return ExitLabel{}, invalid("reasons need the outcome %q or %q", ExitPartial, ExitUnresolved)
	}
	for _, c := range exitReasons {
		if seen[c.ID] {
			out.Reasons = append(out.Reasons, c.ID)
		}
	}
	if utf8.RuneCountInString(out.Note) > maxExitNote {
		return ExitLabel{}, invalid("the note is longer than %d characters", maxExitNote)
	}
	return out, nil
}

// ExitInput is what the removal adds to what the store reads itself.
type ExitInput struct {
	Origin         string
	Asked          bool
	AskSkip        string
	Label          ExitLabel
	SessionsPurged bool
	WorkPurged     bool
}

// ExitLaunch is a CLI agent's terminal launch record, without secrets:
// the overrides keep the names of environment variables, never values.
type ExitLaunch struct {
	Executable   string              `json:"executable,omitempty"`
	Args         []string            `json:"args,omitempty"`
	EnvKeys      []string            `json:"envKeys,omitempty"`
	RemovedEnv   []string            `json:"removedEnv,omitempty"`
	Path         []string            `json:"path,omitempty"`
	Integration  *bool               `json:"integration,omitempty"`
	Tools        []string            `json:"tools,omitempty"`
	Applied      *clilaunch.Snapshot `json:"applied,omitempty"`
	HasOverrides bool                `json:"hasOverrides"`
}

// ExitConfig is the frozen setup beside the promoted provider/model.
type ExitConfig struct {
	Thinking         string      `json:"thinking,omitempty"`
	OpMode           string      `json:"opMode,omitempty"`
	Checklist        string      `json:"checklist,omitempty"`
	Packages         []string    `json:"packages"`
	PackagesIsolated bool        `json:"packagesIsolated"`
	ExtraPrompt      string      `json:"extraPrompt,omitempty"`
	WorkPath         string      `json:"workPath,omitempty"`
	Launch           *ExitLaunch `json:"launch,omitempty"`
}

// ExitSignals is what PiCode observed of the agent's life.
type ExitSignals struct {
	LastStatus     string  `json:"lastStatus"`
	LastStatusAt   *string `json:"lastStatusAt,omitempty"`
	LastStartedAt  *string `json:"lastStartedAt,omitempty"`
	InboxItems     int     `json:"inboxItems"`
	InboxBlocking  int     `json:"inboxBlocking"`
	ChecklistDone  int     `json:"checklistDone"`
	ChecklistTotal int     `json:"checklistTotal"`
}

// ExitSessions points at where the agent's sessions were.
type ExitSessions struct {
	PiSessionPath  string `json:"piSessionPath,omitempty"`
	CLISessionID   string `json:"cliSessionId,omitempty"`
	CLISessionPath string `json:"cliSessionPath,omitempty"`
}

// AgentExit is one row of agent_exits.
type AgentExit struct {
	ID              string       `json:"id"`
	AgentID         string       `json:"agentId"`
	AgentName       string       `json:"agentName"`
	WorkspaceID     string       `json:"workspaceId"`
	WorkspaceName   string       `json:"workspaceName"`
	CLI             string       `json:"cli"`
	Provider        string       `json:"provider"`
	Model           string       `json:"model"`
	Config          ExitConfig   `json:"config"`
	CreatedAt       string       `json:"createdAt"`
	RemovedAt       string       `json:"removedAt"`
	LifetimeS       int64        `json:"lifetimeS"`
	Turns           *int64       `json:"turns"`
	FirstWorkedAt   *string      `json:"firstWorkedAt"`
	LastWorkedAt    *string      `json:"lastWorkedAt"`
	Signals         ExitSignals  `json:"signals"`
	Sessions        ExitSessions `json:"sessions"`
	SessionsPurged  bool         `json:"sessionsPurged"`
	WorkPurged      bool         `json:"workPurged"`
	Origin          string       `json:"origin"`
	Asked           bool         `json:"asked"`
	AskSkip         string       `json:"askSkip"`
	Outcome         string       `json:"outcome"`
	Reasons         []string     `json:"reasons"`
	Note            string       `json:"note"`
	LabeledAt       *string      `json:"labeledAt"`
	Taxonomy        int          `json:"taxonomy"`
	UndoneAt        *string      `json:"undoneAt"`
	RestoredAgentID string       `json:"restoredAgentId,omitempty"`
}

// AgentActivity is the agent's own work record (ADR-0194).
type AgentActivity struct {
	Turns         *int64  `json:"turns"` // nil = not measured
	FirstWorkedAt *string `json:"firstWorkedAt"`
	LastWorkedAt  *string `json:"lastWorkedAt"`
}

// Worked is true when the agent is known to have worked, or when it predates
// the counter: an unmeasured agent gets the benefit of the doubt.
func (a AgentActivity) Worked() bool {
	return a.Turns == nil || *a.Turns > 0 || a.LastWorkedAt != nil
}

// AgentActivityOf reads an agent's turn counter.
func (s *Store) AgentActivityOf(id string) (AgentActivity, error) {
	return agentActivityOf(s.db, id)
}

func agentActivityOf(q txRunner, id string) (AgentActivity, error) {
	var v AgentActivity
	var turns sql.NullInt64
	var first, last sql.NullString
	err := q.QueryRow(`SELECT turns, first_worked_at, last_worked_at FROM agents WHERE id = ?`, id).Scan(&turns, &first, &last)
	if errors.Is(err, sql.ErrNoRows) {
		return v, ErrNotFound
	}
	if err != nil {
		return v, fmt.Errorf("store: agent activity: %w", err)
	}
	if turns.Valid {
		n := turns.Int64
		v.Turns = &n
	}
	v.FirstWorkedAt = nullStr(first)
	v.LastWorkedAt = nullStr(last)
	return v, nil
}

// NoteAgentTurn counts one transition into work. An agent born before the
// counter stays "not measured"; its worked times still move.
func (s *Store) NoteAgentTurn(id string) error {
	now := nowUTC()
	res, err := s.db.Exec(`UPDATE agents SET turns = CASE WHEN turns IS NULL THEN NULL ELSE turns + 1 END,
		first_worked_at = COALESCE(first_worked_at, ?), last_worked_at = ? WHERE id = ?`, now, now, id)
	if err != nil {
		return fmt.Errorf("store: agent turn: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// ExitAskDecision says whether removing this agent asks the question and,
// when it does not, why (ADR-0194's decision table).
func ExitAskDecision(askOn bool, act AgentActivity, createdAt string, now time.Time) (bool, string) {
	if !askOn {
		return false, ExitSkipOff
	}
	if !act.Worked() {
		return false, ExitSkipIdle
	}
	if born, err := time.Parse(time.RFC3339Nano, createdAt); err == nil && now.Sub(born) < ExitAskMinLifetime {
		return false, ExitSkipBrief
	}
	return true, ""
}

// ExitAskOn reads the person's "ask when removing agents" switch.
func (s *Store) ExitAskOn() (bool, error) {
	v, ok, err := s.GetSetting(exitAskSetting)
	if err != nil {
		return true, err
	}
	return !ok || v != "0", nil
}

// SetExitAskOn turns the question on or off.
func (s *Store) SetExitAskOn(on bool) error {
	v := "1"
	if !on {
		v = "0"
	}
	return s.SetSetting(exitAskSetting, v)
}

// RemoveAgentWithExit removes the agent and writes its exit in one
// transaction. The bound terminal goes after the commit, as with DeleteAgent.
func (s *Store) RemoveAgentWithExit(id string, in ExitInput) (AgentExit, error) {
	ex, err := s.removeAgent(id, &in)
	if err != nil {
		return AgentExit{}, err
	}
	return *ex, nil
}

// removeAgent is the one removal path. in == nil removes without an exit:
// rollbacks of a launch, a handoff or an adoption, where nobody decided.
func (s *Store) removeAgent(id string, in *ExitInput) (*AgentExit, error) {
	a, err := s.GetAgent(id)
	if err != nil {
		return nil, err
	}
	var ex *AgentExit
	if in != nil {
		built, err := s.buildExit(a, *in, time.Now())
		if err != nil {
			return nil, err
		}
		ex = &built
	}
	// An open needs-you prompt names this agent (ADR-0160 Fatia E); with
	// the agent gone it has no addressee, so it closes.
	if open, err := s.ActiveInboxBySourceReason(InboxFromAgent, id, InboxNeedsYouReason); err == nil {
		for _, it := range open {
			_, _ = s.SetInboxItemState(it.ID, InboxDone, nil)
		}
	}
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	if ex != nil {
		if err := insertExitTx(tx, *ex); err != nil {
			s.rollback(tx)
			return nil, err
		}
		if err := s.AppendEventTx(tx, "agent_exit.recorded", nil, nil, ex); err != nil {
			s.rollback(tx)
			return nil, err
		}
	}
	if _, err := tx.Exec(`DELETE FROM agent_checklists WHERE agent_id = ?`, id); err != nil {
		s.rollback(tx)
		return nil, fmt.Errorf("store: delete agent checklist: %w", err)
	}
	res, err := tx.Exec(`DELETE FROM agents WHERE id = ?`, id)
	if err != nil {
		s.rollback(tx)
		return nil, fmt.Errorf("store: delete agent: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		s.rollback(tx)
		return nil, ErrNotFound
	}
	if err := s.AppendEventTx(tx, "agent.deleted", nil, nil, idData(id)); err != nil {
		s.rollback(tx)
		return nil, err
	}
	if err := s.commit(tx); err != nil {
		return nil, err
	}
	if a.TerminalID != nil && strings.TrimSpace(*a.TerminalID) != "" {
		_ = s.DeleteTerminal(*a.TerminalID)
	}
	return ex, nil
}

// buildExit reads everything the exit freezes. It runs before the
// transaction: the store has one connection, and a query on s.db inside
// a transaction would wait on itself.
func (s *Store) buildExit(a Agent, in ExitInput, now time.Time) (AgentExit, error) {
	label, err := normalizeExitLabel(in.Label)
	if err != nil {
		return AgentExit{}, err
	}
	act, err := s.AgentActivityOf(a.ID)
	if err != nil {
		return AgentExit{}, err
	}
	removed := now.UTC()
	ex := AgentExit{
		ID:             newID(a.Name, "exit"),
		AgentID:        a.ID,
		AgentName:      a.Name,
		WorkspaceID:    a.WorkspaceID,
		CLI:            a.CLI,
		Provider:       deref(a.Provider),
		Model:          deref(a.Model),
		CreatedAt:      a.CreatedAt,
		RemovedAt:      removed.Format(time.RFC3339Nano),
		Turns:          act.Turns,
		FirstWorkedAt:  act.FirstWorkedAt,
		LastWorkedAt:   act.LastWorkedAt,
		SessionsPurged: in.SessionsPurged,
		WorkPurged:     in.WorkPurged,
		Origin:         exitOrigin(in.Origin),
		Asked:          in.Asked,
		Outcome:        label.Outcome,
		Reasons:        label.Reasons,
		Note:           label.Note,
		Taxonomy:       ExitTaxonomyVersion,
	}
	if !in.Asked {
		ex.AskSkip = in.AskSkip
	}
	if label.Outcome != "" {
		ex.LabeledAt = &ex.RemovedAt
	}
	if born, err := time.Parse(time.RFC3339Nano, a.CreatedAt); err == nil && removed.After(born) {
		ex.LifetimeS = int64(removed.Sub(born) / time.Second)
	}
	if ws, err := s.GetWorkspace(a.WorkspaceID); err == nil {
		ex.WorkspaceName = ws.Name
	}
	ex.Config = ExitConfig{
		Thinking:         deref(a.Thinking),
		OpMode:           deref(a.OpMode),
		Checklist:        a.Checklist,
		Packages:         append([]string{}, a.Packages...),
		PackagesIsolated: a.PackagesIsolated,
		ExtraPrompt:      deref(a.ExtraPrompt),
		WorkPath:         deref(a.WorkPath),
	}
	ex.Signals = ExitSignals{LastStatus: a.LastStatus, LastStatusAt: a.LastStatusAt, LastStartedAt: a.LastStartedAt}
	ex.Sessions.PiSessionPath = deref(a.SessionPath)
	if a.TerminalID != nil && strings.TrimSpace(*a.TerminalID) != "" {
		if l, err := s.TerminalLaunch(*a.TerminalID); err == nil && l != nil {
			ex.Config.Launch = exitLaunchOf(l)
			if l.LastSession != nil {
				ex.Sessions.CLISessionID = l.LastSession.SessionID
				ex.Sessions.CLISessionPath = l.LastSession.Path
			}
		}
	}
	if err := s.db.QueryRow(`SELECT COUNT(1), COALESCE(SUM(blocking), 0) FROM inbox_items WHERE source_kind = ? AND source_id = ?`,
		InboxFromAgent, a.ID).Scan(&ex.Signals.InboxItems, &ex.Signals.InboxBlocking); err != nil {
		return AgentExit{}, fmt.Errorf("store: exit inbox: %w", err)
	}
	var items string
	err = s.db.QueryRow(`SELECT items FROM agent_checklists WHERE agent_id = ?`, a.ID).Scan(&items)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return AgentExit{}, fmt.Errorf("store: exit checklist: %w", err)
	}
	if err == nil {
		var steps []struct {
			Status string `json:"status"`
		}
		if json.Unmarshal([]byte(items), &steps) == nil {
			ex.Signals.ChecklistTotal = len(steps)
			for _, st := range steps {
				if st.Status == "completed" {
					ex.Signals.ChecklistDone++
				}
			}
		}
	}
	return ex, nil
}

func exitOrigin(o string) string {
	switch o {
	case ExitFromDesktop, ExitFromMobile:
		return o
	}
	return ExitFromAPI
}

// exitLaunchOf copies a launch record without environment values: an
// override can carry a key, and the exit must never hold one.
func exitLaunchOf(l *TerminalLaunch) *ExitLaunch {
	o := l.Overrides
	out := &ExitLaunch{Applied: l.Applied, Integration: o.Integration}
	if o.Executable != nil {
		out.Executable = *o.Executable
	}
	if o.Args != nil {
		out.Args = append([]string{}, (*o.Args)...)
	}
	if o.Path != nil {
		out.Path = append([]string{}, (*o.Path)...)
	}
	if o.Tools != nil {
		out.Tools = append([]string{}, (*o.Tools)...)
	}
	for k, v := range o.Env {
		if v == nil {
			out.RemovedEnv = append(out.RemovedEnv, k)
		} else {
			out.EnvKeys = append(out.EnvKeys, k)
		}
	}
	sort.Strings(out.EnvKeys)
	sort.Strings(out.RemovedEnv)
	out.HasOverrides = o.Executable != nil || o.Args != nil || o.Path != nil || o.Tools != nil || o.Integration != nil || len(o.Env) > 0
	return out
}

const exitCols = `id, agent_id, agent_name, workspace_id, workspace_name, cli, provider, model, config, created_at, removed_at,
	lifetime_s, turns, first_worked_at, last_worked_at, signals, sessions, sessions_purged, work_purged, origin, asked, ask_skip,
	outcome, reasons, note, labeled_at, taxonomy, undone_at, restored_agent_id`

func insertExitTx(tx *sql.Tx, ex AgentExit) error {
	config, err := marshalJSON(ex.Config)
	if err != nil {
		return err
	}
	signals, err := marshalJSON(ex.Signals)
	if err != nil {
		return err
	}
	sessions, err := marshalJSON(ex.Sessions)
	if err != nil {
		return err
	}
	var turns any
	if ex.Turns != nil {
		turns = *ex.Turns
	}
	_, err = tx.Exec(`INSERT INTO agent_exits (`+exitCols+`) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		ex.ID, ex.AgentID, ex.AgentName, ex.WorkspaceID, ex.WorkspaceName, ex.CLI, ex.Provider, ex.Model, config, ex.CreatedAt, ex.RemovedAt,
		ex.LifetimeS, turns, orNull(ex.FirstWorkedAt), orNull(ex.LastWorkedAt), signals, sessions, boolInt(ex.SessionsPurged), boolInt(ex.WorkPurged),
		ex.Origin, boolInt(ex.Asked), ex.AskSkip, ex.Outcome, encodePackages(ex.Reasons), ex.Note, orNull(ex.LabeledAt), ex.Taxonomy,
		orNull(ex.UndoneAt), ex.RestoredAgentID)
	if err != nil {
		return fmt.Errorf("store: insert exit: %w", err)
	}
	return nil
}

func scanExit(row interface{ Scan(...any) error }) (AgentExit, error) {
	var ex AgentExit
	var config, signals, sessions, reasons string
	var turns sql.NullInt64
	var first, last, labeled, undone sql.NullString
	var purgedS, purgedW, asked int
	err := row.Scan(&ex.ID, &ex.AgentID, &ex.AgentName, &ex.WorkspaceID, &ex.WorkspaceName, &ex.CLI, &ex.Provider, &ex.Model, &config,
		&ex.CreatedAt, &ex.RemovedAt, &ex.LifetimeS, &turns, &first, &last, &signals, &sessions, &purgedS, &purgedW, &ex.Origin, &asked,
		&ex.AskSkip, &ex.Outcome, &reasons, &ex.Note, &labeled, &ex.Taxonomy, &undone, &ex.RestoredAgentID)
	if err != nil {
		return AgentExit{}, err
	}
	_ = json.Unmarshal([]byte(config), &ex.Config)
	if ex.Config.Packages == nil {
		ex.Config.Packages = []string{}
	}
	_ = json.Unmarshal([]byte(signals), &ex.Signals)
	_ = json.Unmarshal([]byte(sessions), &ex.Sessions)
	ex.Reasons = decodePackages(reasons)
	if turns.Valid {
		n := turns.Int64
		ex.Turns = &n
	}
	ex.FirstWorkedAt = nullStr(first)
	ex.LastWorkedAt = nullStr(last)
	ex.LabeledAt = nullStr(labeled)
	ex.UndoneAt = nullStr(undone)
	ex.SessionsPurged = purgedS != 0
	ex.WorkPurged = purgedW != 0
	ex.Asked = asked != 0
	return ex, nil
}

// GetAgentExit returns one exit.
func (s *Store) GetAgentExit(id string) (AgentExit, error) {
	ex, err := scanExit(s.db.QueryRow(`SELECT `+exitCols+` FROM agent_exits WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return AgentExit{}, ErrNotFound
	}
	if err != nil {
		return AgentExit{}, fmt.Errorf("store: get exit: %w", err)
	}
	return ex, nil
}

// ExitOutcomeUnanswered filters the exits with no answer.
const ExitOutcomeUnanswered = "unanswered"

// ExitFilter narrows the catalog. Undone exits are never listed.
type ExitFilter struct {
	WorkspaceID string
	CLI         string
	Outcome     string // an outcome code, ExitOutcomeUnanswered, or "" for all
	Since       string // removed_at >= Since (RFC 3339), "" for all
	Before      string // removed_at < Before, the page cursor
	Limit       int
}

func (f ExitFilter) where() (string, []any) {
	conds := []string{"undone_at IS NULL"}
	var args []any
	if f.WorkspaceID != "" {
		conds = append(conds, "workspace_id = ?")
		args = append(args, f.WorkspaceID)
	}
	if f.CLI != "" {
		conds = append(conds, "cli = ?")
		args = append(args, f.CLI)
	}
	switch f.Outcome {
	case "":
	case ExitOutcomeUnanswered:
		conds = append(conds, "outcome = ''")
	default:
		conds = append(conds, "outcome = ?")
		args = append(args, f.Outcome)
	}
	if f.Since != "" {
		conds = append(conds, "removed_at >= ?")
		args = append(args, f.Since)
	}
	if f.Before != "" {
		conds = append(conds, "removed_at < ?")
		args = append(args, f.Before)
	}
	return " WHERE " + strings.Join(conds, " AND "), args
}

// ListAgentExits returns the catalog, newest first.
func (s *Store) ListAgentExits(f ExitFilter) ([]AgentExit, error) {
	if f.Limit <= 0 || f.Limit > 200 {
		f.Limit = 50
	}
	where, args := f.where()
	rows, err := s.db.Query(`SELECT `+exitCols+` FROM agent_exits`+where+` ORDER BY removed_at DESC, id DESC LIMIT ?`, append(args, f.Limit)...)
	if err != nil {
		return nil, fmt.Errorf("store: list exits: %w", err)
	}
	defer rows.Close()
	out := []AgentExit{}
	for rows.Next() {
		ex, err := scanExit(rows)
		if err != nil {
			return nil, fmt.Errorf("store: scan exit: %w", err)
		}
		out = append(out, ex)
	}
	return out, rows.Err()
}

// EachAgentExit walks every exit, undone ones included, oldest first: the
// export hands the person the whole record.
func (s *Store) EachAgentExit(fn func(AgentExit) error) error {
	rows, err := s.db.Query(`SELECT ` + exitCols + ` FROM agent_exits ORDER BY removed_at, id`)
	if err != nil {
		return fmt.Errorf("store: export exits: %w", err)
	}
	var all []AgentExit
	for rows.Next() {
		ex, err := scanExit(rows)
		if err != nil {
			rows.Close()
			return fmt.Errorf("store: scan exit: %w", err)
		}
		all = append(all, ex)
	}
	if err := rows.Close(); err != nil {
		return err
	}
	// The callback writes to a response; the rows are closed first so a
	// slow reader never holds the store's one connection.
	for _, ex := range all {
		if err := fn(ex); err != nil {
			return err
		}
	}
	return nil
}

// LabelAgentExit sets (or clears) the person's answer after the removal.
func (s *Store) LabelAgentExit(id string, l ExitLabel) (AgentExit, error) {
	label, err := normalizeExitLabel(l)
	if err != nil {
		return AgentExit{}, err
	}
	var labeled any
	if label.Outcome != "" {
		labeled = nowUTC()
	}
	return s.updateExit(id, `outcome = ?, reasons = ?, note = ?, labeled_at = ?`,
		label.Outcome, encodePackages(label.Reasons), label.Note, labeled)
}

// MarkAgentExitUndone records that an Undo brought the agent back; the exit
// stays for the record and leaves every count.
func (s *Store) MarkAgentExitUndone(id, restoredAgentID string) (AgentExit, error) {
	return s.updateExit(id, `undone_at = COALESCE(undone_at, ?), restored_agent_id = ?`, nowUTC(), strings.TrimSpace(restoredAgentID))
}

func (s *Store) updateExit(id, set string, args ...any) (AgentExit, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return AgentExit{}, err
	}
	res, err := tx.Exec(`UPDATE agent_exits SET `+set+` WHERE id = ?`, append(args, id)...)
	if err != nil {
		s.rollback(tx)
		return AgentExit{}, fmt.Errorf("store: update exit: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		s.rollback(tx)
		return AgentExit{}, ErrNotFound
	}
	ex, err := scanExit(tx.QueryRow(`SELECT `+exitCols+` FROM agent_exits WHERE id = ?`, id))
	if err != nil {
		s.rollback(tx)
		return AgentExit{}, fmt.Errorf("store: reread exit: %w", err)
	}
	if err := s.AppendEventTx(tx, "agent_exit.updated", nil, nil, ex); err != nil {
		s.rollback(tx)
		return AgentExit{}, err
	}
	if err := s.commit(tx); err != nil {
		return AgentExit{}, err
	}
	return ex, nil
}

// DeleteAgentExit removes one exit from the catalog.
func (s *Store) DeleteAgentExit(id string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	res, err := tx.Exec(`DELETE FROM agent_exits WHERE id = ?`, id)
	if err != nil {
		s.rollback(tx)
		return fmt.Errorf("store: delete exit: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		s.rollback(tx)
		return ErrNotFound
	}
	if err := s.AppendEventTx(tx, "agent_exit.deleted", nil, nil, idData(id)); err != nil {
		s.rollback(tx)
		return err
	}
	return s.commit(tx)
}

// ExitCLIRow is one CLI's outcomes.
type ExitCLIRow struct {
	CLI        string `json:"cli"`
	Total      int    `json:"total"`
	Resolved   int    `json:"resolved"`
	Partial    int    `json:"partial"`
	Unresolved int    `json:"unresolved"`
	Trial      int    `json:"trial"`
	Unanswered int    `json:"unanswered"`
}

// ExitCount is one reason's count.
type ExitCount struct {
	ID    string `json:"id"`
	Count int    `json:"count"`
}

// ExitSummary is the catalog's numbers for a window.
type ExitSummary struct {
	Total    int            `json:"total"`
	Asked    int            `json:"asked"`
	Answered int            `json:"answered"`
	Outcomes map[string]int `json:"outcomes"`
	ByCLI    []ExitCLIRow   `json:"byCli"`
	Reasons  []ExitCount    `json:"reasons"`
	// MedianLifetimeS is over every exit in the window; MedianTurns only
	// over the exits whose turns were measured (nil when none were).
	MedianLifetimeS int64    `json:"medianLifetimeS"`
	MedianTurns     *float64 `json:"medianTurns"`
	MeasuredTurns   int      `json:"measuredTurns"`
}

// AgentExitSummary counts the catalog; f.Limit and f.Before are ignored.
func (s *Store) AgentExitSummary(f ExitFilter) (ExitSummary, error) {
	f.Before, f.Limit = "", 0
	where, args := f.where()
	rows, err := s.db.Query(`SELECT cli, asked, outcome, reasons, lifetime_s, turns FROM agent_exits`+where, args...)
	if err != nil {
		return ExitSummary{}, fmt.Errorf("store: exit summary: %w", err)
	}
	defer rows.Close()
	sum := ExitSummary{Outcomes: map[string]int{ExitResolved: 0, ExitPartial: 0, ExitUnresolved: 0, ExitTrial: 0, ExitOutcomeUnanswered: 0}}
	byCLI := map[string]*ExitCLIRow{}
	reasons := map[string]int{}
	var lifetimes []int64
	var turns []int64
	for rows.Next() {
		var cli, outcome, rs string
		var asked int
		var life int64
		var t sql.NullInt64
		if err := rows.Scan(&cli, &asked, &outcome, &rs, &life, &t); err != nil {
			return ExitSummary{}, fmt.Errorf("store: exit summary: %w", err)
		}
		sum.Total++
		if asked != 0 {
			sum.Asked++
		}
		row := byCLI[cli]
		if row == nil {
			row = &ExitCLIRow{CLI: cli}
			byCLI[cli] = row
		}
		row.Total++
		switch outcome {
		case ExitResolved:
			row.Resolved++
		case ExitPartial:
			row.Partial++
		case ExitUnresolved:
			row.Unresolved++
		case ExitTrial:
			row.Trial++
		default:
			row.Unanswered++
		}
		if outcome == "" {
			sum.Outcomes[ExitOutcomeUnanswered]++
		} else {
			sum.Answered++
			sum.Outcomes[outcome]++
		}
		for _, r := range decodePackages(rs) {
			reasons[r]++
		}
		lifetimes = append(lifetimes, life)
		if t.Valid {
			turns = append(turns, t.Int64)
		}
	}
	if err := rows.Err(); err != nil {
		return ExitSummary{}, err
	}
	for _, row := range byCLI {
		sum.ByCLI = append(sum.ByCLI, *row)
	}
	sort.Slice(sum.ByCLI, func(i, j int) bool {
		if sum.ByCLI[i].Total != sum.ByCLI[j].Total {
			return sum.ByCLI[i].Total > sum.ByCLI[j].Total
		}
		return sum.ByCLI[i].CLI < sum.ByCLI[j].CLI
	})
	if sum.ByCLI == nil {
		sum.ByCLI = []ExitCLIRow{}
	}
	sum.Reasons = []ExitCount{}
	for _, c := range exitReasons {
		if n := reasons[c.ID]; n > 0 {
			sum.Reasons = append(sum.Reasons, ExitCount{ID: c.ID, Count: n})
		}
	}
	sort.SliceStable(sum.Reasons, func(i, j int) bool { return sum.Reasons[i].Count > sum.Reasons[j].Count })
	if len(lifetimes) > 0 {
		sort.Slice(lifetimes, func(i, j int) bool { return lifetimes[i] < lifetimes[j] })
		sum.MedianLifetimeS = int64(medianOf(lifetimes))
	}
	sum.MeasuredTurns = len(turns)
	if len(turns) > 0 {
		sort.Slice(turns, func(i, j int) bool { return turns[i] < turns[j] })
		m := medianOf(turns)
		sum.MedianTurns = &m
	}
	return sum, nil
}

func medianOf(sorted []int64) float64 {
	n := len(sorted)
	if n%2 == 1 {
		return float64(sorted[n/2])
	}
	return float64(sorted[n/2-1]+sorted[n/2]) / 2
}

func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func nullStr(v sql.NullString) *string {
	if !v.Valid {
		return nil
	}
	s := v.String
	return &s
}
