package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A Send is a set: the whole set arrives in one call and leaves as ONE note
// plus one crop per pin. The owner's call (2026-09-19) is that the number of
// annotations is the human's business — nothing in the transport may cap it —
// so this test pins both halves: many pins deliver as one package, and the
// package is the only bound (the per-file size cap).
func TestAnnotationBatchStagesOneNoteForTheWholeSend(t *testing.T) {
	st := testStore(t)
	deps := Deps{Store: st, DataDir: t.TempDir()}
	ts := httptest.NewServer(New("127.0.0.1:0", deps).Handler)
	t.Cleanup(ts.Close)
	dir := t.TempDir()
	_, agent, err := storeWorkspaceWithAgent(st, "App", dir)
	if err != nil {
		t.Fatal(err)
	}
	term, err := st.CreateTerminalIn(agent.WorkspaceID, "a", dir)
	if err != nil {
		t.Fatal(err)
	}

	// Twelve pins: more than any transport's file cap, which is the point.
	const pins = 12
	items := make([]map[string]string, 0, pins)
	for i := 1; i <= pins; i++ {
		items = append(items, map[string]string{
			"selector": fmt.Sprintf("#e%d", i),
			"comment":  fmt.Sprintf("pin %d", i),
			"dom":      fmt.Sprintf(`<b id="e%d">x</b>`, i),
			"css":      "color: red",
			"image":    "iVBORw0KGgo=",
		})
	}
	payload, err := json.Marshal(map[string]any{
		"terminalId": term.ID,
		"url":        "https://example.com/page",
		"title":      "Page",
		"items":      items,
	})
	if err != nil {
		t.Fatal(err)
	}
	body := string(payload)
	code, page := postRaw(t, ts, "/api/browser/annotations", body)
	if code != http.StatusOK {
		t.Fatalf("batch = %d %v", code, page)
	}
	rows, _ := page["annotations"].([]any)
	if len(rows) != pins {
		t.Fatalf("every pin keeps a row: %d", len(rows))
	}
	paths, _ := page["paths"].([]any)
	if len(paths) != pins+1 {
		t.Fatalf("one note + one crop per pin: %d", len(paths))
	}
	notePath, _ := paths[0].(string)
	raw, err := os.ReadFile(notePath)
	if err != nil {
		t.Fatalf("the note is staged where it says: %v", err)
	}
	note := string(raw)
	if !strings.Contains(note, fmt.Sprintf("%d annotations on https://example.com/page", pins)) {
		t.Fatalf("the note counts the set:\n%s", note)
	}
	for i := 1; i <= pins; i++ {
		if !strings.Contains(note, fmt.Sprintf("## %d. pin %d", i, i)) {
			t.Fatalf("pin %d is in the note, in order", i)
		}
		shot := filepath.Base(paths[i].(string))
		if !strings.Contains(note, shot) {
			t.Fatalf("pin %d names its crop %q:\n%s", i, shot, note)
		}
		if _, err := os.Stat(paths[i].(string)); err != nil {
			t.Fatalf("crop %d is on disk: %v", i, err)
		}
	}
	// Every row points at the SAME note (it is the package) and at its own crop.
	listed, err := st.ListBrowserAnnotations(50, "")
	if err != nil || len(listed) != pins {
		t.Fatalf("rows = %d, %v", len(listed), err)
	}
	first := filepath.Base(notePath)
	for _, row := range listed {
		if row.Note != first {
			t.Fatalf("row note = %q, want %q", row.Note, first)
		}
		if row.Shot == "" || row.Note == row.Shot {
			t.Fatalf("row crop = %q", row.Shot)
		}
	}
}

// A Send that cannot be staged whole stages nothing: a bad image in the
// middle must not leave half a package on disk and half in the store.
func TestAnnotationBatchRefusesWholeSendOnOneBadItem(t *testing.T) {
	st := testStore(t)
	deps := Deps{Store: st, DataDir: t.TempDir()}
	ts := httptest.NewServer(New("127.0.0.1:0", deps).Handler)
	t.Cleanup(ts.Close)
	dir := t.TempDir()
	_, agent, err := storeWorkspaceWithAgent(st, "App", dir)
	if err != nil {
		t.Fatal(err)
	}
	term, err := st.CreateTerminalIn(agent.WorkspaceID, "a", dir)
	if err != nil {
		t.Fatal(err)
	}
	body := fmt.Sprintf(`{"terminalId":"%s","url":"https://example.com/","items":[{"selector":"a","image":"iVBORw0KGgo="},{"selector":"b","image":"not base64!!"}]}`, term.ID)
	code, page := postRaw(t, ts, "/api/browser/annotations", body)
	if code != http.StatusBadRequest {
		t.Fatalf("a bad item must refuse the Send: %d %v", code, page)
	}
	listed, err := st.ListBrowserAnnotations(50, "")
	if err != nil || len(listed) != 0 {
		t.Fatalf("nothing staged: %d rows, %v", len(listed), err)
	}
	if entries, _ := os.ReadDir(filepath.Join(dir, ".picode", "drop")); len(entries) != 0 {
		t.Fatalf("nothing on disk: %d entries", len(entries))
	}
	// An empty set is not a Send either — and it says so.
	code, page = postRaw(t, ts, "/api/browser/annotations", fmt.Sprintf(`{"terminalId":"%s","url":"u","items":[]}`, term.ID))
	if code != http.StatusBadRequest || !strings.Contains(fmt.Sprint(page["error"]), "at least one") {
		t.Fatalf("empty set = %d %v", code, page)
	}
}

// The JSON body of the batch is what the strip sends: items, not one pin.
func TestAnnotationBatchBodyShape(t *testing.T) {
	var payload struct {
		Items []struct {
			Selector string `json:"selector"`
			Image    string `json:"image"`
		} `json:"items"`
	}
	raw := `{"terminalId":"t","url":"u","items":[{"selector":"#a","comment":"c","dom":"<i>","css":"color: red","image":"AAA"}]}`
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Items) != 1 || payload.Items[0].Selector != "#a" || payload.Items[0].Image != "AAA" {
		t.Fatalf("payload = %+v", payload)
	}
}

// The camera-off row of the ADR's decision table: pins still stage, the note
// is the whole package, and nothing pretends a picture exists.
func TestAnnotationBatchWithoutScreenshotsStagesTheNoteAlone(t *testing.T) {
	st := testStore(t)
	deps := Deps{Store: st, DataDir: t.TempDir()}
	ts := httptest.NewServer(New("127.0.0.1:0", deps).Handler)
	t.Cleanup(ts.Close)
	dir := t.TempDir()
	_, agent, err := storeWorkspaceWithAgent(st, "App", dir)
	if err != nil {
		t.Fatal(err)
	}
	term, err := st.CreateTerminalIn(agent.WorkspaceID, "a", dir)
	if err != nil {
		t.Fatal(err)
	}
	body := fmt.Sprintf(`{"terminalId":"%s","url":"https://example.com/","items":[{"selector":"#a","comment":"one"},{"selector":"#b","comment":"two"}]}`, term.ID)
	code, page := postRaw(t, ts, "/api/browser/annotations", body)
	if code != http.StatusOK {
		t.Fatalf("no-image batch = %d %v", code, page)
	}
	paths, _ := page["paths"].([]any)
	if len(paths) != 1 {
		t.Fatalf("only the note is staged: %v", paths)
	}
	note, _ := os.ReadFile(paths[0].(string))
	if strings.Contains(string(note), "Screenshot:") {
		t.Fatal("no crop means no screenshot line")
	}
	if !strings.Contains(string(note), "## 2. two") {
		t.Fatalf("both pins are in the note:\n%s", note)
	}
}
