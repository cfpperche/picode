package server

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/store"
)

func pinReqJSON(t *testing.T, ts *httptest.Server, method, path string, body any, headers map[string]string) (int, map[string]any) {
	t.Helper()
	raw, _ := json.Marshal(body)
	req, _ := http.NewRequest(method, ts.URL+path, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	res := do(t, ts.Client(), req)
	defer res.Body.Close()
	out := map[string]any{}
	_ = json.NewDecoder(res.Body).Decode(&out)
	return res.StatusCode, out
}

func pinMultipart(t *testing.T, ts *httptest.Server, path string, fields map[string]string, files map[string][3]string) (int, map[string]any) {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	for k, v := range fields {
		_ = mw.WriteField(k, v)
	}
	for field, f := range files { // f = {filename, content-type, bytes}
		h := make(map[string][]string)
		h["Content-Disposition"] = []string{`form-data; name="` + field + `"; filename="` + f[0] + `"`}
		h["Content-Type"] = []string{f[1]}
		w, err := mw.CreatePart(h)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(f[2]))
	}
	_ = mw.Close()
	req, _ := http.NewRequest(http.MethodPost, ts.URL+path, &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	res := do(t, ts.Client(), req)
	defer res.Body.Close()
	out := map[string]any{}
	_ = json.NewDecoder(res.Body).Decode(&out)
	return res.StatusCode, out
}

func newPin(t *testing.T, ts *httptest.Server, title string) map[string]any {
	t.Helper()
	code, p := pinReqJSON(t, ts, http.MethodPost, "/api/pins", map[string]any{"title": title, "tags": []string{}, "body": ""}, nil)
	if code != http.StatusOK {
		t.Fatalf("create pin = %d %v", code, p)
	}
	return p
}

// Limits and preconditions come back as the status the caller can act on.
func TestPinStatuses(t *testing.T) {
	ts, _, _ := cleanupServer(t)
	code, out := pinReqJSON(t, ts, http.MethodPost, "/api/pins", map[string]any{"title": strings.Repeat("x", 201)}, nil)
	if code != http.StatusBadRequest || !strings.Contains(out["error"].(string), "too long") {
		t.Fatalf("long title = %d %v", code, out)
	}
	p := newPin(t, ts, "One")
	id := p["id"].(string)
	stale := p["updatedAt"].(string)
	code, _ = pinReqJSON(t, ts, http.MethodPatch, "/api/pins/"+id, map[string]any{"title": "Two", "ifUpdatedAt": stale}, nil)
	if code != http.StatusOK {
		t.Fatalf("first patch = %d", code)
	}
	code, out = pinReqJSON(t, ts, http.MethodPatch, "/api/pins/"+id, map[string]any{"title": "Three", "ifUpdatedAt": stale}, nil)
	if code != http.StatusConflict {
		t.Fatalf("stale patch = %d %v", code, out)
	}
	code, _ = pinReqJSON(t, ts, http.MethodPatch, "/api/pins/"+id, map[string]any{"title": "Three"}, map[string]string{"If-Match": stale})
	if code != http.StatusConflict {
		t.Fatalf("stale If-Match = %d", code)
	}
	code, _ = pinReqJSON(t, ts, http.MethodPatch, "/api/pins/nope-000000", map[string]any{"title": "x"}, nil)
	if code != http.StatusNotFound {
		t.Fatalf("missing patch = %d", code)
	}
	var got map[string]any
	getJSON(t, ts, "/api/pins/"+id, &got)
	if got["title"] != "Two" {
		t.Fatalf("stale write landed: %v", got["title"])
	}
	var list struct{ Pins []map[string]any }
	getJSON(t, ts, "/api/pins", &list)
	if len(list.Pins) != 1 {
		t.Fatalf("list = %v", list)
	}
	if _, has := list.Pins[0]["body"]; has {
		t.Fatalf("list carries body: %v", list.Pins[0])
	}
}

func TestPinUploadLimitsAndNames(t *testing.T) {
	ts, _, _ := cleanupServer(t)
	p := newPin(t, ts, "Files")
	id := p["id"].(string)
	base := "/api/pins/" + id + "/files"
	code, out := pinMultipart(t, ts, base, nil, map[string][3]string{"file": {"run.exe", "application/octet-stream", "MZ"}})
	if code != http.StatusBadRequest || !strings.Contains(out["error"].(string), "not allowed") {
		t.Fatalf("exe = %d %v", code, out)
	}
	code, out = pinMultipart(t, ts, base, nil, map[string][3]string{"file": {"big.png", "image/png", strings.Repeat("p", store.MaxPinImageSize+1)}})
	if code != http.StatusBadRequest || !strings.Contains(out["error"].(string), "too large") {
		t.Fatalf("big image = %d %v", code, out)
	}
	code, out = pinMultipart(t, ts, base, nil, map[string][3]string{"file": {"café.png", "image/png", "PNG"}})
	if code != http.StatusOK {
		t.Fatalf("upload = %d %v", code, out)
	}
	fid := out["id"].(string)
	res, err := ts.Client().Get(ts.URL + base + "/" + fid)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	disp := res.Header.Get("Content-Disposition")
	if res.StatusCode != http.StatusOK || !strings.HasPrefix(disp, "inline;") || !strings.Contains(disp, "filename*=utf-8''caf%C3%A9.png") {
		t.Fatalf("bytes = %d disposition %q", res.StatusCode, disp)
	}
	if ct := res.Header.Get("Content-Type"); ct != "image/png" {
		t.Fatalf("content-type = %q", ct)
	}
	code, _ = pinMultipart(t, ts, "/api/pins/../etc/files", nil, map[string][3]string{"file": {"a.txt", "text/plain", "x"}})
	if code == http.StatusOK {
		t.Fatal("path-shaped pin id accepted")
	}
}

// The sketch round-trip: annotate an image by reference, read the scene
// back, edit it, and see the preview's version move (findings 4–6).
func TestPinSketchRoundTrip(t *testing.T) {
	ts, dataDir, _ := cleanupServer(t)
	p := newPin(t, ts, "Board")
	id := p["id"].(string)
	files := "/api/pins/" + id + "/files"
	sketches := "/api/pins/" + id + "/sketches"
	_, img := pinMultipart(t, ts, files, nil, map[string][3]string{"file": {"shot.png", "image/png", "PNG-BYTES"}})
	imgID := img["id"].(string)

	scene := `{"type":"excalidraw","elements":[{"type":"image","fileId":"bg:x"}],"files":{}}`
	code, out := pinMultipart(t, ts, sketches, map[string]string{"source": "annotate", "baseFileId": "nope", "name": "S"},
		map[string][3]string{"scene": {"scene.json", "application/json", scene}, "preview": {"p.png", "image/png", "PNG1"}})
	if code != http.StatusBadRequest {
		t.Fatalf("bad base = %d %v", code, out)
	}
	code, out = pinMultipart(t, ts, sketches, map[string]string{"source": "annotate", "baseFileId": imgID, "name": "S"},
		map[string][3]string{"scene": {"scene.json", "application/json", scene}, "preview": {"p.png", "image/png", "PNG1"}})
	if code != http.StatusOK || out["kind"] != "sketch" || out["baseFileId"] != imgID {
		t.Fatalf("add sketch = %d %v", code, out)
	}
	sid := out["id"].(string)
	v1 := out["updatedAt"].(string)
	if !exists(filepath.Join(dataDir, "pins", id, sid)) || !exists(filepath.Join(dataDir, "pins", id, sid+".scene")) {
		t.Fatal("sketch bytes missing")
	}
	if entries, _ := os.ReadDir(filepath.Join(dataDir, "pins", id)); len(entries) != 3 {
		t.Fatalf("temp files left: %d entries", len(entries))
	}

	res, _ := ts.Client().Get(ts.URL + files + "/" + sid + "/scene")
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != http.StatusOK || string(body) != scene {
		t.Fatalf("scene = %d %s", res.StatusCode, body)
	}
	res, _ = ts.Client().Get(ts.URL + files + "/" + imgID + "/scene")
	res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("scene of an image = %d", res.StatusCode)
	}

	scene2 := strings.Replace(scene, "bg:x", "bg:y", 1)
	code, out = pinMultipart(t, ts, sketches, map[string]string{"id": sid, "name": "S2"},
		map[string][3]string{"scene": {"scene.json", "application/json", scene2}, "preview": {"p.png", "image/png", "PNG2"}})
	if code != http.StatusOK || out["id"] != sid || out["name"] != "S2" || out["updatedAt"] == v1 {
		t.Fatalf("update sketch = %d %v (v1 %s)", code, out, v1)
	}
	res, _ = ts.Client().Get(ts.URL + files + "/" + sid)
	body, _ = io.ReadAll(res.Body)
	res.Body.Close()
	if string(body) != "PNG2" {
		t.Fatalf("preview after edit = %q", body)
	}

	code, out = pinMultipart(t, ts, sketches, map[string]string{"id": imgID},
		map[string][3]string{"scene": {"scene.json", "application/json", scene}, "preview": {"p.png", "image/png", "PNG"}})
	if code != http.StatusNotFound {
		t.Fatalf("update image as sketch = %d %v", code, out)
	}
	code, out = pinMultipart(t, ts, sketches, map[string]string{"source": "blank"},
		map[string][3]string{"scene": {"scene.json", "application/json", strings.Repeat("s", store.MaxPinSceneSize+1)}, "preview": {"p.png", "image/png", "PNG"}})
	if code != http.StatusBadRequest || !strings.Contains(out["error"].(string), "too large") {
		t.Fatalf("huge scene = %d %v", code, out)
	}
	var pin map[string]any
	getJSON(t, ts, "/api/pins/"+id, &pin)
	if n := len(pin["files"].([]any)); n != 2 {
		t.Fatalf("files after refused saves = %d", n)
	}
}

// Boot removes pins/<id> directories whose row is gone and nothing else.
func TestPinSweepOrphanDirs(t *testing.T) {
	root := t.TempDir()
	st, err := store.Open(filepath.Join(root, "picode.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	live, _ := st.CreatePin("Live", nil, "")
	for _, d := range []string{live.ID, "gone-abc123", "not a pin id", ".tmp"} {
		if err := os.MkdirAll(filepath.Join(root, "pins", d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	_ = os.WriteFile(filepath.Join(root, "pins", "gone-abc123", "f"), []byte("x"), 0o644)
	sweepPinDirs(root, st)
	if !exists(filepath.Join(root, "pins", live.ID)) {
		t.Fatal("live pin dir removed")
	}
	if exists(filepath.Join(root, "pins", "gone-abc123")) {
		t.Fatal("orphan dir kept")
	}
	if !exists(filepath.Join(root, "pins", "not a pin id")) || !exists(filepath.Join(root, "pins", ".tmp")) {
		t.Fatal("swept a directory that is not a pin id")
	}
}

func TestPinListV2Routes(t *testing.T) {
	ts, _, _ := cleanupServer(t)
	a := newPin(t, ts, "Alpha")
	b := newPin(t, ts, "Beta")
	code, out := pinReqJSON(t, ts, http.MethodPost, "/api/pins/"+b["id"].(string)+"/starred", map[string]any{"starred": true}, nil)
	if code != http.StatusOK || out["starred"] != true {
		t.Fatalf("star = %d %v", code, out)
	}
	code, out = pinReqJSON(t, ts, http.MethodPost, "/api/pins/"+a["id"].(string)+"/archived", map[string]any{"archived": true}, nil)
	if code != http.StatusOK || out["archivedAt"] == nil {
		t.Fatalf("archive = %d %v", code, out)
	}
	var list struct {
		Pins     []map[string]any
		Archived int
	}
	getJSON(t, ts, "/api/pins", &list)
	if len(list.Pins) != 1 || list.Pins[0]["id"] != b["id"] || list.Archived != 1 {
		t.Fatalf("live list = %+v", list)
	}
	list = struct {
		Pins     []map[string]any
		Archived int
	}{}
	getJSON(t, ts, "/api/pins?archived=1", &list)
	if len(list.Pins) != 1 || list.Pins[0]["id"] != a["id"] {
		t.Fatalf("archived list = %+v", list)
	}
	list = struct {
		Pins     []map[string]any
		Archived int
	}{}
	getJSON(t, ts, "/api/pins?q=alpha", &list)
	if len(list.Pins) != 1 || list.Pins[0]["archivedAt"] == nil {
		t.Fatalf("search = %+v", list)
	}
	code, _ = pinReqJSON(t, ts, http.MethodPost, "/api/pins/nope-000000/starred", map[string]any{"starred": true}, nil)
	if code != http.StatusNotFound {
		t.Fatalf("star missing = %d", code)
	}
}
