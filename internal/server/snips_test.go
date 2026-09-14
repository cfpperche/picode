package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/clilaunch"
	"github.com/cfpperche/picode/internal/rpc"
	"github.com/cfpperche/picode/internal/snips"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

func snipJSON(t *testing.T, tsURL, method, path string, body any) (int, map[string]any) {
	t.Helper()
	var rdr *bytes.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		rdr = bytes.NewReader(raw)
	} else {
		rdr = bytes.NewReader(nil)
	}
	req, _ := http.NewRequest(method, tsURL+path, rdr)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res := do(t, http.DefaultClient, req)
	defer res.Body.Close()
	out := map[string]any{}
	_ = json.NewDecoder(res.Body).Decode(&out)
	return res.StatusCode, out
}

func TestSnipsHTTP(t *testing.T) {
	ts, _, _ := cleanupServer(t)
	code, out := snipJSON(t, ts.URL, http.MethodPost, "/api/snips", map[string]any{
		"title": "Review PR", "body": "Look at {{pr}}",
	})
	if code != http.StatusOK {
		t.Fatalf("create = %d %v", code, out)
	}
	id := out["id"].(string)
	if out["slug"] != "review-pr" {
		t.Fatalf("slug = %v", out["slug"])
	}

	code, list := snipJSON(t, ts.URL, http.MethodGet, "/api/snips", nil)
	if code != http.StatusOK {
		t.Fatalf("list = %d", code)
	}
	raw, _ := json.Marshal(list)
	if strings.Contains(string(raw), `"body":`) {
		t.Fatalf("list body: %s", raw)
	}

	code, pick := snipJSON(t, ts.URL, http.MethodGet, "/api/snips/picker", nil)
	if code != http.StatusOK {
		t.Fatalf("picker = %d %v", code, pick)
	}
	snips, _ := pick["snips"].([]any)
	if len(snips) != 1 {
		t.Fatalf("picker rows = %v", pick)
	}

	code, got := snipJSON(t, ts.URL, http.MethodGet, "/api/snips/"+id, nil)
	if code != http.StatusOK || got["body"] != "Look at {{pr}}" {
		t.Fatalf("get = %d %v", code, got)
	}

	stale := out["updatedAt"].(string)
	code, _ = snipJSON(t, ts.URL, http.MethodPatch, "/api/snips/"+id, map[string]any{
		"title": "Review PR", "body": "v2", "ifUpdatedAt": stale,
	})
	if code != http.StatusOK {
		t.Fatalf("patch = %d", code)
	}
	code, _ = snipJSON(t, ts.URL, http.MethodPatch, "/api/snips/"+id, map[string]any{
		"title": "Review PR", "body": "v3", "ifUpdatedAt": stale,
	})
	if code != http.StatusConflict {
		t.Fatalf("stale patch = %d", code)
	}

	code, _ = snipJSON(t, ts.URL, http.MethodPost, "/api/snips", map[string]any{"title": "Review PR", "body": "x"})
	if code != http.StatusBadRequest {
		t.Fatalf("dup slug = %d", code)
	}

	code, _ = snipJSON(t, ts.URL, http.MethodPost, "/api/snips/"+id+"/starred", map[string]any{"starred": true})
	if code != http.StatusOK {
		t.Fatalf("star = %d", code)
	}
	code, _ = snipJSON(t, ts.URL, http.MethodPost, "/api/snips/"+id+"/archived", map[string]any{"archived": true})
	if code != http.StatusOK {
		t.Fatalf("archive = %d", code)
	}
	code, pick = snipJSON(t, ts.URL, http.MethodGet, "/api/snips/picker", nil)
	if code != http.StatusOK {
		t.Fatal(code)
	}
	if rows, _ := pick["snips"].([]any); len(rows) != 0 {
		t.Fatalf("archived still in picker: %v", pick)
	}

	req, _ := http.NewRequest(http.MethodDelete, ts.URL+"/api/snips/"+id, nil)
	res := do(t, http.DefaultClient, req)
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("delete = %d", res.StatusCode)
	}
	code, _ = snipJSON(t, ts.URL, http.MethodGet, "/api/snips/"+id, nil)
	if code != http.StatusNotFound {
		t.Fatalf("get deleted = %d", code)
	}
}

func TestSnipsPickerIsNotAnID(t *testing.T) {
	ts, _, _ := cleanupServer(t)
	code, _ := snipJSON(t, ts.URL, http.MethodGet, "/api/snips/picker", nil)
	if code != http.StatusOK {
		t.Fatalf("picker treated as id: %d", code)
	}
}

func TestSnipRunShellIntoTerminal(t *testing.T) {
	newHarness := func(t *testing.T) (*store.Store, *httptest.Server, *TermRuntimes, store.Terminal, store.Snip) {
		t.Helper()
		t.Cleanup(resetPromptInFlight)
		cwd := t.TempDir()
		st := testStore(t)
		runtimes := NewTermRuntimes()
		ts := httptest.NewServer(New("127.0.0.1:0", Deps{
			Store: st, Tmux: tmux.New(), Runtime: rpc.NewRuntime("cat", st, nil), AgentCmd: "cat",
			TermRuntimes: runtimes,
		}).Handler)
		t.Cleanup(ts.Close)
		term, err := st.CreateTerminal("sh", cwd)
		if err != nil {
			t.Fatal(err)
		}
		p, err := st.CreateSnip(store.SnipParams{Title: "Deploy", Kind: "shell", Body: "echo deploy {{env=dev}} in {{cwd}}"})
		if err != nil {
			t.Fatal(err)
		}
		return st, ts, runtimes, term, p
	}
	stubPane := func(t *testing.T, cmd string) {
		t.Helper()
		orig := paneCommandFn
		paneCommandFn = func(context.Context, Deps, string) string { return cmd }
		t.Cleanup(func() { paneCommandFn = orig })
	}
	run := func(t *testing.T, ts *httptest.Server, p store.Snip, target map[string]string, values map[string]string, confirm bool) (int, map[string]any) {
		return snipJSON(t, ts.URL, http.MethodPost, "/api/snips/"+p.ID+"/run", map[string]any{
			"target": target, "values": values, "confirm": confirm,
		})
	}

	t.Run("confirm is required", func(t *testing.T) {
		_, ts, _, term, p := newHarness(t)
		fakeTmuxBin(t)
		stubPane(t, "bash")
		code, out := run(t, ts, p, map[string]string{"type": "terminal", "id": term.ID}, nil, false)
		if code != http.StatusBadRequest || !strings.Contains(out["error"].(string), "confirm") {
			t.Fatalf("no confirm = %d %v", code, out)
		}
	})

	t.Run("agent target refused as kind", func(t *testing.T) {
		_, ts, _, _, p := newHarness(t)
		code, out := run(t, ts, p, map[string]string{"type": "agent", "id": "a"}, nil, true)
		if code != http.StatusConflict || out["reason"] != "kind" {
			t.Fatalf("agent = %d %v", code, out)
		}
	})

	t.Run("live CLI lease refuses", func(t *testing.T) {
		_, ts, runtimes, term, p := newHarness(t)
		fakeTmuxBin(t)
		stubPane(t, "sh")
		runtimes.Start(term.ID, TermRuntime{CLI: "pi", RunID: "r-1", Source: "wrapper"})
		code, out := run(t, ts, p, map[string]string{"type": "terminal", "id": term.ID}, nil, true)
		if code != http.StatusConflict || out["reason"] != "cli" {
			t.Fatalf("lease = %d %v", code, out)
		}
	})

	t.Run("non-shell foreground refuses", func(t *testing.T) {
		_, ts, _, term, p := newHarness(t)
		fakeTmuxBin(t)
		stubPane(t, "vim")
		code, out := run(t, ts, p, map[string]string{"type": "terminal", "id": term.ID}, nil, true)
		if code != http.StatusConflict || out["reason"] != "foreground" {
			t.Fatalf("vim = %d %v", code, out)
		}
	})

	t.Run("preview expands live context without delivering", func(t *testing.T) {
		paste := fakeTmuxBin(t)
		_, ts, _, term, p := newHarness(t)
		stubPane(t, "bash")
		code, out := snipJSON(t, ts.URL, http.MethodPost, "/api/snips/"+p.ID+"/run", map[string]any{
			"target": map[string]string{"type": "terminal", "id": term.ID}, "values": map[string]string{"env": "stg"}, "confirm": true, "preview": true,
		})
		if code != http.StatusOK || out["preview"] != true {
			t.Fatalf("preview = %d %v", code, out)
		}
		if !strings.Contains(out["text"].(string), "echo deploy stg in "+term.Cwd) {
			t.Fatalf("preview text = %v", out["text"])
		}
		if raw, err := os.ReadFile(paste); err == nil && len(raw) > 0 {
			t.Fatalf("preview delivered: %q", raw)
		}
	})

	t.Run("preview without confirm is 400", func(t *testing.T) {
		_, ts, _, term, p := newHarness(t)
		code, _ := snipJSON(t, ts.URL, http.MethodPost, "/api/snips/"+p.ID+"/run", map[string]any{
			"target": map[string]string{"type": "terminal", "id": term.ID}, "preview": true,
		})
		if code != http.StatusBadRequest {
			t.Fatalf("preview no confirm = %d", code)
		}
	})

	t.Run("preview on prompt kind is 400", func(t *testing.T) {
		st, ts, _, _, _ := newHarness(t)
		pp, err := st.CreateSnip(store.SnipParams{Title: "Ask", Body: "look {{x}}"})
		if err != nil {
			t.Fatal(err)
		}
		code, _ := snipJSON(t, ts.URL, http.MethodPost, "/api/snips/"+pp.ID+"/run", map[string]any{
			"target": map[string]string{"type": "agent", "id": "a"}, "values": map[string]string{"x": "1"}, "preview": true,
		})
		if code != http.StatusBadRequest {
			t.Fatalf("prompt preview = %d", code)
		}
	})

	t.Run("branch resolves from the live folder", func(t *testing.T) {
		st, ts, _, term, _ := newHarness(t)
		fakeTmuxBin(t)
		stubPane(t, "bash")
		cwd := term.Cwd
		if out, err := exec.Command("git", "init", "-q", cwd).CombinedOutput(); err != nil {
			t.Fatalf("git init: %v %s", err, out)
		}
		want, _ := exec.Command("git", "-C", cwd, "branch", "--show-current").Output()
		p, err := st.CreateSnip(store.SnipParams{Title: "Where", Kind: "shell", Body: "on {{branch}}"})
		if err != nil {
			t.Fatal(err)
		}
		code, out := snipJSON(t, ts.URL, http.MethodPost, "/api/snips/"+p.ID+"/run", map[string]any{
			"target": map[string]string{"type": "terminal", "id": term.ID}, "confirm": true, "preview": true,
		})
		if code != http.StatusOK {
			t.Fatalf("branch preview = %d %v", code, out)
		}
		if out["text"] != "on "+strings.TrimSpace(string(want)) {
			t.Fatalf("text = %v want branch %q", out["text"], string(want))
		}
	})

	t.Run("enum mismatch is 400", func(t *testing.T) {
		st, ts, _, term, _ := newHarness(t)
		fakeTmuxBin(t)
		stubPane(t, "bash")
		p, err := st.CreateSnip(store.SnipParams{
			Title: "Ship", Kind: "shell", Body: "echo {{env}}",
			Placeholders: []snips.Placeholder{{Name: "env", Enum: []string{"dev", "prod"}}},
		})
		if err != nil {
			t.Fatal(err)
		}
		code, out := snipJSON(t, ts.URL, http.MethodPost, "/api/snips/"+p.ID+"/run", map[string]any{
			"target": map[string]string{"type": "terminal", "id": term.ID}, "values": map[string]string{"env": "staging"}, "confirm": true,
		})
		if code != http.StatusBadRequest || !strings.Contains(out["error"].(string), "allowed choices") {
			t.Fatalf("enum = %d %v", code, out)
		}
	})

	t.Run("runs on a bare shell", func(t *testing.T) {
		paste := fakeTmuxBin(t)
		_, ts, _, term, p := newHarness(t)
		stubPane(t, "bash")
		code, out := run(t, ts, p, map[string]string{"type": "terminal", "id": term.ID}, map[string]string{"env": "prod"}, true)
		if code != http.StatusOK || out["typed"] != true {
			t.Fatalf("run = %d %v", code, out)
		}
		got, err := os.ReadFile(paste)
		if err != nil || !strings.Contains(string(got), "echo deploy prod in "+term.Cwd) {
			t.Fatalf("paste %q %v", got, err)
		}
	})
}

func TestSnipRunIntoTerminal(t *testing.T) {
	newHarness := func(t *testing.T) (*store.Store, *httptest.Server, store.Terminal, store.Snip) {
		t.Helper()
		t.Cleanup(resetPromptInFlight)
		cwd := t.TempDir()
		st := testStore(t)
		ts := httptest.NewServer(New("127.0.0.1:0", Deps{
			Store: st, Tmux: tmux.New(), Runtime: rpc.NewRuntime("cat", st, nil), AgentCmd: "cat",
		}).Handler)
		t.Cleanup(ts.Close)
		term, err := st.CreateTerminal("cli", cwd)
		if err != nil {
			t.Fatal(err)
		}
		if err := st.SetTerminalLaunch(term.ID, "pi", clilaunch.Overrides{}); err != nil {
			t.Fatal(err)
		}
		p, err := st.CreateSnip(store.SnipParams{Title: "Look", Body: "Look at {{pr}} in {{cwd}}"})
		if err != nil {
			t.Fatal(err)
		}
		return st, ts, term, p
	}
	stubPane := func(t *testing.T, cmd string) {
		t.Helper()
		orig := paneCommandFn
		paneCommandFn = func(context.Context, Deps, string) string { return cmd }
		t.Cleanup(func() { paneCommandFn = orig })
	}
	run := func(t *testing.T, ts *httptest.Server, p store.Snip, target map[string]string, values map[string]string) (int, map[string]any) {
		return snipJSON(t, ts.URL, http.MethodPost, "/api/snips/"+p.ID+"/run", map[string]any{
			"target": target, "values": values,
		})
	}

	t.Run("plain shell refused", func(t *testing.T) {
		st, ts, _, p := newHarness(t)
		shell, err := st.CreateTerminal("sh", t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		code, out := run(t, ts, p, map[string]string{"type": "terminal", "id": shell.ID}, map[string]string{"pr": "1"})
		if code != http.StatusConflict || out["reason"] != "cli" {
			t.Fatalf("plain shell = %d %v", code, out)
		}
	})

	t.Run("cli exited leaves a shell: refused as kind (B6b)", func(t *testing.T) {
		_, ts, term, p := newHarness(t)
		fakeTmuxBin(t)
		stubPane(t, "bash")
		code, out := run(t, ts, p, map[string]string{"type": "terminal", "id": term.ID}, map[string]string{"pr": "1"})
		if code != http.StatusConflict || out["reason"] != "kind" {
			t.Fatalf("b6b = %d %v", code, out)
		}
	})

	t.Run("missing field is 400 before any paste", func(t *testing.T) {
		_, ts, term, p := newHarness(t)
		fakeTmuxBin(t)
		stubPane(t, "pi")
		code, out := run(t, ts, p, map[string]string{"type": "terminal", "id": term.ID}, map[string]string{})
		if code != http.StatusBadRequest {
			t.Fatalf("missing = %d %v", code, out)
		}
	})

	t.Run("pane dead is 409 closed", func(t *testing.T) {
		_, ts, term, p := newHarness(t)
		shimTmux(t)
		stubPane(t, "pi")
		code, _ := run(t, ts, p, map[string]string{"type": "terminal", "id": term.ID}, map[string]string{"pr": "1"})
		if code != http.StatusConflict {
			t.Fatalf("dead = %d", code)
		}
	})

	t.Run("send pastes the expanded text", func(t *testing.T) {
		paste := fakeTmuxBin(t)
		_, ts, term, p := newHarness(t)
		stubPane(t, "pi")
		code, out := run(t, ts, p, map[string]string{"type": "terminal", "id": term.ID}, map[string]string{"pr": "9"})
		if code != http.StatusOK || out["typed"] != true {
			t.Fatalf("send = %d %v", code, out)
		}
		got, err := os.ReadFile(paste)
		if err != nil || !strings.Contains(string(got), "Look at 9 in "+term.Cwd) {
			t.Fatalf("paste %q %v", got, err)
		}
	})
}

func TestSnipExpandAndRun(t *testing.T) {
	ts, _, _ := cleanupServer(t)
	code, created := snipJSON(t, ts.URL, http.MethodPost, "/api/snips", map[string]any{
		"title": "Review PR", "body": "Look at {{pr}} in {{cwd}}",
	})
	if code != http.StatusOK {
		t.Fatalf("create = %d %v", code, created)
	}
	id := created["id"].(string)

	code, exp := snipJSON(t, ts.URL, http.MethodPost, "/api/snips/"+id+"/expand", map[string]any{
		"values": map[string]string{"pr": "12"}, "context": map[string]string{"cwd": "/tmp/app"},
	})
	if code != http.StatusOK {
		t.Fatalf("expand = %d %v", code, exp)
	}
	if exp["text"] != "Look at 12 in /tmp/app" {
		t.Fatalf("text = %v", exp["text"])
	}

	code, miss := snipJSON(t, ts.URL, http.MethodPost, "/api/snips/"+id+"/expand", map[string]any{"values": map[string]string{}})
	if code != http.StatusOK {
		t.Fatalf("expand missing = %d", code)
	}
	if raw, _ := json.Marshal(miss["missing"]); !strings.Contains(string(raw), "pr") {
		t.Fatalf("missing = %v", miss["missing"])
	}

	code, shell := snipJSON(t, ts.URL, http.MethodPost, "/api/snips", map[string]any{
		"title": "Echo", "kind": "shell", "body": "echo hi",
	})
	if code != http.StatusOK {
		t.Fatalf("shell create = %d %v", code, shell)
	}
	code, out := snipJSON(t, ts.URL, http.MethodPost, "/api/snips/"+shell["id"].(string)+"/run", map[string]any{
		"target": map[string]string{"type": "agent", "id": "x"}, "values": map[string]string{}, "confirm": true,
	})
	// A command never runs inside an agent — 409 kind, not unimplemented.
	if code != http.StatusConflict || out["reason"] != "kind" {
		t.Fatalf("shell run = %d %v", code, out)
	}

	code, out = snipJSON(t, ts.URL, http.MethodPost, "/api/snips/"+id+"/run", map[string]any{
		"target": map[string]string{"type": "terminal", "id": "t1"}, "values": map[string]string{"pr": "1"},
	})
	// A terminal target now reaches the door; an unknown id is an honest 404.
	if code != http.StatusNotFound {
		t.Fatalf("missing terminal run = %d %v", code, out)
	}
	code, out = snipJSON(t, ts.URL, http.MethodPost, "/api/snips/"+id+"/run", map[string]any{
		"target": map[string]string{"type": "folder", "id": "x"}, "values": map[string]string{"pr": "1"},
	})
	if code != http.StatusBadRequest {
		t.Fatalf("bad target type = %d %v", code, out)
	}

	proj := t.TempDir()
	if err := os.WriteFile(proj+"/a.txt", []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	wk := addWorkspaceWithAgent(t, ts, "App", proj)
	code, out = snipJSON(t, ts.URL, http.MethodPost, "/api/snips/"+id+"/run", map[string]any{
		"target": map[string]string{"type": "agent", "id": wk.Agent.ID}, "values": map[string]string{"pr": "9"},
	})
	if code != http.StatusConflict || out["reason"] != "stopped" {
		t.Fatalf("stopped run = %d %v", code, out)
	}

	code, out = snipJSON(t, ts.URL, http.MethodPost, "/api/snips/"+id+"/run", map[string]any{
		"target": map[string]string{"type": "agent", "id": wk.Agent.ID}, "values": map[string]string{},
	})
	if code != http.StatusBadRequest {
		t.Fatalf("missing run = %d %v", code, out)
	}
}

// GET /api/snips/templates (snippets v2, F6): the starters ship with the
// binary, so the door must answer without a store row — and must not be
// swallowed by GET /api/snips/{id}.
func TestSnipTemplates(t *testing.T) {
	ts, _, _ := cleanupServer(t)

	code, out := snipJSON(t, ts.URL, http.MethodGet, "/api/snips/templates", nil)
	if code != http.StatusOK {
		t.Fatalf("templates: got %d", code)
	}
	items, _ := out["items"].([]any)
	if len(items) < 6 {
		t.Fatalf("expected at least six starters, got %d", len(items))
	}
	for _, raw := range items {
		item, _ := raw.(map[string]any)
		body, _ := item["body"].(string)
		if body == "" {
			t.Fatalf("starter %v has an empty body", item["id"])
		}
		if _, err := snips.Parse(body); err != nil {
			t.Fatalf("starter %v does not parse: %v", item["id"], err)
		}
	}
}
