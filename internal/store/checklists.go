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

// truncateRunes cuts at max code points; a byte slice can split a UTF-8
// rune and store a broken character.
func truncateRunes(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max])
}

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
		text = truncateRunes(text, maxChecklistText)
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

// ClearChecklist drops the agent's row and announces the empty state, so
// shells drop their entry (the checklist line renders nothing) until the
// next real list arrives. Idempotent: clearing with no row is fine.
func (s *Store) ClearChecklist(agentID string) (Checklist, error) {
	if _, err := s.GetAgent(agentID); err != nil {
		return Checklist{}, err
	}
	if _, err := s.db.Exec(`DELETE FROM agent_checklists WHERE agent_id = ?`, agentID); err != nil {
		return Checklist{}, fmt.Errorf("store: clear checklist: %w", err)
	}
	c := Checklist{AgentID: agentID, Items: []ChecklistItem{}, Absent: false}
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

// TerminalChecklist is the latest list a pi inside a PiCode terminal
// published (ADR-0055 extended to Agent CLI terminals): the extension
// posts under PICODE_TERM_ID, and the terminal's card shows the same line
// an agent's card does. Shape and vocabulary match Checklist.
type TerminalChecklist struct {
	TerminalID string          `json:"termId"`
	SessionID  string          `json:"sessionId,omitempty"`
	Items      []ChecklistItem `json:"items"`
	Absent     bool            `json:"absent"`
	UpdatedAt  string          `json:"updatedAt"`
}

// SetTerminalChecklist replaces a terminal's checklist and announces
// terminal.checklist. The terminal must exist — the extension can only
// publish from a live PiCode terminal.
func (s *Store) SetTerminalChecklist(termID, sessionID string, items []ChecklistItem, absent bool) (TerminalChecklist, error) {
	if _, err := s.GetTerminal(termID); err != nil {
		return TerminalChecklist{}, err
	}
	items, err := ValidateChecklistItems(items)
	if err != nil {
		return TerminalChecklist{}, err
	}
	raw, _ := json.Marshal(items)
	c := TerminalChecklist{TerminalID: termID, SessionID: strings.TrimSpace(sessionID), Items: items, Absent: absent, UpdatedAt: time.Now().UTC().Format(time.RFC3339)}
	ab := 0
	if absent {
		ab = 1
	}
	_, err = s.db.Exec(`INSERT INTO terminal_checklists (terminal_id, session_id, items, absent, updated_at) VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(terminal_id) DO UPDATE SET session_id=excluded.session_id, items=excluded.items, absent=excluded.absent, updated_at=excluded.updated_at`,
		c.TerminalID, c.SessionID, string(raw), ab, c.UpdatedAt)
	if err != nil {
		return TerminalChecklist{}, fmt.Errorf("store: set terminal checklist: %w", err)
	}
	s.note("terminal.checklist", nil, nil, c)
	return c, nil
}

func scanTerminalChecklist(row interface{ Scan(...any) error }) (TerminalChecklist, error) {
	var c TerminalChecklist
	var raw string
	var ab int
	if err := row.Scan(&c.TerminalID, &c.SessionID, &raw, &ab, &c.UpdatedAt); err != nil {
		return TerminalChecklist{}, err
	}
	c.Absent = ab != 0
	c.Items = []ChecklistItem{}
	_ = json.Unmarshal([]byte(raw), &c.Items)
	if c.Items == nil {
		c.Items = []ChecklistItem{}
	}
	return c, nil
}

const terminalChecklistCols = `terminal_id, session_id, items, absent, updated_at`

// GetTerminalChecklist returns the terminal's latest checklist; ErrNotFound when none.
func (s *Store) GetTerminalChecklist(termID string) (TerminalChecklist, error) {
	c, err := scanTerminalChecklist(s.db.QueryRow(`SELECT `+terminalChecklistCols+` FROM terminal_checklists WHERE terminal_id = ?`, termID))
	if errors.Is(err, sql.ErrNoRows) {
		return TerminalChecklist{}, ErrNotFound
	}
	if err != nil {
		return TerminalChecklist{}, fmt.Errorf("store: get terminal checklist: %w", err)
	}
	return c, nil
}

// ClearTerminalChecklist drops the terminal's row and announces the empty
// state, so shells drop their line until the next real list arrives. Used
// by the reset marker (a fresh session must not show the old task) and
// idempotent like its agent-side sibling.
func (s *Store) ClearTerminalChecklist(termID string) (TerminalChecklist, error) {
	if _, err := s.GetTerminal(termID); err != nil {
		return TerminalChecklist{}, err
	}
	if _, err := s.db.Exec(`DELETE FROM terminal_checklists WHERE terminal_id = ?`, termID); err != nil {
		return TerminalChecklist{}, fmt.Errorf("store: clear terminal checklist: %w", err)
	}
	c := TerminalChecklist{TerminalID: termID, Items: []ChecklistItem{}, Absent: false}
	s.note("terminal.checklist", nil, nil, c)
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
