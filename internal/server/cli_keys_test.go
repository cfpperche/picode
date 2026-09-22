package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/cfpperche/picode/internal/clikeys"
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

// A CLI the flat engine writes is served through the same envelope pi answers
// on: its own file, its own catalog, its own platform vocabulary, and a
// revision a write checks so a file that moved is refused rather than
// overwritten (ADR-0174).
func TestCLIKeysServesAndWritesAGuestMap(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PI_CONFIG_DIR", root)
	t.Setenv("OMP_PROFILE", "")
	ts := newTestServer(t, "cat")
	get := func() map[string]any {
		t.Helper()
		res, err := ts.Client().Get(ts.URL + "/api/cli-keys?cli=omp")
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK {
			t.Fatalf("GET %d", res.StatusCode)
		}
		var out map[string]any
		if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
			t.Fatal(err)
		}
		return out
	}
	put := func(body map[string]any) int {
		t.Helper()
		raw, _ := json.Marshal(body)
		req, _ := http.NewRequest(http.MethodPut, ts.URL+"/api/cli-keys", bytes.NewReader(raw))
		res, err := ts.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		return res.StatusCode
	}
	path := filepath.Join(root, "agent", "keybindings.yml")
	first := get()
	if first["state"] != "shipped" || first["writable"] != true || first["keymap"] != "flat" || first["pickup"] != "restart" {
		t.Fatalf("the envelope is not the CLI's: %v", first)
	}
	if first["file"] != path || first["exists"] != false {
		t.Fatalf("file=%v exists=%v", first["file"], first["exists"])
	}
	if rows, _ := first["actions"].([]any); len(rows) != 70 {
		t.Fatalf("the catalog did not survive the envelope: %d rows", len(rows))
	}
	if first["platform"] != "linux" && first["platform"] != "win32" && first["platform"] != "darwin" {
		t.Fatalf("the platform is the CLI's own vocabulary, got %v", first["platform"])
	}
	if code := put(map[string]any{"cli": "omp", "action": "app.exit", "keys": []string{"ctrl+q"}}); code != http.StatusOK {
		t.Fatalf("PUT %d", code)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(raw, []byte(`app.exit: ["ctrl+q"]`)) {
		t.Fatalf("the file says:\n%s", raw)
	}
	second := get()
	user, _ := second["user"].(map[string]any)
	if got, _ := user["app.exit"].([]any); len(got) != 1 || got[0] != "ctrl+q" {
		t.Fatalf("the row was not read back: %v", second["user"])
	}
	revision, _ := second["revision"].(string)
	if revision == "" {
		t.Fatal("a file that exists carries a revision")
	}
	// A file that moved under the editor is a conflict, not an overwrite.
	if err := os.WriteFile(path, []byte("app.exit: ctrl+d\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if code := put(map[string]any{"cli": "omp", "action": "app.exit", "keys": []string{"ctrl+w"}, "revision": revision}); code != http.StatusConflict {
		t.Fatalf("a stale revision must be a 409, got %d", code)
	}
	// With the revision the pane now holds, reset hands the row back.
	now, _ := get()["revision"].(string)
	if code := put(map[string]any{"cli": "omp", "action": "app.exit", "reset": true, "revision": now}); code != http.StatusOK {
		t.Fatalf("reset %d", code)
	}
	if bytes.Contains(mustRead(t, path), []byte("app.exit")) {
		t.Fatalf("the row survived a reset:\n%s", mustRead(t, path))
	}
	// And a row the catalog does not know is refused by name.
	if code := put(map[string]any{"cli": "omp", "action": "app.nonesuch", "keys": []string{"ctrl+q"}}); code != http.StatusBadRequest {
		t.Fatalf("an unknown action must be refused, got %d", code)
	}
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// A nested map (codex) is served through the same envelope: rows addressed by
// `context.action`, a write that renders the pane's captured chord into the
// file's own vocabulary, and the settings key beside it left alone.
func TestCLIKeysServesAndWritesANestedGuestMap(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".codex"), 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(home, ".codex", "config.toml")
	if err := os.WriteFile(path, []byte("model = \"gpt-5\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ts := newTestServer(t, "cat")
	get := func() map[string]any {
		res, err := ts.Client().Get(ts.URL + "/api/cli-keys?cli=codex")
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		var out map[string]any
		if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
			t.Fatal(err)
		}
		return out
	}
	first := get()
	if first["state"] != "shipped" || first["keymap"] != "nested" || first["file"] != path {
		t.Fatalf("the envelope is not codex's: %v", first)
	}
	if rows, _ := first["actions"].([]any); len(rows) != 149 {
		t.Fatalf("the catalog did not survive the envelope: %d rows", len(rows))
	}
	if ctxs, _ := first["contexts"].([]any); len(ctxs) != 12 {
		t.Fatalf("the contexts did not survive: %d", len(ctxs))
	}
	body, _ := json.Marshal(map[string]any{"cli": "codex", "action": "composer.submit", "keys": []string{"ctrl+alt+m"}})
	req, _ := http.NewRequest(http.MethodPut, ts.URL+"/api/cli-keys", bytes.NewReader(body))
	res, err := ts.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("PUT %d", res.StatusCode)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(raw, []byte("[tui.keymap.composer]")) || !bytes.Contains(raw, []byte(`submit = ["ctrl-alt-m"]`)) {
		t.Fatalf("the row did not land in codex's own spelling:\n%s", raw)
	}
	if !bytes.Contains(raw, []byte(`model = "gpt-5"`)) {
		t.Fatalf("the settings key did not survive:\n%s", raw)
	}
	// A chord codex cannot express is refused, and the file is untouched.
	bad, _ := json.Marshal(map[string]any{"cli": "codex", "action": "composer.submit", "keys": []string{"super+m"}})
	req, _ = http.NewRequest(http.MethodPut, ts.URL+"/api/cli-keys", bytes.NewReader(bad))
	res, err = ts.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	var e struct {
		Error string `json:"error"`
	}
	json.NewDecoder(res.Body).Decode(&e)
	res.Body.Close()
	if res.StatusCode != http.StatusBadRequest || !bytes.Contains([]byte(e.Error), []byte("super")) {
		t.Fatalf("a chord codex cannot express must be refused by name: %d %q", res.StatusCode, e.Error)
	}
}

// A third shape through the same envelope: OpenCode's keybinds object, whose
// rows live at keybinds.<action> in the user's tui.json.
func TestCLIKeysServesAndWritesTheFlatGuestMap(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".config", "opencode"), 0o755); err != nil {
		t.Fatal(err)
	}
	ts := newTestServer(t, "cat")
	res, err := ts.Client().Get(ts.URL + "/api/cli-keys?cli=opencode")
	if err != nil {
		t.Fatal(err)
	}
	var first struct {
		State    string           `json:"state"`
		Keymap   string           `json:"keymap"`
		Writable bool             `json:"writable"`
		File     string           `json:"file"`
		Exists   bool             `json:"exists"`
		Actions  []map[string]any `json:"actions"`
	}
	if err := json.NewDecoder(res.Body).Decode(&first); err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if first.State != "shipped" || !first.Writable || first.Keymap != "flat" || first.File == "" || first.Exists {
		t.Fatalf("the envelope is not opencode's: %+v", first)
	}
	if len(first.Actions) != 162 {
		t.Fatalf("the catalog did not survive the envelope: %d rows", len(first.Actions))
	}
	body, _ := json.Marshal(map[string]any{"cli": "opencode", "action": "command_list", "keys": []string{"ctrl+alt+p"}})
	req, _ := http.NewRequest(http.MethodPut, ts.URL+"/api/cli-keys", bytes.NewReader(body))
	res, err = ts.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("PUT %d", res.StatusCode)
	}
	raw, err := os.ReadFile(first.File)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(raw, []byte(`"command_list": ["ctrl+alt+p"]`)) {
		t.Fatalf("the row did not land in the tui config:\n%s", raw)
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

	// Every CLI whose editor has not shipped, not a sample of them: the answer
	// is per row, and the refusal says which of the three it is — a vendor that
	// does not allow remapping, a CLI with no key map file at all (the few keys
	// it allows are rows in Settings), or an adapter not written yet.
	for _, cli := range clikeys.Registry {
		if cli.State == clikeys.Shipped {
			continue
		}
		wantIn := "cannot write"
		switch {
		case cli.State == clikeys.Refused:
			wantIn = "does not allow"
		case cli.Keymap == clikeys.Partial:
			wantIn = "keeps no key map file"
		}
		tc := struct{ cli, state, wantIn string }{cli.ID, string(cli.State), wantIn}
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
