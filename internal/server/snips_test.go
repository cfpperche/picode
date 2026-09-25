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
	"github.com/cfpperche/picode/internal/feed"
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

// The shell door refuses to type into a repository another PiCode agent or
// terminal is writing in — the rule Run command… in a terminal already
// follows. The preview stays allowed: it types nothing, and the sheet has to
// show the command before a human can confirm it.
// What counts as "a snippet ran" for the feed is a boundary, not an accident:
// the notice is published after the run route has identified a snippet, so a
// request for an id that does not exist (or a malformed body) is not a run and
// stays out of the feed, while a snippet that ran and failed — here because its
// target is gone — is published with ok:false. Pinned because the defer that
// publishes it sits after those early returns, which reads like an oversight
// until a test says otherwise.
func TestSnipRanFeedBoundary(t *testing.T) {
	st := testStore(t)
	f := &feed.Feed{Store: st}
	var got []string
	f.Listen(func(ev store.Event) {
		if ev.Type == "snip.ran" {
			got = append(got, string(ev.Data))
		}
	})
	ts := httptest.NewServer(New("127.0.0.1:0", Deps{Store: st, Feed: f}).Handler)
	t.Cleanup(ts.Close)

	p, err := st.CreateSnip(store.SnipParams{Title: "Ask", Body: "look at {{x}}"})
	if err != nil {
		t.Fatal(err)
	}

	// A snippet that does not exist: no run, no notice.
	code, _ := snipJSON(t, ts.URL, http.MethodPost, "/api/snips/nope/run", map[string]any{
		"target": map[string]string{"type": "agent", "id": "a1"},
	})
	if code != http.StatusNotFound {
		t.Fatalf("missing snippet = %d, want 404", code)
	}
	if len(got) != 0 {
		t.Fatalf("the feed logged a run for a snippet that does not exist: %v", got)
	}

	// A snippet that ran and could not deliver: published, with ok:false.
	code, _ = snipJSON(t, ts.URL, http.MethodPost, "/api/snips/"+p.ID+"/run", map[string]any{
		"target": map[string]string{"type": "agent", "id": "gone"},
	})
	if code == http.StatusOK {
		t.Fatalf("run into a missing agent = %d, want a failure", code)
	}
	if len(got) != 1 {
		t.Fatalf("feed notices = %v, want exactly one", got)
	}
	if !strings.Contains(got[0], `"ok":false`) || !strings.Contains(got[0], p.ID) {
		t.Fatalf("notice = %s, want ok:false with the snippet id", got[0])
	}
}

func TestSnipRunShellRefusesBusyRepository(t *testing.T) {
	t.Cleanup(resetPromptInFlight)
	repo := gitRepo(t)
	st := testStore(t)
	ws, agent, err := storeWorkspaceWithAgent(st, "App", repo)
	if err != nil {
		t.Fatal(err)
	}
	term, err := st.CreateTerminalIn(ws.ID, "QA", repo)
	if err != nil {
		t.Fatal(err)
	}
	p, err := st.CreateSnip(store.SnipParams{Title: "Deploy", Kind: "shell", Body: "echo hi in {{cwd}}"})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(New("127.0.0.1:0", Deps{
		Store: st, Tmux: tmux.New(), Runtime: rpc.NewRuntime("cat", st, nil), AgentCmd: "cat",
		TermRuntimes: NewTermRuntimes(),
	}).Handler)
	t.Cleanup(ts.Close)
	paste := fakeTmuxBin(t)
	swapProbes(t, map[string]string{}, map[string]string{agent.ID: "mid-turn"})

	send := func(extra map[string]any) (int, map[string]any) {
		t.Helper()
		req := map[string]any{"target": map[string]string{"type": "terminal", "id": term.ID}, "confirm": true}
		for k, v := range extra {
			req[k] = v
		}
		return snipJSON(t, ts.URL, http.MethodPost, "/api/snips/"+p.ID+"/run", req)
	}

	code, out := send(map[string]any{"preview": true})
	if code != http.StatusOK || out["preview"] != true {
		t.Fatalf("preview while busy = %d %v, want 200 preview", code, out)
	}
	if raw, err := os.ReadFile(paste); err == nil && len(raw) > 0 {
		t.Fatalf("the preview delivered: %q", raw)
	}

	code, out = send(nil)
	if code != http.StatusConflict || out["reason"] != "busy" {
		t.Fatalf("busy repository = %d %v, want 409 busy", code, out)
	}
	busy, _ := out["busy"].([]any)
	if len(busy) != 1 || busy[0].(map[string]any)["id"] != agent.ID || busy[0].(map[string]any)["kind"] != "agent" {
		t.Fatalf("busy list = %v, want the agent", busy)
	}
	if msg, _ := out["error"].(string); !strings.HasSuffix(msg, " is mid-turn in this repository.") {
		t.Fatalf("busy message = %q", msg)
	}
	if raw, err := os.ReadFile(paste); err == nil && len(raw) > 0 {
		t.Fatalf("the command was delivered into a busy repository: %q", raw)
	}
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

func TestSnipRunIntoInteractiveAgent(t *testing.T) {
	manager := tmux.New()
	if !manager.Available() {
		t.Skip("tmux missing")
	}
	st := testStore(t)
	deps := Deps{
		Store: st, Tmux: manager, DataDir: t.TempDir(),
		Runtime: rpc.NewRuntime("cat", st, nil), AgentCmd: "cat",
		Replies: NewTuiReplies(),
	}
	ts := httptest.NewServer(New("127.0.0.1:0", deps).Handler)
	t.Cleanup(ts.Close)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	dir := t.TempDir()
	_, agent, err := storeWorkspaceWithAgent(st, "App", dir)
	if err != nil {
		t.Fatal(err)
	}
	name := tmux.SessionName(agent.ID)
	if err := manager.NewSessionEnv(context.Background(), name, dir, nil, "/bin/sh", "-c", "sleep 300"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.KillSession(context.Background(), name) })
	if err := st.SetAgentRuntimeMode(agent.ID, store.StatusRunning, "interactive"); err != nil {
		t.Fatal(err)
	}
	p, err := st.CreateSnip(store.SnipParams{Title: "Look", Body: "Look at {{pr}}"})
	if err != nil {
		t.Fatal(err)
	}

	code, out := snipJSON(t, ts.URL, http.MethodPost, "/api/snips/"+p.ID+"/run", map[string]any{
		"target": map[string]string{"type": "agent", "id": agent.ID}, "values": map[string]string{"pr": "9"},
	})
	if code != http.StatusOK || out["typed"] != true {
		t.Fatalf("interactive run = %d %v", code, out)
	}
	if text, _ := out["text"].(string); !strings.Contains(text, "Look at 9") {
		t.Fatalf("text = %v", out["text"])
	}
	pane, err := manager.CaptureTail(context.Background(), name, 20)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(pane, "Look at 9") {
		t.Fatalf("pane after paste = %q", pane)
	}

	shell, err := st.CreateSnip(store.SnipParams{Title: "Echo", Kind: "shell", Body: "echo hi"})
	if err != nil {
		t.Fatal(err)
	}
	code, out = snipJSON(t, ts.URL, http.MethodPost, "/api/snips/"+shell.ID+"/run", map[string]any{
		"target": map[string]string{"type": "agent", "id": agent.ID}, "confirm": true,
	})
	if code != http.StatusConflict || out["reason"] != "kind" {
		t.Fatalf("shell into agent = %d %v", code, out)
	}

	deps.Replies.mu.Lock()
	deps.Replies.active[agent.ID] = true
	deps.Replies.mu.Unlock()
	code, out = snipJSON(t, ts.URL, http.MethodPost, "/api/snips/"+p.ID+"/run", map[string]any{
		"target": map[string]string{"type": "agent", "id": agent.ID}, "values": map[string]string{"pr": "1"},
	})
	if code != http.StatusConflict || out["reason"] != "busy" {
		t.Fatalf("busy = %d %v", code, out)
	}
	deps.Replies.mu.Lock()
	delete(deps.Replies.active, agent.ID)
	deps.Replies.mu.Unlock()

	_ = manager.KillSession(context.Background(), name)
	code, out = snipJSON(t, ts.URL, http.MethodPost, "/api/snips/"+p.ID+"/run", map[string]any{
		"target": map[string]string{"type": "agent", "id": agent.ID}, "values": map[string]string{"pr": "1"},
	})
	if code != http.StatusConflict || out["reason"] != "stopped" {
		t.Fatalf("stopped after kill = %d %v", code, out)
	}
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
		"target": map[string]string{"type": "agent", "id": wk.Agents[0].ID}, "values": map[string]string{"pr": "9"},
	})
	if code != http.StatusConflict || out["reason"] != "stopped" {
		t.Fatalf("stopped run = %d %v", code, out)
	}

	code, out = snipJSON(t, ts.URL, http.MethodPost, "/api/snips/"+id+"/run", map[string]any{
		"target": map[string]string{"type": "agent", "id": wk.Agents[0].ID}, "values": map[string]string{},
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

// GET /api/snips/slug/{slug} (snippets v2, F8): the editor asks while the
// reader types, so the answer must be the one a save would reach — free is
// 204, taken is 409 — and a snippet's own slug is free for itself.
func TestSnipSlugAvailability(t *testing.T) {
	ts, _, _ := cleanupServer(t)

	code, out := snipJSON(t, ts.URL, http.MethodPost, "/api/snips", map[string]any{
		"title": "Review PR", "body": "Look at {{pr}}",
	})
	if code != http.StatusOK {
		t.Fatalf("create = %d %v", code, out)
	}
	id, _ := out["id"].(string)

	free := func(path string) int {
		t.Helper()
		req, _ := http.NewRequest(http.MethodGet, ts.URL+path, nil)
		res := do(t, http.DefaultClient, req)
		defer res.Body.Close()
		return res.StatusCode
	}

	if got := free("/api/snips/slug/brand-new"); got != http.StatusNoContent {
		t.Fatalf("free slug = %d, want 204", got)
	}
	if got := free("/api/snips/slug/review-pr"); got != http.StatusConflict {
		t.Fatalf("taken slug = %d, want 409", got)
	}
	if got := free("/api/snips/slug/Review%20PR"); got != http.StatusConflict {
		t.Fatalf("normalized taken slug = %d, want 409", got)
	}
	if got := free("/api/snips/slug/review-pr?except=" + id); got != http.StatusNoContent {
		t.Fatalf("own slug = %d, want 204", got)
	}
	// The check must not be eaten by GET /api/snips/{id}: that route would
	// answer "not found" for a slug segment.
	if got := free("/api/snips/slug/review-pr"); got == http.StatusNotFound {
		t.Fatal("the slug route was matched by /api/snips/{id}")
	}
}

// Three layers now guard a snips body, and the test has to tell them apart:
// under the limit it is stored, a body past the store's 100 KB is refused by
// the store ("body is too long"), and a request past the decode cap is refused
// before validation ever sees it ("invalid JSON body"). Without the cap the
// last row is a 400 too — same status, different layer — so the message is the
// evidence that the cap is what stopped it.
func TestSnipDecodeIsCapped(t *testing.T) {
	ts, _, _ := cleanupServer(t)

	code, out := snipJSON(t, ts.URL, http.MethodPost, "/api/snips", map[string]any{
		"title": "Existing", "body": "one",
	})
	if code != http.StatusOK {
		t.Fatalf("create = %d %v", code, out)
	}
	id, _ := out["id"].(string)

	post := func(method, path, body string) (int, map[string]any) {
		t.Helper()
		req, _ := http.NewRequest(method, ts.URL+path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		res := do(t, http.DefaultClient, req)
		defer res.Body.Close()
		var parsed map[string]any
		_ = json.NewDecoder(res.Body).Decode(&parsed)
		return res.StatusCode, parsed
	}
	bodyOf := func(n int) string {
		return `{"title":"sized","body":"` + strings.Repeat("x", n) + `"}`
	}

	// Just past the store's limit: the store refuses it, and says so.
	code, out = post(http.MethodPost, "/api/snips", bodyOf(100_001))
	if code != http.StatusBadRequest || !strings.Contains(errText(out), "body is too long") {
		t.Fatalf("past the store limit = %d %v, want 400 body is too long", code, out)
	}
	// Past the decode cap: refused before validation, so the message is the
	// decoder's, not the store's.
	code, out = post(http.MethodPost, "/api/snips", bodyOf(snipDecodeLimit))
	if code != http.StatusBadRequest || !strings.Contains(errText(out), "invalid JSON body") {
		t.Fatalf("past the cap = %d %v, want 400 invalid JSON body", code, out)
	}
	code, out = post(http.MethodPatch, "/api/snips/"+id, bodyOf(snipDecodeLimit))
	if code != http.StatusBadRequest || !strings.Contains(errText(out), "invalid JSON body") {
		t.Fatalf("oversized PATCH = %d %v, want 400 invalid JSON body", code, out)
	}

	// Nothing was written by any refused request.
	code, out = snipJSON(t, ts.URL, http.MethodGet, "/api/snips", nil)
	if code != http.StatusOK {
		t.Fatalf("list = %d %v", code, out)
	}
	list, _ := out["snips"].([]any)
	if len(list) != 1 {
		t.Fatalf("after refused requests the store holds %d snippets, want 1", len(list))
	}
	if first, _ := list[0].(map[string]any); first["title"] != "Existing" {
		t.Fatalf("the existing snippet changed: %v", first)
	}
}

func errText(out map[string]any) string {
	s, _ := out["error"].(string)
	return s
}
