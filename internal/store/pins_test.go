package store

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestPinsCRUD(t *testing.T) {
	s := openTest(t)
	p, err := s.CreatePin("  Hello  ", []string{"#Foo", "foo", "Bar", "a b  c"}, "body")
	if err != nil {
		t.Fatal(err)
	}
	if p.Title != "Hello" {
		t.Fatalf("title = %q", p.Title)
	}
	if strings.Join(p.Tags, ",") != "foo,bar,a-b-c" {
		t.Fatalf("tags = %v", p.Tags)
	}

	if _, err := s.CreatePin("  ", nil, ""); !errors.Is(err, ErrInvalid) {
		t.Fatalf("empty title: %v", err)
	}

	got, err := s.GetPin(p.ID)
	if err != nil || got.Body != "body" {
		t.Fatalf("get = %+v %v", got, err)
	}

	upd, err := s.UpdatePin(p.ID, "Bye", []string{"x"}, "next", "")
	if err != nil || upd.Title != "Bye" || upd.Body != "next" {
		t.Fatalf("update = %+v %v", upd, err)
	}

	list, err := s.ListPins()
	if err != nil || len(list) != 1 || list[0].Title != "Bye" {
		t.Fatalf("list = %v %v", list, err)
	}
	if b, _ := json.Marshal(list); strings.Contains(string(b), "\"body\"") {
		t.Fatalf("list carries bodies: %s", b)
	}

	if err := s.DeletePin(p.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetPin(p.ID); err != ErrNotFound {
		t.Fatalf("deleted get = %v", err)
	}
	if _, err := s.UpdatePin(p.ID, "x", nil, "", ""); err != ErrNotFound {
		t.Fatalf("update missing = %v", err)
	}
}

// Limits refuse instead of truncating: a byte-sliced title once left
// invalid UTF-8 in the database (pins v2 review, finding 1).
func TestPinLimitsRefuse(t *testing.T) {
	s := openTest(t)
	cases := []struct {
		name  string
		title string
		tags  []string
		body  string
		want  string
	}{
		{"title 201 runes", strings.Repeat("é", 201), nil, "", "title is too long"},
		{"body over 100 KB", "t", nil, strings.Repeat("b", maxPinBody) + "ç", "note is too long"},
		{"tag over 40 runes", "t", []string{strings.Repeat("x", 41)}, "", "is too long"},
		{"17 tags", "t", strings.Split(strings.Repeat("a,", 17), ",")[:0], "", ""},
		{"invalid utf8 title", "ab\xffc", nil, "", "not valid UTF-8"},
	}
	tags17 := make([]string, 17)
	for i := range tags17 {
		tags17[i] = "t" + strings.Repeat("x", i)
	}
	cases[3].tags = tags17
	cases[3].want = "at most 16 tags"
	for _, c := range cases {
		_, err := s.CreatePin(c.title, c.tags, c.body)
		if !errors.Is(err, ErrInvalid) || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: err = %v", c.name, err)
		}
	}
	// Exactly at the limit is fine, and multibyte survives intact.
	p, err := s.CreatePin(strings.Repeat("é", 200), []string{strings.Repeat("ü", 40)}, strings.Repeat("ç", maxPinBody/2))
	if err != nil {
		t.Fatal(err)
	}
	if !utf8.ValidString(p.Title) || !utf8.ValidString(p.Body) || utf8.RuneCountInString(p.Title) != 200 {
		t.Fatalf("limit pin mangled: %d runes", utf8.RuneCountInString(p.Title))
	}
	if list, _ := s.ListPins(); len(list) != 1 {
		t.Fatalf("refused pins were stored: %d", len(list))
	}
}

// Two writers: the one holding a stale updatedAt gets ErrConflict and
// overwrites nothing.
func TestPinUpdateConflict(t *testing.T) {
	s := openTest(t)
	p, _ := s.CreatePin("Draft", nil, "v1")
	first, err := s.UpdatePin(p.ID, "Draft", nil, "v2", p.UpdatedAt)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.UpdatePin(p.ID, "Draft", nil, "v3", p.UpdatedAt); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale write: %v", err)
	}
	got, _ := s.GetPin(p.ID)
	if got.Body != "v2" || got.UpdatedAt != first.UpdatedAt {
		t.Fatalf("stale write landed: %+v", got)
	}
	if _, err := s.UpdatePin(p.ID, "Draft", nil, "v3", first.UpdatedAt); err != nil {
		t.Fatalf("fresh write: %v", err)
	}
	if _, err := s.UpdatePin(p.ID, "Draft", nil, "v4", ""); err != nil {
		t.Fatalf("no precondition: %v", err)
	}
}

// Every pin.created / pin.updated is the summary — one shape, no body —
// whether it came from the editor or a file route (finding 3).
func TestPinEventsCarrySummary(t *testing.T) {
	s := openTest(t)
	var evs []Event
	s.OnEvent = func(ev Event) { evs = append(evs, ev) }
	p, err := s.CreatePin("Note", []string{"a"}, strings.Repeat("x", 50_000))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.UpdatePin(p.ID, "Note", nil, strings.Repeat("y", 50_000), ""); err != nil {
		t.Fatal(err)
	}
	f, err := s.AddPinFile(p.ID, "a.png", "image/png", 10)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.AddPinSketch(p.ID, "S", "annotate", f.ID, 10); err != nil {
		t.Fatal(err)
	}
	if err := s.DeletePinFile(p.ID, f.ID); err != nil {
		t.Fatal(err)
	}
	if len(evs) != 5 {
		t.Fatalf("events = %d", len(evs))
	}
	for i, ev := range evs {
		var sum PinSummary
		if err := json.Unmarshal(ev.Data, &sum); err != nil || sum.ID != p.ID {
			t.Fatalf("event %d %s: %s", i, ev.Type, ev.Data)
		}
		if len(ev.Data) > 2_000 || strings.Contains(string(ev.Data), "\"body\"") {
			t.Fatalf("event %d %s carries a body: %d bytes", i, ev.Type, len(ev.Data))
		}
	}
	if evs[0].Type != "pin.created" || evs[1].Type != "pin.updated" {
		t.Fatalf("types = %s %s", evs[0].Type, evs[1].Type)
	}
	var last PinSummary
	_ = json.Unmarshal(evs[4].Data, &last)
	if last.FileCount != 1 {
		t.Fatalf("fileCount after delete = %d", last.FileCount)
	}
}

func TestPinFiles(t *testing.T) {
	s := openTest(t)
	p, err := s.CreatePin("Shot", nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.AddPinFile(p.ID, "x.exe", "application/octet-stream", 10); !errors.Is(err, ErrInvalid) {
		t.Fatalf("exe: %v", err)
	}
	if _, err := s.AddPinFile(p.ID, "big.png", "image/png", MaxPinImageSize+1); !errors.Is(err, ErrInvalid) {
		t.Fatalf("big image: %v", err)
	}
	f, err := s.AddPinFile(p.ID, "shot.png", "image/png", 120)
	if err != nil || f.Kind != "image" || f.UpdatedAt != f.CreatedAt {
		t.Fatalf("image = %+v %v", f, err)
	}
	got, err := s.GetPin(p.ID)
	if err != nil || got.FileCount != 1 || len(got.Files) != 1 {
		t.Fatalf("get files = %+v %v", got, err)
	}
	list, _ := s.ListPins()
	if list[0].FileCount != 1 {
		t.Fatalf("list count = %d", list[0].FileCount)
	}
	// Annotation must point at an image of this pin.
	if _, err := s.AddPinSketch(p.ID, "Board", "annotate", "nope", 200); !errors.Is(err, ErrInvalid) {
		t.Fatalf("bad base: %v", err)
	}
	sk, err := s.AddPinSketch(p.ID, "Board", "annotate", f.ID, 200)
	if err != nil || sk.Kind != "sketch" || sk.BaseFileID != f.ID {
		t.Fatalf("sketch = %+v %v", sk, err)
	}
	// Editing moves updatedAt (the preview URL's version).
	upd, err := s.UpdatePinSketch(p.ID, sk.ID, "Board 2", 300)
	if err != nil || upd.Name != "Board 2" || upd.Size != 300 || upd.UpdatedAt == sk.UpdatedAt {
		t.Fatalf("update sketch = %+v %v (was %s)", upd, err, sk.UpdatedAt)
	}
	if _, err := s.UpdatePinSketch(p.ID, f.ID, "x", 1); !errors.Is(err, ErrInvalid) {
		t.Fatalf("update image as sketch: %v", err)
	}
	if err := s.DeletePinFile(p.ID, f.ID); err != nil {
		t.Fatal(err)
	}
	ids, _ := s.PinIDs()
	if len(ids) != 1 || ids[0] != p.ID {
		t.Fatalf("ids = %v", ids)
	}
}
