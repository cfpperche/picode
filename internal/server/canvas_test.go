package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/cfpperche/picode/internal/rpc"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

// canvasServer is cleanupServer with the store in hand, so a test can read
// the rows a route appended to the change log — the proof behind the feed.
func canvasServer(t *testing.T) (*httptest.Server, *store.Store) {
	t.Helper()
	root := t.TempDir()
	home := filepath.Join(root, "home")
	dataDir := filepath.Join(root, "data")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_DATA_HOME", "")
	st, err := store.Open(filepath.Join(dataDir, "picode.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	ts := httptest.NewServer(New("127.0.0.1:0", Deps{
		Store:    st,
		Tmux:     tmux.New(),
		Runtime:  rpc.NewRuntime("cat", st, nil),
		AgentCmd: "cat",
		DataDir:  dataDir,
	}).Handler)
	t.Cleanup(ts.Close)
	return ts, st
}

func canvasReq(t *testing.T, ts *httptest.Server, method, path string, body any) (int, map[string]any) {
	t.Helper()
	return pinReqJSON(t, ts, method, path, body, nil)
}

func newCanvas(t *testing.T, ts *httptest.Server, name string) map[string]any {
	t.Helper()
	code, m := canvasReq(t, ts, http.MethodPost, "/api/canvases", map[string]any{"name": name})
	if code != http.StatusCreated {
		t.Fatalf("create canvas = %d %v", code, m)
	}
	return m
}

func addPanelReq(t *testing.T, ts *httptest.Server, id, kind, ref string, x, y, w, h int) (int, map[string]any) {
	t.Helper()
	return canvasReq(t, ts, http.MethodPost, "/api/canvases/"+id+"/panels", map[string]any{"kind": kind, "ref": ref, "x": x, "y": y, "w": w, "h": h})
}

func mustPanel(t *testing.T, ts *httptest.Server, id, kind, ref string, x, y, w, h int) (string, string) {
	t.Helper()
	code, out := addPanelReq(t, ts, id, kind, ref, x, y, w, h)
	if code != http.StatusCreated {
		t.Fatalf("add panel %s %s = %d %v", kind, ref, code, out)
	}
	return out["panel"].(map[string]any)["id"].(string), out["updatedAt"].(string)
}

func canvasDetail(t *testing.T, ts *httptest.Server, id string) map[string]any {
	t.Helper()
	var d map[string]any
	if code := getJSON(t, ts, "/api/canvases/"+id, &d); code != http.StatusOK {
		t.Fatalf("GET canvas = %d %v", code, d)
	}
	return d
}

func canvasEvents(t *testing.T, st *store.Store) []store.Event {
	t.Helper()
	evs, err := st.ListEventsSince(0, 100_000)
	if err != nil {
		t.Fatal(err)
	}
	out := []store.Event{}
	for _, ev := range evs {
		if strings.HasPrefix(ev.Type, "canvas.") {
			out = append(out, ev)
		}
	}
	return out
}

func eventTypes(evs []store.Event) []string {
	out := make([]string, 0, len(evs))
	for _, ev := range evs {
		out = append(out, ev.Type)
	}
	return out
}

func errorOf(out map[string]any) string {
	s, _ := out["error"].(string)
	return s
}

// Every route once, the happy path: statuses, shapes and the six events in
// order (rows "create ok", "add panel ok", "layout ok", "update ok",
// "remove panel ok", "delete canvas ok").
func TestCanvasRoutes(t *testing.T) {
	ts, st := canvasServer(t)
	code, out := canvasReq(t, ts, http.MethodGet, "/api/canvases", nil)
	if list, _ := out["canvases"].([]any); code != http.StatusOK || list == nil || len(list) != 0 {
		t.Fatalf("empty list = %d %v", code, out)
	}
	m := newCanvas(t, ts, "Ops")
	id := m["id"].(string)
	if m["name"] != "Ops" || m["compact"] != "vertical" || m["panelCount"] != float64(0) || m["updatedAt"] == "" || m["createdAt"] != m["updatedAt"] {
		t.Fatalf("created = %v", m)
	}
	if _, has := m["panels"]; has {
		t.Fatalf("summary carries panels: %v", m)
	}

	code, added := addPanelReq(t, ts, id, "terminal", "term-1", 0, 0, 32, 28)
	if code != http.StatusCreated || added["id"] != id || added["updatedAt"] == m["updatedAt"] {
		t.Fatalf("add panel = %d %v", code, added)
	}
	panel := added["panel"].(map[string]any)
	pid := panel["id"].(string)
	if panel["kind"] != "terminal" || panel["ref"] != "term-1" || panel["x"] != float64(0) || panel["w"] != float64(32) || panel["h"] != float64(28) || panel["createdAt"] == "" {
		t.Fatalf("panel = %v", panel)
	}

	d := canvasDetail(t, ts, id)
	if panels, _ := d["panels"].([]any); len(panels) != 1 || d["panelCount"] != float64(1) || d["updatedAt"] != added["updatedAt"] || d["name"] != "Ops" {
		t.Fatalf("detail = %v", d)
	}

	code, lay := canvasReq(t, ts, http.MethodPatch, "/api/canvases/"+id+"/layout", map[string]any{
		"ifUpdatedAt": added["updatedAt"],
		"panels":      []map[string]any{{"id": pid, "x": 40, "y": 32, "w": 64, "h": 56}},
	})
	if moved, _ := lay["panels"].([]any); code != http.StatusOK || lay["id"] != id || lay["updatedAt"] == added["updatedAt"] || len(moved) != 1 {
		t.Fatalf("layout = %d %v", code, lay)
	}

	code, upd := canvasReq(t, ts, http.MethodPatch, "/api/canvases/"+id, map[string]any{"name": "Ops board", "compact": "none", "ifUpdatedAt": lay["updatedAt"]})
	if code != http.StatusOK || upd["name"] != "Ops board" || upd["compact"] != "none" || upd["panelCount"] != float64(1) || upd["updatedAt"] == lay["updatedAt"] {
		t.Fatalf("update = %d %v", code, upd)
	}

	var list struct{ Canvases []map[string]any }
	getJSON(t, ts, "/api/canvases", &list)
	if len(list.Canvases) != 1 || list.Canvases[0]["name"] != "Ops board" || list.Canvases[0]["panelCount"] != float64(1) || list.Canvases[0]["updatedAt"] != upd["updatedAt"] {
		t.Fatalf("list = %v", list.Canvases)
	}

	if code, out := canvasReq(t, ts, http.MethodDelete, "/api/canvases/"+id+"/panels/"+pid, nil); code != http.StatusNoContent {
		t.Fatalf("remove panel = %d %v", code, out)
	}
	d = canvasDetail(t, ts, id)
	if panels, _ := d["panels"].([]any); len(panels) != 0 || d["panelCount"] != float64(0) || d["updatedAt"] == upd["updatedAt"] {
		t.Fatalf("detail after remove = %v", d)
	}

	if code, out := canvasReq(t, ts, http.MethodDelete, "/api/canvases/"+id, nil); code != http.StatusNoContent {
		t.Fatalf("delete canvas = %d %v", code, out)
	}
	if code := getJSON(t, ts, "/api/canvases/"+id, nil); code != http.StatusNotFound {
		t.Fatalf("deleted canvas GET = %d", code)
	}

	want := []string{"canvas.created", "canvas.panel.added", "canvas.layout", "canvas.updated", "canvas.panel.removed", "canvas.deleted"}
	if got := eventTypes(canvasEvents(t, st)); !reflect.DeepEqual(got, want) {
		t.Fatalf("events = %v, want %v", got, want)
	}
}

// Rows "create: 64 canvases exist" and "create: name empty / > 80 runes /
// only spaces" over HTTP: 400 naming the limit, never truncated.
func TestCanvasCreateStatuses(t *testing.T) {
	ts, _ := canvasServer(t)
	cases := []struct {
		name string
		body any
		want string
	}{
		{"empty", map[string]any{"name": ""}, "name is required"},
		{"only spaces", map[string]any{"name": "   "}, "name is required"},
		{"81 runes", map[string]any{"name": strings.Repeat("é", 81)}, "name is too long (max 80 characters)"},
	}
	for _, c := range cases {
		if code, out := canvasReq(t, ts, http.MethodPost, "/api/canvases", c.body); code != http.StatusBadRequest || !strings.Contains(errorOf(out), c.want) {
			t.Errorf("%s = %d %v", c.name, code, out)
		}
	}
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/canvases", strings.NewReader("{not json"))
	req.Header.Set("Content-Type", "application/json")
	if res := do(t, ts.Client(), req); res.StatusCode != http.StatusBadRequest {
		t.Fatalf("malformed body = %d", res.StatusCode)
	}
	m := newCanvas(t, ts, strings.Repeat("é", 80))
	if name := m["name"].(string); utf8.RuneCountInString(name) != 80 || !utf8.ValidString(name) {
		t.Fatalf("80-rune name mangled: %q", name)
	}
	for i := 1; i < store.MaxCanvases; i++ {
		newCanvas(t, ts, fmt.Sprintf("m%02d", i))
	}
	if code, out := canvasReq(t, ts, http.MethodPost, "/api/canvases", map[string]any{"name": "one more"}); code != http.StatusBadRequest || !strings.Contains(errorOf(out), "limit: 64 canvases") {
		t.Fatalf("65th canvas = %d %v", code, out)
	}
	var list struct{ Canvases []map[string]any }
	getJSON(t, ts, "/api/canvases", &list)
	if len(list.Canvases) != store.MaxCanvases {
		t.Fatalf("canvases = %d", len(list.Canvases))
	}
}

// Rows "add panel: same (kind, ref)" → 409, "add panel: 500 panels exist"
// → 400, and the seven placement rules → 400 naming the rule.
func TestCanvasPanelStatuses(t *testing.T) {
	ts, st := canvasServer(t)
	m := newCanvas(t, ts, "Ops")
	id := m["id"].(string)
	mustPanel(t, ts, id, "terminal", "t1", 0, 0, 32, 28)
	if code, out := addPanelReq(t, ts, id, "terminal", "t1", 40, 0, 32, 28); code != http.StatusConflict || !strings.Contains(errorOf(out), "already on this canvas") {
		t.Fatalf("duplicate binding = %d %v", code, out)
	}
	cases := []struct {
		name, kind, ref string
		x, y, w, h      int
		want            string
	}{
		{"w < 32", "terminal", "t", 0, 0, 31, 28, "w must be at least 32 canvas units"},
		{"h < 28", "terminal", "t", 0, 0, 32, 27, "h must be at least 28 canvas units"},
		{"kind", "pin", "t", 0, 0, 32, 28, "kind must be agent, terminal, note, file or diff"},
		{"ref empty", "terminal", "  ", 0, 0, 32, 28, "ref is required"},
		{"note ref empty", "note", " ", 0, 0, 32, 28, "ref is required"},
		{"file ref without a path", "file", "t:term-1", 0, 0, 32, 28, "ref must be <owner>:<id>:<path> with owner t, a or w"},
		{"file ref with an empty path", "file", "t:term-1:", 0, 0, 32, 28, "ref must be <owner>:<id>:<path> with owner t, a or w"},
		{"file ref with an unknown owner letter", "file", "x:term-1:main.go", 0, 0, 32, 28, "ref must be <owner>:<id>:<path> with owner t, a or w"},
		{"diff ref without a path", "diff", "a:agent-1", 0, 0, 32, 28, "ref must be <owner>:<id>:<path> with owner t, a or w"},
	}
	for _, c := range cases {
		if code, out := addPanelReq(t, ts, id, c.kind, c.ref, c.x, c.y, c.w, c.h); code != http.StatusBadRequest || !strings.Contains(errorOf(out), c.want) {
			t.Errorf("%s = %d %v", c.name, code, out)
		}
	}
	// Half a cell is not a placement: the body does not decode.
	if code, out := canvasReq(t, ts, http.MethodPost, "/api/canvases/"+id+"/panels", map[string]any{"kind": "terminal", "ref": "t9", "x": 0.5, "y": 0, "w": 32, "h": 28}); code != http.StatusBadRequest || errorOf(out) != "invalid JSON body" {
		t.Fatalf("fractional x = %d %v", code, out)
	}
	if d := canvasDetail(t, ts, id); d["panelCount"] != float64(1) {
		t.Fatalf("refused panels were stored: %v", d["panelCount"])
	}
	for i := 1; i < store.MaxCanvasPanels; i++ {
		mustPanel(t, ts, id, "terminal", fmt.Sprintf("t%d", i+1), 0, i*32, 32, 28)
	}
	if code, out := addPanelReq(t, ts, id, "terminal", "one more", 0, 0, 32, 28); code != http.StatusBadRequest || !strings.Contains(errorOf(out), "limit: 500 panels per canvas") {
		t.Fatalf("501st panel = %d %v", code, out)
	}
	if d := canvasDetail(t, ts, id); d["panelCount"] != float64(store.MaxCanvasPanels) {
		t.Fatalf("panels = %v", d["panelCount"])
	}
	if n := len(canvasEvents(t, st)); n != 1+store.MaxCanvasPanels {
		t.Fatalf("events = %d (a refusal must announce nothing)", n)
	}
}

// C3 (docs/plans/matrix-canvas.md §4.2) through the handlers: a note binds a
// pin id, and the store is ignorant of it (ADR-0108) — a pin that does not
// exist is accepted and read back verbatim, the same pin twice is a 409.
func TestCanvasPanelKindNote(t *testing.T) {
	ts, _ := canvasServer(t)
	m := newCanvas(t, ts, "Ops")
	id := m["id"].(string)
	for i, ref := range []string{"pin-abc123", "pin-never-existed"} {
		code, out := addPanelReq(t, ts, id, "note", ref, 0, i*32, 32, 28)
		p, _ := out["panel"].(map[string]any)
		if code != http.StatusCreated || p == nil || p["kind"] != "note" || p["ref"] != ref {
			t.Fatalf("note %q = %d %v", ref, code, out)
		}
	}
	if code, out := addPanelReq(t, ts, id, "note", "pin-abc123", 0, 320, 32, 28); code != http.StatusConflict || !strings.Contains(errorOf(out), "already on this canvas") {
		t.Fatalf("duplicate note = %d %v", code, out)
	}
	if d := canvasDetail(t, ts, id); d["panelCount"] != float64(2) {
		t.Fatalf("panels = %v", d["panelCount"])
	}
}

// The same rows for a file and a diff panel: the shape is the rule, the
// owner and the file are not the store's business, the same binding twice is
// a 409, and the same path as a file and as a diff is two panels.
func TestCanvasPanelKindsFileAndDiff(t *testing.T) {
	ts, _ := canvasServer(t)
	m := newCanvas(t, ts, "Ops")
	id := m["id"].(string)
	rows := []struct{ kind, ref string }{
		{"file", "t:term-1:src/main.go"},
		{"file", "w:ws-1:docs/architecture/canvas.md"},
		{"file", "a:agent-gone:never/written.txt"},
		{"diff", "t:term-1:src/main.go"},
	}
	for i, r := range rows {
		code, out := addPanelReq(t, ts, id, r.kind, r.ref, 0, i*32, 32, 28)
		p, _ := out["panel"].(map[string]any)
		if code != http.StatusCreated || p == nil || p["kind"] != r.kind || p["ref"] != r.ref {
			t.Fatalf("%s %q = %d %v", r.kind, r.ref, code, out)
		}
	}
	if code, out := addPanelReq(t, ts, id, "file", "t:term-1:src/main.go", 0, 320, 32, 28); code != http.StatusConflict || !strings.Contains(errorOf(out), "already on this canvas") {
		t.Fatalf("duplicate file = %d %v", code, out)
	}
	if code, out := addPanelReq(t, ts, id, "diff", "t:term-1:src/main.go", 0, 320, 32, 28); code != http.StatusConflict {
		t.Fatalf("duplicate diff = %d %v", code, out)
	}
	if d := canvasDetail(t, ts, id); d["panelCount"] != float64(len(rows)) {
		t.Fatalf("panels = %v", d["panelCount"])
	}
}

// Rows "layout patch: stale → 409", "a panel id not in this canvas, or a
// position out of bounds → 400, nothing written", "ok subset → 200 and one
// canvas.layout event carrying exactly the subset".
func TestCanvasLayoutStatuses(t *testing.T) {
	ts, st := canvasServer(t)
	m := newCanvas(t, ts, "Ops")
	id := m["id"].(string)
	aid, atA := mustPanel(t, ts, id, "terminal", "a", 0, 0, 32, 28)
	bid, at := mustPanel(t, ts, id, "terminal", "b", 40, 0, 32, 28)
	move := func(pid string, x, y, w, h int) map[string]any {
		return map[string]any{"id": pid, "x": x, "y": y, "w": w, "h": h}
	}
	layout := func(ifUpdatedAt string, panels ...map[string]any) (int, map[string]any) {
		return canvasReq(t, ts, http.MethodPatch, "/api/canvases/"+id+"/layout", map[string]any{"ifUpdatedAt": ifUpdatedAt, "panels": panels})
	}
	before := canvasDetail(t, ts, id)
	unchanged := func(what string) {
		t.Helper()
		if got := canvasDetail(t, ts, id); !reflect.DeepEqual(got, before) {
			t.Fatalf("%s wrote something:\n got %v\nwant %v", what, got, before)
		}
	}

	if code, out := layout(atA, move(aid, 0, 32, 32, 28)); code != http.StatusConflict || !strings.Contains(errorOf(out), "changed elsewhere") {
		t.Fatalf("stale = %d %v", code, out)
	}
	unchanged("a stale patch")
	if code, out := layout(at, move(aid, 0, 32, 32, 28), move("panel-nope", 0, 0, 32, 28)); code != http.StatusBadRequest || !strings.Contains(errorOf(out), "panel panel-nope is not on this canvas") {
		t.Fatalf("foreign id = %d %v", code, out)
	}
	unchanged("a foreign id")
	if code, out := layout(at, move(aid, 0, 32, 32, 28), move(bid, 100001, 0, 32, 28)); code != http.StatusBadRequest || !strings.Contains(errorOf(out), "x must be between -100000 and 100000 canvas units") {
		t.Fatalf("out of bounds = %d %v", code, out)
	}
	unchanged("an out-of-bounds row")
	if code, _ := pinReqJSON(t, ts, http.MethodPatch, "/api/canvases/"+id+"/layout", map[string]any{"panels": []map[string]any{move(aid, 0, 32, 32, 28)}}, map[string]string{"If-Match": atA}); code != http.StatusConflict {
		t.Fatalf("stale If-Match = %d", code)
	}
	unchanged("a stale If-Match")
	if code, out := layout(at); code != http.StatusBadRequest || !strings.Contains(errorOf(out), "panels is required") {
		t.Fatalf("empty subset = %d %v", code, out)
	}

	code, out := layout(at, move(aid, 0, 32, 64, 56))
	if moved, _ := out["panels"].([]any); code != http.StatusOK || out["id"] != id || out["updatedAt"] == at || len(moved) != 1 {
		t.Fatalf("subset = %d %v", code, out)
	}
	after := canvasDetail(t, ts, id)
	if after["updatedAt"] != out["updatedAt"] {
		t.Fatalf("updatedAt: answer %v, canvas %v", out["updatedAt"], after["updatedAt"])
	}
	for _, p := range after["panels"].([]any) {
		pm := p.(map[string]any)
		switch pm["id"] {
		case aid:
			if pm["x"] != float64(0) || pm["y"] != float64(32) || pm["w"] != float64(64) || pm["h"] != float64(56) {
				t.Fatalf("a did not move: %v", pm)
			}
		case bid:
			if pm["x"] != float64(40) || pm["y"] != float64(0) || pm["w"] != float64(32) || pm["h"] != float64(28) {
				t.Fatalf("b moved: %v", pm)
			}
		}
	}
	var layouts []store.Event
	for _, ev := range canvasEvents(t, st) {
		if ev.Type == "canvas.layout" {
			layouts = append(layouts, ev)
		}
	}
	if len(layouts) != 1 {
		t.Fatalf("canvas.layout events = %d, want 1", len(layouts))
	}
	var lay store.CanvasLayout
	if err := json.Unmarshal(layouts[0].Data, &lay); err != nil {
		t.Fatal(err)
	}
	if lay.ID != id || lay.UpdatedAt != out["updatedAt"] || !reflect.DeepEqual(lay.Panels, []store.PanelPlacement{{ID: aid, X: 0, Y: 32, W: 64, H: 56}}) {
		t.Fatalf("canvas.layout = %+v", lay)
	}
}

// Rows "update: rename ok / compact ∈ {vertical, none} → 200" and "update:
// compact other / name over limit / stale ifUpdatedAt → 400 / 400 / 409".
func TestCanvasUpdateStatuses(t *testing.T) {
	ts, st := canvasServer(t)
	m := newCanvas(t, ts, "Ops")
	id := m["id"].(string)
	at := m["updatedAt"].(string)
	patch := func(body map[string]any) (int, map[string]any) {
		return canvasReq(t, ts, http.MethodPatch, "/api/canvases/"+id, body)
	}
	code, out := patch(map[string]any{"name": "Ops board", "ifUpdatedAt": at})
	if code != http.StatusOK || out["name"] != "Ops board" || out["compact"] != "vertical" || out["updatedAt"] == at {
		t.Fatalf("rename = %d %v", code, out)
	}
	at2 := out["updatedAt"].(string)
	code, out = patch(map[string]any{"compact": "none", "ifUpdatedAt": at2})
	if code != http.StatusOK || out["compact"] != "none" || out["name"] != "Ops board" {
		t.Fatalf("compact none = %d %v", code, out)
	}
	at3 := out["updatedAt"].(string)
	if code, out = patch(map[string]any{"compact": "vertical"}); code != http.StatusOK || out["compact"] != "vertical" {
		t.Fatalf("compact vertical, no precondition = %d %v", code, out)
	}
	if code, out = patch(map[string]any{"compact": "diagonal"}); code != http.StatusBadRequest || !strings.Contains(errorOf(out), "compact must be vertical or none") {
		t.Fatalf("compact other = %d %v", code, out)
	}
	if code, out = patch(map[string]any{"name": strings.Repeat("x", 81)}); code != http.StatusBadRequest || !strings.Contains(errorOf(out), "name is too long (max 80 characters)") {
		t.Fatalf("name over limit = %d %v", code, out)
	}
	if code, out = patch(map[string]any{"name": "Late", "ifUpdatedAt": at}); code != http.StatusConflict || !strings.Contains(errorOf(out), "changed elsewhere") {
		t.Fatalf("stale = %d %v", code, out)
	}
	if code, _ = pinReqJSON(t, ts, http.MethodPatch, "/api/canvases/"+id, map[string]any{"name": "Late"}, map[string]string{"If-Match": at3}); code != http.StatusConflict {
		t.Fatalf("stale If-Match = %d", code)
	}
	if code, out = patch(map[string]any{}); code != http.StatusBadRequest || !strings.Contains(errorOf(out), "nothing to update") {
		t.Fatalf("empty patch = %d %v", code, out)
	}
	if d := canvasDetail(t, ts, id); d["name"] != "Ops board" || d["compact"] != "vertical" {
		t.Fatalf("a refused patch landed: %v", d)
	}
	want := []string{"canvas.created", "canvas.updated", "canvas.updated", "canvas.updated"}
	if got := eventTypes(canvasEvents(t, st)); !reflect.DeepEqual(got, want) {
		t.Fatalf("events = %v, want %v", got, want)
	}
}

// Rows "any: unknown canvas id → 404" and "remove panel: unknown panel →
// 404", with the message naming what is missing.
func TestCanvasUnknownIDs(t *testing.T) {
	ts, st := canvasServer(t)
	m := newCanvas(t, ts, "Ops")
	id := m["id"].(string)
	pid, _ := mustPanel(t, ts, id, "terminal", "t1", 0, 0, 32, 28)
	placement := []map[string]any{{"id": pid, "x": 0, "y": 8, "w": 32, "h": 28}}
	cases := []struct {
		method, path string
		body         any
		want         string
	}{
		{http.MethodGet, "/api/canvases/canvas-nope", nil, "canvas not found"},
		{http.MethodPatch, "/api/canvases/canvas-nope", map[string]any{"name": "x"}, "canvas not found"},
		{http.MethodPatch, "/api/canvases/canvas-nope/layout", map[string]any{"panels": placement}, "canvas not found"},
		{http.MethodPost, "/api/canvases/canvas-nope/panels", map[string]any{"kind": "terminal", "ref": "t", "x": 0, "y": 0, "w": 32, "h": 28}, "canvas not found"},
		{http.MethodDelete, "/api/canvases/canvas-nope/panels/" + pid, nil, "canvas not found"},
		{http.MethodDelete, "/api/canvases/canvas-nope", nil, "canvas not found"},
		{http.MethodDelete, "/api/canvases/" + id + "/panels/panel-nope", nil, "panel not found"},
	}
	for _, c := range cases {
		if code, out := canvasReq(t, ts, c.method, c.path, c.body); code != http.StatusNotFound || !strings.Contains(errorOf(out), c.want) {
			t.Errorf("%s %s = %d %v", c.method, c.path, code, out)
		}
	}
	if d := canvasDetail(t, ts, id); d["panelCount"] != float64(1) {
		t.Fatalf("a miss touched the real canvas: %v", d)
	}
	if got := eventTypes(canvasEvents(t, st)); !reflect.DeepEqual(got, []string{"canvas.created", "canvas.panel.added"}) {
		t.Fatalf("events = %v", got)
	}
}

// ---- the plane -----------------------------------------------------------

func patchCanvas(t *testing.T, ts *httptest.Server, id string, body map[string]any) (int, map[string]any) {
	t.Helper()
	return canvasReq(t, ts, http.MethodPatch, "/api/canvases/"+id, body)
}

// rectsOf reads the canvas's panels as {id: [x, y, w, h]}.
func rectsOf(t *testing.T, ts *httptest.Server, id string) map[string][4]float64 {
	t.Helper()
	out := map[string][4]float64{}
	for _, p := range canvasDetail(t, ts, id)["panels"].([]any) {
		pm := p.(map[string]any)
		out[pm["id"].(string)] = [4]float64{pm["x"].(float64), pm["y"].(float64), pm["w"].(float64), pm["h"].(float64)}
	}
	return out
}

// Rows "add panel | w < 32 or h < 28 → 400 naming the limit", "| negative
// x/y → accepted", "| |x| > 100000 → 400 naming the bound" and "layout
// patch | mixed → 400, nothing written": the plane's rules at the HTTP
// boundary (ADR-0118 left one set).
func TestCanvasPlanePanelStatuses(t *testing.T) {
	ts, st := canvasServer(t)
	canvas := newCanvas(t, ts, "Canvas")
	id := canvas["id"].(string)
	cases := []struct {
		name       string
		x, y, w, h int
		want       string
	}{
		{"w < 32", 0, 0, 31, 42, "w must be at least 32 canvas units"},
		{"h < 28", 0, 0, 32, 27, "h must be at least 28 canvas units"},
		{"w over the bound", 0, 0, 4097, 42, "w must be at most 4096 canvas units"},
		{"x off the plane", 100001, 0, 32, 42, "x must be between -100000 and 100000 canvas units"},
		{"y off the plane", 0, -100001, 32, 42, "y must be between -100000 and 100000 canvas units"},
	}
	for _, c := range cases {
		if code, out := addPanelReq(t, ts, id, "terminal", "t-"+c.name, c.x, c.y, c.w, c.h); code != http.StatusBadRequest || !strings.Contains(errorOf(out), c.want) {
			t.Errorf("%s = %d %v", c.name, code, out)
		}
	}
	// Negative coordinates are the plane's: accepted, and read back as sent.
	code, added := addPanelReq(t, ts, id, "terminal", "neg", -4000, -2500, 32, 28)
	if code != http.StatusCreated {
		t.Fatalf("negative placement = %d %v", code, added)
	}
	panel := added["panel"].(map[string]any)
	if panel["x"] != float64(-4000) || panel["y"] != float64(-2500) {
		t.Fatalf("negative placement = %v", panel)
	}
	negID := panel["id"].(string)
	okID, _ := mustPanel(t, ts, id, "agent", "ok", 200, 0, 40, 30)

	// A rectangle in the old grid units is refused in the plane's words.
	if code, out := addPanelReq(t, ts, id, "terminal", "small", 0, 0, 4, 8); code != http.StatusBadRequest || !strings.Contains(errorOf(out), "w must be at least 32 canvas units") {
		t.Fatalf("grid-sized rectangle = %d %v", code, out)
	}

	// A mixed layout patch: one legal rectangle, one not — 400, nothing
	// written; the batch is all or nothing.
	before := rectsOf(t, ts, id)
	body := map[string]any{"panels": []map[string]any{
		{"id": negID, "x": -8, "y": -8, "w": 40, "h": 40},
		{"id": okID, "x": 0, "y": 0, "w": 32, "h": 27},
	}}
	if code, out := canvasReq(t, ts, http.MethodPatch, "/api/canvases/"+id+"/layout", body); code != http.StatusBadRequest || !strings.Contains(errorOf(out), "panel "+okID+": h must be at least 28 canvas units") {
		t.Fatalf("mixed batch = %d %v", code, out)
	}
	if got := rectsOf(t, ts, id); !reflect.DeepEqual(got, before) {
		t.Fatalf("a refused batch wrote something:\n got %v\nwant %v", got, before)
	}
	body = map[string]any{"panels": []map[string]any{{"id": negID, "x": -8, "y": -8, "w": 40, "h": 40}}}
	if code, out := canvasReq(t, ts, http.MethodPatch, "/api/canvases/"+id+"/layout", body); code != http.StatusOK {
		t.Fatalf("canvas layout patch = %d %v", code, out)
	}
	if got := rectsOf(t, ts, id)[negID]; got != [4]float64{-8, -8, 40, 40} {
		t.Fatalf("canvas panel did not move: %v", got)
	}
	want := []string{"canvas.created", "canvas.panel.added", "canvas.panel.added", "canvas.layout"}
	if got := eventTypes(canvasEvents(t, st)); !reflect.DeepEqual(got, want) {
		t.Fatalf("events = %v, want %v", got, want)
	}
}

// ---- edges (ADR-0116) ----------------------------------------------------
//
// An edge grants exactly ADR-0104's mailbox contact: the owner API is the
// only way one is drawn, and these rows are its decision table at the HTTP
// boundary. What the grant *means* is proved in the store
// (internal/store/canvas_edges_test.go); here it is statuses, shapes,
// messages and events.

func addEdgeReq(t *testing.T, ts *httptest.Server, id, a, b string) (int, map[string]any) {
	t.Helper()
	return canvasReq(t, ts, http.MethodPost, "/api/canvases/"+id+"/edges", map[string]any{"aPanel": a, "bPanel": b})
}

func edgeList(t *testing.T, ts *httptest.Server, id string) []any {
	t.Helper()
	code, out := canvasReq(t, ts, http.MethodGet, "/api/canvases/"+id+"/edges", nil)
	if code != http.StatusOK {
		t.Fatalf("GET edges = %d %v", code, out)
	}
	list, _ := out["edges"].([]any)
	if list == nil {
		t.Fatalf("GET edges has no list: %v", out)
	}
	return list
}

// Rows: an edge between two agent/terminal panels (201, event, both
// directions dedupe to one row), the reverse (409 "already linked"), the
// list, the canvas read that carries it, and the removal (204, event) that
// takes the grant with it.
func TestCanvasEdgeRoutes(t *testing.T) {
	ts, st := canvasServer(t)
	m := newCanvas(t, ts, "Ops")
	id := m["id"].(string)
	term, _ := mustPanel(t, ts, id, "terminal", "term-1", 0, 0, 32, 28)
	agent, upd := mustPanel(t, ts, id, "agent", "agent-1", 40, 0, 32, 28)

	// Drawn from the agent to the terminal; the store orders the pair.
	code, added := addEdgeReq(t, ts, id, agent, term)
	if code != http.StatusCreated || added["id"] != id || added["updatedAt"] == upd {
		t.Fatalf("add edge = %d %v", code, added)
	}
	edge := added["edge"].(map[string]any)
	eid := edge["id"].(string)
	lo, hi := term, agent
	if lo > hi {
		lo, hi = hi, lo
	}
	if edge["aPanel"] != lo || edge["bPanel"] != hi || edge["createdAt"] == "" {
		t.Fatalf("edge = %v, want ordered (%s, %s)", edge, lo, hi)
	}

	// The same pair drawn backwards is the 409, and the message is the one
	// the client repeats.
	if code, out := addEdgeReq(t, ts, id, term, agent); code != http.StatusConflict || !strings.Contains(errorOf(out), "already linked") {
		t.Fatalf("reverse edge = %d %v", code, out)
	}
	if list := edgeList(t, ts, id); len(list) != 1 {
		t.Fatalf("edges = %v, want one row", list)
	}
	// One read still opens a canvas.
	d := canvasDetail(t, ts, id)
	edges, _ := d["edges"].([]any)
	if len(edges) != 1 || edges[0].(map[string]any)["id"] != eid {
		t.Fatalf("detail edges = %v", d["edges"])
	}

	if code, out := canvasReq(t, ts, http.MethodDelete, "/api/canvases/"+id+"/edges/"+eid, nil); code != http.StatusNoContent {
		t.Fatalf("remove edge = %d %v", code, out)
	}
	if list := edgeList(t, ts, id); len(list) != 0 {
		t.Fatalf("edges after removal = %v", list)
	}
	if code, out := canvasReq(t, ts, http.MethodDelete, "/api/canvases/"+id+"/edges/"+eid, nil); code != http.StatusNotFound || !strings.Contains(errorOf(out), "edge not found") {
		t.Fatalf("remove twice = %d %v", code, out)
	}
	want := []string{"canvas.created", "canvas.panel.added", "canvas.panel.added", "canvas.edge.added", "canvas.edge.removed"}
	if got := eventTypes(canvasEvents(t, st)); !reflect.DeepEqual(got, want) {
		t.Fatalf("events = %v, want %v", got, want)
	}
}

// Rows: the same panel twice, a panel of another canvas, a note panel (the
// message names the kind), a canvas that does not exist, and a body that is
// not JSON. Each is the status the client acts on and the sentence it shows.
func TestCanvasEdgeRefusalRoutes(t *testing.T) {
	ts, _ := canvasServer(t)
	m := newCanvas(t, ts, "Ops")
	id := m["id"].(string)
	other := newCanvas(t, ts, "Other")["id"].(string)
	term, _ := mustPanel(t, ts, id, "terminal", "term-1", 0, 0, 32, 28)
	agent, _ := mustPanel(t, ts, id, "agent", "agent-1", 40, 0, 32, 28)
	note, _ := mustPanel(t, ts, id, "note", "pin-1", 80, 0, 32, 28)
	elsewhere, _ := mustPanel(t, ts, other, "agent", "agent-2", 0, 0, 32, 28)

	for _, c := range []struct {
		name, a, b string
		status     int
		want       string
	}{
		{"the same panel twice", term, term, http.StatusBadRequest, "an edge needs two different panels"},
		{"a missing id", term, "", http.StatusBadRequest, "aPanel and bPanel are required"},
		{"a panel from another canvas", term, elsewhere, http.StatusBadRequest, "is not on this canvas"},
		{"a note panel", agent, note, http.StatusBadRequest, "is a note panel and has no mailbox"},
	} {
		t.Run(c.name, func(t *testing.T) {
			code, out := addEdgeReq(t, ts, id, c.a, c.b)
			if code != c.status || !strings.Contains(errorOf(out), c.want) {
				t.Fatalf("= %d %v, want %d containing %q", code, out, c.status, c.want)
			}
		})
	}
	t.Run("a canvas that does not exist", func(t *testing.T) {
		if code, out := addEdgeReq(t, ts, "canvas-nope", term, agent); code != http.StatusNotFound {
			t.Fatalf("= %d %v", code, out)
		}
		if code, out := canvasReq(t, ts, http.MethodGet, "/api/canvases/canvas-nope/edges", nil); code != http.StatusNotFound {
			t.Fatalf("GET = %d %v", code, out)
		}
		if code, out := canvasReq(t, ts, http.MethodDelete, "/api/canvases/canvas-nope/edges/edge-x", nil); code != http.StatusNotFound {
			t.Fatalf("DELETE = %d %v", code, out)
		}
	})
	if list := edgeList(t, ts, id); len(list) != 0 {
		t.Fatalf("a refusal wrote a row: %v", list)
	}
}

// Row: removing a panel takes its edges with it (the store cascades; the
// feed carries canvas.panel.removed and no event per edge, so the client
// drops them the same way).
func TestCanvasEdgeFollowsItsPanel(t *testing.T) {
	ts, _ := canvasServer(t)
	id := newCanvas(t, ts, "Ops")["id"].(string)
	term, _ := mustPanel(t, ts, id, "terminal", "term-1", 0, 0, 32, 28)
	agent, _ := mustPanel(t, ts, id, "agent", "agent-1", 40, 0, 32, 28)
	if code, out := addEdgeReq(t, ts, id, term, agent); code != http.StatusCreated {
		t.Fatalf("add edge = %d %v", code, out)
	}
	if code, out := canvasReq(t, ts, http.MethodDelete, "/api/canvases/"+id+"/panels/"+term, nil); code != http.StatusNoContent {
		t.Fatalf("remove panel = %d %v", code, out)
	}
	if list := edgeList(t, ts, id); len(list) != 0 {
		t.Fatalf("edges outlived their panel: %v", list)
	}
}

// Row: the cap is reached → 400 naming it. The fill is a fixture (the store
// is the same writer the route uses); the last edge is the route's answer.
func TestCanvasEdgeCapRoute(t *testing.T) {
	ts, st := canvasServer(t)
	id := newCanvas(t, ts, "Ops")["id"].(string)
	const n = 46 // n*(n-1)/2 = 1035 pairs > the cap
	panels := make([]string, 0, n)
	for i := range n {
		p, err := st.AddCanvasPanel(id, "terminal", fmt.Sprintf("term-%d", i), 0, 0, 32, 28)
		if err != nil {
			t.Fatalf("panel %d: %v", i, err)
		}
		panels = append(panels, p.Panel.ID)
	}
	drawn := 0
	for i := 0; i < n && drawn < store.MaxCanvasEdges; i++ {
		for j := i + 1; j < n && drawn < store.MaxCanvasEdges; j++ {
			if _, err := st.AddCanvasEdge(id, panels[i], panels[j]); err != nil {
				t.Fatalf("fill (%d,%d): %v", i, j, err)
			}
			drawn++
		}
	}
	code, out := addEdgeReq(t, ts, id, panels[n-2], panels[n-1])
	if code != http.StatusBadRequest || !strings.Contains(errorOf(out), fmt.Sprintf("limit: %d edges per canvas", store.MaxCanvasEdges)) {
		t.Fatalf("at the cap = %d %v", code, out)
	}
}
