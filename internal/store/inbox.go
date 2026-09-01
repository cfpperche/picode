package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// ErrAgentInteractive is returned by RespondAndForward when the target
// agent is currently running in an interactive (TUI/tmux) session: the
// only thing that drains a queued follow_up task is the RPC runtime's
// deliverLoop, which a TUI session has none of. Enqueueing anyway would
// tell the human "sent" for a message that silently sits forever.
var ErrAgentInteractive = errors.New("store: agent is running interactively; replies are not delivered automatically")

// AgentDeliverable answers whether a queued reply for this agent will be
// drained automatically. The store package has no tmux or RPC-runtime
// import, so the caller (internal/server, internal/apps) supplies this —
// nil means "assume yes" (tests, the demo app, any caller that hasn't
// wired reachability).
type AgentDeliverable func(agentID string) bool

// Inbox (ADR-0037): the durable mailbox between agents/terminals and the
// human. Items carry provenance (source, reason) and a triage state;
// responding to a blocking item IS its done — no separate step.

// Item kinds.
const (
	InboxFYI      = "fyi"
	InboxQuestion = "question"
	InboxApproval = "approval"
	InboxResult   = "result"
)

// Response verbs (Agent Inbox convention: per-item allow flags).
const (
	VerbAccept  = "accept"
	VerbEdit    = "edit"
	VerbRespond = "respond"
	VerbIgnore  = "ignore"
)

// Triage states.
const (
	InboxUnread = "unread"
	InboxRead   = "read"
	InboxDone   = "done"
)

// Source kinds.
const (
	InboxFromAgent    = "agent"
	InboxFromTerminal = "terminal"
	InboxFromSystem   = "system"
	// InboxFromAutomation items are filed by the automations engine
	// (ADR-0045); SourceID is the automation id. Nobody replies to them,
	// so RespondAndForward never forwards for this kind.
	InboxFromAutomation = "automation"
)

const (
	maxInboxTitle = 200
	maxInboxBody  = 100_000
)

// InboxItem is one mailbox row.
type InboxItem struct {
	ID          string   `json:"id"`
	Kind        string   `json:"kind"`
	SourceKind  string   `json:"sourceKind"`
	SourceID    string   `json:"sourceId,omitempty"`
	WorkspaceID string   `json:"workspaceId,omitempty"`
	Reason      string   `json:"reason"`
	Title       string   `json:"title"`
	Body        string   `json:"body,omitempty"`
	Blocking    bool     `json:"blocking"`
	Allowed     []string `json:"allowedResponses"`
	State       string   `json:"state"`
	Snoozed     *string  `json:"snoozedUntil,omitempty"`
	Response    *string  `json:"response,omitempty"`
	RespondedAt *string  `json:"respondedAt,omitempty"`
	CreatedAt   string   `json:"createdAt"`
	UpdatedAt   string   `json:"updatedAt"`
}

// InboxItemParams is CreateInboxItem's input.
type InboxItemParams struct {
	Kind        string
	SourceKind  string
	SourceID    string
	WorkspaceID string
	Reason      string
	Title       string
	Body        string
	Blocking    bool
	Allowed     []string
}

// InboxFilter narrows ListInboxItems.
type InboxFilter struct {
	State          string // "" = not-done (unread+read)
	Blocking       *bool
	IncludeSnoozed bool
	Limit          int
}

const inboxCols = `id, kind, source_kind, source_id, workspace_id, reason, title, body,
	blocking, allowed_responses, state, snoozed_until, response, responded_at, created_at, updated_at`

func scanInboxItem(row interface{ Scan(...any) error }, it *InboxItem) error {
	var blocking int
	var allowed string
	if err := row.Scan(&it.ID, &it.Kind, &it.SourceKind, &it.SourceID, &it.WorkspaceID,
		&it.Reason, &it.Title, &it.Body, &blocking, &allowed, &it.State,
		&it.Snoozed, &it.Response, &it.RespondedAt, &it.CreatedAt, &it.UpdatedAt); err != nil {
		return err
	}
	it.Blocking = blocking != 0
	it.Allowed = decodeVerbs(allowed)
	return nil
}

func decodeVerbs(s string) []string {
	out := []string{}
	_ = json.Unmarshal([]byte(s), &out)
	return out
}

func validVerb(v string) bool {
	switch v {
	case VerbAccept, VerbEdit, VerbRespond, VerbIgnore:
		return true
	}
	return false
}

func defaultVerbs(kind string) []string {
	switch kind {
	case InboxQuestion:
		return []string{VerbRespond, VerbIgnore}
	case InboxApproval:
		return []string{VerbAccept, VerbRespond, VerbIgnore}
	}
	return []string{VerbIgnore}
}

func normalizeInboxItem(p InboxItemParams) (InboxItemParams, error) {
	switch p.Kind {
	case InboxFYI, InboxQuestion, InboxApproval, InboxResult:
	default:
		return p, fmt.Errorf("kind must be fyi, question, approval or result")
	}
	switch p.SourceKind {
	case InboxFromAgent, InboxFromTerminal, InboxFromSystem, InboxFromAutomation:
	default:
		return p, fmt.Errorf("sourceKind must be agent, terminal, system or automation")
	}
	p.Reason = strings.TrimSpace(p.Reason)
	if p.Reason == "" {
		return p, fmt.Errorf("reason is required")
	}
	p.Title = strings.TrimSpace(p.Title)
	if p.Title == "" {
		return p, fmt.Errorf("title is required")
	}
	if len(p.Title) > maxInboxTitle {
		p.Title = p.Title[:maxInboxTitle]
	}
	if len(p.Body) > maxInboxBody {
		p.Body = p.Body[:maxInboxBody]
	}
	if p.Kind == InboxQuestion && strings.TrimSpace(p.Body) == "" {
		return p, fmt.Errorf("a question needs a body")
	}
	if len(p.Allowed) == 0 {
		p.Allowed = defaultVerbs(p.Kind)
	}
	seen := map[string]bool{}
	verbs := []string{}
	for _, v := range p.Allowed {
		v = strings.ToLower(strings.TrimSpace(v))
		if !validVerb(v) {
			return p, fmt.Errorf("allowed response %q unknown", v)
		}
		if !seen[v] {
			seen[v] = true
			verbs = append(verbs, v)
		}
	}
	p.Allowed = verbs
	// Questions and approvals block by nature unless the caller says otherwise.
	if p.Kind == InboxQuestion || p.Kind == InboxApproval {
		p.Blocking = true
	}
	return p, nil
}

// CreateInboxItem files one item.
func (s *Store) CreateInboxItem(p InboxItemParams) (InboxItem, error) {
	p, err := normalizeInboxItem(p)
	if err != nil {
		return InboxItem{}, err
	}
	now := nowUTC()
	allowed, _ := json.Marshal(p.Allowed)
	it := InboxItem{
		ID: newID(p.Title, "inb"), Kind: p.Kind, SourceKind: p.SourceKind,
		SourceID: p.SourceID, WorkspaceID: p.WorkspaceID, Reason: p.Reason,
		Title: p.Title, Body: p.Body, Blocking: p.Blocking, Allowed: p.Allowed,
		State: InboxUnread, CreatedAt: now, UpdatedAt: now,
	}
	if _, err := s.db.Exec(`INSERT INTO inbox_items
		(id, kind, source_kind, source_id, workspace_id, reason, title, body, blocking, allowed_responses, state, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		it.ID, it.Kind, it.SourceKind, it.SourceID, it.WorkspaceID, it.Reason,
		it.Title, it.Body, boolInt(it.Blocking), string(allowed), it.State, it.CreatedAt, it.UpdatedAt); err != nil {
		return InboxItem{}, fmt.Errorf("store: create inbox item: %w", err)
	}
	if s.OnInboxCreated != nil {
		s.OnInboxCreated(it)
	}
	return it, nil
}

// GetInboxItem returns one item.
func (s *Store) GetInboxItem(id string) (InboxItem, error) {
	var it InboxItem
	err := scanInboxItem(s.db.QueryRow(`SELECT `+inboxCols+` FROM inbox_items WHERE id = ?`, id), &it)
	if err == sql.ErrNoRows {
		return InboxItem{}, ErrNotFound
	}
	if err != nil {
		return InboxItem{}, fmt.Errorf("store: get inbox item: %w", err)
	}
	return it, nil
}

// ListInboxItems returns items newest first. Snoozed items are hidden
// until due unless IncludeSnoozed. snoozed_until is stored at second
// precision (RFC3339, never Nano) so the lexicographic comparison below
// is correct against a second-precision now.
func (s *Store) ListInboxItems(f InboxFilter) ([]InboxItem, error) {
	q := `SELECT ` + inboxCols + ` FROM inbox_items WHERE 1=1`
	args := []any{}
	if f.State != "" {
		q += ` AND state = ?`
		args = append(args, f.State)
	} else {
		q += ` AND state != ?`
		args = append(args, InboxDone)
	}
	if f.Blocking != nil {
		q += ` AND blocking = ?`
		args = append(args, boolInt(*f.Blocking))
	}
	if !f.IncludeSnoozed {
		q += ` AND (snoozed_until IS NULL OR snoozed_until <= ?)`
		args = append(args, time.Now().UTC().Format(time.RFC3339))
	}
	q += ` ORDER BY created_at DESC`
	if f.Limit > 0 {
		q += ` LIMIT ?`
		args = append(args, f.Limit)
	}
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("store: list inbox: %w", err)
	}
	defer rows.Close()
	out := []InboxItem{}
	for rows.Next() {
		var it InboxItem
		if err := scanInboxItem(rows, &it); err != nil {
			return nil, fmt.Errorf("store: scan inbox item: %w", err)
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// RespondInboxItem answers an item with one of its allowed verbs and
// marks it done (responding IS the done).
func (s *Store) RespondInboxItem(id, verb, text string) (InboxItem, error) {
	it, err := s.GetInboxItem(id)
	if err != nil {
		return InboxItem{}, err
	}
	if it.State == InboxDone {
		return InboxItem{}, fmt.Errorf("item is already done")
	}
	allowed := false
	for _, v := range it.Allowed {
		if v == verb {
			allowed = true
		}
	}
	if !allowed {
		return InboxItem{}, fmt.Errorf("response %q is not allowed on this item", verb)
	}
	now := nowUTC()
	resp := verb
	if strings.TrimSpace(text) != "" {
		resp = verb + ": " + text
	}
	res, err := s.db.Exec(`UPDATE inbox_items SET state = ?, response = ?, responded_at = ?, updated_at = ? WHERE id = ? AND state != ?`,
		InboxDone, resp, now, now, id, InboxDone)
	if err != nil {
		return InboxItem{}, fmt.Errorf("store: respond inbox item: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return InboxItem{}, ErrNotFound
	}
	return s.GetInboxItem(id)
}

// SetInboxItemState triages an item (unread/read/done) and/or snoozes it.
// Setting a state clears any snooze unless one is provided.
func (s *Store) SetInboxItemState(id, state string, snoozedUntil *string) (InboxItem, error) {
	switch state {
	case InboxUnread, InboxRead, InboxDone:
	case "":
		if snoozedUntil == nil {
			return InboxItem{}, fmt.Errorf("state or snoozedUntil is required")
		}
	default:
		return InboxItem{}, fmt.Errorf("state must be unread, read or done")
	}
	now := nowUTC()
	var res sql.Result
	var err error
	if state == "" {
		res, err = s.db.Exec(`UPDATE inbox_items SET snoozed_until = ?, updated_at = ? WHERE id = ?`, snoozedUntil, now, id)
	} else {
		res, err = s.db.Exec(`UPDATE inbox_items SET state = ?, snoozed_until = ?, updated_at = ? WHERE id = ?`, state, snoozedUntil, now, id)
	}
	if err != nil {
		return InboxItem{}, fmt.Errorf("store: set inbox state: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return InboxItem{}, ErrNotFound
	}
	return s.GetInboxItem(id)
}

// AnnotateInboxItem appends a visible note to the item body (e.g. a
// delivery failure) without inventing a new column.
func (s *Store) AnnotateInboxItem(id, note string) error {
	res, err := s.db.Exec(`UPDATE inbox_items SET body = body || ?, updated_at = ? WHERE id = ?`,
		"\n\n> "+note, nowUTC(), id)
	if err != nil {
		return fmt.Errorf("store: annotate inbox item: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// CountInboxBadge feeds the app badge: how many blocking items are not
// done, and whether any non-blocking unread news exists. Snoozed items
// don't count.
func (s *Store) CountInboxBadge() (blocking int, other bool, err error) {
	now := time.Now().UTC().Format(time.RFC3339)
	var otherN int
	err = s.db.QueryRow(`SELECT
		COUNT(CASE WHEN blocking = 1 THEN 1 END),
		COUNT(CASE WHEN blocking = 0 AND state = 'unread' THEN 1 END)
		FROM inbox_items WHERE state != 'done' AND (snoozed_until IS NULL OR snoozed_until <= ?)`, now).
		Scan(&blocking, &otherN)
	if err != nil {
		return 0, false, fmt.Errorf("store: inbox badge: %w", err)
	}
	return blocking, otherN > 0, nil
}

// DeleteInboxItem permanently removes one item, regardless of state. The
// caller (server/app layer) decides when to expose this — today only on
// already-done items — the store itself doesn't gate by state, same as
// DeletePin.
func (s *Store) DeleteInboxItem(id string) error {
	res, err := s.db.Exec(`DELETE FROM inbox_items WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("store: delete inbox item: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteDoneInboxItems permanently removes every done item and reports
// how many were removed (for a "Cleared N item(s)" toast).
func (s *Store) DeleteDoneInboxItems() (int, error) {
	res, err := s.db.Exec(`DELETE FROM inbox_items WHERE state = ?`, InboxDone)
	if err != nil {
		return 0, fmt.Errorf("store: clear done inbox items: %w", err)
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

// CountInboxItems returns how many items are currently in this exact
// state, for a tab badge. Unlike ListInboxItems' State field, there is
// no "" or "all" special case here — pass InboxUnread, InboxRead or
// InboxDone explicitly; reusing "" with a different meaning in the same
// file would confuse more than it'd save.
func (s *Store) CountInboxItems(state string) (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM inbox_items WHERE state = ?`, state).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("store: count inbox items: %w", err)
	}
	return n, nil
}

// CountAllInboxItems returns the total row count across every state, for
// the "All" tab's badge.
func (s *Store) CountAllInboxItems() (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM inbox_items`).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("store: count all inbox items: %w", err)
	}
	return n, nil
}

// FileAgentResult files a `result` item for an agent run, consolidating
// noise: an unread result from the same source is updated in place, so a
// chatty agent yields one unread item, not a pile.
func (s *Store) FileAgentResult(agentID, workspaceID, title, body, reason string) (InboxItem, error) {
	var existing string
	err := s.db.QueryRow(`SELECT id FROM inbox_items
		WHERE kind = ? AND source_kind = ? AND source_id = ? AND state = ?
		ORDER BY created_at DESC LIMIT 1`,
		InboxResult, InboxFromAgent, agentID, InboxUnread).Scan(&existing)
	if err == nil && existing != "" {
		now := nowUTC()
		if len(body) > maxInboxBody {
			body = body[:maxInboxBody]
		}
		if _, err := s.db.Exec(`UPDATE inbox_items SET title = ?, body = ?, reason = ?, updated_at = ? WHERE id = ?`,
			title, body, reason, now, existing); err != nil {
			return InboxItem{}, fmt.Errorf("store: update result item: %w", err)
		}
		it, err := s.GetInboxItem(existing)
		// A superseded result is news too; the notifier's tag collapses
		// it onto the earlier notification for the same agent.
		if err == nil && s.OnInboxCreated != nil {
			s.OnInboxCreated(it)
		}
		return it, err
	}
	return s.CreateInboxItem(InboxItemParams{
		Kind: InboxResult, SourceKind: InboxFromAgent, SourceID: agentID,
		WorkspaceID: workspaceID, Reason: reason, Title: title, Body: body,
	})
}

// RespondAndForward answers an item and, when it is an agent-sourced
// question/approval, forwards the reply to the agent as a durable
// follow_up task (park-and-wake: a stopped agent drains it on next
// start). An agent row that no longer exists annotates the item and
// leaves it open — a visible failure, never a lost message.
func (s *Store) RespondAndForward(id, verb, text string, deliverable AgentDeliverable) (InboxItem, error) {
	it, err := s.GetInboxItem(id)
	if err != nil {
		return InboxItem{}, err
	}
	// Validate BEFORE forwarding: a rejected verb or an already-done item
	// must never reach the agent's queue.
	if it.State == InboxDone {
		return InboxItem{}, fmt.Errorf("item is already done")
	}
	allowed := false
	for _, v := range it.Allowed {
		if v == verb {
			allowed = true
		}
	}
	if !allowed {
		return InboxItem{}, fmt.Errorf("response %q is not allowed on this item", verb)
	}
	needsForward := it.SourceKind == InboxFromAgent &&
		(it.Kind == InboxQuestion || it.Kind == InboxApproval) &&
		verb != VerbIgnore
	if needsForward {
		if deliverable != nil && !deliverable(it.SourceID) {
			_ = s.AnnotateInboxItem(id, "Reply not delivered: the agent is running in an interactive terminal, "+
				"which does not pick up follow-up messages automatically. Open its terminal and paste the reply, "+
				"or stop the agent and start it managed so replies deliver on their own.")
			return InboxItem{}, ErrAgentInteractive
		}
		payload := fmt.Sprintf("Human reply to your question %q: %s", it.Title, text)
		if verb == VerbAccept && strings.TrimSpace(text) == "" {
			payload = fmt.Sprintf("The human accepted: %q", it.Title)
		}
		if _, err := s.EnqueueTask(it.SourceID, TaskFollowUp, payload, "inbox"); err != nil {
			if err == ErrNotFound {
				_ = s.AnnotateInboxItem(id, "Reply could not be delivered: agent no longer exists.")
				return InboxItem{}, fmt.Errorf("agent no longer exists: %w", ErrNotFound)
			}
			return InboxItem{}, err
		}
	}
	return s.RespondInboxItem(id, verb, text)
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
