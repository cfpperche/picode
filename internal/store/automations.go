package store

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/cron"
)

// Automations (ADR-0046): trigger(s) + prompt + bounds. Every run is an
// ordinary pi session on the automation's own agent (action=start) or a
// follow_up queued into an existing agent (action=message).

// Actions.
const (
	AutomationStart   = "start"
	AutomationMessage = "message"
)

// Run triggers.
const (
	TriggerSchedule = "schedule"
	TriggerWebhook  = "webhook"
	TriggerManual   = "manual"
	TriggerCatchUp  = "catch-up"
)

// Run statuses.
const (
	RunRunning = "running"
	RunDone    = "done"
	RunFailed  = "failed"
	RunSkipped = "skipped"
)

const (
	maxAutomationName   = 60
	maxAutomationPrompt = 100_000
)

// Automation is one row. Webhook is derived from webhook_hash; the secret
// itself is never stored or returned after creation.
type Automation struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	Enabled          bool     `json:"enabled"`
	WorkspaceID      string   `json:"workspaceId"`
	Action           string   `json:"action"`
	TargetAgentID    *string  `json:"targetAgentId,omitempty"`
	AgentID          *string  `json:"agentId,omitempty"`
	Prompt           string   `json:"prompt"`
	Provider         *string  `json:"provider"`
	Model            *string  `json:"model"`
	Thinking         *string  `json:"thinking"`
	Cron             *string  `json:"cron"`
	Webhook          bool     `json:"webhook"`
	MaxCostUSD       *float64 `json:"maxCostUsd"`
	MaxRuns          *int     `json:"maxRuns"`
	MaxRunsWindowMin *int     `json:"maxRunsWindowMin"`
	LastFiredAt      *string  `json:"lastFiredAt,omitempty"`
	CreatedAt        string   `json:"createdAt"`
	UpdatedAt        string   `json:"updatedAt"`

	webhookHash *string
}

// AutomationParams is CreateAutomation's input. Zero values mean
// "unset"; MaxRuns and MaxRunsWindowMin go together.
type AutomationParams struct {
	Name             string
	WorkspaceID      string
	Action           string
	TargetAgentID    string
	Prompt           string
	Provider         string
	Model            string
	Thinking         string
	Cron             string
	Webhook          bool
	MaxCostUSD       float64
	MaxRuns          int
	MaxRunsWindowMin int
}

// AutomationPatch is a partial update. Nil = unchanged. For nullable
// columns an empty string / zero clears (as AgentPatch does).
type AutomationPatch struct {
	Name             *string
	Enabled          *bool
	WorkspaceID      *string
	Action           *string
	TargetAgentID    *string
	Prompt           *string
	Provider         *string
	Model            *string
	Thinking         *string
	Cron             *string
	MaxCostUSD       *float64
	MaxRuns          *int
	MaxRunsWindowMin *int
}

// Run is one invocation. Skipped rows record why nothing ran.
type Run struct {
	ID           string  `json:"id"`
	AutomationID string  `json:"automationId"`
	Trigger      string  `json:"trigger"`
	Status       string  `json:"status"`
	Reason       string  `json:"reason,omitempty"`
	SessionPath  *string `json:"sessionPath,omitempty"`
	CostUSD      float64 `json:"costUsd"`
	FiredAt      string  `json:"firedAt"`
	FinishedAt   *string `json:"finishedAt,omitempty"`
}

const automationCols = `id, name, enabled, workspace_id, action, target_agent_id, agent_id, prompt,
	provider, model, thinking, cron, webhook_hash, max_cost_usd, max_runs, max_runs_window_min,
	last_fired_at, created_at, updated_at`

func scanAutomation(row interface{ Scan(...any) error }, a *Automation) error {
	var enabled int
	if err := row.Scan(&a.ID, &a.Name, &enabled, &a.WorkspaceID, &a.Action, &a.TargetAgentID, &a.AgentID,
		&a.Prompt, &a.Provider, &a.Model, &a.Thinking, &a.Cron, &a.webhookHash, &a.MaxCostUSD,
		&a.MaxRuns, &a.MaxRunsWindowMin, &a.LastFiredAt, &a.CreatedAt, &a.UpdatedAt); err != nil {
		return err
	}
	a.Enabled = enabled != 0
	a.Webhook = a.webhookHash != nil && *a.webhookHash != ""
	return nil
}

const runCols = `id, automation_id, trigger, status, reason, session_path, cost_usd, fired_at, finished_at`

func scanRun(row interface{ Scan(...any) error }, r *Run) error {
	return row.Scan(&r.ID, &r.AutomationID, &r.Trigger, &r.Status, &r.Reason, &r.SessionPath,
		&r.CostUSD, &r.FiredAt, &r.FinishedAt)
}

// validateAutomation checks the invariants shared by create and update.
func validateAutomation(a Automation) error {
	a.Name = strings.TrimSpace(a.Name)
	if a.Name == "" {
		return fmt.Errorf("name is required")
	}
	if len(a.Name) > maxAutomationName {
		return fmt.Errorf("name is longer than %d characters", maxAutomationName)
	}
	switch a.Action {
	case AutomationStart:
	case AutomationMessage:
		if a.TargetAgentID == nil || strings.TrimSpace(*a.TargetAgentID) == "" {
			return fmt.Errorf("a message automation needs a target agent")
		}
	default:
		return fmt.Errorf("action must be start or message")
	}
	if strings.TrimSpace(a.Prompt) == "" {
		return fmt.Errorf("prompt is required")
	}
	if len(a.Prompt) > maxAutomationPrompt {
		return fmt.Errorf("prompt is too long")
	}
	if a.Cron != nil {
		if _, err := cron.Parse(*a.Cron); err != nil {
			return err
		}
	}
	if a.Cron == nil && !a.Webhook {
		return fmt.Errorf("an automation needs a schedule or a webhook")
	}
	if a.MaxCostUSD != nil && *a.MaxCostUSD <= 0 {
		return fmt.Errorf("max cost must be greater than zero")
	}
	if (a.MaxRuns == nil) != (a.MaxRunsWindowMin == nil) {
		return fmt.Errorf("max runs and its window go together")
	}
	if a.MaxRuns != nil && (*a.MaxRuns < 1 || *a.MaxRunsWindowMin < 1) {
		return fmt.Errorf("max runs and its window must be at least 1")
	}
	return nil
}

func optFloat(v float64) *float64 {
	if v <= 0 {
		return nil
	}
	return &v
}

func optInt(v int) *int {
	if v <= 0 {
		return nil
	}
	return &v
}

// newWebhookSecret returns (plaintext, sha256 hex).
func newWebhookSecret() (string, string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("store: webhook secret: %w", err)
	}
	secret := hex.EncodeToString(b)
	return secret, hashSecret(secret), nil
}

func hashSecret(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

// CreateAutomation inserts one automation. When p.Webhook is set the
// plaintext secret is returned exactly once; only its hash is stored.
func (s *Store) CreateAutomation(p AutomationParams) (Automation, string, error) {
	if p.MaxCostUSD < 0 || p.MaxRuns < 0 || p.MaxRunsWindowMin < 0 {
		return Automation{}, "", fmt.Errorf("limits cannot be negative")
	}
	now := nowUTC()
	ws := strings.TrimSpace(p.WorkspaceID)
	if ws == "" {
		ws = FreeWorkspaceID
	}
	a := Automation{
		ID: newID(p.Name, "aut"), Name: strings.TrimSpace(p.Name), Enabled: true, WorkspaceID: ws,
		Action: p.Action, TargetAgentID: emptyToNil(p.TargetAgentID), Prompt: p.Prompt,
		Provider: emptyToNil(p.Provider), Model: emptyToNil(p.Model), Thinking: emptyToNil(p.Thinking),
		Cron: emptyToNil(p.Cron), Webhook: p.Webhook,
		MaxCostUSD: optFloat(p.MaxCostUSD), MaxRuns: optInt(p.MaxRuns), MaxRunsWindowMin: optInt(p.MaxRunsWindowMin),
		CreatedAt: now, UpdatedAt: now,
	}
	if a.Cron != nil {
		if sch, err := cron.Parse(*a.Cron); err == nil {
			norm := sch.String()
			a.Cron = &norm
		}
	}
	if err := validateAutomation(a); err != nil {
		return Automation{}, "", err
	}
	secret := ""
	if p.Webhook {
		plain, hash, err := newWebhookSecret()
		if err != nil {
			return Automation{}, "", err
		}
		secret, a.webhookHash = plain, &hash
	}
	if _, err := s.db.Exec(`INSERT INTO automations (`+automationCols+`)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		a.ID, a.Name, boolInt(a.Enabled), a.WorkspaceID, a.Action, a.TargetAgentID, a.AgentID, a.Prompt,
		a.Provider, a.Model, a.Thinking, a.Cron, a.webhookHash, a.MaxCostUSD, a.MaxRuns, a.MaxRunsWindowMin,
		a.LastFiredAt, a.CreatedAt, a.UpdatedAt); err != nil {
		return Automation{}, "", fmt.Errorf("store: create automation: %w", err)
	}
	return a, secret, nil
}

// GetAutomation returns one automation.
func (s *Store) GetAutomation(id string) (Automation, error) {
	var a Automation
	err := scanAutomation(s.db.QueryRow(`SELECT `+automationCols+` FROM automations WHERE id = ?`, id), &a)
	if errors.Is(err, sql.ErrNoRows) {
		return Automation{}, ErrNotFound
	}
	if err != nil {
		return Automation{}, fmt.Errorf("store: get automation: %w", err)
	}
	return a, nil
}

// ListAutomations returns every automation, newest first.
func (s *Store) ListAutomations() ([]Automation, error) {
	rows, err := s.db.Query(`SELECT ` + automationCols + ` FROM automations ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("store: list automations: %w", err)
	}
	defer rows.Close()
	out := []Automation{}
	for rows.Next() {
		var a Automation
		if err := scanAutomation(rows, &a); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// UpdateAutomation applies a patch and returns the row after the write.
func (s *Store) UpdateAutomation(id string, p AutomationPatch) (Automation, error) {
	a, err := s.GetAutomation(id)
	if err != nil {
		return Automation{}, err
	}
	if p.Name != nil {
		a.Name = strings.TrimSpace(*p.Name)
	}
	if p.Enabled != nil {
		a.Enabled = *p.Enabled
	}
	if p.WorkspaceID != nil {
		if ws := strings.TrimSpace(*p.WorkspaceID); ws != "" {
			a.WorkspaceID = ws
		}
	}
	if p.Action != nil {
		a.Action = *p.Action
	}
	if p.TargetAgentID != nil {
		a.TargetAgentID = emptyToNil(*p.TargetAgentID)
	}
	if p.Prompt != nil {
		a.Prompt = *p.Prompt
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
	if p.Cron != nil {
		a.Cron = emptyToNil(*p.Cron)
		if a.Cron != nil {
			if sch, err := cron.Parse(*a.Cron); err == nil {
				norm := sch.String()
				a.Cron = &norm
			}
		}
	}
	if p.MaxCostUSD != nil {
		if *p.MaxCostUSD < 0 {
			return Automation{}, fmt.Errorf("limits cannot be negative")
		}
		a.MaxCostUSD = optFloat(*p.MaxCostUSD)
	}
	if p.MaxRuns != nil {
		a.MaxRuns = optInt(*p.MaxRuns)
	}
	if p.MaxRunsWindowMin != nil {
		a.MaxRunsWindowMin = optInt(*p.MaxRunsWindowMin)
	}
	if err := validateAutomation(a); err != nil {
		return Automation{}, err
	}
	a.UpdatedAt = nowUTC()
	if _, err := s.db.Exec(`UPDATE automations SET name=?, enabled=?, workspace_id=?, action=?, target_agent_id=?,
		prompt=?, provider=?, model=?, thinking=?, cron=?, max_cost_usd=?, max_runs=?, max_runs_window_min=?, updated_at=?
		WHERE id=?`,
		a.Name, boolInt(a.Enabled), a.WorkspaceID, a.Action, a.TargetAgentID, a.Prompt, a.Provider, a.Model,
		a.Thinking, a.Cron, a.MaxCostUSD, a.MaxRuns, a.MaxRunsWindowMin, a.UpdatedAt, id); err != nil {
		return Automation{}, fmt.Errorf("store: update automation: %w", err)
	}
	return s.GetAutomation(id)
}

// SetAutomationWebhook turns the webhook on (minting a fresh secret,
// returned once) or off. Turning it on when it is already on rotates the
// secret — that is also how "Regenerate" works.
func (s *Store) SetAutomationWebhook(id string, on bool) (string, error) {
	a, err := s.GetAutomation(id)
	if err != nil {
		return "", err
	}
	if !on {
		if a.Cron == nil {
			return "", fmt.Errorf("an automation needs a schedule or a webhook")
		}
		_, err = s.db.Exec(`UPDATE automations SET webhook_hash=NULL, updated_at=? WHERE id=?`, nowUTC(), id)
		return "", err
	}
	plain, hash, err := newWebhookSecret()
	if err != nil {
		return "", err
	}
	if _, err := s.db.Exec(`UPDATE automations SET webhook_hash=?, updated_at=? WHERE id=?`, hash, nowUTC(), id); err != nil {
		return "", fmt.Errorf("store: set webhook: %w", err)
	}
	return plain, nil
}

// VerifyWebhookSecret answers whether secret opens automation id. The
// comparison is constant-time on the hashes; a missing webhook is false.
func (s *Store) VerifyWebhookSecret(id, secret string) (Automation, bool, error) {
	a, err := s.GetAutomation(id)
	if err != nil {
		return Automation{}, false, err
	}
	if a.webhookHash == nil || secret == "" {
		return a, false, nil
	}
	ok := subtle.ConstantTimeCompare([]byte(*a.webhookHash), []byte(hashSecret(secret))) == 1
	return a, ok, nil
}

// SetAutomationAgent records the lazily created agent for action=start.
func (s *Store) SetAutomationAgent(id, agentID string) error {
	res, err := s.db.Exec(`UPDATE automations SET agent_id=?, updated_at=? WHERE id=?`, emptyToNil(agentID), nowUTC(), id)
	if err != nil {
		return fmt.Errorf("store: set automation agent: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// TouchAutomationFired stamps last_fired_at (schedule + catch-up bookkeeping).
func (s *Store) TouchAutomationFired(id string, t time.Time) error {
	_, err := s.db.Exec(`UPDATE automations SET last_fired_at=? WHERE id=?`, t.UTC().Format(time.RFC3339Nano), id)
	return err
}

// DeleteAutomation removes the automation and its runs. The agent it
// created stays — it is an ordinary agent with sessions the user may want.
func (s *Store) DeleteAutomation(id string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.Exec(`DELETE FROM automation_runs WHERE automation_id = ?`, id); err != nil {
		return fmt.Errorf("store: delete runs: %w", err)
	}
	res, err := tx.Exec(`DELETE FROM automations WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("store: delete automation: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return tx.Commit()
}

// CreateRun records an invocation. status is running for a real run, or
// skipped/failed with a reason when the decision table stopped it.
func (s *Store) CreateRun(automationID, trigger, status, reason string) (Run, error) {
	switch trigger {
	case TriggerSchedule, TriggerWebhook, TriggerManual, TriggerCatchUp:
	default:
		return Run{}, fmt.Errorf("store: invalid trigger %q", trigger)
	}
	switch status {
	case RunRunning, RunDone, RunFailed, RunSkipped:
	default:
		return Run{}, fmt.Errorf("store: invalid run status %q", status)
	}
	now := nowUTC()
	r := Run{ID: newID("run", "run"), AutomationID: automationID, Trigger: trigger, Status: status, Reason: reason, FiredAt: now}
	if status != RunRunning {
		r.FinishedAt = &now
	}
	if _, err := s.db.Exec(`INSERT INTO automation_runs (id, automation_id, trigger, status, reason, fired_at, finished_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`, r.ID, r.AutomationID, r.Trigger, r.Status, r.Reason, r.FiredAt, r.FinishedAt); err != nil {
		return Run{}, fmt.Errorf("store: create run: %w", err)
	}
	return r, nil
}

// SetRunSession attaches the pi session file once the agent has one.
func (s *Store) SetRunSession(id, sessionPath string) error {
	_, err := s.db.Exec(`UPDATE automation_runs SET session_path=? WHERE id=?`, emptyToNil(sessionPath), id)
	return err
}

// FinishRun closes a running row. Idempotent on already-finished rows
// (the first writer wins: a watchdog and a settle may race).
func (s *Store) FinishRun(id, status, reason string, cost float64) error {
	switch status {
	case RunDone, RunFailed, RunSkipped:
	default:
		return fmt.Errorf("store: invalid finish status %q", status)
	}
	_, err := s.db.Exec(`UPDATE automation_runs SET status=?, reason=?, cost_usd=?, finished_at=?
		WHERE id=? AND status=?`, status, reason, cost, nowUTC(), id, RunRunning)
	if err != nil {
		return fmt.Errorf("store: finish run: %w", err)
	}
	return nil
}

// GetRun returns one run.
func (s *Store) GetRun(id string) (Run, error) {
	var r Run
	err := scanRun(s.db.QueryRow(`SELECT `+runCols+` FROM automation_runs WHERE id = ?`, id), &r)
	if errors.Is(err, sql.ErrNoRows) {
		return Run{}, ErrNotFound
	}
	return r, err
}

// ListRuns returns an automation's runs, newest first.
func (s *Store) ListRuns(automationID string, limit int) ([]Run, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.Query(`SELECT `+runCols+` FROM automation_runs WHERE automation_id = ?
		ORDER BY fired_at DESC LIMIT ?`, automationID, limit)
	if err != nil {
		return nil, fmt.Errorf("store: list runs: %w", err)
	}
	defer rows.Close()
	out := []Run{}
	for rows.Next() {
		var r Run
		if err := scanRun(rows, &r); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// LastRun returns the newest run, or nil when there is none.
func (s *Store) LastRun(automationID string) (*Run, error) {
	runs, err := s.ListRuns(automationID, 1)
	if err != nil || len(runs) == 0 {
		return nil, err
	}
	return &runs[0], nil
}

// RunningRun returns the in-flight run, or nil (concurrency = 1).
func (s *Store) RunningRun(automationID string) (*Run, error) {
	var r Run
	err := scanRun(s.db.QueryRow(`SELECT `+runCols+` FROM automation_runs WHERE automation_id = ? AND status = ?
		ORDER BY fired_at DESC LIMIT 1`, automationID, RunRunning), &r)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("store: running run: %w", err)
	}
	return &r, nil
}

// CountRunsSince counts runs that actually started (running, done,
// failed) since t — the rate-cap numerator. Skips do not count against
// the cap, or a busy automation could lock itself out.
func (s *Store) CountRunsSince(automationID string, t time.Time) (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(1) FROM automation_runs WHERE automation_id = ? AND status != ? AND fired_at >= ?`,
		automationID, RunSkipped, t.UTC().Format(time.RFC3339Nano)).Scan(&n)
	return n, err
}

// RunCountsByDay returns one bucket per local day for the last `days`
// days (oldest first), counting started runs — the list sparkline.
func (s *Store) RunCountsByDay(automationID string, days int, now time.Time) ([]int, error) {
	if days <= 0 {
		days = 30
	}
	out := make([]int, days)
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -(days - 1))
	rows, err := s.db.Query(`SELECT fired_at FROM automation_runs WHERE automation_id = ? AND status != ? AND fired_at >= ?`,
		automationID, RunSkipped, start.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return nil, fmt.Errorf("store: run counts: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var fired string
		if err := rows.Scan(&fired); err != nil {
			return nil, err
		}
		t, err := time.Parse(time.RFC3339Nano, fired)
		if err != nil {
			continue
		}
		t = t.In(now.Location())
		day := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, now.Location())
		i := int(day.Sub(start).Hours() / 24)
		if i >= 0 && i < days {
			out[i]++
		}
	}
	return out, rows.Err()
}

// FailStaleRuns closes every running row — called once at boot, since a
// run cannot survive the process that was watching it (binwatch re-execs
// on deploy). Returns the closed runs so the caller can notify.
func (s *Store) FailStaleRuns(reason string) ([]Run, error) {
	rows, err := s.db.Query(`SELECT `+runCols+` FROM automation_runs WHERE status = ?`, RunRunning)
	if err != nil {
		return nil, fmt.Errorf("store: stale runs: %w", err)
	}
	stale := []Run{}
	for rows.Next() {
		var r Run
		if err := scanRun(rows, &r); err != nil {
			rows.Close()
			return nil, err
		}
		stale = append(stale, r)
	}
	rows.Close()
	for i := range stale {
		if err := s.FinishRun(stale[i].ID, RunFailed, reason, 0); err != nil {
			return nil, err
		}
		stale[i].Status, stale[i].Reason = RunFailed, reason
	}
	return stale, nil
}
