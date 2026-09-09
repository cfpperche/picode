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

// matrixServer is cleanupServer with the store in hand, so a test can read
// the rows a route appended to the change log — the proof behind the feed.
func matrixServer(t *testing.T) (*httptest.Server, *store.Store) {
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

func matrixReq(t *testing.T, ts *httptest.Server, method, path string, body any) (int, map[string]any) {
	t.Helper()
	return pinReqJSON(t, ts, method, path, body, nil)
}

func newMatrix(t *testing.T, ts *httptest.Server, name string) map[string]any {
	t.Helper()
	code, m := matrixReq(t, ts, http.MethodPost, "/api/matrices", map[string]any{"name": name})
	if code != http.StatusCreated {
		t.Fatalf("create matrix = %d %v", code, m)
	}
	return m
}

func addPanelReq(t *testing.T, ts *httptest.Server, id, kind, ref string, x, y, w, h int) (int, map[string]any) {
	t.Helper()
	return matrixReq(t, ts, http.MethodPost, "/api/matrices/"+id+"/panels", map[string]any{"kind": kind, "ref": ref, "x": x, "y": y, "w": w, "h": h})
}

func mustPanel(t *testing.T, ts *httptest.Server, id, kind, ref string, x, y, w, h int) (string, string) {
	t.Helper()
	code, out := addPanelReq(t, ts, id, kind, ref, x, y, w, h)
	if code != http.StatusCreated {
		t.Fatalf("add panel %s %s = %d %v", kind, ref, code, out)
	}
	return out["panel"].(map[string]any)["id"].(string), out["updatedAt"].(string)
}

func matrixDetail(t *testing.T, ts *httptest.Server, id string) map[string]any {
	t.Helper()
	var d map[string]any
	if code := getJSON(t, ts, "/api/matrices/"+id, &d); code != http.StatusOK {
		t.Fatalf("GET matrix = %d %v", code, d)
	}
	return d
}

func matrixEvents(t *testing.T, st *store.Store) []store.Event {
	t.Helper()
	evs, err := st.ListEventsSince(0, 100_000)
	if err != nil {
		t.Fatal(err)
	}
	out := []store.Event{}
	for _, ev := range evs {
		if strings.HasPrefix(ev.Type, "matrix.") {
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
// "remove panel ok", "delete matrix ok").
func TestMatrixRoutes(t *testing.T) {
	ts, st := matrixServer(t)
	code, out := matrixReq(t, ts, http.MethodGet, "/api/matrices", nil)
	if list, _ := out["matrices"].([]any); code != http.StatusOK || list == nil || len(list) != 0 {
		t.Fatalf("empty list = %d %v", code, out)
	}
	m := newMatrix(t, ts, "Ops")
	id := m["id"].(string)
	if m["name"] != "Ops" || m["compact"] != "vertical" || m["panelCount"] != float64(0) || m["updatedAt"] == "" || m["createdAt"] != m["updatedAt"] {
		t.Fatalf("created = %v", m)
	}
	if _, has := m["panels"]; has {
		t.Fatalf("summary carries panels: %v", m)
	}

	code, added := addPanelReq(t, ts, id, "terminal", "term-1", 0, 0, 4, 8)
	if code != http.StatusCreated || added["id"] != id || added["updatedAt"] == m["updatedAt"] {
		t.Fatalf("add panel = %d %v", code, added)
	}
	panel := added["panel"].(map[string]any)
	pid := panel["id"].(string)
	if panel["kind"] != "terminal" || panel["ref"] != "term-1" || panel["x"] != float64(0) || panel["w"] != float64(4) || panel["h"] != float64(8) || panel["createdAt"] == "" {
		t.Fatalf("panel = %v", panel)
	}

	d := matrixDetail(t, ts, id)
	if panels, _ := d["panels"].([]any); len(panels) != 1 || d["panelCount"] != float64(1) || d["updatedAt"] != added["updatedAt"] || d["name"] != "Ops" {
		t.Fatalf("detail = %v", d)
	}

	code, lay := matrixReq(t, ts, http.MethodPatch, "/api/matrices/"+id+"/layout", map[string]any{
		"ifUpdatedAt": added["updatedAt"],
		"panels":      []map[string]any{{"id": pid, "x": 4, "y": 8, "w": 8, "h": 16}},
	})
	if moved, _ := lay["panels"].([]any); code != http.StatusOK || lay["id"] != id || lay["updatedAt"] == added["updatedAt"] || len(moved) != 1 {
		t.Fatalf("layout = %d %v", code, lay)
	}

	code, upd := matrixReq(t, ts, http.MethodPatch, "/api/matrices/"+id, map[string]any{"name": "Ops board", "compact": "none", "ifUpdatedAt": lay["updatedAt"]})
	if code != http.StatusOK || upd["name"] != "Ops board" || upd["compact"] != "none" || upd["panelCount"] != float64(1) || upd["updatedAt"] == lay["updatedAt"] {
		t.Fatalf("update = %d %v", code, upd)
	}

	var list struct{ Matrices []map[string]any }
	getJSON(t, ts, "/api/matrices", &list)
	if len(list.Matrices) != 1 || list.Matrices[0]["name"] != "Ops board" || list.Matrices[0]["panelCount"] != float64(1) || list.Matrices[0]["updatedAt"] != upd["updatedAt"] {
		t.Fatalf("list = %v", list.Matrices)
	}

	if code, out := matrixReq(t, ts, http.MethodDelete, "/api/matrices/"+id+"/panels/"+pid, nil); code != http.StatusNoContent {
		t.Fatalf("remove panel = %d %v", code, out)
	}
	d = matrixDetail(t, ts, id)
	if panels, _ := d["panels"].([]any); len(panels) != 0 || d["panelCount"] != float64(0) || d["updatedAt"] == upd["updatedAt"] {
		t.Fatalf("detail after remove = %v", d)
	}

	if code, out := matrixReq(t, ts, http.MethodDelete, "/api/matrices/"+id, nil); code != http.StatusNoContent {
		t.Fatalf("delete matrix = %d %v", code, out)
	}
	if code := getJSON(t, ts, "/api/matrices/"+id, nil); code != http.StatusNotFound {
		t.Fatalf("deleted matrix GET = %d", code)
	}

	want := []string{"matrix.created", "matrix.panel.added", "matrix.layout", "matrix.updated", "matrix.panel.removed", "matrix.deleted"}
	if got := eventTypes(matrixEvents(t, st)); !reflect.DeepEqual(got, want) {
		t.Fatalf("events = %v, want %v", got, want)
	}
}

// Rows "create: 64 matrices exist" and "create: name empty / > 80 runes /
// only spaces" over HTTP: 400 naming the limit, never truncated.
func TestMatrixCreateStatuses(t *testing.T) {
	ts, _ := matrixServer(t)
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
		if code, out := matrixReq(t, ts, http.MethodPost, "/api/matrices", c.body); code != http.StatusBadRequest || !strings.Contains(errorOf(out), c.want) {
			t.Errorf("%s = %d %v", c.name, code, out)
		}
	}
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/matrices", strings.NewReader("{not json"))
	req.Header.Set("Content-Type", "application/json")
	if res := do(t, ts.Client(), req); res.StatusCode != http.StatusBadRequest {
		t.Fatalf("malformed body = %d", res.StatusCode)
	}
	m := newMatrix(t, ts, strings.Repeat("é", 80))
	if name := m["name"].(string); utf8.RuneCountInString(name) != 80 || !utf8.ValidString(name) {
		t.Fatalf("80-rune name mangled: %q", name)
	}
	for i := 1; i < store.MaxMatrices; i++ {
		newMatrix(t, ts, fmt.Sprintf("m%02d", i))
	}
	if code, out := matrixReq(t, ts, http.MethodPost, "/api/matrices", map[string]any{"name": "one more"}); code != http.StatusBadRequest || !strings.Contains(errorOf(out), "limit: 64 matrices") {
		t.Fatalf("65th matrix = %d %v", code, out)
	}
	var list struct{ Matrices []map[string]any }
	getJSON(t, ts, "/api/matrices", &list)
	if len(list.Matrices) != store.MaxMatrices {
		t.Fatalf("matrices = %d", len(list.Matrices))
	}
}

// Rows "add panel: same (kind, ref)" → 409, "add panel: 500 panels exist"
// → 400, and the seven placement rules → 400 naming the rule.
func TestMatrixPanelStatuses(t *testing.T) {
	ts, st := matrixServer(t)
	m := newMatrix(t, ts, "Ops")
	id := m["id"].(string)
	mustPanel(t, ts, id, "terminal", "t1", 0, 0, 4, 8)
	if code, out := addPanelReq(t, ts, id, "terminal", "t1", 4, 0, 4, 8); code != http.StatusConflict || !strings.Contains(errorOf(out), "already on this matrix") {
		t.Fatalf("duplicate binding = %d %v", code, out)
	}
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
		if code, out := addPanelReq(t, ts, id, c.kind, c.ref, c.x, c.y, c.w, c.h); code != http.StatusBadRequest || !strings.Contains(errorOf(out), c.want) {
			t.Errorf("%s = %d %v", c.name, code, out)
		}
	}
	// Half a cell is not a placement: the body does not decode.
	if code, out := matrixReq(t, ts, http.MethodPost, "/api/matrices/"+id+"/panels", map[string]any{"kind": "terminal", "ref": "t9", "x": 0.5, "y": 0, "w": 4, "h": 8}); code != http.StatusBadRequest || errorOf(out) != "invalid JSON body" {
		t.Fatalf("fractional x = %d %v", code, out)
	}
	if d := matrixDetail(t, ts, id); d["panelCount"] != float64(1) {
		t.Fatalf("refused panels were stored: %v", d["panelCount"])
	}
	for i := 1; i < store.MaxMatrixPanels; i++ {
		mustPanel(t, ts, id, "terminal", fmt.Sprintf("t%d", i+1), 0, i*8, 4, 8)
	}
	if code, out := addPanelReq(t, ts, id, "terminal", "one more", 0, 0, 4, 8); code != http.StatusBadRequest || !strings.Contains(errorOf(out), "limit: 500 panels per matrix") {
		t.Fatalf("501st panel = %d %v", code, out)
	}
	if d := matrixDetail(t, ts, id); d["panelCount"] != float64(store.MaxMatrixPanels) {
		t.Fatalf("panels = %v", d["panelCount"])
	}
	if n := len(matrixEvents(t, st)); n != 1+store.MaxMatrixPanels {
		t.Fatalf("events = %d (a refusal must announce nothing)", n)
	}
}

// Rows "layout patch: stale → 409", "a panel id not in this matrix, or a
// position out of bounds → 400, nothing written", "ok subset → 200 and one
// matrix.layout event carrying exactly the subset".
func TestMatrixLayoutStatuses(t *testing.T) {
	ts, st := matrixServer(t)
	m := newMatrix(t, ts, "Ops")
	id := m["id"].(string)
	aid, atA := mustPanel(t, ts, id, "terminal", "a", 0, 0, 4, 8)
	bid, at := mustPanel(t, ts, id, "terminal", "b", 4, 0, 4, 8)
	move := func(pid string, x, y, w, h int) map[string]any {
		return map[string]any{"id": pid, "x": x, "y": y, "w": w, "h": h}
	}
	layout := func(ifUpdatedAt string, panels ...map[string]any) (int, map[string]any) {
		return matrixReq(t, ts, http.MethodPatch, "/api/matrices/"+id+"/layout", map[string]any{"ifUpdatedAt": ifUpdatedAt, "panels": panels})
	}
	before := matrixDetail(t, ts, id)
	unchanged := func(what string) {
		t.Helper()
		if got := matrixDetail(t, ts, id); !reflect.DeepEqual(got, before) {
			t.Fatalf("%s wrote something:\n got %v\nwant %v", what, got, before)
		}
	}

	if code, out := layout(atA, move(aid, 0, 8, 4, 8)); code != http.StatusConflict || !strings.Contains(errorOf(out), "changed elsewhere") {
		t.Fatalf("stale = %d %v", code, out)
	}
	unchanged("a stale patch")
	if code, out := layout(at, move(aid, 0, 8, 4, 8), move("panel-nope", 0, 0, 4, 8)); code != http.StatusBadRequest || !strings.Contains(errorOf(out), "panel panel-nope is not on this matrix") {
		t.Fatalf("foreign id = %d %v", code, out)
	}
	unchanged("a foreign id")
	if code, out := layout(at, move(aid, 0, 8, 4, 8), move(bid, 10, 0, 4, 8)); code != http.StatusBadRequest || !strings.Contains(errorOf(out), "x + w must be at most 12 columns") {
		t.Fatalf("out of bounds = %d %v", code, out)
	}
	unchanged("an out-of-bounds row")
	if code, _ := pinReqJSON(t, ts, http.MethodPatch, "/api/matrices/"+id+"/layout", map[string]any{"panels": []map[string]any{move(aid, 0, 8, 4, 8)}}, map[string]string{"If-Match": atA}); code != http.StatusConflict {
		t.Fatalf("stale If-Match = %d", code)
	}
	unchanged("a stale If-Match")
	if code, out := layout(at); code != http.StatusBadRequest || !strings.Contains(errorOf(out), "panels is required") {
		t.Fatalf("empty subset = %d %v", code, out)
	}

	code, out := layout(at, move(aid, 0, 8, 8, 16))
	if moved, _ := out["panels"].([]any); code != http.StatusOK || out["id"] != id || out["updatedAt"] == at || len(moved) != 1 {
		t.Fatalf("subset = %d %v", code, out)
	}
	after := matrixDetail(t, ts, id)
	if after["updatedAt"] != out["updatedAt"] {
		t.Fatalf("updatedAt: answer %v, matrix %v", out["updatedAt"], after["updatedAt"])
	}
	for _, p := range after["panels"].([]any) {
		pm := p.(map[string]any)
		switch pm["id"] {
		case aid:
			if pm["x"] != float64(0) || pm["y"] != float64(8) || pm["w"] != float64(8) || pm["h"] != float64(16) {
				t.Fatalf("a did not move: %v", pm)
			}
		case bid:
			if pm["x"] != float64(4) || pm["y"] != float64(0) || pm["w"] != float64(4) || pm["h"] != float64(8) {
				t.Fatalf("b moved: %v", pm)
			}
		}
	}
	var layouts []store.Event
	for _, ev := range matrixEvents(t, st) {
		if ev.Type == "matrix.layout" {
			layouts = append(layouts, ev)
		}
	}
	if len(layouts) != 1 {
		t.Fatalf("matrix.layout events = %d, want 1", len(layouts))
	}
	var lay store.MatrixLayout
	if err := json.Unmarshal(layouts[0].Data, &lay); err != nil {
		t.Fatal(err)
	}
	if lay.ID != id || lay.UpdatedAt != out["updatedAt"] || !reflect.DeepEqual(lay.Panels, []store.PanelPlacement{{ID: aid, X: 0, Y: 8, W: 8, H: 16}}) {
		t.Fatalf("matrix.layout = %+v", lay)
	}
}

// Rows "update: rename ok / compact ∈ {vertical, none} → 200" and "update:
// compact other / name over limit / stale ifUpdatedAt → 400 / 400 / 409".
func TestMatrixUpdateStatuses(t *testing.T) {
	ts, st := matrixServer(t)
	m := newMatrix(t, ts, "Ops")
	id := m["id"].(string)
	at := m["updatedAt"].(string)
	patch := func(body map[string]any) (int, map[string]any) {
		return matrixReq(t, ts, http.MethodPatch, "/api/matrices/"+id, body)
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
	if code, _ = pinReqJSON(t, ts, http.MethodPatch, "/api/matrices/"+id, map[string]any{"name": "Late"}, map[string]string{"If-Match": at3}); code != http.StatusConflict {
		t.Fatalf("stale If-Match = %d", code)
	}
	if code, out = patch(map[string]any{}); code != http.StatusBadRequest || !strings.Contains(errorOf(out), "nothing to update") {
		t.Fatalf("empty patch = %d %v", code, out)
	}
	if d := matrixDetail(t, ts, id); d["name"] != "Ops board" || d["compact"] != "vertical" {
		t.Fatalf("a refused patch landed: %v", d)
	}
	want := []string{"matrix.created", "matrix.updated", "matrix.updated", "matrix.updated"}
	if got := eventTypes(matrixEvents(t, st)); !reflect.DeepEqual(got, want) {
		t.Fatalf("events = %v, want %v", got, want)
	}
}

// Rows "any: unknown matrix id → 404" and "remove panel: unknown panel →
// 404", with the message naming what is missing.
func TestMatrixUnknownIDs(t *testing.T) {
	ts, st := matrixServer(t)
	m := newMatrix(t, ts, "Ops")
	id := m["id"].(string)
	pid, _ := mustPanel(t, ts, id, "terminal", "t1", 0, 0, 4, 8)
	placement := []map[string]any{{"id": pid, "x": 0, "y": 8, "w": 4, "h": 8}}
	cases := []struct {
		method, path string
		body         any
		want         string
	}{
		{http.MethodGet, "/api/matrices/matrix-nope", nil, "matrix not found"},
		{http.MethodPatch, "/api/matrices/matrix-nope", map[string]any{"name": "x"}, "matrix not found"},
		{http.MethodPatch, "/api/matrices/matrix-nope/layout", map[string]any{"panels": placement}, "matrix not found"},
		{http.MethodPost, "/api/matrices/matrix-nope/panels", map[string]any{"kind": "terminal", "ref": "t", "x": 0, "y": 0, "w": 4, "h": 8}, "matrix not found"},
		{http.MethodDelete, "/api/matrices/matrix-nope/panels/" + pid, nil, "matrix not found"},
		{http.MethodDelete, "/api/matrices/matrix-nope", nil, "matrix not found"},
		{http.MethodDelete, "/api/matrices/" + id + "/panels/panel-nope", nil, "panel not found"},
	}
	for _, c := range cases {
		if code, out := matrixReq(t, ts, c.method, c.path, c.body); code != http.StatusNotFound || !strings.Contains(errorOf(out), c.want) {
			t.Errorf("%s %s = %d %v", c.method, c.path, code, out)
		}
	}
	if d := matrixDetail(t, ts, id); d["panelCount"] != float64(1) {
		t.Fatalf("a miss touched the real matrix: %v", d)
	}
	if got := eventTypes(matrixEvents(t, st)); !reflect.DeepEqual(got, []string{"matrix.created", "matrix.panel.added"}) {
		t.Fatalf("events = %v", got)
	}
}
