package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/clilaunch"
	"github.com/cfpperche/picode/internal/rpc"
	"github.com/cfpperche/picode/internal/session"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

// A pi CLI terminal in a repository, with its receiver having said hello and
// named a real session file — the state ADR-0089's amendment asks into.
func piTerminalFixture(t *testing.T) (*httptest.Server, Deps, *store.Store, store.Terminal, string, string) {
	t.Helper()
	st := testStore(t)
	// New() takes Deps by value and lazily creates TermRuntimes in its own
	// copy; a presence registered here must be the one the handlers read.
	deps := Deps{
		Store: st, Tmux: tmux.New(), DataDir: t.TempDir(),
		Runtime: rpc.NewRuntime("cat", st, nil), AgentCmd: "cat",
		Replies: newTuiReplies(), TermRuntimes: NewTermRuntimes(),
	}
	ts := httptest.NewServer(New("127.0.0.1:0", deps).Handler)
	t.Cleanup(ts.Close)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	repo := gitRepo(t)
	ws, _, err := storeWorkspaceWithAgent(st, "App", repo)
	if err != nil {
		t.Fatal(err)
	}
	term, err := st.CreateTerminalIn(ws.ID, "Pi", repo)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.SetTerminalLaunch(term.ID, "pi", clilaunch.Overrides{}); err != nil {
		t.Fatal(err)
	}
	deps.TermRuntimes.Start(term.ID, TermRuntime{CLI: "pi", Source: "test", RunID: "r1", StartedAt: time.Now()})
	sessionPath := filepath.Join(session.Dir(repo), "exact.jsonl")
	if err := os.MkdirAll(filepath.Dir(sessionPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sessionPath, []byte(`{"type":"session","id":"exact"}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return ts, deps, st, term, repo, sessionPath
}

func TestTerminalHelloRecordsTheSessionItShows(t *testing.T) {
	ts, deps, _, term, _, sessionPath := piTerminalFixture(t)
	code, _ := postRaw(t, ts, "/api/terminals/"+term.ID+"/tui-hello", `{"session":`+jsonString(sessionPath)+`}`)
	if code != http.StatusNoContent {
		t.Fatalf("hello: %d", code)
	}
	if got := deps.Replies.receiverSession(termReplyKey(term.ID)); got != sessionPath {
		t.Errorf("recorded session = %q; want %q", got, sessionPath)
	}
	if code, _ := postRaw(t, ts, "/api/terminals/nope/tui-hello", `{}`); code != http.StatusNotFound {
		t.Errorf("unknown terminal hello: %d; want 404", code)
	}
}

// Every refusal names itself, and none of them types into the pane.
func TestTerminalAskRefusals(t *testing.T) {
	ts, deps, st, term, repo, sessionPath := piTerminalFixture(t)
	ask := func(path, root string) (int, string) {
		code, page := postRaw(t, ts, path, askBody("merge feat/x", root))
		reason, _ := page["reason"].(string)
		return code, reason
	}
	// No hello yet: no receiver.
	if code, reason := ask("/api/terminals/"+term.ID+"/ask", repo); code != http.StatusConflict || reason != "no-receiver" {
		t.Errorf("without a hello: %d %q; want 409 no-receiver", code, reason)
	}
	// A hello with no session: alive, but nothing to ask into.
	deps.Replies.HelloSession(termReplyKey(term.ID), "")
	if code, reason := ask("/api/terminals/"+term.ID+"/ask", repo); code != http.StatusConflict || reason != "no-session" {
		t.Errorf("hello without a session: %d %q; want 409 no-session", code, reason)
	}
	// A session outside this folder's session tree is not trusted.
	deps.Replies.HelloSession(termReplyKey(term.ID), filepath.Join(t.TempDir(), "elsewhere.jsonl"))
	if code, reason := ask("/api/terminals/"+term.ID+"/ask", repo); code != http.StatusConflict || reason != "no-session" {
		t.Errorf("session outside the tree: %d %q; want 409 no-session", code, reason)
	}
	deps.Replies.HelloSession(termReplyKey(term.ID), sessionPath)
	// Another repository as root: moved.
	if code, reason := ask("/api/terminals/"+term.ID+"/ask", gitRepo(t)); code != http.StatusConflict || reason != "moved" {
		t.Errorf("other repository: %d %q; want 409 moved", code, reason)
	}
	// pi not running right now: stopped.
	deps.TermRuntimes.Drop(term.ID)
	if code, reason := ask("/api/terminals/"+term.ID+"/ask", repo); code != http.StatusConflict || reason != "stopped" {
		t.Errorf("pi down: %d %q; want 409 stopped", code, reason)
	}
	// A plain shell, or another CLI, is never asked — ADR-0062 stands.
	plain, err := st.CreateTerminalIn(term.WorkspaceID, "sh", repo)
	if err != nil {
		t.Fatal(err)
	}
	deps.Replies.HelloSession(termReplyKey(plain.ID), sessionPath)
	if code, reason := ask("/api/terminals/"+plain.ID+"/ask", repo); code != http.StatusConflict || reason != "cli" {
		t.Errorf("plain shell: %d %q; want 409 cli", code, reason)
	}
	if code, _ := postRaw(t, ts, "/api/terminals/"+term.ID+"/ask", `{"text":"","root":"`+repo+`"}`); code != http.StatusBadRequest {
		t.Errorf("empty text: %d; want 400", code)
	}
}

// The happy path is ADR-0060's: a reply file under the terminal's own key,
// acknowledged by the receiver, recorded as an event — never a task row,
// never a paste.
func TestTerminalAskDeliversThroughTheReceiver(t *testing.T) {
	ts, deps, st, term, repo, sessionPath := piTerminalFixture(t)
	deps.Replies.HelloSession(termReplyKey(term.ID), sessionPath)

	type result struct {
		code int
		page map[string]any
	}
	done := make(chan result, 1)
	go func() {
		code, page := postRaw(t, ts, "/api/terminals/"+term.ID+"/ask", askBody("please merge feat/x", repo))
		done <- result{code, page}
	}()

	dir := replyDir(deps.DataDir, termReplyKey(term.ID))
	var file string
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) && file == "" {
		if ents, err := os.ReadDir(dir); err == nil {
			for _, e := range ents {
				if filepath.Ext(e.Name()) == ".json" {
					file = filepath.Join(dir, e.Name())
				}
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	if file == "" {
		t.Fatal("no reply file reached the terminal's receiver directory")
	}
	raw, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	var doc replyFile
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if doc.Payload != "please merge feat/x" || doc.SessionPath != sessionPath {
		t.Errorf("reply file = %+v", doc)
	}
	// The receiver acks through the terminal route, like the agent one.
	if code, _ := postRaw(t, ts, "/api/terminals/"+term.ID+"/tui-ack", `{"nonce":`+jsonString(doc.Nonce)+`,"ok":true}`); code != http.StatusNoContent {
		t.Fatalf("ack: %d", code)
	}
	res := <-done
	if res.code != http.StatusOK {
		t.Fatalf("ask: %d %v", res.code, res.page)
	}
	if res.page["mode"] != "interactive" || res.page["via"] != "receiver" || res.page["proof"] != true {
		t.Errorf("result = %v", res.page)
	}
	// Provenance is an event, and the agents' task queue stays untouched.
	tasks, err := st.ListTasks("", 50)
	if err == nil && len(tasks) != 0 {
		t.Errorf("a terminal ask must not enqueue an agent task: %v", tasks)
	}
}

// A pi terminal is an occupant of its worktree, marked as one, with its
// presence — so the graph can offer "Ask Pi" beside the agents.
func TestGraphListsPiTerminalsAsOccupants(t *testing.T) {
	ts, deps, st, term, repo, _ := piTerminalFixture(t)
	ws, err := st.GetWorkspace(term.WorkspaceID)
	if err != nil {
		t.Fatal(err)
	}
	// A plain shell in the same folder is not an occupant: it has nobody to ask.
	if _, err := st.CreateTerminalIn(ws.ID, "sh", repo); err != nil {
		t.Fatal(err)
	}
	g := getGraph(t, ts, "/api/workspaces/"+ws.ID+"/git")
	var found *occupant
	for _, wt := range g.Worktrees {
		for i := range wt.Agents {
			if wt.Agents[i].ID == term.ID {
				found = &wt.Agents[i]
			}
			if wt.Agents[i].Kind == "terminal" && wt.Agents[i].ID != term.ID {
				t.Errorf("a plain shell was listed as an occupant: %+v", wt.Agents[i])
			}
		}
	}
	if found == nil {
		t.Fatalf("the pi terminal is not an occupant of its worktree: %+v", g.Worktrees)
	}
	if found.Kind != "terminal" || !found.Live || found.Name != "Pi" {
		t.Errorf("occupant = %+v; want kind terminal, live, named Pi", *found)
	}
	deps.TermRuntimes.Drop(term.ID)
	g = getGraph(t, ts, "/api/workspaces/"+ws.ID+"/git")
	for _, wt := range g.Worktrees {
		for _, o := range wt.Agents {
			if o.ID == term.ID && o.Live {
				t.Error("a terminal whose pi exited is still marked live")
			}
		}
	}
}
