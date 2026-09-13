package store

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/snips"
)

func TestSnipsCRUD(t *testing.T) {
	s := openTest(t)
	p, err := s.CreateSnip(SnipParams{Title: "  Review PR  ", Body: "Look at {{pr}}", Tags: []string{"#Git"}})
	if err != nil {
		t.Fatal(err)
	}
	if p.Slug != "review-pr" || p.Kind != "prompt" || p.Title != "Review PR" {
		t.Fatalf("created = %+v", p)
	}
	if len(p.Placeholders) != 1 || p.Placeholders[0].Name != "pr" {
		t.Fatalf("placeholders = %+v", p.Placeholders)
	}

	got, err := s.GetSnip(p.ID)
	if err != nil || got.Body != "Look at {{pr}}" {
		t.Fatalf("get = %+v %v", got, err)
	}

	upd, err := s.UpdateSnip(p.ID, SnipParams{Title: "Review PR", Slug: "review-pr", Body: "Look at {{pr=1}}"}, "")
	if err != nil || !upd.Placeholders[0].Optional || upd.Placeholders[0].Default != "1" {
		t.Fatalf("update = %+v %v", upd, err)
	}

	list, err := s.ListSnips(SnipListFilter{})
	if err != nil || len(list) != 1 || list[0].Title != "Review PR" {
		t.Fatalf("list = %v %v", list, err)
	}
	if b, _ := json.Marshal(list); strings.Contains(string(b), `"body"`) {
		t.Fatalf("list carries bodies: %s", b)
	}

	if _, err := s.CreateSnip(SnipParams{Title: "Other", Slug: "review-pr", Body: "x"}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("dup slug: %v", err)
	}

	if err := s.DeleteSnip(p.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetSnip(p.ID); err != ErrNotFound {
		t.Fatalf("deleted get = %v", err)
	}
}

func TestSnipSlugFromTitleAndArchiveReuse(t *testing.T) {
	s := openTest(t)
	a, err := s.CreateSnip(SnipParams{Title: "Review PR", Body: "a"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.SetSnipArchived(a.ID, true); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateSnip(SnipParams{Title: "Review PR", Body: "b"}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("archived slug reused: %v", err)
	}
	if err := s.DeleteSnip(a.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateSnip(SnipParams{Title: "Review PR", Body: "c"}); err != nil {
		t.Fatalf("after delete: %v", err)
	}
}

func TestSnipLimitsAndKind(t *testing.T) {
	s := openTest(t)
	if _, err := s.CreateSnip(SnipParams{Title: "", Body: "x"}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("empty title: %v", err)
	}
	if _, err := s.CreateSnip(SnipParams{Title: "t", Kind: "other", Body: "x"}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("kind: %v", err)
	}
	if _, err := s.CreateSnip(SnipParams{Title: "t", Body: "{{"}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("bad body: %v", err)
	}
	if _, err := s.CreateSnip(SnipParams{Title: strings.Repeat("é", 201), Body: "x"}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("long title: %v", err)
	}
	p, err := s.CreateSnip(SnipParams{Title: "Shell", Kind: "shell", Body: "echo {{msg}}"})
	if err != nil || p.Kind != "shell" {
		t.Fatalf("shell kind: %+v %v", p, err)
	}
}

func TestSnipStarArchiveAndPicker(t *testing.T) {
	s := openTest(t)
	a, _ := s.CreateSnip(SnipParams{Title: "Alpha", Body: "a {{x}}"})
	b, _ := s.CreateSnip(SnipParams{Title: "Beta", Kind: "shell", Body: "echo 1"})
	if _, err := s.SetSnipStarred(a.ID, true); err != nil {
		t.Fatal(err)
	}
	got, _ := s.GetSnip(a.ID)
	if !got.Starred || got.UpdatedAt != a.UpdatedAt {
		t.Fatalf("star bumped updated_at: %+v vs %+v", got, a)
	}
	if _, err := s.SetSnipArchived(a.ID, true); err != nil {
		t.Fatal(err)
	}
	live, _ := s.ListSnips(SnipListFilter{})
	if len(live) != 1 || live[0].ID != b.ID {
		t.Fatalf("live = %+v", live)
	}
	arch, _ := s.ListSnips(SnipListFilter{Archived: true})
	if len(arch) != 1 || arch[0].ID != a.ID {
		t.Fatalf("archived = %+v", arch)
	}
	n, _ := s.CountArchivedSnips()
	if n != 1 {
		t.Fatalf("count archived = %d", n)
	}
	pick, err := s.ListSnipPicker()
	if err != nil || len(pick) != 0 {
		t.Fatalf("picker should omit archived prompt and all shell, got %+v %v", pick, err)
	}
	_, _ = s.SetSnipArchived(a.ID, false)
	pick, _ = s.ListSnipPicker()
	if len(pick) != 1 || pick[0].Slug != "alpha" || pick[0].Kind != "prompt" {
		t.Fatalf("picker = %+v", pick)
	}
}

func TestSnipUpdateConflict(t *testing.T) {
	s := openTest(t)
	p, _ := s.CreateSnip(SnipParams{Title: "Draft", Body: "v1"})
	first, err := s.UpdateSnip(p.ID, SnipParams{Title: "Draft", Body: "v2"}, p.UpdatedAt)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.UpdateSnip(p.ID, SnipParams{Title: "Draft", Body: "v3"}, p.UpdatedAt); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale write: %v", err)
	}
	got, _ := s.GetSnip(p.ID)
	if got.Body != "v2" {
		t.Fatalf("stale write landed: %+v", got)
	}
	if _, err := s.UpdateSnip(p.ID, SnipParams{Title: "Draft", Body: "v3"}, first.UpdatedAt); err != nil {
		t.Fatal(err)
	}
}

func TestSnipEventsCarrySummary(t *testing.T) {
	s := openTest(t)
	var evs []Event
	s.OnEvent = func(ev Event) { evs = append(evs, ev) }
	p, err := s.CreateSnip(SnipParams{Title: "Note", Body: strings.Repeat("x", 1000) + " {{z}}"})
	if err != nil {
		t.Fatal(err)
	}
	if len(evs) != 1 || evs[0].Type != "snip.created" {
		t.Fatalf("events = %+v", evs)
	}
	if strings.Contains(string(evs[0].Data), `"body"`) {
		t.Fatalf("created event carries body: %s", evs[0].Data)
	}
	_, _ = s.SetSnipStarred(p.ID, true)
	if evs[len(evs)-1].Type != "snip.updated" {
		t.Fatalf("star event = %s", evs[len(evs)-1].Type)
	}
}

func TestSnipPlaceholderMergeEnum(t *testing.T) {
	s := openTest(t)
	p, err := s.CreateSnip(SnipParams{
		Title: "Env",
		Body:  "deploy {{env}}",
		Placeholders: []snips.Placeholder{{
			Name: "env", Enum: []string{"dev", "prod"},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Placeholders) != 1 || strings.Join(p.Placeholders[0].Enum, ",") != "dev,prod" {
		t.Fatalf("enum = %+v", p.Placeholders)
	}
}
