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
