package server

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/cfpperche/picode/internal/session"
)

func writeTailSession(t *testing.T, cwd string, n int) string {
	t.Helper()
	dir := session.Dir(cwd)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, "tail.jsonl")
	body := ""
	for i := 1; i <= n; i++ {
		body += `{"type":"message","message":{"role":"user","content":[{"type":"text","text":"m` + strconv.Itoa(i) + `"}]}}` + "\n"
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestTranscriptTailWindow(t *testing.T) {
	ts, _, home := cleanupServer(t)
	proj := filepath.Join(home, "p")
	if err := os.MkdirAll(proj, 0o755); err != nil {
		t.Fatal(err)
	}
	src := writeTailSession(t, proj, 6)
	ws := postJSON(t, ts, "/api/workspaces", map[string]string{"name": "App", "path": proj})
	if ws.StatusCode != http.StatusCreated {
		t.Fatalf("workspace = %d", ws.StatusCode)
	}
	var wsv workspaceView
	if err := json.NewDecoder(ws.Body).Decode(&wsv); err != nil {
		t.Fatal(err)
	}
	base := ts.URL + "/api/workspaces/" + wsv.ID + "/sessions/transcript?path=" + src

	get := func(q string) map[string]any {
		res := do(t, ts.Client(), mustGet(t, base+q))
		if res.StatusCode != http.StatusOK {
			t.Fatalf("GET %s = %d", q, res.StatusCode)
		}
		var body map[string]any
		if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		return body
	}
	count := func(b map[string]any) int { return len(b["events"].([]any)) }
	texts := func(b map[string]any) (string, string) {
		arr := b["events"].([]any)
		if len(arr) == 0 {
			return "", ""
		}
		first := arr[0].(map[string]any)
		last := arr[len(arr)-1].(map[string]any)
		return first["text"].(string), last["text"].(string)
	}

	full := get("")
	if count(full) != 6 || full["total"].(float64) != 6 || full["remaining"].(float64) != 0 {
		t.Fatalf("full = total %v remaining %v n %d", full["total"], full["remaining"], count(full))
	}
	tail2 := get("&tail=2")
	if count(tail2) != 2 || tail2["remaining"].(float64) != 4 {
		t.Fatalf("tail2 remaining=%v n=%d", tail2["remaining"], count(tail2))
	}
	if f, l := texts(tail2); f != "m5" || l != "m6" {
		t.Fatalf("tail2 = %s..%s, want m5..m6", f, l)
	}
	win := get("&tail=2&skip=2")
	if count(win) != 2 || win["remaining"].(float64) != 2 {
		t.Fatalf("win remaining=%v n=%d", win["remaining"], count(win))
	}
	if f, l := texts(win); f != "m3" || l != "m4" {
		t.Fatalf("win = %s..%s, want m3..m4", f, l)
	}
	skipBig := get("&tail=2&skip=99")
	if count(skipBig) != 2 || skipBig["remaining"].(float64) != 0 {
		t.Fatalf("skipBig remaining=%v n=%d", skipBig["remaining"], count(skipBig))
	}
	if f, l := texts(skipBig); f != "m1" || l != "m2" {
		t.Fatalf("skipBig = %s..%s, want m1..m2", f, l)
	}
	if b, _ := skipBig["bytes"].(float64); b <= 0 {
		t.Fatalf("bytes = %v", skipBig["bytes"])
	}
	if skipBig["compacted"] != false {
		t.Fatalf("uncompacted session reported compacted=%v", skipBig["compacted"])
	}
}

func writeCompactSession(t *testing.T, cwd string) string {
	t.Helper()
	dir := session.Dir(cwd)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, "compact.jsonl")
	body := ``
	for i := 1; i <= 3; i++ {
		body += `{"type":"message","message":{"role":"user","content":[{"type":"text","text":"old` + strconv.Itoa(i) + `"}]}}` + "\n"
	}
	body += `{"type":"compaction","summary":"SUMMARY: the old turns","timestamp":1759000000}` + "\n"
	body += `{"type":"message","message":{"role":"user","content":[{"type":"text","text":"new1"}]}}` + "\n"
	body += `{"type":"message","message":{"role":"user","content":[{"type":"text","text":"new2"}]}}` + "\n"
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// After a compaction the window must start at the boundary: pre-compaction
// history stays inside the summary, exactly like pi replays it.
func TestTranscriptCompactionBoundary(t *testing.T) {
	ts, _, home := cleanupServer(t)
	proj := filepath.Join(home, "p2")
	if err := os.MkdirAll(proj, 0o755); err != nil {
		t.Fatal(err)
	}
	src := writeCompactSession(t, proj)
	ws := postJSON(t, ts, "/api/workspaces", map[string]string{"name": "App2", "path": proj})
	var wsv workspaceView
	if err := json.NewDecoder(ws.Body).Decode(&wsv); err != nil {
		t.Fatal(err)
	}
	base := ts.URL + "/api/workspaces/" + wsv.ID + "/sessions/transcript?path=" + src
	res := do(t, ts.Client(), mustGet(t, base))
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET = %d", res.StatusCode)
	}
	var body map[string]any
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	events := body["events"].([]any)
	if len(events) != 3 {
		t.Fatalf("events = %d, want 3 (summary + new1 + new2)", len(events))
	}
	first := events[0].(map[string]any)
	if first["kind"] != "compaction" || first["text"] != "SUMMARY: the old turns" {
		t.Fatalf("first = %v %v", first["kind"], first["text"])
	}
	for _, e := range events[1:] {
		if txt := e.(map[string]any)["text"]; txt == "old1" || txt == "old2" || txt == "old3" {
			t.Fatalf("pre-compaction event leaked: %v", txt)
		}
	}
	if body["compacted"] != true || body["total"].(float64) != 3 {
		t.Fatalf("compacted=%v total=%v", body["compacted"], body["total"])
	}
	// Load earlier stops at the boundary: no pre-compaction remaining.
	res2 := do(t, ts.Client(), mustGet(t, base+"&tail=1"))
	var b2 map[string]any
	if err := json.NewDecoder(res2.Body).Decode(&b2); err != nil {
		t.Fatal(err)
	}
	if b2["remaining"].(float64) != 2 {
		t.Fatalf("tail=1 remaining=%v, want 2", b2["remaining"])
	}
}
