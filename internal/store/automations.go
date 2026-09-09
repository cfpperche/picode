package store

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/cron"
)

// Automations (ADR-0045): trigger(s) + prompt + bounds. Every run is an
// ordinary pi session on the automation's own agent (action=start) or a
// follow_up queued into an existing agent (action=message). Since the
// 2026-09-09 amendment an automation has zero or more schedules, one row
// each in automation_schedules; the automation row carries the what,
// the schedule rows carry the when.

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
	maxAutomationName      = 60
	maxAutomationPrompt    = 100_000
	maxScheduleLabel       = 40
	maxAutomationSchedules = 10
)

// Automation is one row. Webhook is derived from webhook_hash; the secret
// itself is never stored or returned after creation. Schedules are the
// automation's rules, in position order.
type Automation struct {
	ID               string     `json:"id"`
	Name             string     `json:"name"`
	Enabled          bool       `json:"enabled"`
	WorkspaceID      string     `json:"workspaceId"`
	Action           string     `json:"action"`
	TargetAgentID    *string    `json:"targetAgentId,omitempty"`
	AgentID          *string    `json:"agentId,omitempty"`
	Prompt           string     `json:"prompt"`
	Provider         *string    `json:"provider"`
	Model            *string    `json:"model"`
	Thinking         *string    `json:"thinking"`
	Schedules        []Schedule `json:"schedules"`
	Webhook          bool       `json:"webhook"`
	NotifyURL        *string    `json:"notifyUrl"` // POSTed a Slack-style message when a run ends
	MaxCostUSD       *float64   `json:"maxCostUsd"`
	MaxRuns          *int       `json:"maxRuns"`
	MaxRunsWindowMin *int       `json:"maxRunsWindowMin"`
	CreatedAt        string     `json:"createdAt"`
	UpdatedAt        string     `json:"updatedAt"`

	webhookHash *string
}

// Schedule is one rule of an automation (ADR-0045 amendment 2026-09-09):
// a cron evaluated in a zone, with its own switch and its own last fire,
// so two rules never share a catch-up or a jitter.
type Schedule struct {
	ID           string  `json:"id"`
	AutomationID string  `json:"automationId"`
	Label        string  `json:"label"`
	Cron         string  `json:"cron"`
	TZ           string  `json:"tz"` // IANA zone; "" = the daemon's local zone
	Enabled      bool    `json:"enabled"`
	LastFiredAt  *string `json:"lastFiredAt,omitempty"`
	Position     int     `json:"position"`
	CreatedAt    string  `json:"createdAt"`
	UpdatedAt    string  `json:"updatedAt"`
}

// Location is the zone the schedule's wall clock is read in. An empty or
// (since validation) unknown zone is the daemon's local zone.
func (sc Schedule) Location() *time.Location {
	if sc.TZ != "" {
		if loc, err := time.LoadLocation(sc.TZ); err == nil {
			return loc
		}
	}
	return time.Local
}

// ScheduleParams is one schedule as the caller wants it. A non-empty ID
// names one of the automation's existing rows, updated in place: its
// last fire survives unless Cron or TZ changed.
type ScheduleParams struct {
	ID      string
	Label   string
	Cron    string
	TZ      string
	Enabled bool
}

// AutomationParams is CreateAutomation's input. Zero values mean
// "unset"; MaxRuns and MaxRunsWindowMin go together. Cron is the
// one-schedule convenience (daemon zone, enabled) used when Schedules
// is empty — templates and the /automate fence speak it.
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
	Schedules        []ScheduleParams
	Webhook          bool
	NotifyURL        string
	MaxCostUSD       float64
	MaxRuns          int
	MaxRunsWindowMin int
}

// AutomationPatch is a partial update. Nil = unchanged. For nullable
// columns an empty string / zero clears (as AgentPatch does). Schedules
// is the whole list (empty = none); Cron is the one-schedule convenience
// ("" = none) and is read only when Schedules is nil.
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
	Schedules        *[]ScheduleParams
	NotifyURL        *string
	MaxCostUSD       *float64
	MaxRuns          *int
	MaxRunsWindowMin *int
}

// Run is one invocation. Skipped rows record why nothing ran. ScheduleID
// names the rule that fired; nil for webhook and Run now.
type Run struct {
	ID           string  `json:"id"`
	AutomationID string  `json:"automationId"`
	ScheduleID   *string `json:"scheduleId,omitempty"`
	Trigger      string  `json:"trigger"`
	Status       string  `json:"status"`
	Reason       string  `json:"reason,omitempty"`
	SessionPath  *string `json:"sessionPath,omitempty"`
	CostUSD      float64 `json:"costUsd"`
	FiredAt      string  `json:"firedAt"`
	FinishedAt   *string `json:"finishedAt,omitempty"`
}

// RunParams is CreateRun's input.
type RunParams struct {
	AutomationID string
	ScheduleID   string // the rule that fired; "" for webhook and Run now
	Trigger      string
	Status       string
	Reason       string
}

const automationCols = `id, name, enabled, workspace_id, action, target_agent_id, agent_id, prompt,
	provider, model, thinking, webhook_hash, max_cost_usd, max_runs, max_runs_window_min,
	created_at, updated_at, notify_url`

func scanAutomation(row interface{ Scan(...any) error }, a *Automation) error {
	var enabled int
	if err := row.Scan(&a.ID, &a.Name, &enabled, &a.WorkspaceID, &a.Action, &a.TargetAgentID, &a.AgentID,
		&a.Prompt, &a.Provider, &a.Model, &a.Thinking, &a.webhookHash, &a.MaxCostUSD,
		&a.MaxRuns, &a.MaxRunsWindowMin, &a.CreatedAt, &a.UpdatedAt, &a.NotifyURL); err != nil {
		return err
	}
	a.Enabled = enabled != 0
	a.Webhook = a.webhookHash != nil && *a.webhookHash != ""
	a.Schedules = []Schedule{}
	return nil
}

const scheduleCols = `id, automation_id, label, cron, tz, enabled, last_fired_at, position, created_at, updated_at`

func scanSchedule(row interface{ Scan(...any) error }, sc *Schedule) error {
	var enabled int
	if err := row.Scan(&sc.ID, &sc.AutomationID, &sc.Label, &sc.Cron, &sc.TZ, &enabled, &sc.LastFiredAt,
		&sc.Position, &sc.CreatedAt, &sc.UpdatedAt); err != nil {
		return err
	}
	sc.Enabled = enabled != 0
	return nil
}

// schedulesFor loads the schedules of one or every automation (id ""),
// keyed by automation, in position order.
func (s *Store) schedulesFor(q interface {
	Query(query string, args ...any) (*sql.Rows, error)
}, automationID string) (map[string][]Schedule, error) {
	query := `SELECT ` + scheduleCols + ` FROM automation_schedules`
	args := []any{}
	if automationID != "" {
		query += ` WHERE automation_id = ?`
		args = append(args, automationID)
	}
	rows, err := q.Query(query+` ORDER BY automation_id, position, created_at`, args...)
	if err != nil {
		return nil, fmt.Errorf("store: list schedules: %w", err)
	}
	defer rows.Close()
	out := map[string][]Schedule{}
	for rows.Next() {
		var sc Schedule
		if err := scanSchedule(rows, &sc); err != nil {
			return nil, err
		}
		out[sc.AutomationID] = append(out[sc.AutomationID], sc)
	}
	return out, rows.Err()
}

const runCols = `id, automation_id, schedule_id, trigger, status, reason, session_path, cost_usd, fired_at, finished_at`

func scanRun(row interface{ Scan(...any) error }, r *Run) error {
	return row.Scan(&r.ID, &r.AutomationID, &r.ScheduleID, &r.Trigger, &r.Status, &r.Reason, &r.SessionPath,
		&r.CostUSD, &r.FiredAt, &r.FinishedAt)
}

// validateAutomation checks the invariants shared by create and update.
func validateAutomation(a Automation) error {
	if a.NotifyURL != nil {
		u, err := url.Parse(*a.NotifyURL)
		if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
			return fmt.Errorf("notify URL must be an http(s) address")
		}
	}
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
	if err := validateSchedules(a.Schedules); err != nil {
		return err
	}
	if len(a.Schedules) == 0 && !a.Webhook {
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

// validateSchedules checks every rule: a parseable cron, a known zone (or
// none), a short label, at most maxAutomationSchedules rows and no two
// rows saying the same thing (they would only collide as busy).
func validateSchedules(list []Schedule) error {
	if len(list) > maxAutomationSchedules {
		return fmt.Errorf("at most %d schedules per automation", maxAutomationSchedules)
	}
	seen := map[string]bool{}
	for _, sc := range list {
		if _, err := cron.Parse(sc.Cron); err != nil {
			return err
		}
		if sc.TZ != "" {
			if _, err := time.LoadLocation(sc.TZ); err != nil {
				return fmt.Errorf("unknown time zone %q", sc.TZ)
			}
		}
		if len(sc.Label) > maxScheduleLabel {
			return fmt.Errorf("schedule label is longer than %d characters", maxScheduleLabel)
		}
		key := sc.Cron + " @ " + sc.TZ
		if seen[key] {
			return fmt.Errorf("two schedules say the same thing (%s)", sc.Cron)
		}
		seen[key] = true
	}
	return nil
}

// scheduleParams turns the caller's rows into the shape validation and
// the writers use: trimmed, cron normalised, positions by index. Rows
// naming an existing id keep that row's creation time and — unless the
// rule changed — its last fire: an edited rule starts over and never
// catches up a slot it was not yet asked for (ADR-0100 does the same).
func scheduleRows(automationID string, params []ScheduleParams, existing []Schedule, now string) ([]Schedule, error) {
	byID := map[string]Schedule{}
	for _, sc := range existing {
		byID[sc.ID] = sc
	}
	out := make([]Schedule, 0, len(params))
	for i, p := range params {
		sc := Schedule{
			AutomationID: automationID, Label: strings.TrimSpace(p.Label), Cron: strings.Join(strings.Fields(p.Cron), " "),
			TZ: strings.TrimSpace(p.TZ), Enabled: p.Enabled, Position: i, CreatedAt: now, UpdatedAt: now,
		}
		if sch, err := cron.Parse(sc.Cron); err == nil {
			sc.Cron = sch.String()
		}
		if p.ID != "" {
			old, ok := byID[p.ID]
			if !ok {
				return nil, fmt.Errorf("unknown schedule %q", p.ID)
			}
			sc.ID, sc.CreatedAt = old.ID, old.CreatedAt
			if old.Cron == sc.Cron && old.TZ == sc.TZ {
				sc.LastFiredAt = old.LastFiredAt
			}
		} else {
			sc.ID = newID(sc.Label, "sch")
		}
		out = append(out, sc)
	}
	return out, nil
}

// oneSchedule is the Cron convenience: one enabled rule in the daemon zone.
func oneSchedule(cronExpr string) []ScheduleParams {
	cronExpr = strings.TrimSpace(cronExpr)
	if cronExpr == "" {
		return nil
	}
	return []ScheduleParams{{Cron: cronExpr, Enabled: true}}
}

// writeSchedules replaces the automation's rows with list inside tx.
func writeSchedules(tx execer, automationID string, list []Schedule) error {
	if _, err := tx.Exec(`DELETE FROM automation_schedules WHERE automation_id = ?`, automationID); err != nil {
		return fmt.Errorf("store: clear schedules: %w", err)
	}
	for _, sc := range list {
		if _, err := tx.Exec(`INSERT INTO automation_schedules (`+scheduleCols+`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			sc.ID, sc.AutomationID, sc.Label, sc.Cron, sc.TZ, boolInt(sc.Enabled), sc.LastFiredAt, sc.Position,
			sc.CreatedAt, sc.UpdatedAt); err != nil {
			return fmt.Errorf("store: insert schedule: %w", err)
		}
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

// CreateAutomation inserts one automation and its schedules. When
// p.Webhook is set the plaintext secret is returned exactly once; only
// its hash is stored.
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
		Webhook: p.Webhook, NotifyURL: emptyToNil(strings.TrimSpace(p.NotifyURL)),
		MaxCostUSD: optFloat(p.MaxCostUSD), MaxRuns: optInt(p.MaxRuns), MaxRunsWindowMin: optInt(p.MaxRunsWindowMin),
		CreatedAt: now, UpdatedAt: now,
	}
	params := p.Schedules
	if len(params) == 0 {
		params = oneSchedule(p.Cron)
	}
	rows, err := scheduleRows(a.ID, params, nil, now)
	if err != nil {
		return Automation{}, "", err
	}
	a.Schedules = rows
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
	tx, err := s.db.Begin()
	if err != nil {
		return Automation{}, "", err
	}
	defer func() { s.rollback(tx) }()
	if _, err := tx.Exec(`INSERT INTO automations (`+automationCols+`)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		a.ID, a.Name, boolInt(a.Enabled), a.WorkspaceID, a.Action, a.TargetAgentID, a.AgentID, a.Prompt,
		a.Provider, a.Model, a.Thinking, a.webhookHash, a.MaxCostUSD, a.MaxRuns, a.MaxRunsWindowMin,
		a.CreatedAt, a.UpdatedAt, a.NotifyURL); err != nil {
		return Automation{}, "", fmt.Errorf("store: create automation: %w", err)
	}
	if err := writeSchedules(tx, a.ID, a.Schedules); err != nil {
		return Automation{}, "", err
	}
	if err := s.commit(tx); err != nil {
		return Automation{}, "", err
	}
	s.note("automation.created", nil, nil, a) // workspace may be gone: not a workspaces FK
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
	byAut, err := s.schedulesFor(s.db, id)
	if err != nil {
		return Automation{}, err
	}
	if list := byAut[id]; list != nil {
		a.Schedules = list
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
	if err := rows.Err(); err != nil {
		return nil, err
	}
	byAut, err := s.schedulesFor(s.db, "")
	if err != nil {
		return nil, err
	}
	for i := range out {
		if list := byAut[out[i].ID]; list != nil {
			out[i].Schedules = list
		}
	}
	return out, nil
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
	a.UpdatedAt = nowUTC()
	var params *[]ScheduleParams
	if p.Schedules != nil {
		params = p.Schedules
	} else if p.Cron != nil {
		one := oneSchedule(*p.Cron)
		params = &one
	}
	if params != nil {
		rows, err := scheduleRows(a.ID, *params, a.Schedules, a.UpdatedAt)
		if err != nil {
			return Automation{}, err
		}
		a.Schedules = rows
	}
	if p.NotifyURL != nil {
		a.NotifyURL = emptyToNil(strings.TrimSpace(*p.NotifyURL))
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
	tx, err := s.db.Begin()
	if err != nil {
		return Automation{}, err
	}
	defer func() { s.rollback(tx) }()
	if _, err := tx.Exec(`UPDATE automations SET name=?, enabled=?, workspace_id=?, action=?, target_agent_id=?,
		prompt=?, provider=?, model=?, thinking=?, max_cost_usd=?, max_runs=?, max_runs_window_min=?, updated_at=?,
		notify_url=?
		WHERE id=?`,
		a.Name, boolInt(a.Enabled), a.WorkspaceID, a.Action, a.TargetAgentID, a.Prompt, a.Provider, a.Model,
		a.Thinking, a.MaxCostUSD, a.MaxRuns, a.MaxRunsWindowMin, a.UpdatedAt, a.NotifyURL, id); err != nil {
		return Automation{}, fmt.Errorf("store: update automation: %w", err)
	}
	if params != nil {
		if err := writeSchedules(tx, a.ID, a.Schedules); err != nil {
			return Automation{}, err
		}
	}
	if err := s.commit(tx); err != nil {
		return Automation{}, err
	}
	return s.automationChanged(id)
}

// automationChanged reloads the row and announces automation.updated.
func (s *Store) automationChanged(id string) (Automation, error) {
	a, err := s.GetAutomation(id)
	if err != nil {
		return Automation{}, err
	}
	s.note("automation.updated", nil, nil, a)
	return a, nil
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
		if len(a.Schedules) == 0 {
			return "", fmt.Errorf("an automation needs a schedule or a webhook")
		}
		if _, err = s.db.Exec(`UPDATE automations SET webhook_hash=NULL, updated_at=? WHERE id=?`, nowUTC(), id); err != nil {
			return "", err
		}
		_, err = s.automationChanged(id)
		return "", err
	}
	plain, hash, err := newWebhookSecret()
	if err != nil {
		return "", err
	}
	if _, err := s.db.Exec(`UPDATE automations SET webhook_hash=?, updated_at=? WHERE id=?`, hash, nowUTC(), id); err != nil {
		return "", fmt.Errorf("store: set webhook: %w", err)
	}
	if _, err := s.automationChanged(id); err != nil {
		return "", err
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
	_, err = s.automationChanged(id)
	return err
}

// TouchScheduleFired stamps the schedule's last_fired_at (schedule +
// catch-up bookkeeping) and announces the automation as updated.
func (s *Store) TouchScheduleFired(scheduleID string, t time.Time) error {
	var automationID string
	err := s.db.QueryRow(`SELECT automation_id FROM automation_schedules WHERE id = ?`, scheduleID).Scan(&automationID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("store: touch schedule: %w", err)
	}
	if _, err := s.db.Exec(`UPDATE automation_schedules SET last_fired_at=? WHERE id=?`, t.UTC().Format(time.RFC3339Nano), scheduleID); err != nil {
		return err
	}
	_, err = s.automationChanged(automationID)
	return err
}

// DeleteAutomation removes the automation and its runs. The agent it
// created stays — it is an ordinary agent with sessions the user may want.
func (s *Store) DeleteAutomation(id string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { s.rollback(tx) }()
	if _, err := tx.Exec(`DELETE FROM automation_runs WHERE automation_id = ?`, id); err != nil {
		return fmt.Errorf("store: delete runs: %w", err)
	}
	if _, err := tx.Exec(`DELETE FROM automation_schedules WHERE automation_id = ?`, id); err != nil {
		return fmt.Errorf("store: delete schedules: %w", err)
	}
	res, err := tx.Exec(`DELETE FROM automations WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("store: delete automation: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	if err := s.AppendEventTx(tx, "automation.deleted", nil, nil, idData(id)); err != nil {
		return err
	}
	return s.commit(tx)
}

// CreateRun records an invocation. Status is running for a real run, or
// skipped/failed with a reason when the decision table stopped it.
func (s *Store) CreateRun(p RunParams) (Run, error) {
	switch p.Trigger {
	case TriggerSchedule, TriggerWebhook, TriggerManual, TriggerCatchUp:
	default:
		return Run{}, fmt.Errorf("store: invalid trigger %q", p.Trigger)
	}
	switch p.Status {
	case RunRunning, RunDone, RunFailed, RunSkipped:
	default:
		return Run{}, fmt.Errorf("store: invalid run status %q", p.Status)
	}
	now := nowUTC()
	r := Run{ID: newID("run", "run"), AutomationID: p.AutomationID, ScheduleID: emptyToNil(p.ScheduleID),
		Trigger: p.Trigger, Status: p.Status, Reason: p.Reason, FiredAt: now}
	if p.Status != RunRunning {
		r.FinishedAt = &now
	}
	if _, err := s.db.Exec(`INSERT INTO automation_runs (id, automation_id, schedule_id, trigger, status, reason, fired_at, finished_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, r.ID, r.AutomationID, r.ScheduleID, r.Trigger, r.Status, r.Reason, r.FiredAt, r.FinishedAt); err != nil {
		return Run{}, fmt.Errorf("store: create run: %w", err)
	}
	s.note("run.created", nil, nil, r)
	return r, nil
}

// SetRunSession attaches the pi session file once the agent has one.
func (s *Store) SetRunSession(id, sessionPath string) error {
	if _, err := s.db.Exec(`UPDATE automation_runs SET session_path=? WHERE id=?`, emptyToNil(sessionPath), id); err != nil {
		return err
	}
	if r, err := s.GetRun(id); err == nil {
		s.note("run.updated", nil, nil, r)
	}
	return nil
}

// FinishRun closes a running row. Idempotent on already-finished rows
// (the first writer wins: a watchdog and a settle may race).
func (s *Store) FinishRun(id, status, reason string, cost float64) error {
	switch status {
	case RunDone, RunFailed, RunSkipped:
	default:
		return fmt.Errorf("store: invalid finish status %q", status)
	}
	res, err := s.db.Exec(`UPDATE automation_runs SET status=?, reason=?, cost_usd=?, finished_at=?
		WHERE id=? AND status=?`, status, reason, cost, nowUTC(), id, RunRunning)
	if err != nil {
		return fmt.Errorf("store: finish run: %w", err)
	}
	if n, _ := res.RowsAffected(); n > 0 {
		if r, err := s.GetRun(id); err == nil {
			s.note("run.finished", nil, nil, r)
		}
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
// run cannot survive the process that was watching it (a deploy restarts
// the daemon). costOf (optional) prices a run from its session file so the
// row keeps the real number. Returns the closed runs so the caller can notify.
func (s *Store) FailStaleRuns(reason string, costOf func(sessionPath string) float64) ([]Run, error) {
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
		cost := 0.0
		if costOf != nil && stale[i].SessionPath != nil {
			cost = costOf(*stale[i].SessionPath)
		}
		if err := s.FinishRun(stale[i].ID, RunFailed, reason, cost); err != nil {
			return nil, err
		}
		stale[i].Status, stale[i].Reason, stale[i].CostUSD = RunFailed, reason, cost
	}
	return stale, nil
}
