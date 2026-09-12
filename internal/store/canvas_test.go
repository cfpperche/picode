package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

func recordCanvasEvents(s *Store) *[]Event {
	evs := &[]Event{}
	s.OnEvent = func(ev Event) { *evs = append(*evs, ev) }
	return evs
}

func lastEvent(t *testing.T, evs *[]Event, typ string) Event {
	t.Helper()
	if len(*evs) == 0 {
		t.Fatalf("no events, want %s", typ)
	}
	ev := (*evs)[len(*evs)-1]
	if ev.Type != typ {
		t.Fatalf("last event = %s, want %s", ev.Type, typ)
	}
	return ev
}

func decodeEvent(t *testing.T, ev Event, into any) {
	t.Helper()
	if err := json.Unmarshal(ev.Data, into); err != nil {
		t.Fatalf("%s data %s: %v", ev.Type, ev.Data, err)
	}
}

func countType(evs *[]Event, typ string) int {
	n := 0
	for _, ev := range *evs {
		if ev.Type == typ {
			n++
		}
	}
	return n
}

func canvasIDs(list []Canvas) []string {
	out := make([]string, 0, len(list))
	for _, m := range list {
		out = append(out, m.ID)
	}
	return out
}

func panelsByID(list []CanvasPanel) map[string]CanvasPanel {
	out := map[string]CanvasPanel{}
	for _, p := range list {
		out[p.ID] = p
	}
	return out
}

func addPanel(t *testing.T, s *Store, id, kind, ref string, x, y, w, h int) CanvasPanel {
	t.Helper()
	added, err := s.AddCanvasPanel(id, kind, ref, x, y, w, h)
	if err != nil {
		t.Fatalf("add %s %s: %v", kind, ref, err)
	}
	return added.Panel
}

// Decision table rows "create ok", "update ok", "delete ok", "any unknown
// id": the summary is what the list, the event and the patch answer carry.
func TestCanvasCRUD(t *testing.T) {
	s := openTest(t)
	evs := recordCanvasEvents(s)
	m, err := s.CreateCanvas("  Ops  ")
	if err != nil {
		t.Fatal(err)
	}
	if m.Name != "Ops" || m.Compact != CanvasCompactVertical || m.PanelCount != 0 || m.CreatedAt != m.UpdatedAt || !strings.HasPrefix(m.ID, "ops-") {
		t.Fatalf("created = %+v", m)
	}
	if at, err := time.Parse(time.RFC3339Nano, m.UpdatedAt); err != nil || at.Location() != time.UTC || !strings.HasSuffix(m.UpdatedAt, "Z") {
		t.Fatalf("updatedAt %q is not RFC3339 UTC: %v", m.UpdatedAt, err)
	}
	var created Canvas
	decodeEvent(t, lastEvent(t, evs, "canvas.created"), &created)
	if created != m {
		t.Fatalf("canvas.created = %+v, want %+v", created, m)
	}

	// The list is by name (case-folded), then creation.
	b, _ := s.CreateCanvas("beta")
	a, _ := s.CreateCanvas("Alpha")
	list, err := s.ListCanvases()
	if err != nil || !reflect.DeepEqual(canvasIDs(list), []string{a.ID, b.ID, m.ID}) {
		t.Fatalf("list = %v %v", canvasIDs(list), err)
	}

	// Get carries the panels — an empty list, never null.
	d, err := s.GetCanvas(m.ID)
	if err != nil || d.Canvas != m || d.Panels == nil || len(d.Panels) != 0 {
		t.Fatalf("get = %+v %v", d, err)
	}
	if js, _ := json.Marshal(d); !strings.Contains(string(js), `"panels":[]`) || !strings.Contains(string(js), `"panelCount":0`) {
		t.Fatalf("detail json = %s", js)
	}

	// Rename and compact in one patch; updatedAt moves, createdAt stays.
	name, compact := "Ops board", CanvasCompactNone
	upd, err := s.UpdateCanvas(m.ID, CanvasPatch{Name: &name, Compact: &compact}, m.UpdatedAt)
	if err != nil || upd.Name != name || upd.Compact != compact || upd.UpdatedAt == m.UpdatedAt || upd.CreatedAt != m.CreatedAt {
		t.Fatalf("update = %+v %v", upd, err)
	}
	var updated Canvas
	decodeEvent(t, lastEvent(t, evs, "canvas.updated"), &updated)
	if updated != upd {
		t.Fatalf("canvas.updated = %+v, want %+v", updated, upd)
	}

	if err := s.DeleteCanvas(m.ID); err != nil {
		t.Fatal(err)
	}
	var gone map[string]string
	decodeEvent(t, lastEvent(t, evs, "canvas.deleted"), &gone)
	if gone["id"] != m.ID || len(gone) != 1 {
		t.Fatalf("canvas.deleted = %v", gone)
	}
	if _, err := s.GetCanvas(m.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("deleted get = %v", err)
	}
	if list, _ := s.ListCanvases(); len(list) != 2 {
		t.Fatalf("list after delete = %v", canvasIDs(list))
	}
}

func TestCanvasUnknownIDIsNotFound(t *testing.T) {
	s := openTest(t)
	name := "x"
	checks := map[string]error{}
	_, checks["get"] = s.GetCanvas("canvas-nope")
	_, checks["update"] = s.UpdateCanvas("canvas-nope", CanvasPatch{Name: &name}, "")
	_, checks["layout"] = s.PatchCanvasLayout("canvas-nope", []PanelPlacement{{ID: "panel-x", X: 0, Y: 0, W: 32, H: 28}}, "")
	_, checks["add"] = s.AddCanvasPanel("canvas-nope", "terminal", "t", 0, 0, 32, 28)
	checks["remove"] = s.RemoveCanvasPanel("canvas-nope", "panel-x")
	checks["delete"] = s.DeleteCanvas("canvas-nope")
	for op, err := range checks {
		if !errors.Is(err, ErrNotFound) || !strings.Contains(err.Error(), "canvas not found") {
			t.Errorf("%s on an unknown canvas: %v", op, err)
		}
	}
}

// Rows "create: 64 canvases exist" and "create: name empty / > 80 runes /
// only spaces": refused with the limit named, never truncated.
func TestCanvasLimitsRefuse(t *testing.T) {
	s := openTest(t)
	cases := []struct{ name, in, want string }{
		{"empty", "", "name is required"},
		{"only spaces", "   ", "name is required"},
		{"81 runes", strings.Repeat("é", 81), "name is too long (max 80 characters)"},
		{"invalid utf8", "ab\xffc", "not valid UTF-8"},
	}
	for _, c := range cases {
		if _, err := s.CreateCanvas(c.in); !errors.Is(err, ErrInvalid) || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: err = %v", c.name, err)
		}
	}
	// Exactly 80 multibyte runes survive intact.
	m, err := s.CreateCanvas(strings.Repeat("é", 80))
	if err != nil || utf8.RuneCountInString(m.Name) != 80 || !utf8.ValidString(m.Name) {
		t.Fatalf("80-rune name: %+v %v", m, err)
	}
	if list, _ := s.ListCanvases(); len(list) != 1 {
		t.Fatalf("refused canvases were stored: %d", len(list))
	}
	for i := 1; i < MaxCanvases; i++ {
		if _, err := s.CreateCanvas(fmt.Sprintf("m%02d", i)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.CreateCanvas("one more"); !errors.Is(err, ErrInvalid) || !strings.Contains(err.Error(), "limit: 64 canvases") {
		t.Fatalf("65th canvas: %v", err)
	}
	if list, _ := s.ListCanvases(); len(list) != MaxCanvases {
		t.Fatalf("canvases = %d", len(list))
	}
}

// Row "add panel: kind not agent/terminal/note/file/diff, ref empty, ref
// of the wrong shape": 400 naming the rule; nothing written. The rectangle
// rules are TestCanvasPanelPlaneRules'.
func TestCanvasPanelRulesRefuse(t *testing.T) {
	s := openTest(t)
	m, _ := s.CreateCanvas("Ops")
	evs := recordCanvasEvents(s)
	cases := []struct {
		name, kind, ref string
		x, y, w, h      int
		want            string
	}{
		{"kind", "pin", "t", 0, 0, 32, 28, "kind must be agent, terminal, note, file or diff"},
		{"ref empty", "terminal", "  ", 0, 0, 32, 28, "ref is required"},
		{"note ref empty", "note", " ", 0, 0, 32, 28, "ref is required"},
		{"file ref without a path", "file", "t:term-1", 0, 0, 32, 28, "ref must be <owner>:<id>:<path> with owner t, a or w"},
		{"file ref with an empty path", "file", "t:term-1:", 0, 0, 32, 28, "ref must be <owner>:<id>:<path> with owner t, a or w"},
		{"file ref with an empty id", "file", "t::main.go", 0, 0, 32, 28, "ref must be <owner>:<id>:<path> with owner t, a or w"},
		{"file ref with an unknown owner letter", "file", "x:term-1:main.go", 0, 0, 32, 28, "ref must be <owner>:<id>:<path> with owner t, a or w"},
		{"file ref that is only a path", "file", "src/main.go", 0, 0, 32, 28, "ref must be <owner>:<id>:<path> with owner t, a or w"},
		{"diff ref without a path", "diff", "a:agent-1", 0, 0, 32, 28, "ref must be <owner>:<id>:<path> with owner t, a or w"},
		{"diff ref with an unknown owner letter", "diff", "q:agent-1:main.go", 0, 0, 32, 28, "ref must be <owner>:<id>:<path> with owner t, a or w"},
	}
	for _, c := range cases {
		_, err := s.AddCanvasPanel(m.ID, c.kind, c.ref, c.x, c.y, c.w, c.h)
		if !errors.Is(err, ErrInvalid) || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: err = %v", c.name, err)
		}
	}
	if d, _ := s.GetCanvas(m.ID); len(d.Panels) != 0 || d.UpdatedAt != m.UpdatedAt || len(*evs) != 0 {
		t.Fatalf("a refused panel left a trace: %+v, %d events", d, len(*evs))
	}
	// A legal binding lands: the plane takes a panel anywhere, including a
	// wide one far down.
	addPanel(t, s, m.ID, "terminal", "edge", 80, 0, 32, 28)
	addPanel(t, s, m.ID, "agent", "wide", 0, 40, 96, 120)
}

// C3 (docs/plans/matrix-canvas.md §4.2): a file and a diff panel bind
// "<owner>:<id>:<path>" — the three owner letters the file APIs take
// (ADR-0030) — and the store checks the shape and nothing else: whether that
// owner or that file exists is the UI's question.
func TestCanvasPanelKindsFileAndDiff(t *testing.T) {
	s := openTest(t)
	m, _ := s.CreateCanvas("Ops")
	refs := []string{
		"t:term-1:src/main.go",
		"a:agent-1:README.md",
		"w:ws-1:docs/architecture/canvas.md",
		"t:term-1:weird/name:with:colons.txt",
		"t:term-gone:never/written.txt",
	}
	for i, ref := range refs {
		added, err := s.AddCanvasPanel(m.ID, "file", ref, 0, i*32, 32, 28)
		if err != nil {
			t.Fatalf("file %q: %v", ref, err)
		}
		if added.Panel.Ref != ref {
			t.Fatalf("file %q stored as %q", ref, added.Panel.Ref)
		}
	}
	if _, err := s.AddCanvasPanel(m.ID, "file", "t:term-1:src/main.go", 0, 320, 32, 28); !errors.Is(err, ErrConflict) {
		t.Fatalf("the same file twice: %v", err)
	}
	// The same path as a file and as a diff is two panels: the unique index
	// is per (kind, ref), and reading a file and reading its changes are
	// two things to have open at once.
	if _, err := s.AddCanvasPanel(m.ID, "diff", "t:term-1:src/main.go", 0, 320, 32, 28); err != nil {
		t.Fatalf("the same path as a diff: %v", err)
	}
	if _, err := s.AddCanvasPanel(m.ID, "diff", "t:term-1:src/main.go", 0, 360, 32, 28); !errors.Is(err, ErrConflict) {
		t.Fatalf("the same diff twice: %v", err)
	}
	if d, _ := s.GetCanvas(m.ID); len(d.Panels) != len(refs)+1 {
		t.Fatalf("panels = %d", len(d.Panels))
	}
}

// C3 (docs/plans/matrix-canvas.md §4.2): a note is a binding like any
// other — its ref is a pin id. The store never asks whether that pin exists
// (ADR-0108), so a deleted pin's id is accepted and stored verbatim and the
// UI is what renders the panel as gone; the same pin twice on one canvas is
// the unique index's conflict.
func TestCanvasPanelKindNote(t *testing.T) {
	s := openTest(t)
	m, _ := s.CreateCanvas("Ops")
	for i, ref := range []string{"pin-abc123", "a pin that was deleted an hour ago"} {
		added, err := s.AddCanvasPanel(m.ID, "note", ref, 0, i*32, 32, 28)
		if err != nil {
			t.Fatalf("note %q: %v", ref, err)
		}
		if added.Panel.Kind != "note" || added.Panel.Ref != ref {
			t.Fatalf("note %q stored as %q %q", ref, added.Panel.Kind, added.Panel.Ref)
		}
	}
	if _, err := s.AddCanvasPanel(m.ID, "note", "pin-abc123", 0, 64, 32, 28); !errors.Is(err, ErrConflict) {
		t.Fatalf("the same pin twice: %v", err)
	}
	// A note and a terminal may share a ref: the unique index is per (kind, ref).
	if _, err := s.AddCanvasPanel(m.ID, "terminal", "pin-abc123", 0, 64, 32, 28); err != nil {
		t.Fatalf("same ref, other kind: %v", err)
	}
	if d, _ := s.GetCanvas(m.ID); len(d.Panels) != 3 {
		t.Fatalf("panels = %d", len(d.Panels))
	}
}

// Rows "add panel ok" and "remove panel unknown / ok": the panel id is a
// slot, updatedAt bumps, the events carry the client's view.
func TestCanvasPanelsAddRemove(t *testing.T) {
	s := openTest(t)
	m, _ := s.CreateCanvas("Ops")
	evs := recordCanvasEvents(s)
	added, err := s.AddCanvasPanel(m.ID, "terminal", " term-1 ", 0, 0, 32, 28)
	if err != nil {
		t.Fatal(err)
	}
	p := added.Panel
	if added.ID != m.ID || added.UpdatedAt == m.UpdatedAt || p.Kind != "terminal" || p.Ref != "term-1" || p.CreatedAt != added.UpdatedAt {
		t.Fatalf("added = %+v", added)
	}
	if !strings.HasPrefix(p.ID, "panel-") || len(p.ID) != len("panel-")+12 || strings.Contains(p.ID, "term") {
		t.Fatalf("panel id %q is not a random slot", p.ID)
	}
	var ev CanvasPanelAdded
	decodeEvent(t, lastEvent(t, evs, "canvas.panel.added"), &ev)
	if ev != added {
		t.Fatalf("canvas.panel.added = %+v, want %+v", ev, added)
	}
	d, _ := s.GetCanvas(m.ID)
	if d.PanelCount != 1 || len(d.Panels) != 1 || d.Panels[0] != p || d.UpdatedAt != added.UpdatedAt {
		t.Fatalf("get after add = %+v", d)
	}
	if list, _ := s.ListCanvases(); list[0].PanelCount != 1 || list[0].UpdatedAt != added.UpdatedAt {
		t.Fatalf("list after add = %+v", list)
	}

	// Unknown panel: not found, nothing announced, updatedAt stays.
	if err := s.RemoveCanvasPanel(m.ID, "panel-nope"); !errors.Is(err, ErrNotFound) || !strings.Contains(err.Error(), "panel not found") {
		t.Fatalf("remove unknown = %v", err)
	}
	if n := len(*evs); n != 1 {
		t.Fatalf("events after a refused remove = %d", n)
	}
	if err := s.RemoveCanvasPanel(m.ID, p.ID); err != nil {
		t.Fatal(err)
	}
	var rem CanvasPanelRemoved
	decodeEvent(t, lastEvent(t, evs, "canvas.panel.removed"), &rem)
	after, _ := s.GetCanvas(m.ID)
	if rem.ID != m.ID || rem.PanelID != p.ID || rem.UpdatedAt != after.UpdatedAt || rem.UpdatedAt == d.UpdatedAt {
		t.Fatalf("canvas.panel.removed = %+v (canvas now %s)", rem, after.UpdatedAt)
	}
	if after.PanelCount != 0 || len(after.Panels) != 0 {
		t.Fatalf("panel still there: %+v", after)
	}
	// Removing it twice is a miss, not a no-op.
	if err := s.RemoveCanvasPanel(m.ID, p.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("second remove = %v", err)
	}
}

// Rows "add panel: same (kind, ref) already on this canvas" and "add
// panel: 500 panels exist".
func TestCanvasPanelDuplicateAndLimit(t *testing.T) {
	s := openTest(t)
	m, _ := s.CreateCanvas("Ops")
	other, _ := s.CreateCanvas("Other")
	addPanel(t, s, m.ID, "terminal", "t1", 0, 0, 32, 28)
	_, err := s.AddCanvasPanel(m.ID, "terminal", "t1", 40, 0, 32, 28)
	if !errors.Is(err, ErrConflict) || !strings.Contains(err.Error(), "already on this canvas") {
		t.Fatalf("duplicate binding: %v", err)
	}
	// The same ref as an agent, and the same binding on another canvas,
	// are different panels.
	addPanel(t, s, m.ID, "agent", "t1", 40, 0, 32, 28)
	addPanel(t, s, other.ID, "terminal", "t1", 0, 0, 32, 28)
	if d, _ := s.GetCanvas(m.ID); d.PanelCount != 2 {
		t.Fatalf("panels after duplicate = %d", d.PanelCount)
	}
	for i := 2; i < MaxCanvasPanels; i++ {
		addPanel(t, s, m.ID, "terminal", fmt.Sprintf("t%d", i), 0, i*32, 32, 28)
	}
	_, err = s.AddCanvasPanel(m.ID, "terminal", "one more", 0, 0, 32, 28)
	if !errors.Is(err, ErrInvalid) || !strings.Contains(err.Error(), "limit: 500 panels per canvas") {
		t.Fatalf("501st panel: %v", err)
	}
	if d, _ := s.GetCanvas(m.ID); d.PanelCount != MaxCanvasPanels || len(d.Panels) != MaxCanvasPanels {
		t.Fatalf("panels = %d / %d", d.PanelCount, len(d.Panels))
	}
	// The other canvas has its own cap.
	addPanel(t, s, other.ID, "terminal", "t2", 40, 0, 32, 28)
}

// Rows "layout patch: stale / foreign id or out of bounds / ok subset".
func TestCanvasLayoutPatch(t *testing.T) {
	s := openTest(t)
	m, _ := s.CreateCanvas("Ops")
	a := addPanel(t, s, m.ID, "terminal", "a", 0, 0, 32, 28)
	b := addPanel(t, s, m.ID, "terminal", "b", 40, 0, 32, 28)
	c := addPanel(t, s, m.ID, "agent", "c", 80, 0, 32, 28)
	before, _ := s.GetCanvas(m.ID)
	evs := recordCanvasEvents(s)

	unchanged := func(want CanvasDetail) {
		t.Helper()
		got, err := s.GetCanvas(m.ID)
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Fatalf("a refused patch wrote something:\n got %+v\nwant %+v (%v)", got, want, err)
		}
	}

	// Only the listed panels move; the event carries exactly them.
	subset := []PanelPlacement{{ID: a.ID, X: 0, Y: 32, W: 64, H: 56}, {ID: c.ID, X: 80, Y: 32, W: 32, H: 28}}
	lay, err := s.PatchCanvasLayout(m.ID, subset, before.UpdatedAt)
	if err != nil {
		t.Fatal(err)
	}
	after, _ := s.GetCanvas(m.ID)
	if lay.ID != m.ID || lay.UpdatedAt != after.UpdatedAt || lay.UpdatedAt == before.UpdatedAt || !reflect.DeepEqual(lay.Panels, subset) {
		t.Fatalf("layout = %+v (canvas %s → %s)", lay, before.UpdatedAt, after.UpdatedAt)
	}
	var ev CanvasLayout
	decodeEvent(t, lastEvent(t, evs, "canvas.layout"), &ev)
	if !reflect.DeepEqual(ev, lay) {
		t.Fatalf("canvas.layout = %+v, want %+v", ev, lay)
	}
	got := panelsByID(after.Panels)
	if got[a.ID].Y != 32 || got[a.ID].W != 64 || got[a.ID].H != 56 || got[c.ID].Y != 32 || got[b.ID] != b {
		t.Fatalf("panels after patch = %+v", after.Panels)
	}

	// Stale precondition: conflict, nothing written.
	_, err = s.PatchCanvasLayout(m.ID, []PanelPlacement{{ID: b.ID, X: 0, Y: 160, W: 32, H: 28}}, before.UpdatedAt)
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("stale patch: %v", err)
	}
	unchanged(after)

	// An id that is not on this canvas refuses the whole batch — the valid
	// row before it is not applied either.
	_, err = s.PatchCanvasLayout(m.ID, []PanelPlacement{{ID: b.ID, X: 0, Y: 160, W: 32, H: 28}, {ID: "panel-nope", X: 0, Y: 0, W: 32, H: 28}}, after.UpdatedAt)
	if !errors.Is(err, ErrInvalid) || !strings.Contains(err.Error(), "panel panel-nope is not on this canvas") {
		t.Fatalf("foreign id: %v", err)
	}
	unchanged(after)
	other, _ := s.CreateCanvas("Other")
	o := addPanel(t, s, other.ID, "terminal", "o", 0, 0, 32, 28)
	if _, err = s.PatchCanvasLayout(m.ID, []PanelPlacement{{ID: o.ID, X: 0, Y: 0, W: 32, H: 28}}, ""); !errors.Is(err, ErrInvalid) {
		t.Fatalf("another canvas's panel: %v", err)
	}
	unchanged(after)

	// Off the plane names the rule and the panel; nothing written.
	_, err = s.PatchCanvasLayout(m.ID, []PanelPlacement{{ID: b.ID, X: 0, Y: 160, W: 32, H: 28}, {ID: a.ID, X: 100001, Y: 0, W: 32, H: 28}}, "")
	if !errors.Is(err, ErrInvalid) || !strings.Contains(err.Error(), "panel "+a.ID+": x must be between -100000 and 100000 canvas units") {
		t.Fatalf("out of bounds: %v", err)
	}
	unchanged(after)

	// An empty subset and a repeated id are client bugs, named.
	if _, err := s.PatchCanvasLayout(m.ID, nil, ""); !errors.Is(err, ErrInvalid) || !strings.Contains(err.Error(), "panels is required") {
		t.Fatalf("empty subset: %v", err)
	}
	if _, err := s.PatchCanvasLayout(m.ID, []PanelPlacement{{ID: b.ID, X: 0, Y: 32, W: 32, H: 28}, {ID: b.ID, X: 40, Y: 32, W: 32, H: 28}}, ""); !errors.Is(err, ErrInvalid) || !strings.Contains(err.Error(), "listed twice") {
		t.Fatalf("repeated id: %v", err)
	}
	if n := countType(evs, "canvas.layout"); n != 1 {
		t.Fatalf("refused patches announced: %d canvas.layout events", n)
	}
	// No precondition writes unconditionally.
	if _, err := s.PatchCanvasLayout(m.ID, []PanelPlacement{{ID: b.ID, X: 0, Y: 160, W: 32, H: 28}}, ""); err != nil {
		t.Fatal(err)
	}
}

// Rows "update: rename ok / compact ∈ {vertical, none}" and "update:
// compact other / name over limit / stale ifUpdatedAt".
func TestCanvasUpdateRulesAndConflict(t *testing.T) {
	s := openTest(t)
	m, _ := s.CreateCanvas("Ops")
	evs := recordCanvasEvents(s)
	long, bad, empty, none := strings.Repeat("x", 81), "diagonal", "  ", CanvasCompactNone
	cases := []struct {
		name string
		p    CanvasPatch
		want string
	}{
		{"compact other", CanvasPatch{Compact: &bad}, "compact must be vertical or none"},
		{"name over limit", CanvasPatch{Name: &long}, "name is too long (max 80 characters)"},
		{"name empty", CanvasPatch{Name: &empty}, "name is required"},
		{"empty patch", CanvasPatch{}, "nothing to update"},
	}
	for _, c := range cases {
		if _, err := s.UpdateCanvas(m.ID, c.p, ""); !errors.Is(err, ErrInvalid) || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: err = %v", c.name, err)
		}
	}
	// Two writers: the stale one gets ErrConflict and overwrites nothing.
	first, err := s.UpdateCanvas(m.ID, CanvasPatch{Compact: &none}, m.UpdatedAt)
	if err != nil {
		t.Fatal(err)
	}
	name := "Late"
	if _, err := s.UpdateCanvas(m.ID, CanvasPatch{Name: &name}, m.UpdatedAt); !errors.Is(err, ErrConflict) || !strings.Contains(err.Error(), "changed elsewhere") {
		t.Fatalf("stale write: %v", err)
	}
	got, _ := s.GetCanvas(m.ID)
	if got.Name != "Ops" || got.Compact != none || got.UpdatedAt != first.UpdatedAt {
		t.Fatalf("stale write landed: %+v", got)
	}
	if _, err := s.UpdateCanvas(m.ID, CanvasPatch{Name: &name}, first.UpdatedAt); err != nil {
		t.Fatalf("fresh write: %v", err)
	}
	vertical := CanvasCompactVertical
	if _, err := s.UpdateCanvas(m.ID, CanvasPatch{Compact: &vertical}, ""); err != nil {
		t.Fatalf("no precondition: %v", err)
	}
	if n := countType(evs, "canvas.updated"); n != 3 {
		t.Fatalf("canvas.updated events = %d, want 3", n)
	}
}

// Row "delete canvas ok": the panels go with it (cascade proven by a
// query), other canvases keep theirs, one event.
func TestCanvasDeleteCascadesPanels(t *testing.T) {
	s := openTest(t)
	m, _ := s.CreateCanvas("Ops")
	keep, _ := s.CreateCanvas("Keep")
	for i := 0; i < 3; i++ {
		addPanel(t, s, m.ID, "terminal", fmt.Sprintf("t%d", i), 0, i*32, 32, 28)
	}
	addPanel(t, s, keep.ID, "terminal", "t0", 0, 0, 32, 28)
	evs := recordCanvasEvents(s)
	if err := s.DeleteCanvas(m.ID); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(1) FROM canvas_panels WHERE canvas_id = ?`, m.ID).Scan(&n); err != nil || n != 0 {
		t.Fatalf("panels not cascaded: %d rows remain (%v)", n, err)
	}
	if err := s.db.QueryRow(`SELECT COUNT(1) FROM canvas_panels`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("another canvas lost panels: %d rows (%v)", n, err)
	}
	if len(*evs) != 1 || (*evs)[0].Type != "canvas.deleted" {
		t.Fatalf("events = %+v (the cascade announces nothing per panel)", *evs)
	}
	if err := s.DeleteCanvas(m.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("second delete = %v", err)
	}
}

// Row "agent or terminal deleted elsewhere": the store stays ignorant of
// the binding — the panel row survives and no canvas event is announced.
func TestCanvasPanelSurvivesItsTargetDeletion(t *testing.T) {
	s := openTest(t)
	a, err := s.AddAgent(FreeWorkspaceID, "a", "")
	if err != nil {
		t.Fatal(err)
	}
	tm, err := s.CreateTerminalIn("", "t", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	m, _ := s.CreateCanvas("Ops")
	addPanel(t, s, m.ID, "agent", a.ID, 0, 0, 32, 28)
	addPanel(t, s, m.ID, "terminal", tm.ID, 40, 0, 32, 28)
	before, _ := s.GetCanvas(m.ID)
	evs := recordCanvasEvents(s)
	if err := s.DeleteAgent(a.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteTerminal(tm.ID); err != nil {
		t.Fatal(err)
	}
	after, err := s.GetCanvas(m.ID)
	if err != nil || !reflect.DeepEqual(after, before) {
		t.Fatalf("canvas changed with its targets:\n got %+v\nwant %+v (%v)", after, before, err)
	}
	for _, ev := range *evs {
		if strings.HasPrefix(ev.Type, "canvas.") {
			t.Fatalf("canvas event on target deletion: %s", ev.Type)
		}
	}
}

// ---- the plane (ADR-0113 rules, the only rules since ADR-0118) ---------

// panelRects reads the canvas's panels as rectangles keyed by id.
func panelRects(t *testing.T, s *Store, id string) map[string]PanelPlacement {
	t.Helper()
	d, err := s.GetCanvas(id)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]PanelPlacement{}
	for _, p := range d.Panels {
		out[p.ID] = PanelPlacement{ID: p.ID, X: p.X, Y: p.Y, W: p.W, H: p.H}
	}
	return out
}

// Rows "add panel: w < 32, h < 28, w or h over 4096, |x| or |y| over
// 100000": 400 naming the bound; negative coordinates are the plane's and
// are accepted. These are the only rectangle rules left (ADR-0118).
func TestCanvasPanelPlaneRules(t *testing.T) {
	s := openTest(t)
	canvas, _ := s.CreateCanvas("Canvas")
	cases := []struct {
		name       string
		x, y, w, h int
		want       string
	}{
		{"w < 32", 0, 0, 31, 42, "w must be at least 32 canvas units"},
		{"h < 28", 0, 0, 32, 27, "h must be at least 28 canvas units"},
		{"w over the bound", 0, 0, 4097, 42, "w must be at most 4096 canvas units"},
		{"h over the bound", 0, 0, 32, 4097, "h must be at most 4096 canvas units"},
		{"x off the plane", 100001, 0, 32, 42, "x must be between -100000 and 100000 canvas units"},
		{"x off the plane, negative", -100001, 0, 32, 42, "x must be between -100000 and 100000 canvas units"},
		{"y off the plane", 0, -100001, 32, 42, "y must be between -100000 and 100000 canvas units"},
	}
	for _, c := range cases {
		_, err := s.AddCanvasPanel(canvas.ID, "terminal", "t-"+c.name, c.x, c.y, c.w, c.h)
		if !errors.Is(err, ErrInvalid) || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: err = %v", c.name, err)
		}
	}
	// Negative coordinates are the plane's, not a mistake.
	neg := addPanel(t, s, canvas.ID, "terminal", "neg", -4000, -2500, 32, 28)
	if neg.X != -4000 || neg.Y != -2500 {
		t.Fatalf("negative placement = %+v", neg)
	}
	edge := addPanel(t, s, canvas.ID, "agent", "edge", MaxCanvasCoord, -MaxCanvasCoord, MaxCanvasPanel, MaxCanvasPanel)
	if edge.X != MaxCanvasCoord || edge.W != MaxCanvasPanel {
		t.Fatalf("edge placement = %+v", edge)
	}

	// Row "layout patch | mixed: one rectangle legal, one not": 400,
	// nothing written — all or nothing.
	before := panelRects(t, s, canvas.ID)
	_, err := s.PatchCanvasLayout(canvas.ID, []PanelPlacement{
		{ID: neg.ID, X: -8, Y: -8, W: 40, H: 40},
		{ID: edge.ID, X: 0, Y: 0, W: 32, H: 27},
	}, "")
	if !errors.Is(err, ErrInvalid) || !strings.Contains(err.Error(), "panel "+edge.ID+": h must be at least 28 canvas units") {
		t.Fatalf("mixed batch = %v", err)
	}
	if got := panelRects(t, s, canvas.ID); !reflect.DeepEqual(got, before) {
		t.Fatalf("a refused batch wrote something:\n got %+v\nwant %+v", got, before)
	}
	lay, err := s.PatchCanvasLayout(canvas.ID, []PanelPlacement{{ID: neg.ID, X: -8, Y: -8, W: 40, H: 40}}, "")
	if err != nil || len(lay.Panels) != 1 {
		t.Fatalf("canvas layout patch = %+v %v", lay, err)
	}
	if got := panelRects(t, s, canvas.ID)[neg.ID]; got.X != -8 || got.Y != -8 || got.W != 40 || got.H != 40 {
		t.Fatalf("canvas panel did not move: %+v", got)
	}
}
