package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/cfpperche/picode/internal/pipkg"
)

func TestPiKeysRoundTrip(t *testing.T) {
	home := t.TempDir()
	old := pipkg.UserDir
	pipkg.UserDir = func() string { return filepath.Join(home, ".pi", "agent") }
	t.Cleanup(func() { pipkg.UserDir = old })
	ts := newTestServer(t, "cat")

	res, err := ts.Client().Get(ts.URL + "/api/pi-keys")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET %d", res.StatusCode)
	}
	var got map[string]any
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	acts, _ := got["actions"].([]any)
	if len(acts) < 10 {
		t.Fatalf("catalog %d", len(acts))
	}

	body, _ := json.Marshal(map[string]any{
		"action": "tui.editor.deleteWordBackward",
		"keys":   []string{"ctrl+backspace"},
	})
	req, _ := http.NewRequest(http.MethodPut, ts.URL+"/api/pi-keys", bytes.NewReader(body))
	put, err := ts.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer put.Body.Close()
	if put.StatusCode != http.StatusOK {
		t.Fatalf("PUT %d", put.StatusCode)
	}
	raw, err := os.ReadFile(filepath.Join(home, ".pi", "agent", "keybindings.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(raw, []byte("ctrl+backspace")) {
		t.Fatalf("%s", raw)
	}

	reset, _ := json.Marshal(map[string]any{"action": "tui.editor.deleteWordBackward", "reset": true})
	req2, _ := http.NewRequest(http.MethodPut, ts.URL+"/api/pi-keys", bytes.NewReader(reset))
	out, err := ts.Client().Do(req2)
	if err != nil {
		t.Fatal(err)
	}
	defer out.Body.Close()
	if out.StatusCode != http.StatusOK {
		t.Fatalf("reset %d", out.StatusCode)
	}
}

// The pane names the file it writes and picks the default pi binds on this
// host. Both come from the report: a web page can know neither, and a pane
// that guesses prints a key the machine does not use (nine of pi's rows carry
// a Windows or WSL alternate).
func TestPiKeysReportNamesTheFileAndThePlatform(t *testing.T) {
	home := t.TempDir()
	old := pipkg.UserDir
	dir := filepath.Join(home, ".pi", "agent")
	pipkg.UserDir = func() string { return dir }
	t.Cleanup(func() { pipkg.UserDir = old })
	ts := newTestServer(t, "cat")

	var rep struct {
		File     string `json:"file"`
		Exists   bool   `json:"exists"`
		Platform string `json:"platform"`
		Actions  []struct {
			ID  string              `json:"id"`
			Alt map[string][]string `json:"alt"`
		} `json:"actions"`
	}
	res, err := ts.Client().Get(ts.URL + "/api/pi-keys")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if err := json.NewDecoder(res.Body).Decode(&rep); err != nil {
		t.Fatal(err)
	}
	if rep.File != filepath.Join(dir, "keybindings.json") {
		t.Fatalf("file %q", rep.File)
	}
	if rep.Exists {
		t.Fatal("the file does not exist yet")
	}
	switch rep.Platform {
	case "linux", "windows", "darwin", "wsl":
	default:
		t.Fatalf("platform %q", rep.Platform)
	}
	// The declared-empty Windows row has to reach the client as a present but
	// empty list, and as a key the client can tell from "no alternate".
	var suspendAlt map[string][]string
	for _, a := range rep.Actions {
		if a.ID == "app.suspend" {
			suspendAlt = a.Alt
		}
	}
	if suspendAlt == nil {
		t.Fatal("app.suspend must carry its alt map")
	}
	windows, declared := suspendAlt["windows"]
	if !declared || len(windows) != 0 {
		t.Fatalf("app.suspend alt.windows = %v (declared %v)", windows, declared)
	}
	if _, declared := suspendAlt["wsl"]; declared {
		t.Fatal("app.suspend keeps the base binding on WSL")
	}
}

// Reset all clears every override this catalog knows in one call, and leaves
// a key the catalog does not know where it was.
func TestPiKeysResetAllLeavesUnknownKeys(t *testing.T) {
	home := t.TempDir()
	old := pipkg.UserDir
	dir := filepath.Join(home, ".pi", "agent")
	pipkg.UserDir = func() string { return dir }
	t.Cleanup(func() { pipkg.UserDir = old })
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "keybindings.json")
	seed := `{"future.thing":"x","tui.input.submit":["enter"],"tui.editor.undo":["ctrl+z"]}` + "\n"
	if err := os.WriteFile(path, []byte(seed), 0o644); err != nil {
		t.Fatal(err)
	}
	ts := newTestServer(t, "cat")

	body, _ := json.Marshal(map[string]any{"resetAll": true})
	req, _ := http.NewRequest(http.MethodPut, ts.URL+"/api/pi-keys", bytes.NewReader(body))
	res, err := ts.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("PUT %d", res.StatusCode)
	}
	var rep struct {
		User map[string][]string `json:"user"`
	}
	if err := json.NewDecoder(res.Body).Decode(&rep); err != nil {
		t.Fatal(err)
	}
	if len(rep.User) != 1 || len(rep.User["future.thing"]) != 1 {
		t.Fatalf("after reset all: %v", rep.User)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(raw, []byte("future.thing")) {
		t.Fatalf("the unknown key was deleted: %s", raw)
	}
	if bytes.Contains(raw, []byte("tui.input.submit")) || bytes.Contains(raw, []byte("tui.editor.undo")) {
		t.Fatalf("a known override survived: %s", raw)
	}
}

func TestPiKeysUnknownAction(t *testing.T) {
	home := t.TempDir()
	old := pipkg.UserDir
	pipkg.UserDir = func() string { return filepath.Join(home, ".pi", "agent") }
	t.Cleanup(func() { pipkg.UserDir = old })
	ts := newTestServer(t, "cat")
	body, _ := json.Marshal(map[string]any{"action": "nope", "keys": []string{"a"}})
	req, _ := http.NewRequest(http.MethodPut, ts.URL+"/api/pi-keys", bytes.NewReader(body))
	res, err := ts.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("status %d", res.StatusCode)
	}
}
