package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/tmux"
)

func TestTerminalsNeedTmuxOrCRUD(t *testing.T) {
	ts, _, _ := cleanupServer(t)
	listed := do(t, ts.Client(), mustGet(t, ts.URL+"/api/terminals"))
	if listed.StatusCode != http.StatusOK {
		t.Fatalf("list = %d", listed.StatusCode)
	}
	if !tmux.New().Available() {
		t.Skip("tmux not installed — create/open gated (accepted)")
	}
	created := postJSON(t, ts, "/api/terminals", map[string]any{})
	if created.StatusCode != http.StatusCreated {
		t.Fatalf("create = %d", created.StatusCode)
	}
	var page map[string]any
	if err := json.NewDecoder(created.Body).Decode(&page); err != nil {
		t.Fatal(err)
	}
	id, _ := page["id"].(string)
	sess, _ := page["session"].(string)
	if id == "" || sess == "" {
		t.Fatalf("page=%+v", page)
	}
	t.Cleanup(func() { _ = tmux.New().KillSession(context.Background(), sess) })

	again := postJSON(t, ts, "/api/terminals/"+id+"/open", map[string]any{})
	if again.StatusCode != http.StatusOK {
		t.Fatalf("open = %d", again.StatusCode)
	}
	raw, _ := json.Marshal(map[string]string{"name": "build"})
	preq, _ := http.NewRequest(http.MethodPatch, ts.URL+"/api/terminals/"+id, bytes.NewReader(raw))
	preq.Header.Set("Content-Type", "application/json")
	renamed := do(t, ts.Client(), preq)
	if renamed.StatusCode != http.StatusOK {
		t.Fatalf("rename = %d", renamed.StatusCode)
	}
	var renamedPage map[string]any
	_ = json.NewDecoder(renamed.Body).Decode(&renamedPage)
	if renamedPage["name"] != "build" {
		t.Fatalf("name=%v", renamedPage["name"])
	}

	req, _ := http.NewRequest(http.MethodDelete, ts.URL+"/api/terminals/"+id, nil)
	killed := do(t, ts.Client(), req)
	if killed.StatusCode != http.StatusNoContent {
		t.Fatalf("delete = %d", killed.StatusCode)
	}
	listed = do(t, ts.Client(), mustGet(t, ts.URL+"/api/terminals"))
	var bag struct {
		Terminals []map[string]any `json:"terminals"`
	}
	_ = json.NewDecoder(listed.Body).Decode(&bag)
	if len(bag.Terminals) != 0 {
		t.Fatalf("after delete %+v", bag.Terminals)
	}
}

func TestTerminalsGone(t *testing.T) {
	ts, _, _ := cleanupServer(t)
	got := do(t, ts.Client(), mustGet(t, ts.URL+"/api/terminals/nope"))
	if got.StatusCode != http.StatusNotFound && got.StatusCode != http.StatusMethodNotAllowed {
		// GET by id is not a route; DELETE gone is.
	}
	req, _ := http.NewRequest(http.MethodDelete, ts.URL+"/api/terminals/nope", nil)
	del := do(t, ts.Client(), req)
	if del.StatusCode != http.StatusNotFound {
		t.Fatalf("delete missing = %d", del.StatusCode)
	}
}

func TestTerminalText(t *testing.T) {
	ts, _, _ := cleanupServer(t)
	miss := do(t, ts.Client(), mustGet(t, ts.URL+"/api/terminals/nope/text?path=a.go"))
	if miss.StatusCode != http.StatusNotFound {
		t.Fatalf("gone term = %d", miss.StatusCode)
	}
	if !tmux.New().Available() {
		t.Skip("tmux not installed")
	}
	proj := t.TempDir()
	if err := os.WriteFile(filepath.Join(proj, "a.go"), []byte("pkg\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	created := postJSON(t, ts, "/api/terminals", map[string]any{"cwd": proj})
	if created.StatusCode != http.StatusCreated {
		t.Fatalf("create = %d", created.StatusCode)
	}
	var page map[string]any
	_ = json.NewDecoder(created.Body).Decode(&page)
	id, _ := page["id"].(string)
	sess, _ := page["session"].(string)
	t.Cleanup(func() { _ = tmux.New().KillSession(context.Background(), sess) })
	got := do(t, ts.Client(), mustGet(t, ts.URL+"/api/terminals/"+id+"/text?path=a.go"))
	if got.StatusCode != http.StatusOK {
		t.Fatalf("text = %d", got.StatusCode)
	}
	var body map[string]any
	_ = json.NewDecoder(got.Body).Decode(&body)
	if body["text"] != "pkg\n" || body["path"] != "a.go" {
		t.Fatalf("%+v", body)
	}
	out := do(t, ts.Client(), mustGet(t, ts.URL+"/api/terminals/"+id+"/text?path=/etc/passwd"))
	if out.StatusCode != http.StatusBadRequest {
		t.Fatalf("escape = %d", out.StatusCode)
	}
}

func TestTerminalLiveCwd(t *testing.T) {
	ts, _, _ := cleanupServer(t)
	if !tmux.New().Available() {
		t.Skip("tmux not installed")
	}
	start := t.TempDir()
	live := t.TempDir()
	if err := os.WriteFile(filepath.Join(live, "ping.txt"), []byte("pong\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	created := postJSON(t, ts, "/api/terminals", map[string]any{"cwd": start})
	if created.StatusCode != http.StatusCreated {
		t.Fatalf("create = %d", created.StatusCode)
	}
	var page map[string]any
	_ = json.NewDecoder(created.Body).Decode(&page)
	id, _ := page["id"].(string)
	sess, _ := page["session"].(string)
	t.Cleanup(func() { _ = tmux.New().KillSession(context.Background(), sess) })

	cwd := do(t, ts.Client(), mustGet(t, ts.URL+"/api/terminals/"+id+"/cwd"))
	if cwd.StatusCode != http.StatusOK {
		t.Fatalf("cwd = %d", cwd.StatusCode)
	}
	var bag map[string]any
	_ = json.NewDecoder(cwd.Body).Decode(&bag)
	if filepath.Clean(bag["cwd"].(string)) != filepath.Clean(start) {
		t.Fatalf("start cwd=%v want %s", bag["cwd"], start)
	}

	m := tmux.New()
	if err := m.SendKeys(context.Background(), sess, "cd "+live, "Enter"); err != nil {
		t.Fatalf("cd: %v", err)
	}
	deadline := time.Now().Add(2 * time.Second)
	var got string
	for time.Now().Before(deadline) {
		res := do(t, ts.Client(), mustGet(t, ts.URL+"/api/terminals/"+id+"/cwd"))
		var body map[string]any
		_ = json.NewDecoder(res.Body).Decode(&body)
		got, _ = body["cwd"].(string)
		if filepath.Clean(got) == filepath.Clean(live) {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if filepath.Clean(got) != filepath.Clean(live) {
		t.Fatalf("live cwd=%q want %s", got, live)
	}
	text := do(t, ts.Client(), mustGet(t, ts.URL+"/api/terminals/"+id+"/text?path=ping.txt"))
	if text.StatusCode != http.StatusOK {
		t.Fatalf("text after cd = %d", text.StatusCode)
	}
	var file map[string]any
	_ = json.NewDecoder(text.Body).Decode(&file)
	if file["text"] != "pong\n" {
		t.Fatalf("text=%+v", file)
	}
	esc := do(t, ts.Client(), mustGet(t, ts.URL+"/api/terminals/"+id+"/text?path=a.go"))
	if esc.StatusCode != http.StatusNotFound && esc.StatusCode != http.StatusBadRequest {
		t.Fatalf("start file after cd = %d", esc.StatusCode)
	}
}

// A terminal is created with tmux owning the mouse (ADR-0024). That default
// was removed on 2026-08-30 to recover native text selection, which broke the
// wheel in Pi's TUI — it does not enable mouse tracking itself, so nothing was
// left to scroll it. Passthrough stays on alongside it: it is what carries
// OSC 52 out of the pane, and it is what keeps copying possible now that a
// drag belongs to tmux. Both are set per session, so a regression is silent.
func TestTerminalSessionTakesTheMouseAndPassesThrough(t *testing.T) {
	ts, _, _ := cleanupServer(t)
	if !tmux.New().Available() {
		t.Skip("tmux not installed")
	}
	created := postJSON(t, ts, "/api/terminals", map[string]any{"cwd": t.TempDir()})
	if created.StatusCode != http.StatusCreated {
		t.Fatalf("create = %d", created.StatusCode)
	}
	var page map[string]any
	_ = json.NewDecoder(created.Body).Decode(&page)
	sess, _ := page["session"].(string)
	if sess == "" {
		t.Fatalf("no session in %+v", page)
	}
	t.Cleanup(func() { _ = tmux.New().KillSession(context.Background(), sess) })

	if got := tmuxOption(t, sess, "mouse"); got != "on" {
		t.Fatalf("mouse=%q, want on — without it the wheel dies in a TUI that does not track the mouse", got)
	}
	if got := tmuxOption(t, sess, "allow-passthrough"); got != "on" {
		t.Fatalf("allow-passthrough=%q, want on — OSC 52 cannot leave the pane without it", got)
	}
}

func tmuxOption(t *testing.T, session, option string) string {
	t.Helper()
	out, err := exec.Command("tmux", "show-options", "-t", session+":", "-v", option).Output()
	if err != nil {
		return "" // unset reports an error; an unset option is not "on"
	}
	return strings.TrimSpace(string(out))
}

// The list reports where the terminal IS. The stored cwd is only where it was
// born; after a `cd` inside the pane the two disagree, and the git facts in
// the same response were already read from the live path — reporting the
// stale one next to them showed the wrong directory under every terminal in
// the sidebar.
func TestListTerminalsReportsTheLivePaneCwd(t *testing.T) {
	ts, _, _ := cleanupServer(t)
	tm := tmux.New()
	if !tm.Available() {
		t.Skip("tmux not installed")
	}
	born := t.TempDir()
	created := postJSON(t, ts, "/api/terminals", map[string]any{"cwd": born})
	if created.StatusCode != http.StatusCreated {
		t.Fatalf("create = %d", created.StatusCode)
	}
	var page map[string]any
	_ = json.NewDecoder(created.Body).Decode(&page)
	created.Body.Close()
	sess, _ := page["session"].(string)
	t.Cleanup(func() { _ = tm.KillSession(context.Background(), sess) })

	elsewhere := t.TempDir()
	if err := tm.SendKeys(context.Background(), sess, "cd "+elsewhere, "Enter"); err != nil {
		t.Fatalf("cd: %v", err)
	}

	deadline := time.Now().Add(3 * time.Second)
	for {
		res, err := ts.Client().Get(ts.URL + "/api/terminals")
		if err != nil {
			t.Fatal(err)
		}
		var out struct {
			Terminals []map[string]any `json:"terminals"`
		}
		_ = json.NewDecoder(res.Body).Decode(&out)
		res.Body.Close()
		if len(out.Terminals) != 1 {
			t.Fatalf("terminals = %d, want 1", len(out.Terminals))
		}
		cwd, _ := out.Terminals[0]["cwd"].(string)
		// The pane may resolve symlinks (macOS /tmp); compare resolved.
		want, _ := filepath.EvalSymlinks(elsewhere)
		got, _ := filepath.EvalSymlinks(cwd)
		if got == want {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("cwd = %q, want the live pane path %q (stored was %q)", cwd, elsewhere, born)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// Open answers with the same live view the list gives. The app merges the
// open response into its terminal list, so a response carrying the record
// cwd overwrote the live one while the stale git survived the merge — the
// SELECTED terminal (the one thing the user is looking at) showed one
// directory's path beside another directory's branch.
func TestOpenTerminalReportsTheLiveViewToo(t *testing.T) {
	ts, _, _ := cleanupServer(t)
	tm := tmux.New()
	if !tm.Available() {
		t.Skip("tmux not installed")
	}
	born := t.TempDir()
	created := postJSON(t, ts, "/api/terminals", map[string]any{"cwd": born})
	var page map[string]any
	_ = json.NewDecoder(created.Body).Decode(&page)
	created.Body.Close()
	id, _ := page["id"].(string)
	sess, _ := page["session"].(string)
	t.Cleanup(func() { _ = tm.KillSession(context.Background(), sess) })

	elsewhere := t.TempDir()
	if err := tm.SendKeys(context.Background(), sess, "cd "+elsewhere, "Enter"); err != nil {
		t.Fatalf("cd: %v", err)
	}

	want, _ := filepath.EvalSymlinks(elsewhere)
	deadline := time.Now().Add(3 * time.Second)
	for {
		res := postJSON(t, ts, "/api/terminals/"+id+"/open", nil)
		var view map[string]any
		_ = json.NewDecoder(res.Body).Decode(&view)
		res.Body.Close()
		cwd, _ := view["cwd"].(string)
		got, _ := filepath.EvalSymlinks(cwd)
		if got == want {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("open cwd = %q, want the live %q (record was %q)", cwd, elsewhere, born)
		}
		time.Sleep(100 * time.Millisecond)
	}
}
