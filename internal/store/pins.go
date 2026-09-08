package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	maxPinTags   = 16
	maxPinTitle  = 200     // runes
	maxPinTag    = 40      // runes
	maxPinBody   = 100_000 // bytes of UTF-8
	maxPinBodyKB = maxPinBody / 1000
)

// ErrInvalid marks a pin request the caller can fix (a limit, a missing
// title). Handlers map it to 400; everything else from this file is a
// store failure and stays a 500.
var ErrInvalid = errors.New("invalid")

// ErrConflict is the optimistic-concurrency refusal: the caller sent the
// updatedAt it last saw and the row moved on since.
var ErrConflict = errors.New("This pin changed elsewhere. Reload to see the latest version.")

// invalidError reads as its message alone (the studio shows it verbatim)
// and still answers errors.Is(err, ErrInvalid).
type invalidError struct{ msg string }

func (e invalidError) Error() string        { return e.msg }
func (e invalidError) Is(target error) bool { return target == ErrInvalid }

func invalid(format string, args ...any) error {
	return invalidError{fmt.Sprintf(format, args...)}
}

// Pin is a flat machine-scoped note (no folder tree).
type Pin struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Tags      []string  `json:"tags"`
	Body      string    `json:"body"`
	CreatedAt string    `json:"createdAt"`
	UpdatedAt string    `json:"updatedAt"`
	FileCount int       `json:"fileCount"`
	Files     []PinFile `json:"files,omitempty"`
	// Reminder is the pin's cadence, if any (ADR-0100).
	Reminder *PinReminder `json:"reminder,omitempty"`
	// Starred keeps the pin on top of the list; ArchivedAt takes it out of
	// the sidebar (search still finds it; its reminder is paused).
	Starred    bool    `json:"starred"`
	ArchivedAt *string `json:"archivedAt,omitempty"`
}

// PinSummary is what a list and a feed event carry: never the body. The
// sidebar renders title, tags and a count; a 100 KB note has no business
// riding every SSE connection or every tab switch.
type PinSummary struct {
	ID         string       `json:"id"`
	Title      string       `json:"title"`
	Tags       []string     `json:"tags"`
	CreatedAt  string       `json:"createdAt"`
	UpdatedAt  string       `json:"updatedAt"`
	FileCount  int          `json:"fileCount"`
	Reminder   *PinReminder `json:"reminder,omitempty"`
	Starred    bool         `json:"starred"`
	ArchivedAt *string      `json:"archivedAt,omitempty"`
}

func (p Pin) Summary() PinSummary {
	return PinSummary{ID: p.ID, Title: p.Title, Tags: p.Tags, CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt, FileCount: p.FileCount, Reminder: p.Reminder, Starred: p.Starred, ArchivedAt: p.ArchivedAt}
}

// PinListFilter narrows ListPins. Q searches title, tags and body (every
// word must appear) across live and archived pins; without Q the list is
// the live pins, or the archived ones when Archived is set.
type PinListFilter struct {
	Q        string
	Archived bool
}

func scanPin(row interface{ Scan(...any) error }, p *Pin) error {
	var tags string
	var starred int
	var archived sql.NullString
	if err := row.Scan(&p.ID, &p.Title, &tags, &p.Body, &p.CreatedAt, &p.UpdatedAt, &starred, &archived); err != nil {
		return err
	}
	p.Tags = decodePackages(tags)
	p.Starred = starred == 1
	if archived.Valid {
		v := archived.String
		p.ArchivedAt = &v
	}
	return nil
}

const pinSummaryCols = `id, title, tags, created_at, updated_at, starred, archived_at,
		(SELECT COUNT(1) FROM pin_files f WHERE f.pin_id = pins.id)`

func scanPinSummary(row interface{ Scan(...any) error }, p *PinSummary) error {
	var tags string
	var starred int
	var archived sql.NullString
	if err := row.Scan(&p.ID, &p.Title, &tags, &p.CreatedAt, &p.UpdatedAt, &starred, &archived, &p.FileCount); err != nil {
		return err
	}
	p.Tags = decodePackages(tags)
	p.Starred = starred == 1
	if archived.Valid {
		v := archived.String
		p.ArchivedAt = &v
	}
	return nil
}

// normalizeTag lower-cases, strips a leading '#', and folds whitespace runs
// into one '-'. Empty after that means "skip".
func normalizeTag(t string) string {
	t = strings.TrimPrefix(strings.TrimSpace(t), "#")
	return strings.ToLower(strings.Join(strings.Fields(t), "-"))
}

// normalizePin validates and cleans; it never truncates. A byte-sliced
// title left invalid UTF-8 in the database once (pins v2 review, finding
// 1), so limits refuse with a message the studio can show instead.
func normalizePin(title string, tags []string, body string) (string, []string, string, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return "", nil, "", invalid("title is required")
	}
	if !utf8.ValidString(title) || !utf8.ValidString(body) {
		return "", nil, "", invalid("text is not valid UTF-8")
	}
	if utf8.RuneCountInString(title) > maxPinTitle {
		return "", nil, "", invalid("title is too long (max %d characters)", maxPinTitle)
	}
	if len(body) > maxPinBody {
		return "", nil, "", invalid("note is too long (max %d KB)", maxPinBodyKB)
	}
	out := make([]string, 0, len(tags))
	seen := map[string]bool{}
	for _, raw := range tags {
		t := normalizeTag(raw)
		if t == "" || seen[t] {
			continue
		}
		if !utf8.ValidString(t) {
			return "", nil, "", invalid("tag is not valid UTF-8")
		}
		if utf8.RuneCountInString(t) > maxPinTag {
			return "", nil, "", invalid("tag %q is too long (max %d characters)", t, maxPinTag)
		}
		if len(out) >= maxPinTags {
			return "", nil, "", invalid("a pin can have at most %d tags", maxPinTags)
		}
		seen[t] = true
		out = append(out, t)
	}
	return title, out, body, nil
}

// ListPins: starred first, then most recently updated. A search runs over
// live and archived pins alike (an archived note is still the note you
// wrote); every word of the query must appear in the title, a tag or the
// body. SQLite's LIKE is case-insensitive for ASCII only, so both sides
// are lower()ed for the rest.
func (s *Store) ListPins(f PinListFilter) ([]PinSummary, error) {
	q := `SELECT ` + pinSummaryCols + ` FROM pins WHERE 1=1`
	args := []any{}
	if words := strings.Fields(strings.ToLower(f.Q)); len(words) > 0 {
		for _, w := range words {
			like := "%" + escapeLike(w) + "%"
			q += ` AND (lower(title) LIKE ? ESCAPE '\' OR lower(tags) LIKE ? ESCAPE '\' OR lower(body) LIKE ? ESCAPE '\')`
			args = append(args, like, like, like)
		}
	} else if f.Archived {
		q += ` AND archived_at IS NOT NULL`
	} else {
		q += ` AND archived_at IS NULL`
	}
	q += ` ORDER BY starred DESC, (archived_at IS NOT NULL), updated_at DESC`
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("store: list pins: %w", err)
	}
	defer rows.Close()
	out := []PinSummary{}
	for rows.Next() {
		var p PinSummary
		if err := scanPinSummary(rows, &p); err != nil {
			return nil, fmt.Errorf("store: scan pin: %w", err)
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()
	rems, err := s.remindersByPin()
	if err != nil {
		return nil, err
	}
	for i := range out {
		out[i].Reminder = rems[out[i].ID]
	}
	return out, nil
}

// PinIDs lists every pin id, for the boot sweep that removes attachment
// directories whose row is gone.
func (s *Store) PinIDs() ([]string, error) {
	rows, err := s.db.Query(`SELECT id FROM pins`)
	if err != nil {
		return nil, fmt.Errorf("store: pin ids: %w", err)
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

func (s *Store) GetPin(id string) (Pin, error) {
	var p Pin
	err := scanPin(s.db.QueryRow(`SELECT id, title, tags, body, created_at, updated_at, starred, archived_at FROM pins WHERE id = ?`, id), &p)
	if err == sql.ErrNoRows {
		return Pin{}, ErrNotFound
	}
	if err != nil {
		return Pin{}, fmt.Errorf("store: get pin: %w", err)
	}
	files, err := s.ListPinFiles(p.ID)
	if err != nil {
		return Pin{}, err
	}
	p.Files = files
	p.FileCount = len(files)
	if p.Reminder, err = s.GetPinReminder(p.ID); err != nil {
		return Pin{}, err
	}
	return p, nil
}

func (s *Store) pinSummary(id string) (PinSummary, error) {
	var p PinSummary
	err := scanPinSummary(s.db.QueryRow(`SELECT `+pinSummaryCols+` FROM pins WHERE id = ?`, id), &p)
	if err == sql.ErrNoRows {
		return PinSummary{}, ErrNotFound
	}
	if err != nil {
		return PinSummary{}, fmt.Errorf("store: pin summary: %w", err)
	}
	if p.Reminder, err = s.GetPinReminder(id); err != nil {
		return PinSummary{}, err
	}
	return p, nil
}

// notePinUpdated is the one shape every pin.updated carries (ADR-0048):
// the summary, never the body. File routes and the editor emit the same.
func (s *Store) notePinUpdated(id string) {
	if p, err := s.pinSummary(id); err == nil {
		s.note("pin.updated", nil, nil, p)
	}
}

func (s *Store) CreatePin(title string, tags []string, body string) (Pin, error) {
	title, tags, body, err := normalizePin(title, tags, body)
	if err != nil {
		return Pin{}, err
	}
	now := nowUTC()
	p := Pin{ID: newID(title, "pin"), Title: title, Tags: tags, Body: body, CreatedAt: now, UpdatedAt: now}
	if _, err := s.db.Exec(`INSERT INTO pins (id, title, tags, body, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		p.ID, p.Title, encodePackages(p.Tags), p.Body, p.CreatedAt, p.UpdatedAt); err != nil {
		return Pin{}, fmt.Errorf("store: create pin: %w", err)
	}
	s.note("pin.created", nil, nil, p.Summary())
	return p, nil
}

// UpdatePin replaces title, tags and body. ifUpdatedAt, when not empty, is
// the updatedAt the caller last saw: a row that moved on answers
// ErrConflict instead of overwriting the other writer.
func (s *Store) UpdatePin(id, title string, tags []string, body, ifUpdatedAt string) (Pin, error) {
	title, tags, body, err := normalizePin(title, tags, body)
	if err != nil {
		return Pin{}, err
	}
	now := nowUTC()
	res, err := s.db.Exec(`UPDATE pins SET title = ?, tags = ?, body = ?, updated_at = ? WHERE id = ? AND (? = '' OR updated_at = ?)`,
		title, encodePackages(tags), body, now, id, ifUpdatedAt, ifUpdatedAt)
	if err != nil {
		return Pin{}, fmt.Errorf("store: update pin: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		var exists int
		if err := s.db.QueryRow(`SELECT COUNT(1) FROM pins WHERE id = ?`, id).Scan(&exists); err == nil && exists > 0 {
			return Pin{}, ErrConflict
		}
		return Pin{}, ErrNotFound
	}
	p, err := s.GetPin(id)
	if err != nil {
		return Pin{}, err
	}
	s.note("pin.updated", nil, nil, p.Summary())
	return p, nil
}

// DeletePin removes the pin, its files and reminder (cascade) and closes
// any reminder item still open for it — nothing is owed for a pin that
// is gone (ADR-0100).
func (s *Store) DeletePin(id string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	res, err := tx.Exec(`DELETE FROM pins WHERE id = ?`, id)
	if err != nil {
		s.rollback(tx)
		return fmt.Errorf("store: delete pin: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		s.rollback(tx)
		return ErrNotFound
	}
	s.closeReminderItems(tx, id)
	if err := s.AppendEventTx(tx, "pin.deleted", nil, nil, idData(id)); err != nil {
		s.rollback(tx)
		return err
	}
	return s.commit(tx)
}

func escapeLike(w string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return r.Replace(w)
}

// CountArchivedPins feeds the sidebar's "N archived" line.
func (s *Store) CountArchivedPins() (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(1) FROM pins WHERE archived_at IS NOT NULL`).Scan(&n)
	return n, err
}

// SetPinStarred keeps the pin on top (or not). Starring is not an edit:
// updated_at stays, so the order among starred pins is still recency.
func (s *Store) SetPinStarred(id string, starred bool) (Pin, error) {
	res, err := s.db.Exec(`UPDATE pins SET starred = ? WHERE id = ?`, boolInt(starred), id)
	if err != nil {
		return Pin{}, fmt.Errorf("store: star pin: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return Pin{}, ErrNotFound
	}
	p, err := s.GetPin(id)
	if err != nil {
		return Pin{}, err
	}
	s.note("pin.updated", nil, nil, p.Summary())
	return p, nil
}

// SetPinArchived puts the pin away or brings it back. Archiving pauses its
// reminder (the engine skips archived pins) and closes any reminder item
// still open — a note put away is not owed. Unarchiving resumes the rule:
// a slot missed meanwhile fires once, as after downtime.
func (s *Store) SetPinArchived(id string, archived bool) (Pin, error) {
	var at any
	if archived {
		at = nowUTC()
	}
	res, err := s.db.Exec(`UPDATE pins SET archived_at = ? WHERE id = ?`, at, id)
	if err != nil {
		return Pin{}, fmt.Errorf("store: archive pin: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return Pin{}, ErrNotFound
	}
	if archived {
		s.closeReminderItems(s.db, id)
	}
	p, err := s.GetPin(id)
	if err != nil {
		return Pin{}, err
	}
	s.note("pin.updated", nil, nil, p.Summary())
	return p, nil
}
