package store

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/cfpperche/picode/internal/remind"
)

// PinReminder is one pin's cadence (ADR-0100). The row owns *when*; the
// Inbox item it files owns *whether the person has dealt with it*.
type PinReminder struct {
	ID          string  `json:"id"`
	PinID       string  `json:"pinId"`
	Kind        string  `json:"kind"`         // once | interval | cron
	At          string  `json:"at,omitempty"` // once: RFC3339 UTC
	IntervalMin int     `json:"intervalMin,omitempty"`
	Cron        string  `json:"cron,omitempty"`
	Anchor      string  `json:"anchor"` // schedule | completion
	TZ          string  `json:"tz"`
	NextAt      *string `json:"nextAt"` // UTC; null = nothing owed right now
	LastFiredAt *string `json:"lastFiredAt,omitempty"`
	Enabled     bool    `json:"enabled"`
	Label       string  `json:"label"` // the cadence in words
	CreatedAt   string  `json:"createdAt"`
	UpdatedAt   string  `json:"updatedAt"`
}

// PinReminderParams is what the picker sends.
type PinReminderParams struct {
	Kind        string
	At          string // RFC3339, any offset
	IntervalMin int
	Cron        string
	Anchor      string
	TZ          string
	Enabled     *bool
}

const pinReminderCols = `id, pin_id, kind, at, interval_min, cron, anchor, tz, next_at, last_fired_at, enabled, created_at, updated_at`

func (r PinReminder) rule() remind.Rule {
	rule := remind.Rule{Kind: r.Kind, Interval: time.Duration(r.IntervalMin) * time.Minute, Cron: r.Cron, Anchor: r.Anchor, TZ: r.TZ}
	if r.At != "" {
		rule.At, _ = time.Parse(time.RFC3339Nano, r.At)
	}
	return rule
}

func scanPinReminder(row interface{ Scan(...any) error }, r *PinReminder) error {
	var at, next, last sql.NullString
	var enabled int
	if err := row.Scan(&r.ID, &r.PinID, &r.Kind, &at, &r.IntervalMin, &r.Cron, &r.Anchor, &r.TZ, &next, &last, &enabled, &r.CreatedAt, &r.UpdatedAt); err != nil {
		return err
	}
	r.At = at.String
	if next.Valid {
		v := next.String
		r.NextAt = &v
	}
	if last.Valid {
		v := last.String
		r.LastFiredAt = &v
	}
	r.Enabled = enabled == 1
	r.Label = r.rule().Label()
	return nil
}

func (s *Store) GetPinReminder(pinID string) (*PinReminder, error) {
	var r PinReminder
	err := scanPinReminder(s.db.QueryRow(`SELECT `+pinReminderCols+` FROM pin_reminders WHERE pin_id = ?`, pinID), &r)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("store: get reminder: %w", err)
	}
	return &r, nil
}

func (s *Store) getPinReminderByID(id string) (PinReminder, error) {
	var r PinReminder
	err := scanPinReminder(s.db.QueryRow(`SELECT `+pinReminderCols+` FROM pin_reminders WHERE id = ?`, id), &r)
	if err == sql.ErrNoRows {
		return PinReminder{}, ErrNotFound
	}
	if err != nil {
		return PinReminder{}, fmt.Errorf("store: get reminder: %w", err)
	}
	return r, nil
}

// remindersByPin loads every reminder keyed by pin, for the list.
func (s *Store) remindersByPin() (map[string]*PinReminder, error) {
	rows, err := s.db.Query(`SELECT ` + pinReminderCols + ` FROM pin_reminders`)
	if err != nil {
		return nil, fmt.Errorf("store: list reminders: %w", err)
	}
	defer rows.Close()
	out := map[string]*PinReminder{}
	for rows.Next() {
		var r PinReminder
		if err := scanPinReminder(rows, &r); err != nil {
			return nil, err
		}
		rr := r
		out[r.PinID] = &rr
	}
	return out, rows.Err()
}

// SetPinReminder creates or replaces the pin's reminder. A replaced rule
// starts over: next_at is computed from now, an open Inbox item from the
// old rule stays (it is still owed) but reads the new label on its next
// fire.
func (s *Store) SetPinReminder(pinID string, p PinReminderParams) (PinReminder, error) {
	if _, err := s.GetPin(pinID); err != nil {
		return PinReminder{}, err
	}
	now := time.Now().UTC()
	rule := remind.Rule{Kind: strings.TrimSpace(p.Kind), Interval: time.Duration(p.IntervalMin) * time.Minute, Cron: strings.TrimSpace(p.Cron), Anchor: strings.TrimSpace(p.Anchor), TZ: strings.TrimSpace(p.TZ)}
	if rule.Kind == remind.KindInterval && rule.Anchor == "" {
		rule.Anchor = remind.AnchorSchedule
	}
	if rule.Kind != remind.KindInterval {
		rule.Anchor = remind.AnchorSchedule
	}
	if rule.Kind == remind.KindOnce && strings.TrimSpace(p.At) != "" {
		at, err := time.Parse(time.RFC3339, strings.TrimSpace(p.At))
		if err != nil {
			return PinReminder{}, invalid("the date and time must be RFC 3339")
		}
		rule.At = at.UTC()
	}
	if err := rule.Validate(now); err != nil {
		return PinReminder{}, invalid("%s", strings.TrimPrefix(err.Error(), remind.ErrInvalid.Error()+": "))
	}
	enabled := true
	if p.Enabled != nil {
		enabled = *p.Enabled
	}
	var next *string
	if t, ok := rule.First(now); ok {
		v := t.Format(time.RFC3339Nano)
		next = &v
	}
	stamp := now.Format(time.RFC3339Nano)
	r := PinReminder{ID: newID("reminder", "rem"), PinID: pinID, Kind: rule.Kind, IntervalMin: p.IntervalMin, Cron: rule.Cron, Anchor: rule.Anchor, TZ: rule.TZ, NextAt: next, Enabled: enabled, CreatedAt: stamp, UpdatedAt: stamp}
	if rule.Kind == remind.KindOnce {
		r.At = rule.At.Format(time.RFC3339Nano)
	}
	if rule.Kind != remind.KindInterval {
		r.IntervalMin = 0
	}
	if rule.Kind != remind.KindCron {
		r.Cron = ""
	}
	// Keep the id and created_at of an existing row so the event stream
	// and any open item keep pointing at the same reminder.
	if cur, err := s.GetPinReminder(pinID); err == nil && cur != nil {
		r.ID, r.CreatedAt = cur.ID, cur.CreatedAt
		if _, err := s.db.Exec(`UPDATE pin_reminders SET kind=?, at=?, interval_min=?, cron=?, anchor=?, tz=?, next_at=?, last_fired_at=NULL, enabled=?, updated_at=? WHERE id=?`,
			r.Kind, nullIfEmpty(r.At), r.IntervalMin, r.Cron, r.Anchor, r.TZ, r.NextAt, boolInt(r.Enabled), r.UpdatedAt, r.ID); err != nil {
			return PinReminder{}, fmt.Errorf("store: update reminder: %w", err)
		}
	} else {
		if _, err := s.db.Exec(`INSERT INTO pin_reminders (`+pinReminderCols+`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, NULL, ?, ?, ?)`,
			r.ID, r.PinID, r.Kind, nullIfEmpty(r.At), r.IntervalMin, r.Cron, r.Anchor, r.TZ, r.NextAt, boolInt(r.Enabled), r.CreatedAt, r.UpdatedAt); err != nil {
			return PinReminder{}, fmt.Errorf("store: add reminder: %w", err)
		}
	}
	r.Label = rule.Label()
	s.notePinUpdated(pinID)
	return r, nil
}

// DeletePinReminder removes the rule and closes the item it may have open:
// nothing owed once the person said "no more".
func (s *Store) DeletePinReminder(pinID string) error {
	res, err := s.db.Exec(`DELETE FROM pin_reminders WHERE pin_id = ?`, pinID)
	if err != nil {
		return fmt.Errorf("store: delete reminder: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	s.closeReminderItems(s.db, pinID)
	s.notePinUpdated(pinID)
	return nil
}

func nullIfEmpty(v string) any {
	if v == "" {
		return nil
	}
	return v
}

// DueReminderIDs is the engine's question: which enabled rules owe a fire.
func (s *Store) DueReminderIDs(now time.Time) ([]string, error) {
	rows, err := s.db.Query(`SELECT id FROM pin_reminders WHERE enabled = 1 AND next_at IS NOT NULL AND next_at <= ? ORDER BY next_at`, now.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return nil, fmt.Errorf("store: due reminders: %w", err)
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// ReminderFire is the pin.reminded payload.
type ReminderFire struct {
	PinID      string `json:"pinId"`
	ReminderID string `json:"reminderId"`
	InboxID    string `json:"inboxId"`
	Title      string `json:"title"`
	Label      string `json:"label"`
	At         string `json:"at"` // the slot, RFC3339 UTC
	CatchUp    bool   `json:"catchUp"`
}

// openReminderItem finds the pin's reminder item that is not done.
func (s *Store) openReminderItem(pinID string) (InboxItem, bool) {
	var it InboxItem
	err := scanInboxItem(s.db.QueryRow(`SELECT `+inboxCols+` FROM inbox_items WHERE kind = ? AND source_kind = ? AND source_id = ? AND state != ? ORDER BY created_at DESC LIMIT 1`,
		InboxReminder, InboxFromPin, pinID, InboxDone), &it)
	return it, err == nil
}

// FireReminder honours one due slot, in one transaction (ADR-0100):
//   - no open item → file one (unread) and announce pin.reminded
//   - open item, not snoozed → touch it and announce again (one card, never a pile)
//   - open item, snoozed → say nothing; the snooze is the person's word
//   - then advance next_at by the rule; stamp last_fired_at
func (s *Store) FireReminder(id string, now time.Time) error {
	r, err := s.getPinReminderByID(id)
	if err != nil {
		return err
	}
	pin, err := s.GetPin(r.PinID)
	if err != nil {
		return err
	}
	slot := now
	if r.NextAt != nil {
		if t, err := time.Parse(time.RFC3339Nano, *r.NextAt); err == nil {
			slot = t
		}
	}
	rule := r.rule()
	catchUp := remind.CatchUp(slot, now)
	// Read before Begin: the pool hands the transaction its only
	// connection, so a query on s.db inside it would wait forever.
	open, hasOpen := s.openReminderItem(pin.ID)
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	stamp := now.UTC().Format(time.RFC3339Nano)
	fire := ReminderFire{PinID: pin.ID, ReminderID: r.ID, Title: pin.Title, Label: rule.Label(), At: slot.UTC().Format(time.RFC3339), CatchUp: catchUp}
	announce := true
	if hasOpen {
		fire.InboxID = open.ID
		if open.Snoozed != nil && *open.Snoozed > now.UTC().Format(time.RFC3339) {
			announce = false
		} else if _, err := tx.Exec(`UPDATE inbox_items SET updated_at = ?, snoozed_until = NULL WHERE id = ?`, stamp, open.ID); err != nil {
			s.rollback(tx)
			return fmt.Errorf("store: touch reminder item: %w", err)
		}
	} else {
		body := firstProseLine(pin.Body)
		if catchUp {
			body = strings.TrimSpace(body + "\n\nWas due " + slot.In(rule.Location()).Format("Mon 2 Jan 15:04") + ".")
		}
		it := InboxItem{
			ID: newID(pin.Title, "rem"), Kind: InboxReminder, SourceKind: InboxFromPin, SourceID: pin.ID,
			Reason: rule.Label(), Title: pin.Title, Body: body, Allowed: []string{VerbIgnore},
			State: InboxUnread, CreatedAt: stamp, UpdatedAt: stamp,
		}
		if _, err := tx.Exec(`INSERT INTO inbox_items
			(id, kind, source_kind, source_id, workspace_id, reason, title, body, blocking, allowed_responses, state, created_at, updated_at, session_path)
			VALUES (?, ?, ?, ?, '', ?, ?, ?, 0, '["ignore"]', ?, ?, ?, '')`,
			it.ID, it.Kind, it.SourceKind, it.SourceID, it.Reason, it.Title, it.Body, it.State, it.CreatedAt, it.UpdatedAt); err != nil {
			s.rollback(tx)
			return fmt.Errorf("store: file reminder item: %w", err)
		}
		fire.InboxID = it.ID
		if err := s.AppendEventTx(tx, "inbox.created", nil, nil, it); err != nil {
			s.rollback(tx)
			return err
		}
	}
	var next any
	if t, ok := rule.AfterFire(slot, now); ok {
		next = t.Format(time.RFC3339Nano)
	}
	enabled := 1
	if r.Kind == remind.KindOnce {
		enabled = 0
	}
	if _, err := tx.Exec(`UPDATE pin_reminders SET next_at = ?, last_fired_at = ?, enabled = ?, updated_at = ? WHERE id = ?`, next, stamp, enabled, stamp, r.ID); err != nil {
		s.rollback(tx)
		return fmt.Errorf("store: advance reminder: %w", err)
	}
	if announce {
		if err := s.AppendEventTx(tx, "pin.reminded", nil, nil, fire); err != nil {
			s.rollback(tx)
			return err
		}
	}
	if err := s.commit(tx); err != nil {
		return err
	}
	if announce && fire.InboxID != "" {
		_, _ = s.inboxChanged(fire.InboxID)
	}
	s.notePinUpdated(pin.ID)
	return nil
}

// WakeSnoozedReminders re-raises reminder items whose snooze ended:
// snoozed_until is cleared (so this runs once per snooze) and pin.reminded
// is announced with the same item, so every browser shows the card again.
func (s *Store) WakeSnoozedReminders(now time.Time) (int, error) {
	rows, err := s.db.Query(`SELECT `+inboxCols+` FROM inbox_items WHERE kind = ? AND state != ? AND snoozed_until IS NOT NULL AND snoozed_until <= ?`,
		InboxReminder, InboxDone, now.UTC().Format(time.RFC3339))
	if err != nil {
		return 0, fmt.Errorf("store: snoozed reminders: %w", err)
	}
	var items []InboxItem
	for rows.Next() {
		var it InboxItem
		if err := scanInboxItem(rows, &it); err != nil {
			rows.Close()
			return 0, err
		}
		items = append(items, it)
	}
	rows.Close()
	woken := 0
	for _, it := range items {
		r, _ := s.GetPinReminder(it.SourceID)
		stamp := now.UTC().Format(time.RFC3339Nano)
		if _, err := s.db.Exec(`UPDATE inbox_items SET snoozed_until = NULL, updated_at = ? WHERE id = ?`, stamp, it.ID); err != nil {
			return woken, fmt.Errorf("store: wake reminder: %w", err)
		}
		fire := ReminderFire{PinID: it.SourceID, InboxID: it.ID, Title: it.Title, Label: it.Reason, At: stamp}
		if r != nil {
			fire.ReminderID = r.ID
			fire.Label = r.Label
		}
		s.note("pin.reminded", nil, nil, fire)
		_, _ = s.inboxChanged(it.ID)
		woken++
	}
	return woken, nil
}

// reminderClosed runs when a reminder item goes done: a completion-anchored
// interval starts counting from now.
func (s *Store) reminderClosed(it InboxItem, now string) {
	r, err := s.GetPinReminder(it.SourceID)
	if err != nil || r == nil || !r.Enabled {
		return
	}
	closedAt, err := time.Parse(time.RFC3339Nano, now)
	if err != nil {
		closedAt = time.Now().UTC()
	}
	next, ok := r.rule().AfterClose(closedAt)
	if !ok {
		return
	}
	if _, err := s.db.Exec(`UPDATE pin_reminders SET next_at = ?, updated_at = ? WHERE id = ?`, next.Format(time.RFC3339Nano), now, r.ID); err == nil {
		s.notePinUpdated(it.SourceID)
	}
}

type execer interface {
	Exec(query string, args ...any) (sql.Result, error)
}

// closeReminderItems marks the pin's open reminder items done: the pin or
// its rule is gone, so nothing is owed any more.
func (s *Store) closeReminderItems(db execer, pinID string) {
	_, _ = db.Exec(`UPDATE inbox_items SET state = ?, snoozed_until = NULL, updated_at = ? WHERE kind = ? AND source_kind = ? AND source_id = ? AND state != ?`,
		InboxDone, nowUTC(), InboxReminder, InboxFromPin, pinID, InboxDone)
}

// firstProseLine is the first line of prose in a markdown body: fenced
// code is skipped whole, headings and rules too (notice.js's rule).
func firstProseLine(text string) string {
	fenced := false
	for _, raw := range strings.Split(text, "\n") {
		line := strings.TrimSpace(raw)
		if strings.HasPrefix(line, "```") || strings.HasPrefix(line, "~~~") {
			fenced = !fenced
			continue
		}
		if fenced || line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.Trim(line, "-*_= ") == "" {
			continue
		}
		if len(line) > 200 {
			line = line[:200]
			for !utf8.ValidString(line) {
				line = line[:len(line)-1]
			}
			line += "…"
		}
		return line
	}
	return ""
}
