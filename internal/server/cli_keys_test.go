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

// The envelope the Keyboard pane renders, for the one CLI whose editor ships.
func TestCLIKeysServesPiThroughTheEnvelope(t *testing.T) {
	home := t.TempDir()
	old := pipkg.UserDir
	dir := filepath.Join(home, ".pi", "agent")
	pipkg.UserDir = func() string { return dir }
	t.Cleanup(func() { pipkg.UserDir = old })
	ts := newTestServer(t, "cat")

	var rep struct {
		CLI      string `json:"cli"`
		Label    string `json:"label"`
		State    string `json:"state"`
		Writable bool   `json:"writable"`
		Pickup   string `json:"pickup"`
		Vocab    string `json:"vocab"`
		File     string `json:"file"`
		Exists   bool   `json:"exists"`
		Platform string `json:"platform"`
		Actions  []struct {
			ID string `json:"id"`
		} `json:"actions"`
		User map[string][]string `json:"user"`
	}
	res, err := ts.Client().Get(ts.URL + "/api/cli-keys?cli=pi")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET %d", res.StatusCode)
	}
	if err := json.NewDecoder(res.Body).Decode(&rep); err != nil {
		t.Fatal(err)
	}
	if rep.CLI != "pi" || rep.State != "shipped" || !rep.Writable {
		t.Fatalf("envelope: %+v", rep)
	}
	if rep.Pickup != "reload" || rep.Vocab != "pi" {
		t.Fatalf("pickup/vocab: %q %q", rep.Pickup, rep.Vocab)
	}
	if len(rep.Actions) < 90 {
		t.Fatalf("catalog through the envelope: %d actions", len(rep.Actions))
	}
	if rep.File != filepath.Join(dir, "keybindings.json") || rep.Exists {
		t.Fatalf("file: %q exists=%v", rep.File, rep.Exists)
	}
	if rep.Platform == "" {
		t.Fatal("the envelope must carry the host's platform")
	}
}

// A write through the envelope lands in pi's file, and comes back in the same
// report — the pane never has to re-read to see its own change.
func TestCLIKeysWritesThroughTheEnvelope(t *testing.T) {
	home := t.TempDir()
	old := pipkg.UserDir
	dir := filepath.Join(home, ".pi", "agent")
	pipkg.UserDir = func() string { return dir }
	t.Cleanup(func() { pipkg.UserDir = old })
	ts := newTestServer(t, "cat")

	body, _ := json.Marshal(map[string]any{"cli": "pi", "action": "tui.editor.undo", "keys": []string{"ctrl+alt+z"}})
	req, _ := http.NewRequest(http.MethodPut, ts.URL+"/api/cli-keys", bytes.NewReader(body))
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
	if got := rep.User["tui.editor.undo"]; len(got) != 1 || got[0] != "ctrl+alt+z" {
		t.Fatalf("the report after a write: %v", rep.User)
	}

	// resetAll is the pane's Reset all, through the same door.
	body, _ = json.Marshal(map[string]any{"cli": "pi", "resetAll": true})
	req, _ = http.NewRequest(http.MethodPut, ts.URL+"/api/cli-keys", bytes.NewReader(body))
	res, err = ts.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var after struct {
		User map[string][]string `json:"user"`
	}
	if err := json.NewDecoder(res.Body).Decode(&after); err != nil {
		t.Fatal(err)
	}
	if len(after.User) != 0 {
		t.Fatalf("reset all left %v", after.User)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "keybindings.json"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(raw, []byte("ctrl+alt+z")) {
		t.Fatalf("the override survived reset all: %s", raw)
	}
}

// A CLI whose editor has not shipped answers with its state and nothing else,
// and a write to it is refused by name — never silently ignored.
func TestCLIKeysRefusesWhatPiCodeCannotWrite(t *testing.T) {
	home := t.TempDir()
	old := pipkg.UserDir
	pipkg.UserDir = func() string { return filepath.Join(home, ".pi", "agent") }
	t.Cleanup(func() { pipkg.UserDir = old })
	ts := newTestServer(t, "cat")

	for _, tc := range []struct {
		cli, state, wantIn string
	}{
		{"codex", "planned", "cannot write"},
		{"grok", "refused", "does not allow"},
		{"muse", "refused", "does not allow"},
		{"omp", "planned", "cannot write"},
	} {
		res, err := ts.Client().Get(ts.URL + "/api/cli-keys?cli=" + tc.cli)
		if err != nil {
			t.Fatal(err)
		}
		var rep struct {
			State    string `json:"state"`
			Writable bool   `json:"writable"`
			File     string `json:"file"`
			Actions  []any  `json:"actions"`
		}
		if err := json.NewDecoder(res.Body).Decode(&rep); err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if rep.State != tc.state || rep.Writable {
			t.Fatalf("%s: state=%q writable=%v", tc.cli, rep.State, rep.Writable)
		}
		// No file and no catalog: a planned CLI has nothing PiCode can name yet.
		if rep.File != "" || len(rep.Actions) != 0 {
			t.Fatalf("%s: file=%q actions=%d", tc.cli, rep.File, len(rep.Actions))
		}
		body, _ := json.Marshal(map[string]any{"cli": tc.cli, "action": "whatever", "keys": []string{"ctrl+q"}})
		req, _ := http.NewRequest(http.MethodPut, ts.URL+"/api/cli-keys", bytes.NewReader(body))
		put, err := ts.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		var e struct {
			Error string `json:"error"`
		}
		if err := json.NewDecoder(put.Body).Decode(&e); err != nil {
			t.Fatal(err)
		}
		put.Body.Close()
		if put.StatusCode != http.StatusBadRequest {
			t.Fatalf("%s: PUT %d", tc.cli, put.StatusCode)
		}
		if !bytes.Contains([]byte(e.Error), []byte(tc.wantIn)) {
			t.Fatalf("%s: error %q does not say %q", tc.cli, e.Error, tc.wantIn)
		}
	}

	res, err := ts.Client().Get(ts.URL + "/api/cli-keys?cli=nonesuch")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("unknown CLI: %d", res.StatusCode)
	}
	// No cli at all is a bad request too: the endpoint never guesses, because a
	// guessed CLI would write another vendor's file.
	res2, err := ts.Client().Get(ts.URL + "/api/cli-keys")
	if err != nil {
		t.Fatal(err)
	}
	defer res2.Body.Close()
	if res2.StatusCode != http.StatusBadRequest {
		t.Fatalf("missing cli: %d", res2.StatusCode)
	}
}
