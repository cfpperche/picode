package server

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/rpc"
	"github.com/cfpperche/picode/internal/session"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

// replyHarness wires a store, one interactive TUI agent with an exact
// captured session, and an Inbox question bound to that session.
type replyHarness struct {
	deps        Deps
	store       *store.Store
	agent       store.Agent
	sessionPath string
	item        store.InboxItem
	name        string
}

func newReplyHarness(t *testing.T) *replyHarness {
	t.Helper()
	manager := tmux.New()
	if !manager.Available() {
		t.Skip("tmux not installed")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	dataDir := filepath.Join(home, ".picode")
	st, err := store.Open(filepath.Join(dataDir, "picode.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	cwd := t.TempDir()
	ws, _ := st.AddWorkspace("reply workspace", cwd)
	agent, err := st.AddAgent(ws.ID, "reply-agent", cwd)
	if err != nil {
		t.Fatal(err)
	}
	sessionPath := filepath.Join(session.Dir(cwd), "exact.jsonl")
	if err := os.MkdirAll(filepath.Dir(sessionPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sessionPath, []byte("{\"type\":\"session\",\"id\":\"exact\"}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	agent, err = st.UpdateAgent(agent.ID, store.AgentPatch{SessionPath: &sessionPath})
	if err != nil {
		t.Fatal(err)
	}
	if err := st.SetAgentRuntimeMode(agent.ID, store.StatusRunning, "interactive"); err != nil {
		t.Fatal(err)
	}
	item, err := st.CreateInboxItem(store.InboxItemParams{
		Kind: store.InboxQuestion, SourceKind: store.InboxFromAgent, SourceID: agent.ID,
		Reason: "test", Title: "Continue?", Body: "Please answer", SessionPath: sessionPath,
	})
	if err != nil {
		t.Fatal(err)
	}
	name := tmux.SessionName(agent.ID)
	if err := manager.NewSessionEnv(context.Background(), name, cwd, nil, "/bin/sh", "-c", "sleep 300"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.KillSession(context.Background(), name) })
	h := &replyHarness{
		deps: Deps{
			Store: st, Tmux: manager, DataDir: dataDir,
			Runtime: rpc.NewRuntime("/nonexistent-pi", st, nil),
			Replies: newTuiReplies(),
		},
		store: st, agent: agent, sessionPath: sessionPath, item: item, name: name,
	}
	return h
}

func (h *replyHarness) replyFiles(t *testing.T) []string {
	t.Helper()
	dir := replyDir(h.deps.DataDir, h.agent.ID)
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		ents, err := os.ReadDir(dir)
		if err == nil {
			var out []string
			for _, e := range ents {
				if strings.HasSuffix(e.Name(), ".json") {
					out = append(out, filepath.Join(dir, e.Name()))
				}
			}
			if len(out) > 0 {
				return out
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("no reply file appeared")
	return nil
}

func appendUserRow(t *testing.T, path, payload string) {
	t.Helper()
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	row := map[string]any{
		"type":      "message",
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		"message": map[string]any{
			"role": "user", "content": []map[string]string{{"type": "text", "text": payload}},
		},
	}
	if err := json.NewEncoder(f).Encode(row); err != nil {
		t.Fatal(err)
	}
}

func (h *replyHarness) taskFor(t *testing.T, itemID string) store.Task {
	t.Helper()
	tasks, err := h.store.ListTasks(h.agent.ID, 10)
	if err != nil {
		t.Fatal(err)
	}
	for _, task := range tasks {
		if strings.HasSuffix(task.Source, ":"+itemID) &&
			(strings.HasPrefix(task.Source, "inbox-tui:") || strings.HasPrefix(task.Source, "inbox-burst:")) {
			return task
		}
	}
	t.Fatalf("no parked task for item %s", itemID)
	return store.Task{}
}

func (h *replyHarness) awaitTaskStatus(t *testing.T, taskID, status string) store.Task {
	t.Helper()
	deadline := time.Now().Add(6 * time.Second)
	for time.Now().Before(deadline) {
		tasks, _ := h.store.ListTasks(h.agent.ID, 10)
		for _, task := range tasks {
			if task.ID == taskID && task.Status == status {
				return task
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	tasks, _ := h.store.ListTasks(h.agent.ID, 10)
	t.Fatalf("task %s never reached %s; tasks = %+v", taskID, status, tasks)
	return store.Task{}
}

// The receiver channel: hello fresh → the daemon drops a one-shot file, the
// receiver acks, the JSONL row lands, and the item stays done.
func TestDeliverReplyReceiverChannel(t *testing.T) {
	h := newReplyHarness(t)
	h.deps.Replies.Hello(h.agent.ID)
	sent := make(chan error, 1)
	go func() {
		_, err := h.deps.DeliverReply(context.Background(), h.item.ID, store.VerbRespond, "continue in place")
		sent <- err
	}()

	files := h.replyFiles(t)
	raw, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatal(err)
	}
	var doc replyFile
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if doc.SessionPath != h.sessionPath || !strings.Contains(doc.Payload, "continue in place") {
		t.Fatalf("reply file = %+v", doc)
	}
	// A blocking question must have parked the item done already.
	if got, _ := h.store.GetInboxItem(h.item.ID); got.State != store.InboxDone {
		t.Fatalf("item state = %s before ack", got.State)
	}
	h.deps.Replies.resolveAck(doc.Nonce, replyAck{OK: true})
	if err := <-sent; err != nil {
		t.Fatalf("DeliverReply = %v", err)
	}
	if _, err := os.Stat(files[0]); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("reply file survived the ack: %v", err)
	}
	task := h.taskFor(t, h.item.ID)
	if task.Status != store.TaskDelivering {
		t.Fatalf("task after ack = %s, want delivering until the row lands", task.Status)
	}
	appendUserRow(t, h.sessionPath, task.Payload)
	task = h.awaitTaskStatus(t, task.ID, store.TaskDelivered)
}

// The receiver refuses a session mismatch: the item reopens with the response
// preserved, the task fails, and the file is gone.
func TestDeliverReplyReceiverSessionMismatch(t *testing.T) {
	h := newReplyHarness(t)
	h.deps.Replies.Hello(h.agent.ID)
	sent := make(chan error, 1)
	go func() {
		_, err := h.deps.DeliverReply(context.Background(), h.item.ID, store.VerbRespond, "mismatched")
		sent <- err
	}()
	files := h.replyFiles(t)
	raw, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatal(err)
	}
	var doc replyFile
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	h.deps.Replies.resolveAck(doc.Nonce, replyAck{OK: false, Reason: "the terminal is showing a different session"})
	err = <-sent
	if err == nil || !strings.Contains(err.Error(), "different session") {
		t.Fatalf("DeliverReply = %v, want the session-mismatch refusal", err)
	}
	if _, statErr := os.Stat(files[0]); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("reply file survived the nack: %v", statErr)
	}
	got, _ := h.store.GetInboxItem(h.item.ID)
	if got.State != store.InboxUnread || got.Response == nil || !strings.Contains(*got.Response, "mismatched") {
		t.Fatalf("reopened item = %+v", got)
	}
	if !strings.Contains(got.Body, "Send it again") {
		t.Fatalf("reopen note missing: %q", got.Body)
	}
	task := h.taskFor(t, h.item.ID)
	if task.Status != store.TaskFailed {
		t.Fatalf("task after nack = %s", task.Status)
	}
}

// No ack at all: the durable JSONL row still wins, and the reply file is
// cleaned up.
func TestDeliverReplyRowWinsWithoutAck(t *testing.T) {
	h := newReplyHarness(t)
	orig := receiverAckWait
	receiverAckWait = 1200 * time.Millisecond
	t.Cleanup(func() { receiverAckWait = orig })
	h.deps.Replies.Hello(h.agent.ID)

	sent := make(chan error, 1)
	go func() {
		_, err := h.deps.DeliverReply(context.Background(), h.item.ID, store.VerbRespond, "durable proof")
		sent <- err
	}()
	files := h.replyFiles(t)
	task := h.taskFor(t, h.item.ID)
	appendUserRow(t, h.sessionPath, task.Payload)
	if err := <-sent; err != nil {
		t.Fatalf("DeliverReply = %v", err)
	}
	if _, statErr := os.Stat(files[0]); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("reply file survived: %v", statErr)
	}
	h.awaitTaskStatus(t, task.ID, store.TaskDelivered)
	if got, _ := h.store.GetInboxItem(h.item.ID); got.State != store.InboxDone {
		t.Fatalf("item state = %s", got.State)
	}
}

// Ack without a row: the TUI owns the message, but if it never reaches the
// session the item reopens honestly.
func TestDeliverReplyAckWithoutRowReopens(t *testing.T) {
	h := newReplyHarness(t)
	origWait := replyRowWait
	replyRowWait = 400 * time.Millisecond
	t.Cleanup(func() { replyRowWait = origWait })
	h.deps.Replies.Hello(h.agent.ID)

	sent := make(chan error, 1)
	go func() {
		_, err := h.deps.DeliverReply(context.Background(), h.item.ID, store.VerbRespond, "queued then lost")
		sent <- err
	}()
	files := h.replyFiles(t)
	raw, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatal(err)
	}
	var doc replyFile
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	h.deps.Replies.resolveAck(doc.Nonce, replyAck{OK: true})
	if err := <-sent; err != nil {
		t.Fatalf("DeliverReply = %v, want success on ack", err)
	}
	task := h.taskFor(t, h.item.ID)
	h.awaitTaskStatus(t, task.ID, store.TaskFailed)
	got, _ := h.store.GetInboxItem(h.item.ID)
	if got.State != store.InboxUnread || got.Response == nil || !strings.Contains(*got.Response, "queued then lost") {
		t.Fatalf("reopened item = %+v", got)
	}
}

// The fallback channel types the reply into a legacy pane. The fake pane
// turns each submitted line into a JSONL user row, exactly what pi's editor
// submit does with the real session.
func TestDeliverReplyPasteFallback(t *testing.T) {
	h := newReplyHarness(t)
	before, err := h.deps.Tmux.PaneSessionID(context.Background(), h.name)
	if err != nil {
		t.Fatal(err)
	}
	// Replace the sleeper with the fake session writer. A real editor
	// consumes the bracketed-paste markers; this fake strips them the same way.
	script := "while IFS= read -r line; do " +
		`line=${line//$'\e[200~'/}; line=${line//$'\e[201~'/}; line=${line//\"/\\\"}; ` +
		`ts=$(date -u +%Y-%m-%dT%H:%M:%S.000Z); ` +
		`printf '{"type":"message","timestamp":"%s","message":{"role":"user","content":[{"type":"text","text":"%s"}]}}\n' "$ts" "$line" ` +
		`>> "$REPLY_SESSION"; done`
	if err := h.deps.Tmux.RespawnPaneEnv(context.Background(), h.name, ".", []string{"REPLY_SESSION=" + h.sessionPath}, "/bin/bash", "-c", script); err != nil {
		t.Fatal(err)
	}
	time.Sleep(200 * time.Millisecond) // let the shell reach the read loop

	if _, err := h.deps.DeliverReply(context.Background(), h.item.ID, store.VerbRespond, "typed fallback"); err != nil {
		t.Fatalf("DeliverReply = %v", err)
	}
	task := h.taskFor(t, h.item.ID)
	h.awaitTaskStatus(t, task.ID, store.TaskDelivered)
	if got, _ := h.store.GetInboxItem(h.item.ID); got.State != store.InboxDone {
		t.Fatalf("item state = %s", got.State)
	}
	after, err := h.deps.Tmux.PaneSessionID(context.Background(), h.name)
	if err != nil || before != after {
		t.Fatalf("tmux identity changed: %q -> %q (%v)", before, after, err)
	}
}

func TestDeliverReplyRefusals(t *testing.T) {
	h := newReplyHarness(t)

	// A held mutation guard refuses the reply deterministically.
	release := h.deps.Replies.Controls.BeginMutation(h.agent.ID)
	_, err := h.deps.DeliverReply(context.Background(), h.item.ID, store.VerbRespond, "x")
	if err == nil || !strings.Contains(err.Error(), replyControlConflict) {
		t.Fatalf("guarded reply = %v, want the control conflict", err)
	}
	release()

	// A pending send refuses a second one.
	h.deps.Replies.mu.Lock()
	h.deps.Replies.active[h.agent.ID] = true
	h.deps.Replies.mu.Unlock()
	_, err = h.deps.DeliverReply(context.Background(), h.item.ID, store.VerbRespond, "x")
	if err == nil || !strings.Contains(err.Error(), "already has a reply") {
		t.Fatalf("pending reply = %v, want the pending refusal", err)
	}
	h.deps.Replies.mu.Lock()
	delete(h.deps.Replies.active, h.agent.ID)
	h.deps.Replies.mu.Unlock()

	// An unknown pane refuses before anything is parked.
	if err := h.deps.Tmux.KillSession(context.Background(), h.name); err != nil {
		t.Fatal(err)
	}
	_, err = h.deps.DeliverReply(context.Background(), h.item.ID, store.VerbRespond, "x")
	if err == nil || !strings.Contains(err.Error(), "no longer running") {
		t.Fatalf("dead pane reply = %v, want the pane refusal", err)
	}
	if got, _ := h.store.GetInboxItem(h.item.ID); got.State == store.InboxDone {
		t.Fatalf("refused reply must not park the item")
	}
}

// The terminal's exact session is mandatory: no fallback to the agent's
// current pointer, latest file, or pending state.
func TestResolveReplySessionRequiresItemSessionPath(t *testing.T) {
	cwd := t.TempDir()
	path := filepath.Join(session.Dir(cwd), "exact.jsonl")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{\"type\":\"session\"}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	agent := store.Agent{ID: "exact-agent", SessionPath: &path}
	deps := Deps{}
	if got, ok := deps.resolveReplySession(agent, store.InboxItem{}, cwd); ok || got != "" {
		t.Fatalf("missing item session fell back to %q", got)
	}
	if got, ok := deps.resolveReplySession(agent, store.InboxItem{SessionPath: path}, cwd); !ok || got != path {
		t.Fatalf("captured item session = %q, %v", got, ok)
	}
}

// Boot reconciliation: a row proves delivery even though the daemon died;
// silence fails the task, reopens the item, and clears the reply directory.
func TestReconcilePendingReplies(t *testing.T) {
	h := newReplyHarness(t)
	orig := bootRowGrace
	bootRowGrace = 30 * time.Millisecond
	t.Cleanup(func() { bootRowGrace = orig })

	// Delivered-before-crash: park a task, then write the row.
	if _, _, err := (Deps{Store: h.store}).parkForTest(h.item.ID, "row lands first"); err != nil {
		t.Fatal(err)
	}
	taskDelivered := h.taskFor(t, h.item.ID)
	appendUserRow(t, h.sessionPath, taskDelivered.Payload)

	// Lost-in-crash: a second item whose row never appears, with a leftover
	// reply file on disk.
	item2, err := h.store.CreateInboxItem(store.InboxItemParams{
		Kind: store.InboxQuestion, SourceKind: store.InboxFromAgent, SourceID: h.agent.ID,
		Reason: "test", Title: "Again?", Body: "?", SessionPath: h.sessionPath,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := (Deps{Store: h.store}).parkForTest(item2.ID, "never confirmed"); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(replyDir(h.deps.DataDir, h.agent.ID), 0o700); err != nil {
		t.Fatal(err)
	}
	stale := filepath.Join(replyDir(h.deps.DataDir, h.agent.ID), "leftover.json")
	if err := os.WriteFile(stale, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}

	ReconcilePendingReplies(h.store, h.deps.DataDir)

	h.awaitTaskStatus(t, taskDelivered.ID, store.TaskDelivered)
	taskLost := h.taskFor(t, item2.ID)
	if taskLost.Status != store.TaskFailed {
		t.Fatalf("lost task = %s", taskLost.Status)
	}
	got, _ := h.store.GetInboxItem(item2.ID)
	if got.State != store.InboxUnread || got.Response == nil || !strings.Contains(*got.Response, "never confirmed") {
		t.Fatalf("reopened item = %+v", got)
	}
	if _, err := os.Stat(stale); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("stale reply dir survived: %v", err)
	}
	if got2, _ := h.store.GetInboxItem(h.item.ID); got2.State != store.InboxDone {
		t.Fatalf("delivered item state = %s", got2.State)
	}
}

// parkForTest reserves the one synchronous send lane and parks the reply,
// mirroring DeliverReply's store transaction without needing a live terminal.
func (deps Deps) parkForTest(itemID, text string) (store.InboxItem, store.Task, error) {
	return deps.Store.RespondAndPark(itemID, store.VerbRespond, text)
}

func TestAgentControlsExclusivity(t *testing.T) {
	g := newAgentControls()
	if err := g.check("a"); err != nil {
		t.Fatalf("idle check = %v", err)
	}
	release := g.BeginMutation("a")
	if err := g.check("a"); err == nil {
		t.Fatal("mutation guard did not block the reply check")
	}
	if _, err := g.TryBeginMutation("a"); err == nil {
		t.Fatal("TryBeginMutation must be exclusive")
	}
	release()
	release() // idempotent
	if err := g.check("a"); err != nil {
		t.Fatalf("released check = %v", err)
	}
	tryRelease, err := g.TryBeginMutation("a")
	if err != nil {
		t.Fatalf("TryBeginMutation = %v", err)
	}
	tryRelease()
}

// Every interactive agent TUI PiCode spawns carries the reply receiver.
func TestSpawnFlagsInjectReplyReceiver(t *testing.T) {
	h := newReplyHarness(t)
	flags := h.deps.spawnFlags(h.agent)
	found := false
	for i, f := range flags {
		if f == "-e" && i+1 < len(flags) && strings.HasSuffix(flags[i+1], "pi-inbox-reply.ts") {
			found = true
			raw, err := os.ReadFile(flags[i+1])
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(raw), "sendUserMessage") || !strings.Contains(string(raw), "tui-hello") {
				t.Fatalf("receiver extension is missing its contract:\n%s", raw)
			}
		}
	}
	if !found {
		t.Fatalf("spawn flags %v do not inject the reply receiver", flags)
	}
}
