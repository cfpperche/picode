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

	"github.com/gorilla/websocket"

	"github.com/cfpperche/picode/internal/rpc"
	"github.com/cfpperche/picode/internal/session"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

func askBody(text, root string) string {
	b, _ := json.Marshal(map[string]string{"text": text, "root": root})
	return string(b)
}

// askServer builds a server whose Deps carry a DataDir (the receiver channel
// needs one to drop reply files) alongside the usual store/tmux/runtime.
func askServer(t *testing.T, st *store.Store, agentCmd string) (*httptest.Server, Deps) {
	t.Helper()
	deps := Deps{
		Store: st, Tmux: tmux.New(), DataDir: t.TempDir(),
		Runtime: rpc.NewRuntime(agentCmd, st, nil), AgentCmd: agentCmd,
		// New() takes Deps by value: a nil Replies here would be replaced
		// inside its own copy, invisible to this function's caller. Set it
		// up front so the test and the running server share one instance.
		Replies: newTuiReplies(),
	}
	ts := httptest.NewServer(New("127.0.0.1:0", deps).Handler)
	t.Cleanup(ts.Close)
	return ts, deps
}

func awaitTaskStatus(t *testing.T, st *store.Store, agentID, taskID, status string) store.Task {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		tasks, err := st.ListTasks(agentID, 20)
		if err != nil {
			t.Fatal(err)
		}
		for _, task := range tasks {
			if task.ID == taskID && task.Status == status {
				return task
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	tasks, _ := st.ListTasks(agentID, 20)
	t.Fatalf("task %s never reached %s; tasks = %+v", taskID, status, tasks)
	return store.Task{}
}

// The precondition (repository identity) and the plain-input refusals need
// no tmux session and no running agent: they answer before runMode is read.
func TestAskAgentRefusals(t *testing.T) {
	st := testStore(t)
	ts, _ := askServer(t, st, "cat")
	dir := t.TempDir()
	_, agent, err := storeWorkspaceWithAgent(st, "App", dir)
	if err != nil {
		t.Fatal(err)
	}
	askPath := "/api/agents/" + agent.ID + "/ask"

	if code, _ := postRaw(t, ts, "/api/agents/nope/ask", askBody("push", dir)); code != http.StatusNotFound {
		t.Fatalf("unknown agent = %d, want 404", code)
	}
	if code, _ := postRaw(t, ts, askPath, askBody("", dir)); code != http.StatusBadRequest {
		t.Fatalf("empty text = %d, want 400", code)
	}
	if code, _ := postRaw(t, ts, askPath, askBody("push", "")); code != http.StatusBadRequest {
		t.Fatalf("empty root = %d, want 400", code)
	}
	huge := make([]byte, askMaxLen+1)
	for i := range huge {
		huge[i] = 'x'
	}
	if code, _ := postRaw(t, ts, askPath, askBody(string(huge), dir)); code != http.StatusBadRequest {
		t.Fatalf("oversized text = %d, want 400", code)
	}
	if code, page := postRaw(t, ts, askPath, askBody("push", t.TempDir())); code != http.StatusConflict || page["reason"] != "moved" {
		t.Fatalf("wrong repository = %d %v, want 409 moved", code, page)
	}
	if code, page := postRaw(t, ts, askPath, askBody("push", dir)); code != http.StatusConflict || page["reason"] != "stopped" {
		t.Fatalf("stopped agent = %d %v, want 409 stopped", code, page)
	}
	if msg, _ := page(t, ts, askPath, dir); msg != "App is not running. Use the terminal instead." {
		t.Fatalf("stopped message = %q", msg)
	}
}

// page re-sends the ask and returns just the error string, so the message
// wording is asserted once without repeating the request.
func page(t *testing.T, ts *httptest.Server, path, root string) (string, map[string]any) {
	t.Helper()
	_, body := postRaw(t, ts, path, askBody("push", root))
	msg, _ := body["error"].(string)
	return msg, body
}

// A managed agent's ask enqueues a prompt task the runtime's delivery loop
// carries to completion; the response says the queue took it and whether a
// turn was already streaming.
func TestAskAgentQueuesForManagedAgent(t *testing.T) {
	t.Setenv("PICODE_FAKE_RPC", "1")
	st := testStore(t)
	ts, _ := askServer(t, st, os.Args[0])
	dir := t.TempDir()
	_, agent, err := storeWorkspaceWithAgent(st, "App", dir)
	if err != nil {
		t.Fatal(err)
	}
	if code, _ := postRaw(t, ts, "/api/agents/"+agent.ID+"/managed/start", "{}"); code != http.StatusCreated && code != http.StatusOK {
		t.Fatalf("managed/start = %d", code)
	}

	code, res := postRaw(t, ts, "/api/agents/"+agent.ID+"/ask", askBody("please push main", dir))
	if code != http.StatusAccepted || res["mode"] != "managed" || res["via"] != "queue" || res["busy"] != false {
		t.Fatalf("ask managed idle = %d %v", code, res)
	}
	taskID, _ := res["taskId"].(string)
	if taskID == "" {
		t.Fatalf("ask response missing taskId: %v", res)
	}
	awaitTaskStatus(t, st, agent.ID, taskID, store.TaskDelivered)

	// A turn already streaming (an unanswered ASK: dialog) is reported busy;
	// the prompt is still queued — it lands as pi's native follow_up once
	// the dialog is answered.
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws/agent?agent=" + agent.ID
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("ws: %v", err)
	}
	defer ws.Close()
	_ = ws.SetReadDeadline(time.Now().Add(8 * time.Second))
	readWSEvent(t, ws) // snapshot

	if code, _ := postRaw(t, ts, "/api/agents/"+agent.ID+"/tasks", `{"kind":"prompt","payload":"ASK:confirm","source":"user"}`); code != http.StatusCreated {
		t.Fatalf("enqueue ASK:confirm = %d", code)
	}
	waitWSType(t, ws, "extension_ui_request", 5*time.Second)

	code2, res2 := postRaw(t, ts, "/api/agents/"+agent.ID+"/ask", askBody("please push main too", dir))
	if code2 != http.StatusAccepted || res2["mode"] != "managed" || res2["via"] != "queue" || res2["busy"] != true {
		t.Fatalf("ask managed while waiting on a dialog = %d %v, want busy", code2, res2)
	}

	// Unblock the open dialog so the fake agent's process exits cleanly.
	if code3, _ := postRaw(t, ts, "/api/agents/"+agent.ID+"/ui", `{"id":"ui-ask","cancelled":true}`); code3/100 != 2 {
		t.Fatalf("ui cancel = %d", code3)
	}
}

// tuiAskFixture is the shared setup for the two TUI ask tests: an agent whose
// tmux session is up, ready for either the paste or the receiver door.
func tuiAskFixture(t *testing.T) (*httptest.Server, Deps, *store.Store, store.Agent, string) {
	t.Helper()
	manager := tmux.New()
	if !manager.Available() {
		t.Skip("tmux missing")
	}
	st := testStore(t)
	ts, deps := askServer(t, st, "cat")
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
	return ts, deps, st, agent, dir
}

// Without a fresh receiver hello (or with no session file known), asking a
// TUI agent falls back to the bracketed paste + Enter ADR-0060 already uses
// for Inbox replies. No session row to await, so delivery is recorded the
// moment tmux accepts the paste.
func TestAskAgentPastesIntoTUI(t *testing.T) {
	ts, _, st, agent, dir := tuiAskFixture(t)
	code, res := postRaw(t, ts, "/api/agents/"+agent.ID+"/ask", askBody("please pull with --ff-only", dir))
	if code != http.StatusOK || res["mode"] != "interactive" || res["via"] != "paste" || res["proof"] != false {
		t.Fatalf("ask tui paste = %d %v", code, res)
	}
	taskID, _ := res["taskId"].(string)
	pane, err := tmux.New().CaptureTail(context.Background(), tmux.SessionName(agent.ID), 20)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(pane, "please pull with --ff-only") {
		t.Fatalf("pane after paste = %q, want the prompt", pane)
	}
	awaitTaskStatus(t, st, agent.ID, taskID, store.TaskDelivered)
}

// A fresh receiver hello and a known session file take ADR-0060's other
// door: a one-shot reply file, an ack, and the durable JSONL row settle the
// task — exactly the Inbox reply channel, reused for an Inspector ask.
func TestAskAgentUsesTheReceiver(t *testing.T) {
	ts, deps, st, agent, dir := tuiAskFixture(t)
	sessionPath := filepath.Join(session.Dir(dir), "exact.jsonl")
	if err := os.MkdirAll(filepath.Dir(sessionPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sessionPath, []byte(`{"type":"session","id":"exact"}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := st.UpdateAgent(agent.ID, store.AgentPatch{SessionPath: &sessionPath}); err != nil {
		t.Fatal(err)
	}
	deps.Replies.Hello(agent.ID)

	type result struct {
		code int
		page map[string]any
	}
	done := make(chan result, 1)
	go func() {
		code, page := postRaw(t, ts, "/api/agents/"+agent.ID+"/ask", askBody("please open a pull request", dir))
		done <- result{code, page}
	}()

	dir2 := replyDir(deps.DataDir, agent.ID)
	var file string
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		ents, err := os.ReadDir(dir2)
		if err == nil {
			for _, e := range ents {
				if len(e.Name()) > 5 {
					file = filepath.Join(dir2, e.Name())
				}
			}
		}
		if file != "" {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if file == "" {
		t.Fatal("the receiver never received a reply file")
	}
	raw, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	var doc replyFile
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if doc.SessionPath != sessionPath || !contains(doc.Payload, "pull request") {
		t.Fatalf("reply file = %+v", doc)
	}
	deps.Replies.resolveAck(doc.Nonce, replyAck{OK: true})

	res := <-done
	if res.code != http.StatusOK || res.page["mode"] != "interactive" || res.page["via"] != "receiver" || res.page["proof"] != true {
		t.Fatalf("ask tui receiver = %d %v", res.code, res.page)
	}
	taskID, _ := res.page["taskId"].(string)
	task := awaitTaskStatus(t, st, agent.ID, taskID, store.TaskDelivering)
	appendUserRow(t, sessionPath, task.Payload)
	awaitTaskStatus(t, st, agent.ID, taskID, store.TaskDelivered)
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
