package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Checklist obligation levels (ADR-0055), passed to pi-checklist as
// PICODE_CHECKLIST. "changes": a checklist before the first change of a
// task; "always": every task; "never": the tool stays, nothing required.
const (
	ChecklistChanges = "changes"
	ChecklistAlways  = "always"
	ChecklistNever   = "never"
)

// ChecklistEnv is the process env pi-checklist reads for the level.
const ChecklistEnv = "PICODE_CHECKLIST"

// NormalizeChecklist maps user input to a level; "" is the default.
func NormalizeChecklist(raw string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", ChecklistChanges:
		return ChecklistChanges, nil
	case ChecklistAlways:
		return ChecklistAlways, nil
	case ChecklistNever:
		return ChecklistNever, nil
	default:
		return "", fmt.Errorf("store: unknown checklist level %q", raw)
	}
}

// ChecklistLevel is the effective level for a spawn: a read-only agent
// cannot change anything, so it is never asked for a plan.
func (a Agent) ChecklistLevel() string {
	if a.OpMode != nil && *a.OpMode == OpModeReadonly {
		return ChecklistNever
	}
	if lvl, err := NormalizeChecklist(a.Checklist); err == nil {
		return lvl
	}
	return ChecklistChanges
}

// ChecklistItem is one step as the extension sends it.
type ChecklistItem struct {
	Text   string `json:"text"`
	Status string `json:"status"`
}

// Checklist statuses, a closed vocabulary shared with the package.
var checklistStatuses = map[string]bool{"pending": true, "in-progress": true, "completed": true}

// Checklist is the latest list an agent's pi-checklist published.
type Checklist struct {
	AgentID   string          `json:"agentId"`
	SessionID string          `json:"sessionId,omitempty"`
	Items     []ChecklistItem `json:"items"`
	Absent    bool            `json:"absent"`
	UpdatedAt string          `json:"updatedAt"`
}

const maxChecklistItems = 50
const maxChecklistText = 300

// ValidateChecklistItems normalizes what the extension sent; an empty
// list is fine (an "absent" marker carries none).
func ValidateChecklistItems(items []ChecklistItem) ([]ChecklistItem, error) {
	if len(items) > maxChecklistItems {
		return nil, fmt.Errorf("store: checklist holds more than %d steps", maxChecklistItems)
	}
	out := make([]ChecklistItem, 0, len(items))
	for i, it := range items {
		text := strings.Join(strings.Fields(it.Text), " ")
		if text == "" {
			return nil, fmt.Errorf("store: checklist step %d has no text", i)
		}
		if len(text) > maxChecklistText {
			text = text[:maxChecklistText]
		}
		st := strings.ToLower(strings.TrimSpace(it.Status))
		if st == "" {
			st = "pending"
		}
		if !checklistStatuses[st] {
			return nil, fmt.Errorf("store: checklist step %d has status %q", i, it.Status)
		}
		out = append(out, ChecklistItem{Text: text, Status: st})
	}
	return out, nil
}

// SetChecklist replaces an agent's checklist and announces agent.checklist.
func (s *Store) SetChecklist(agentID, sessionID string, items []ChecklistItem, absent bool) (Checklist, error) {
	if _, err := s.GetAgent(agentID); err != nil {
		return Checklist{}, err
	}
	items, err := ValidateChecklistItems(items)
	if err != nil {
		return Checklist{}, err
	}
	raw, _ := json.Marshal(items)
	c := Checklist{AgentID: agentID, SessionID: strings.TrimSpace(sessionID), Items: items, Absent: absent, UpdatedAt: time.Now().UTC().Format(time.RFC3339)}
	ab := 0
	if absent {
		ab = 1
	}
	_, err = s.db.Exec(`INSERT INTO agent_checklists (agent_id, session_id, items, absent, updated_at) VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(agent_id) DO UPDATE SET session_id=excluded.session_id, items=excluded.items, absent=excluded.absent, updated_at=excluded.updated_at`,
		c.AgentID, c.SessionID, string(raw), ab, c.UpdatedAt)
	if err != nil {
		return Checklist{}, fmt.Errorf("store: set checklist: %w", err)
	}
	s.note("agent.checklist", &agentID, nil, c)
	return c, nil
}

func scanChecklist(row interface{ Scan(...any) error }) (Checklist, error) {
	var c Checklist
	var raw string
	var ab int
	if err := row.Scan(&c.AgentID, &c.SessionID, &raw, &ab, &c.UpdatedAt); err != nil {
		return Checklist{}, err
	}
	c.Absent = ab != 0
	c.Items = []ChecklistItem{}
	_ = json.Unmarshal([]byte(raw), &c.Items)
	if c.Items == nil {
		c.Items = []ChecklistItem{}
	}
	return c, nil
}

const checklistCols = `agent_id, session_id, items, absent, updated_at`

// GetChecklist returns the agent's latest checklist; ErrNotFound when none.
func (s *Store) GetChecklist(agentID string) (Checklist, error) {
	c, err := scanChecklist(s.db.QueryRow(`SELECT `+checklistCols+` FROM agent_checklists WHERE agent_id = ?`, agentID))
	if errors.Is(err, sql.ErrNoRows) {
		return Checklist{}, ErrNotFound
	}
	if err != nil {
		return Checklist{}, fmt.Errorf("store: get checklist: %w", err)
	}
	return c, nil
}

// ListChecklists returns every agent's latest checklist (one shell fetch at boot).
func (s *Store) ListChecklists() ([]Checklist, error) {
	rows, err := s.db.Query(`SELECT ` + checklistCols + ` FROM agent_checklists ORDER BY agent_id`)
	if err != nil {
		return nil, fmt.Errorf("store: list checklists: %w", err)
	}
	defer rows.Close()
	out := []Checklist{}
	for rows.Next() {
		c, err := scanChecklist(rows)
		if err != nil {
			return nil, fmt.Errorf("store: list checklists: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
