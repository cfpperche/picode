package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/cfpperche/picode/internal/snips"
)

const (
	maxSnipTitle  = 200
	maxSnipDesc   = 500
	maxSnipBody   = 100_000
	maxSnipBodyKB = maxSnipBody / 1000
)

// Snip is one machine-scoped template (ADR-0130).
type Snip struct {
	ID           string              `json:"id"`
	Slug         string              `json:"slug"`
	Title        string              `json:"title"`
	Description  string              `json:"description"`
	Kind         string              `json:"kind"`
	Body         string              `json:"body"`
	Placeholders []snips.Placeholder `json:"placeholders"`
	Tags         []string            `json:"tags"`
	Starred      bool                `json:"starred"`
	ArchivedAt   *string             `json:"archivedAt,omitempty"`
	CreatedAt    string              `json:"createdAt"`
	UpdatedAt    string              `json:"updatedAt"`
}

// SnipSummary is the list/feed view: never the body.
type SnipSummary struct {
	ID          string   `json:"id"`
	Slug        string   `json:"slug"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Kind        string   `json:"kind"`
	Tags        []string `json:"tags"`
	Starred     bool     `json:"starred"`
	ArchivedAt  *string  `json:"archivedAt,omitempty"`
	CreatedAt   string   `json:"createdAt"`
	UpdatedAt   string   `json:"updatedAt"`
}

func (p Snip) Summary() SnipSummary {
	return SnipSummary{
		ID: p.ID, Slug: p.Slug, Title: p.Title, Description: p.Description,
		Kind: p.Kind, Tags: p.Tags, Starred: p.Starred, ArchivedAt: p.ArchivedAt,
		CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
	}
}

// SnipPicker is a composer/palette row. No body.
type SnipPicker struct {
	ID    string   `json:"id"`
	Slug  string   `json:"slug"`
	Title string   `json:"title"`
	Hint  string   `json:"hint"`
	Kind  string   `json:"kind"`
	Names []string `json:"names,omitempty"`
}

// SnipParams is create/update input.
type SnipParams struct {
	Title        string
	Slug         string
	Description  string
	Kind         string
	Body         string
	Tags         []string
	Placeholders []snips.Placeholder
}

// SnipListFilter mirrors PinListFilter.
type SnipListFilter struct {
	Q        string
	Archived bool
}

func scanSnip(row interface{ Scan(...any) error }, p *Snip) error {
	var tags, phJSON string
	var starred int
	var archived sql.NullString
	if err := row.Scan(&p.ID, &p.Slug, &p.Title, &p.Description, &p.Kind, &p.Body, &phJSON, &tags, &starred, &archived, &p.CreatedAt, &p.UpdatedAt); err != nil {
		return err
	}
	p.Tags = decodePackages(tags)
	p.Placeholders = decodePlaceholders(phJSON)
	p.Starred = starred == 1
	if archived.Valid {
		v := archived.String
		p.ArchivedAt = &v
	}
	return nil
}

func scanSnipSummary(row interface{ Scan(...any) error }, p *SnipSummary) error {
	var tags string
	var starred int
	var archived sql.NullString
	if err := row.Scan(&p.ID, &p.Slug, &p.Title, &p.Description, &p.Kind, &tags, &starred, &archived, &p.CreatedAt, &p.UpdatedAt); err != nil {
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

func decodePlaceholders(raw string) []snips.Placeholder {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "null" {
		return []snips.Placeholder{}
	}
	var out []snips.Placeholder
	if json.Unmarshal([]byte(raw), &out) != nil || out == nil {
		return []snips.Placeholder{}
	}
	return out
}

func encodePlaceholders(src []snips.Placeholder) string {
	if src == nil {
		src = []snips.Placeholder{}
	}
	b, err := json.Marshal(src)
	if err != nil {
		return "[]"
	}
	return string(b)
}

func slugBusy(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "unique")
}

func normalizeSnip(p SnipParams) (SnipParams, error) {
	p.Title = strings.TrimSpace(p.Title)
	p.Description = strings.TrimSpace(p.Description)
	p.Kind = strings.TrimSpace(strings.ToLower(p.Kind))
	if p.Kind == "" {
		p.Kind = "prompt"
	}
	if p.Kind != "prompt" && p.Kind != "shell" {
		return p, invalid("kind must be prompt or shell")
	}
	if p.Title == "" {
		return p, invalid("title is required")
	}
	if !utf8.ValidString(p.Title) || !utf8.ValidString(p.Description) || !utf8.ValidString(p.Body) {
		return p, invalid("text is not valid UTF-8")
	}
	if utf8.RuneCountInString(p.Title) > maxSnipTitle {
		return p, invalid("title is too long (max %d characters)", maxSnipTitle)
	}
	if utf8.RuneCountInString(p.Description) > maxSnipDesc {
		return p, invalid("description is too long (max %d characters)", maxSnipDesc)
	}
	if len(p.Body) > maxSnipBody {
		return p, invalid("body is too long (max %d KB)", maxSnipBodyKB)
	}
	slug := strings.TrimSpace(p.Slug)
	if slug == "" {
		slug = snips.Slug(p.Title)
	} else {
		slug = snips.Slug(slug)
	}
	if slug == "" {
		return p, invalid("slug is required")
	}
	if len(slug) > snips.MaxSlug {
		return p, invalid("slug is too long (max %d characters)", snips.MaxSlug)
	}
	p.Slug = slug

	parsed, err := snips.Parse(p.Body)
	if err != nil {
		return p, invalid("%s", err.Error())
	}
	merged, err := mergeSnipPlaceholders(parsed, p.Placeholders)
	if err != nil {
		return p, err
	}
	p.Placeholders = merged

	title, tags, _, err := normalizePin(p.Title, p.Tags, "")
	if err != nil {
		return p, err
	}
	p.Title = title
	p.Tags = tags
	return p, nil
}

func mergeSnipPlaceholders(parsed, client []snips.Placeholder) ([]snips.Placeholder, error) {
	byName := map[string]snips.Placeholder{}
	for _, c := range client {
		byName[c.Name] = c
	}
	out := make([]snips.Placeholder, len(parsed))
	for i, ph := range parsed {
		if c, ok := byName[ph.Name]; ok {
			if err := validateSnipEnum(c.Enum); err != nil {
				return nil, err
			}
			ph.Enum = c.Enum
			if c.Optional {
				ph.Optional = true
				ph.Default = c.Default
			}
		}
		out[i] = ph
	}
	return out, nil
}

func validateSnipEnum(enum []string) error {
	if len(enum) > snips.MaxEnum {
		return invalid("an enum can have at most %d values", snips.MaxEnum)
	}
	seen := map[string]bool{}
	for _, v := range enum {
		v = strings.TrimSpace(v)
		if v == "" || seen[v] {
			continue
		}
		if utf8.RuneCountInString(v) > snips.MaxEnumLen {
			return invalid("enum value is too long (max %d characters)", snips.MaxEnumLen)
		}
		seen[v] = true
	}
	return nil
}

func (s *Store) CreateSnip(in SnipParams) (Snip, error) {
	in, err := normalizeSnip(in)
	if err != nil {
		return Snip{}, err
	}
	now := nowUTC()
	p := Snip{
		ID: newID(in.Title, "snip"), Slug: in.Slug, Title: in.Title, Description: in.Description,
		Kind: in.Kind, Body: in.Body, Placeholders: in.Placeholders, Tags: in.Tags,
		CreatedAt: now, UpdatedAt: now,
	}
	_, err = s.db.Exec(`INSERT INTO snips (id, slug, title, description, kind, body, placeholders, tags, starred, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, 0, ?, ?)`,
		p.ID, p.Slug, p.Title, p.Description, p.Kind, p.Body, encodePlaceholders(p.Placeholders), encodePackages(p.Tags), p.CreatedAt, p.UpdatedAt)
	if slugBusy(err) {
		return Snip{}, invalid("slug is already used")
	}
	if err != nil {
		return Snip{}, fmt.Errorf("store: create snip: %w", err)
	}
	s.note("snip.created", nil, nil, p.Summary())
	return p, nil
}

func (s *Store) UpdateSnip(id string, in SnipParams, ifUpdatedAt string) (Snip, error) {
	in, err := normalizeSnip(in)
	if err != nil {
		return Snip{}, err
	}
	now := nowUTC()
	res, err := s.db.Exec(`UPDATE snips SET slug = ?, title = ?, description = ?, kind = ?, body = ?, placeholders = ?, tags = ?, updated_at = ?
		WHERE id = ? AND (? = '' OR updated_at = ?)`,
		in.Slug, in.Title, in.Description, in.Kind, in.Body, encodePlaceholders(in.Placeholders), encodePackages(in.Tags), now,
		id, ifUpdatedAt, ifUpdatedAt)
	if slugBusy(err) {
		return Snip{}, invalid("slug is already used")
	}
	if err != nil {
		return Snip{}, fmt.Errorf("store: update snip: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		var exists int
		if err := s.db.QueryRow(`SELECT COUNT(1) FROM snips WHERE id = ?`, id).Scan(&exists); err == nil && exists > 0 {
			return Snip{}, ErrConflict
		}
		return Snip{}, ErrNotFound
	}
	p, err := s.GetSnip(id)
	if err != nil {
		return Snip{}, err
	}
	s.note("snip.updated", nil, nil, p.Summary())
	return p, nil
}

func (s *Store) GetSnip(id string) (Snip, error) {
	var p Snip
	row := s.db.QueryRow(`SELECT id, slug, title, description, kind, body, placeholders, tags, starred, archived_at, created_at, updated_at FROM snips WHERE id = ?`, id)
	if err := scanSnip(row, &p); err != nil {
		if err == sql.ErrNoRows {
			return Snip{}, ErrNotFound
		}
		return Snip{}, fmt.Errorf("store: get snip: %w", err)
	}
	return p, nil
}

func (s *Store) DeleteSnip(id string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	res, err := tx.Exec(`DELETE FROM snips WHERE id = ?`, id)
	if err != nil {
		s.rollback(tx)
		return fmt.Errorf("store: delete snip: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		s.rollback(tx)
		return ErrNotFound
	}
	if err := s.AppendEventTx(tx, "snip.deleted", nil, nil, idData(id)); err != nil {
		s.rollback(tx)
		return err
	}
	return s.commit(tx)
}

func (s *Store) ListSnips(f SnipListFilter) ([]SnipSummary, error) {
	q := `SELECT id, slug, title, description, kind, tags, starred, archived_at, created_at, updated_at FROM snips WHERE 1=1`
	args := []any{}
	if words := strings.Fields(strings.ToLower(f.Q)); len(words) > 0 {
		for _, w := range words {
			like := "%" + escapeLike(w) + "%"
			q += ` AND (lower(title) LIKE ? ESCAPE '\' OR lower(slug) LIKE ? ESCAPE '\' OR lower(tags) LIKE ? ESCAPE '\' OR lower(body) LIKE ? ESCAPE '\' OR lower(description) LIKE ? ESCAPE '\')`
			args = append(args, like, like, like, like, like)
		}
	} else if f.Archived {
		q += ` AND archived_at IS NOT NULL`
	} else {
		q += ` AND archived_at IS NULL`
	}
	q += ` ORDER BY starred DESC, updated_at DESC`
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("store: list snips: %w", err)
	}
	defer rows.Close()
	out := []SnipSummary{}
	for rows.Next() {
		var p SnipSummary
		if err := scanSnipSummary(rows, &p); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) CountArchivedSnips() (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(1) FROM snips WHERE archived_at IS NOT NULL`).Scan(&n)
	return n, err
}

func (s *Store) SetSnipStarred(id string, starred bool) (Snip, error) {
	res, err := s.db.Exec(`UPDATE snips SET starred = ? WHERE id = ?`, boolInt(starred), id)
	if err != nil {
		return Snip{}, fmt.Errorf("store: star snip: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return Snip{}, ErrNotFound
	}
	p, err := s.GetSnip(id)
	if err != nil {
		return Snip{}, err
	}
	s.note("snip.updated", nil, nil, p.Summary())
	return p, nil
}

func (s *Store) SetSnipArchived(id string, archived bool) (Snip, error) {
	var at any
	if archived {
		at = nowUTC()
	}
	res, err := s.db.Exec(`UPDATE snips SET archived_at = ? WHERE id = ?`, at, id)
	if err != nil {
		return Snip{}, fmt.Errorf("store: archive snip: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return Snip{}, ErrNotFound
	}
	p, err := s.GetSnip(id)
	if err != nil {
		return Snip{}, err
	}
	s.note("snip.updated", nil, nil, p.Summary())
	return p, nil
}

// ListSnipPicker is live snippets for the composer and run sheets. Prompt
// only by default (the composer never offers a Command, D2); includeShell
// adds kind shell for the terminal menu's Run command…
func (s *Store) ListSnipPicker(includeShell bool) ([]SnipPicker, error) {
	where := `archived_at IS NULL AND kind = 'prompt'`
	if includeShell {
		where = `archived_at IS NULL`
	}
	rows, err := s.db.Query(`SELECT id, slug, title, description, kind, placeholders FROM snips WHERE ` + where + ` ORDER BY starred DESC, updated_at DESC LIMIT 200`)
	if err != nil {
		return nil, fmt.Errorf("store: snip picker: %w", err)
	}
	defer rows.Close()
	out := []SnipPicker{}
	for rows.Next() {
		var p SnipPicker
		var phJSON string
		if err := rows.Scan(&p.ID, &p.Slug, &p.Title, &p.Hint, &p.Kind, &phJSON); err != nil {
			return nil, err
		}
		if strings.TrimSpace(p.Hint) == "" {
			p.Hint = p.Title
		}
		for _, ph := range decodePlaceholders(phJSON) {
			if snips.Reserved[ph.Name] {
				continue
			}
			p.Names = append(p.Names, ph.Name)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// SnipSlugTaken reports whether slug already belongs to another snippet —
// the editor's debounced "is this address free?". An archived row keeps its
// slug (reuse needs a DELETE), so this deliberately does not filter
// archived_at. `except` is the snippet being edited, whose own slug is not a
// clash; it is empty when creating.
func (s *Store) SnipSlugTaken(slug, except string) (bool, error) {
	slug = snips.Slug(slug)
	if slug == "" {
		return false, invalid("slug is required")
	}
	var one int
	err := s.db.QueryRow(`SELECT 1 FROM snips WHERE slug = ? AND id <> ? LIMIT 1`, slug, except).Scan(&one)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("store: snip slug: %w", err)
	}
	return true, nil
}
