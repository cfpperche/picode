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

func recordMatrixEvents(s *Store) *[]Event {
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

func matrixIDs(list []Matrix) []string {
	out := make([]string, 0, len(list))
	for _, m := range list {
		out = append(out, m.ID)
	}
	return out
}

func panelsByID(list []MatrixPanel) map[string]MatrixPanel {
	out := map[string]MatrixPanel{}
	for _, p := range list {
		out[p.ID] = p
	}
	return out
}

func addPanel(t *testing.T, s *Store, id, kind, ref string, x, y, w, h int) MatrixPanel {
	t.Helper()
	added, err := s.AddMatrixPanel(id, kind, ref, x, y, w, h)
	if err != nil {
		t.Fatalf("add %s %s: %v", kind, ref, err)
	}
	return added.Panel
}

// Decision table rows "create ok", "update ok", "delete ok", "any unknown
// id": the summary is what the list, the event and the patch answer carry.
func TestMatrixCRUD(t *testing.T) {
	s := openTest(t)
	evs := recordMatrixEvents(s)
	m, err := s.CreateMatrix("  Ops  ")
	if err != nil {
		t.Fatal(err)
	}
	if m.Name != "Ops" || m.Compact != MatrixCompactVertical || m.PanelCount != 0 || m.CreatedAt != m.UpdatedAt || !strings.HasPrefix(m.ID, "ops-") {
		t.Fatalf("created = %+v", m)
	}
	if at, err := time.Parse(time.RFC3339Nano, m.UpdatedAt); err != nil || at.Location() != time.UTC || !strings.HasSuffix(m.UpdatedAt, "Z") {
		t.Fatalf("updatedAt %q is not RFC3339 UTC: %v", m.UpdatedAt, err)
	}
	var created Matrix
	decodeEvent(t, lastEvent(t, evs, "matrix.created"), &created)
	if created != m {
		t.Fatalf("matrix.created = %+v, want %+v", created, m)
	}

	// The list is by name (case-folded), then creation.
	b, _ := s.CreateMatrix("beta")
	a, _ := s.CreateMatrix("Alpha")
	list, err := s.ListMatrices()
	if err != nil || !reflect.DeepEqual(matrixIDs(list), []string{a.ID, b.ID, m.ID}) {
		t.Fatalf("list = %v %v", matrixIDs(list), err)
	}

	// Get carries the panels — an empty list, never null.
	d, err := s.GetMatrix(m.ID)
	if err != nil || d.Matrix != m || d.Panels == nil || len(d.Panels) != 0 {
		t.Fatalf("get = %+v %v", d, err)
	}
	if js, _ := json.Marshal(d); !strings.Contains(string(js), `"panels":[]`) || !strings.Contains(string(js), `"panelCount":0`) {
		t.Fatalf("detail json = %s", js)
	}

	// Rename and compact in one patch; updatedAt moves, createdAt stays.
	name, compact := "Ops board", MatrixCompactNone
	upd, err := s.UpdateMatrix(m.ID, MatrixPatch{Name: &name, Compact: &compact}, m.UpdatedAt)
	if err != nil || upd.Name != name || upd.Compact != compact || upd.UpdatedAt == m.UpdatedAt || upd.CreatedAt != m.CreatedAt {
		t.Fatalf("update = %+v %v", upd, err)
	}
	var updated Matrix
	decodeEvent(t, lastEvent(t, evs, "matrix.updated"), &updated)
	if updated != upd {
		t.Fatalf("matrix.updated = %+v, want %+v", updated, upd)
	}

	if err := s.DeleteMatrix(m.ID); err != nil {
		t.Fatal(err)
	}
	var gone map[string]string
	decodeEvent(t, lastEvent(t, evs, "matrix.deleted"), &gone)
	if gone["id"] != m.ID || len(gone) != 1 {
		t.Fatalf("matrix.deleted = %v", gone)
	}
	if _, err := s.GetMatrix(m.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("deleted get = %v", err)
	}
	if list, _ := s.ListMatrices(); len(list) != 2 {
		t.Fatalf("list after delete = %v", matrixIDs(list))
	}
}

func TestMatrixUnknownIDIsNotFound(t *testing.T) {
	s := openTest(t)
	name := "x"
	checks := map[string]error{}
	_, checks["get"] = s.GetMatrix("matrix-nope")
	_, checks["update"] = s.UpdateMatrix("matrix-nope", MatrixPatch{Name: &name}, "")
	_, checks["layout"] = s.PatchMatrixLayout("matrix-nope", []PanelPlacement{{ID: "panel-x", X: 0, Y: 0, W: 4, H: 8}}, "")
	_, checks["add"] = s.AddMatrixPanel("matrix-nope", "terminal", "t", 0, 0, 4, 8)
	checks["remove"] = s.RemoveMatrixPanel("matrix-nope", "panel-x")
	checks["delete"] = s.DeleteMatrix("matrix-nope")
	for op, err := range checks {
		if !errors.Is(err, ErrNotFound) || !strings.Contains(err.Error(), "matrix not found") {
			t.Errorf("%s on an unknown matrix: %v", op, err)
		}
	}
}

// Rows "create: 64 matrices exist" and "create: name empty / > 80 runes /
// only spaces": refused with the limit named, never truncated.
func TestMatrixLimitsRefuse(t *testing.T) {
	s := openTest(t)
	cases := []struct{ name, in, want string }{
		{"empty", "", "name is required"},
		{"only spaces", "   ", "name is required"},
		{"81 runes", strings.Repeat("é", 81), "name is too long (max 80 characters)"},
		{"invalid utf8", "ab\xffc", "not valid UTF-8"},
	}
	for _, c := range cases {
		if _, err := s.CreateMatrix(c.in); !errors.Is(err, ErrInvalid) || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: err = %v", c.name, err)
		}
	}
	// Exactly 80 multibyte runes survive intact.
	m, err := s.CreateMatrix(strings.Repeat("é", 80))
	if err != nil || utf8.RuneCountInString(m.Name) != 80 || !utf8.ValidString(m.Name) {
		t.Fatalf("80-rune name: %+v %v", m, err)
	}
	if list, _ := s.ListMatrices(); len(list) != 1 {
		t.Fatalf("refused matrices were stored: %d", len(list))
	}
	for i := 1; i < MaxMatrices; i++ {
		if _, err := s.CreateMatrix(fmt.Sprintf("m%02d", i)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.CreateMatrix("one more"); !errors.Is(err, ErrInvalid) || !strings.Contains(err.Error(), "limit: 64 matrices") {
		t.Fatalf("65th matrix: %v", err)
	}
	if list, _ := s.ListMatrices(); len(list) != MaxMatrices {
		t.Fatalf("matrices = %d", len(list))
	}
}

// Row "add panel: w < 4, h < 8, x < 0, y < 0, x + w > 12, kind not
// agent/terminal, ref empty": 400 naming the rule; nothing written.
func TestMatrixPanelRulesRefuse(t *testing.T) {
	s := openTest(t)
	m, _ := s.CreateMatrix("Ops")
	evs := recordMatrixEvents(s)
	cases := []struct {
		name, kind, ref string
		x, y, w, h      int
		want            string
	}{
		{"w < 4", "terminal", "t", 0, 0, 3, 8, "w must be at least 4 columns"},
		{"h < 8", "terminal", "t", 0, 0, 4, 7, "h must be at least 8 rows"},
		{"x < 0", "terminal", "t", -1, 0, 4, 8, "x must be 0 or more"},
		{"y < 0", "terminal", "t", 0, -1, 4, 8, "y must be 0 or more"},
		{"x + w > 12", "terminal", "t", 9, 0, 4, 8, "x + w must be at most 12 columns"},
		{"kind", "pin", "t", 0, 0, 4, 8, "kind must be agent or terminal"},
		{"ref empty", "terminal", "  ", 0, 0, 4, 8, "ref is required"},
	}
	for _, c := range cases {
		_, err := s.AddMatrixPanel(m.ID, c.kind, c.ref, c.x, c.y, c.w, c.h)
		if !errors.Is(err, ErrInvalid) || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: err = %v", c.name, err)
		}
	}
	if d, _ := s.GetMatrix(m.ID); len(d.Panels) != 0 || d.UpdatedAt != m.UpdatedAt || len(*evs) != 0 {
		t.Fatalf("a refused panel left a trace: %+v, %d events", d, len(*evs))
	}
	// The edges are fine: the last columns, a full-width tall panel.
	addPanel(t, s, m.ID, "terminal", "edge", 8, 0, 4, 8)
	addPanel(t, s, m.ID, "agent", "wide", 0, 8, 12, 40)
}

// Rows "add panel ok" and "remove panel unknown / ok": the panel id is a
// slot, updatedAt bumps, the events carry the client's view.
func TestMatrixPanelsAddRemove(t *testing.T) {
	s := openTest(t)
	m, _ := s.CreateMatrix("Ops")
	evs := recordMatrixEvents(s)
	added, err := s.AddMatrixPanel(m.ID, "terminal", " term-1 ", 0, 0, 4, 8)
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
	var ev MatrixPanelAdded
	decodeEvent(t, lastEvent(t, evs, "matrix.panel.added"), &ev)
	if ev != added {
		t.Fatalf("matrix.panel.added = %+v, want %+v", ev, added)
	}
	d, _ := s.GetMatrix(m.ID)
	if d.PanelCount != 1 || len(d.Panels) != 1 || d.Panels[0] != p || d.UpdatedAt != added.UpdatedAt {
		t.Fatalf("get after add = %+v", d)
	}
	if list, _ := s.ListMatrices(); list[0].PanelCount != 1 || list[0].UpdatedAt != added.UpdatedAt {
		t.Fatalf("list after add = %+v", list)
	}

	// Unknown panel: not found, nothing announced, updatedAt stays.
	if err := s.RemoveMatrixPanel(m.ID, "panel-nope"); !errors.Is(err, ErrNotFound) || !strings.Contains(err.Error(), "panel not found") {
		t.Fatalf("remove unknown = %v", err)
	}
	if n := len(*evs); n != 1 {
		t.Fatalf("events after a refused remove = %d", n)
	}
	if err := s.RemoveMatrixPanel(m.ID, p.ID); err != nil {
		t.Fatal(err)
	}
	var rem MatrixPanelRemoved
	decodeEvent(t, lastEvent(t, evs, "matrix.panel.removed"), &rem)
	after, _ := s.GetMatrix(m.ID)
	if rem.ID != m.ID || rem.PanelID != p.ID || rem.UpdatedAt != after.UpdatedAt || rem.UpdatedAt == d.UpdatedAt {
		t.Fatalf("matrix.panel.removed = %+v (matrix now %s)", rem, after.UpdatedAt)
	}
	if after.PanelCount != 0 || len(after.Panels) != 0 {
		t.Fatalf("panel still there: %+v", after)
	}
	// Removing it twice is a miss, not a no-op.
	if err := s.RemoveMatrixPanel(m.ID, p.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("second remove = %v", err)
	}
}

// Rows "add panel: same (kind, ref) already on this matrix" and "add
// panel: 500 panels exist".
func TestMatrixPanelDuplicateAndLimit(t *testing.T) {
	s := openTest(t)
	m, _ := s.CreateMatrix("Ops")
	other, _ := s.CreateMatrix("Other")
	addPanel(t, s, m.ID, "terminal", "t1", 0, 0, 4, 8)
	_, err := s.AddMatrixPanel(m.ID, "terminal", "t1", 4, 0, 4, 8)
	if !errors.Is(err, ErrConflict) || !strings.Contains(err.Error(), "already on this matrix") {
		t.Fatalf("duplicate binding: %v", err)
	}
	// The same ref as an agent, and the same binding on another matrix,
	// are different panels.
	addPanel(t, s, m.ID, "agent", "t1", 4, 0, 4, 8)
	addPanel(t, s, other.ID, "terminal", "t1", 0, 0, 4, 8)
	if d, _ := s.GetMatrix(m.ID); d.PanelCount != 2 {
		t.Fatalf("panels after duplicate = %d", d.PanelCount)
	}
	for i := 2; i < MaxMatrixPanels; i++ {
		addPanel(t, s, m.ID, "terminal", fmt.Sprintf("t%d", i), 0, i*8, 4, 8)
	}
	_, err = s.AddMatrixPanel(m.ID, "terminal", "one more", 0, 0, 4, 8)
	if !errors.Is(err, ErrInvalid) || !strings.Contains(err.Error(), "limit: 500 panels per matrix") {
		t.Fatalf("501st panel: %v", err)
	}
	if d, _ := s.GetMatrix(m.ID); d.PanelCount != MaxMatrixPanels || len(d.Panels) != MaxMatrixPanels {
		t.Fatalf("panels = %d / %d", d.PanelCount, len(d.Panels))
	}
	// The other matrix has its own cap.
	addPanel(t, s, other.ID, "terminal", "t2", 4, 0, 4, 8)
}

// Rows "layout patch: stale / foreign id or out of bounds / ok subset".
func TestMatrixLayoutPatch(t *testing.T) {
	s := openTest(t)
	m, _ := s.CreateMatrix("Ops")
	a := addPanel(t, s, m.ID, "terminal", "a", 0, 0, 4, 8)
	b := addPanel(t, s, m.ID, "terminal", "b", 4, 0, 4, 8)
	c := addPanel(t, s, m.ID, "agent", "c", 8, 0, 4, 8)
	before, _ := s.GetMatrix(m.ID)
	evs := recordMatrixEvents(s)

	unchanged := func(want MatrixDetail) {
		t.Helper()
		got, err := s.GetMatrix(m.ID)
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Fatalf("a refused patch wrote something:\n got %+v\nwant %+v (%v)", got, want, err)
		}
	}

	// Only the listed panels move; the event carries exactly them.
	subset := []PanelPlacement{{ID: a.ID, X: 0, Y: 8, W: 8, H: 16}, {ID: c.ID, X: 8, Y: 8, W: 4, H: 8}}
	lay, err := s.PatchMatrixLayout(m.ID, subset, before.UpdatedAt)
	if err != nil {
		t.Fatal(err)
	}
	after, _ := s.GetMatrix(m.ID)
	if lay.ID != m.ID || lay.UpdatedAt != after.UpdatedAt || lay.UpdatedAt == before.UpdatedAt || !reflect.DeepEqual(lay.Panels, subset) {
		t.Fatalf("layout = %+v (matrix %s → %s)", lay, before.UpdatedAt, after.UpdatedAt)
	}
	var ev MatrixLayout
	decodeEvent(t, lastEvent(t, evs, "matrix.layout"), &ev)
	if !reflect.DeepEqual(ev, lay) {
		t.Fatalf("matrix.layout = %+v, want %+v", ev, lay)
	}
	got := panelsByID(after.Panels)
	if got[a.ID].Y != 8 || got[a.ID].W != 8 || got[a.ID].H != 16 || got[c.ID].Y != 8 || got[b.ID] != b {
		t.Fatalf("panels after patch = %+v", after.Panels)
	}

	// Stale precondition: conflict, nothing written.
	_, err = s.PatchMatrixLayout(m.ID, []PanelPlacement{{ID: b.ID, X: 0, Y: 40, W: 4, H: 8}}, before.UpdatedAt)
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("stale patch: %v", err)
	}
	unchanged(after)

	// An id that is not on this matrix refuses the whole batch — the valid
	// row before it is not applied either.
	_, err = s.PatchMatrixLayout(m.ID, []PanelPlacement{{ID: b.ID, X: 0, Y: 40, W: 4, H: 8}, {ID: "panel-nope", X: 0, Y: 0, W: 4, H: 8}}, after.UpdatedAt)
	if !errors.Is(err, ErrInvalid) || !strings.Contains(err.Error(), "panel panel-nope is not on this matrix") {
		t.Fatalf("foreign id: %v", err)
	}
	unchanged(after)
	other, _ := s.CreateMatrix("Other")
	o := addPanel(t, s, other.ID, "terminal", "o", 0, 0, 4, 8)
	if _, err = s.PatchMatrixLayout(m.ID, []PanelPlacement{{ID: o.ID, X: 0, Y: 0, W: 4, H: 8}}, ""); !errors.Is(err, ErrInvalid) {
		t.Fatalf("another matrix's panel: %v", err)
	}
	unchanged(after)

	// Out of bounds names the rule and the panel; nothing written.
	_, err = s.PatchMatrixLayout(m.ID, []PanelPlacement{{ID: b.ID, X: 0, Y: 40, W: 4, H: 8}, {ID: a.ID, X: 10, Y: 0, W: 4, H: 8}}, "")
	if !errors.Is(err, ErrInvalid) || !strings.Contains(err.Error(), "panel "+a.ID+": x + w must be at most 12 columns") {
		t.Fatalf("out of bounds: %v", err)
	}
	unchanged(after)

	// An empty subset and a repeated id are client bugs, named.
	if _, err := s.PatchMatrixLayout(m.ID, nil, ""); !errors.Is(err, ErrInvalid) || !strings.Contains(err.Error(), "panels is required") {
		t.Fatalf("empty subset: %v", err)
	}
	if _, err := s.PatchMatrixLayout(m.ID, []PanelPlacement{{ID: b.ID, X: 0, Y: 8, W: 4, H: 8}, {ID: b.ID, X: 4, Y: 8, W: 4, H: 8}}, ""); !errors.Is(err, ErrInvalid) || !strings.Contains(err.Error(), "listed twice") {
		t.Fatalf("repeated id: %v", err)
	}
	if n := countType(evs, "matrix.layout"); n != 1 {
		t.Fatalf("refused patches announced: %d matrix.layout events", n)
	}
	// No precondition writes unconditionally.
	if _, err := s.PatchMatrixLayout(m.ID, []PanelPlacement{{ID: b.ID, X: 0, Y: 40, W: 4, H: 8}}, ""); err != nil {
		t.Fatal(err)
	}
}

// Rows "update: rename ok / compact ∈ {vertical, none}" and "update:
// compact other / name over limit / stale ifUpdatedAt".
func TestMatrixUpdateRulesAndConflict(t *testing.T) {
	s := openTest(t)
	m, _ := s.CreateMatrix("Ops")
	evs := recordMatrixEvents(s)
	long, bad, empty, none := strings.Repeat("x", 81), "diagonal", "  ", MatrixCompactNone
	cases := []struct {
		name string
		p    MatrixPatch
		want string
	}{
		{"compact other", MatrixPatch{Compact: &bad}, "compact must be vertical or none"},
		{"name over limit", MatrixPatch{Name: &long}, "name is too long (max 80 characters)"},
		{"name empty", MatrixPatch{Name: &empty}, "name is required"},
		{"empty patch", MatrixPatch{}, "nothing to update"},
	}
	for _, c := range cases {
		if _, err := s.UpdateMatrix(m.ID, c.p, ""); !errors.Is(err, ErrInvalid) || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: err = %v", c.name, err)
		}
	}
	// Two writers: the stale one gets ErrConflict and overwrites nothing.
	first, err := s.UpdateMatrix(m.ID, MatrixPatch{Compact: &none}, m.UpdatedAt)
	if err != nil {
		t.Fatal(err)
	}
	name := "Late"
	if _, err := s.UpdateMatrix(m.ID, MatrixPatch{Name: &name}, m.UpdatedAt); !errors.Is(err, ErrConflict) || !strings.Contains(err.Error(), "changed elsewhere") {
		t.Fatalf("stale write: %v", err)
	}
	got, _ := s.GetMatrix(m.ID)
	if got.Name != "Ops" || got.Compact != none || got.UpdatedAt != first.UpdatedAt {
		t.Fatalf("stale write landed: %+v", got)
	}
	if _, err := s.UpdateMatrix(m.ID, MatrixPatch{Name: &name}, first.UpdatedAt); err != nil {
		t.Fatalf("fresh write: %v", err)
	}
	vertical := MatrixCompactVertical
	if _, err := s.UpdateMatrix(m.ID, MatrixPatch{Compact: &vertical}, ""); err != nil {
		t.Fatalf("no precondition: %v", err)
	}
	if n := countType(evs, "matrix.updated"); n != 3 {
		t.Fatalf("matrix.updated events = %d, want 3", n)
	}
}

// Row "delete matrix ok": the panels go with it (cascade proven by a
// query), other matrices keep theirs, one event.
func TestMatrixDeleteCascadesPanels(t *testing.T) {
	s := openTest(t)
	m, _ := s.CreateMatrix("Ops")
	keep, _ := s.CreateMatrix("Keep")
	for i := 0; i < 3; i++ {
		addPanel(t, s, m.ID, "terminal", fmt.Sprintf("t%d", i), 0, i*8, 4, 8)
	}
	addPanel(t, s, keep.ID, "terminal", "t0", 0, 0, 4, 8)
	evs := recordMatrixEvents(s)
	if err := s.DeleteMatrix(m.ID); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(1) FROM matrix_panels WHERE matrix_id = ?`, m.ID).Scan(&n); err != nil || n != 0 {
		t.Fatalf("panels not cascaded: %d rows remain (%v)", n, err)
	}
	if err := s.db.QueryRow(`SELECT COUNT(1) FROM matrix_panels`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("another matrix lost panels: %d rows (%v)", n, err)
	}
	if len(*evs) != 1 || (*evs)[0].Type != "matrix.deleted" {
		t.Fatalf("events = %+v (the cascade announces nothing per panel)", *evs)
	}
	if err := s.DeleteMatrix(m.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("second delete = %v", err)
	}
}

// Row "agent or terminal deleted elsewhere": the store stays ignorant of
// the binding — the panel row survives and no matrix event is announced.
func TestMatrixPanelSurvivesItsTargetDeletion(t *testing.T) {
	s := openTest(t)
	a, err := s.AddAgent(FreeWorkspaceID, "a", "")
	if err != nil {
		t.Fatal(err)
	}
	tm, err := s.CreateTerminalIn("", "t", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	m, _ := s.CreateMatrix("Ops")
	addPanel(t, s, m.ID, "agent", a.ID, 0, 0, 4, 8)
	addPanel(t, s, m.ID, "terminal", tm.ID, 4, 0, 4, 8)
	before, _ := s.GetMatrix(m.ID)
	evs := recordMatrixEvents(s)
	if err := s.DeleteAgent(a.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteTerminal(tm.ID); err != nil {
		t.Fatal(err)
	}
	after, err := s.GetMatrix(m.ID)
	if err != nil || !reflect.DeepEqual(after, before) {
		t.Fatalf("matrix changed with its targets:\n got %+v\nwant %+v (%v)", after, before, err)
	}
	for _, ev := range *evs {
		if strings.HasPrefix(ev.Type, "matrix.") {
			t.Fatalf("matrix event on target deletion: %s", ev.Type)
		}
	}
}
