package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

func TestIsShell(t *testing.T) {
	for cmd, want := range map[string]bool{
		"bash": true, "-zsh": true, "/usr/bin/fish": true, "sh": true, "pwsh": true,
		"vim": false, "sleep": false, "node": false, "git": false, "claude": false, "": false,
	} {
		if got := isShell(cmd); got != want {
			t.Errorf("isShell(%q) = %v, want %v", cmd, got, want)
		}
	}
}

// swapProbes stands in for pi, tmux and the CLI hooks: every pane is at a
// shell unless named otherwise, and no agent is busy unless said so.
func swapProbes(t *testing.T, panes map[string]string, busyAgents map[string]string) {
	t.Helper()
	pane, agent, term := paneCommandFn, agentBusyFn, terminalBusyFn
	paneCommandFn = func(_ context.Context, _ Deps, session string) string {
		if cmd, ok := panes[session]; ok {
			return cmd
		}
		return "bash"
	}
	agentBusyFn = func(_ context.Context, _ Deps, a store.Agent) (bool, string) {
		if why, ok := busyAgents[a.ID]; ok {
			return true, why
		}
		return false, ""
	}
	t.Cleanup(func() { paneCommandFn, agentBusyFn, terminalBusyFn = pane, agent, term })
}

func postRaw(t *testing.T, ts *httptest.Server, path string, body string) (int, map[string]any) {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, ts.URL+path, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	res := do(t, ts.Client(), req)
	var page map[string]any
	_ = json.NewDecoder(res.Body).Decode(&page)
	return res.StatusCode, page
}

// The run route refuses everything the interlock is for: a moved terminal,
// a foreground program in the target pane, an agent mid-turn, another
// terminal working or holding a program — each as a 409 naming who.
func TestTerminalRunRefusals(t *testing.T) {
	repo := gitRepo(t)
	st := testStore(t)
	ts := graphServer(t, st)
	ws, agent, err := storeWorkspaceWithAgent(st, "App", repo)
	if err != nil {
		t.Fatal(err)
	}
	target, err := st.CreateTerminalIn(ws.ID, "QA", repo)
	if err != nil {
		t.Fatal(err)
	}
	other, err := st.CreateTerminalIn(ws.ID, "Other", repo)
	if err != nil {
		t.Fatal(err)
	}
	root := canonDir(repo)
	runPath := "/api/terminals/" + target.ID + "/run"
	ok := `{"text":"git fetch --prune","root":` + jsonString(root) + `}`

	swapProbes(t, map[string]string{}, map[string]string{})
	if code, _ := postRaw(t, ts, runPath, `{"text":"echo hi\n","root":`+jsonString(root)+`}`); code != http.StatusBadRequest {
		t.Fatalf("newline = %d, want 400", code)
	}
	if code, _ := postRaw(t, ts, "/api/terminals/nope/run", ok); code != http.StatusNotFound {
		t.Fatalf("unknown terminal = %d, want 404", code)
	}
	if code, page := postRaw(t, ts, runPath, `{"text":"git fetch --prune"}`); code != http.StatusConflict || page["reason"] != "moved" {
		t.Fatalf("missing root = %d %v, want 409 moved", code, page)
	}
	if code, page := postRaw(t, ts, runPath, `{"text":"git fetch --prune","root":"/somewhere/else"}`); code != http.StatusConflict || page["reason"] != "moved" {
		t.Fatalf("wrong root = %d %v, want 409 moved", code, page)
	}

	swapProbes(t, map[string]string{tmux.ShellSessionName(target.ID): "vim"}, map[string]string{})
	if code, page := postRaw(t, ts, runPath, ok); code != http.StatusConflict || page["reason"] != "foreground" || page["error"] != "This terminal is running vim." {
		t.Fatalf("foreground = %d %v, want 409 running vim", code, page)
	}

	swapProbes(t, map[string]string{}, map[string]string{agent.ID: "mid-turn"})
	code, page := postRaw(t, ts, runPath, ok)
	if code != http.StatusConflict || page["reason"] != "busy" {
		t.Fatalf("busy agent = %d %v, want 409 busy", code, page)
	}
	busy := page["busy"].([]any)
	if len(busy) != 1 || busy[0].(map[string]any)["id"] != agent.ID || busy[0].(map[string]any)["kind"] != "agent" {
		t.Fatalf("busy list = %v, want the agent", busy)
	}
	if msg := page["error"].(string); !strings.HasSuffix(msg, " is mid-turn in this repository.") {
		t.Fatalf("busy message = %q", msg)
	}

	swapProbes(t, map[string]string{tmux.ShellSessionName(other.ID): "sleep"}, map[string]string{})
	if code, page := postRaw(t, ts, runPath, ok); code != http.StatusConflict || page["reason"] != "busy" || page["error"] != "Terminal Other is running sleep in this repository." {
		t.Fatalf("other terminal foreground = %d %v", code, page)
	}

	swapProbes(t, map[string]string{}, map[string]string{})
	if code, _ := postRaw(t, ts, "/api/terminals/"+other.ID+"/state", `{"state":"working","cli":"claude-code"}`); code/100 != 2 {
		t.Fatalf("state report = %d", code)
	}
	if code, page := postRaw(t, ts, runPath, ok); code != http.StatusConflict || page["reason"] != "busy" || page["error"] != "Terminal Other is working in this repository." {
		t.Fatalf("other terminal working = %d %v", code, page)
	}

	// The type route shares the root and foreground guards.
	swapProbes(t, map[string]string{tmux.ShellSessionName(target.ID): "less"}, map[string]string{})
	if code, page := postRaw(t, ts, "/api/terminals/"+target.ID+"/type", ok); code != http.StatusConflict || page["reason"] != "foreground" {
		t.Fatalf("type into a pager = %d %v, want 409 foreground", code, page)
	}
	if code, page := postRaw(t, ts, "/api/terminals/"+target.ID+"/type", `{"text":"git fetch --prune","root":"/elsewhere"}`); code != http.StatusConflict || page["reason"] != "moved" {
		t.Fatalf("type with a wrong root = %d %v, want 409 moved", code, page)
	}
}

func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// With nobody busy, the command is typed into the real shell and submitted:
// the file it creates is the proof, not a captured screen.
func TestTerminalRunTypesAndSubmits(t *testing.T) {
	tm := tmux.New()
	if !tm.Available() {
		t.Skip("tmux missing")
	}
	repo := gitRepo(t)
	st := testStore(t)
	ts := graphServer(t, st)
	ws, _, err := storeWorkspaceWithAgent(st, "App", repo)
	if err != nil {
		t.Fatal(err)
	}
	code, page := postRaw(t, ts, "/api/terminals", `{"name":"QA","cwd":`+jsonString(repo)+`,"workspaceId":`+jsonString(ws.ID)+`}`)
	if code != http.StatusCreated {
		t.Fatalf("create terminal = %d %v", code, page)
	}
	id, _ := page["id"].(string)
	session, _ := page["session"].(string)
	t.Cleanup(func() { _ = tm.KillSession(context.Background(), session) })
	deadline := time.Now().Add(10 * time.Second)
	for {
		cmd, err := tm.PaneCommand(context.Background(), session)
		if err == nil && isShell(cmd) {
			break
		}
		if time.Now().After(deadline) {
			t.Skipf("shell never came up in %s (%v)", session, err)
		}
		time.Sleep(200 * time.Millisecond)
	}
	time.Sleep(700 * time.Millisecond)
	code, page = postRaw(t, ts, "/api/terminals/"+id+"/run", `{"text":"touch ran-marker","root":`+jsonString(canonDir(repo))+`}`)
	if code != http.StatusOK || page["ran"] != true {
		t.Fatalf("run = %d %v", code, page)
	}
	marker := filepath.Join(repo, "ran-marker")
	deadline = time.Now().Add(10 * time.Second)
	for {
		if _, err := os.Stat(marker); err == nil {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("the submitted command never ran: ran-marker missing")
		}
		time.Sleep(200 * time.Millisecond)
	}
}
