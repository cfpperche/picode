package rpc

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/store"
)

func startRuntime(t *testing.T, st *store.Store) *Runtime {
	t.Helper()
	t.Setenv("PICODE_FAKE_RPC", "1") // child re-execs the test binary as fake pi
	rt := NewRuntime(os.Args[0], st, nil)
	t.Cleanup(rt.StopAll)
	return rt
}

// addWorkspaceWithAgent keeps the old AddWorkspace shape for tests:
// workspaces start empty (ADR-0027), so the agent is explicit now.
func addWorkspaceWithAgent(st *store.Store, name, path string) (store.Workspace, store.Agent, error) {
	w, err := st.AddWorkspace(name, path)
	if err != nil {
		return store.Workspace{}, store.Agent{}, err
	}
	a, err := st.AddAgent(w.ID, "default", "")
	return w, a, err
}

func TestManagedDeliveryEngine(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "picode.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	w, agent, err := addWorkspaceWithAgent(st, "Managed", t.TempDir())
	if err != nil {
		t.Fatalf("AddWorkspace: %v", err)
	}

	rt := startRuntime(t, st)
	if err := rt.Start(agent.ID, w.Path); err != nil {
		t.Fatalf("Start: %v", err)
	}

	ma := rt.Get(agent.ID)
	if ma == nil {
		t.Fatal("managed agent not registered")
	}

	// Watch the hub for task_delivered.
	hubCh, unsub := ma.hub.Subscribe()
	defer unsub()

	// Enqueue a prompt task; engine must claim, send, mark delivered.
	if _, err := st.EnqueueTask(agent.ID, store.TaskPrompt, "hello engine", "user"); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	deadline := time.Now().Add(10 * time.Second)
	delivered := false
	for time.Now().Before(deadline) && !delivered {
		select {
		case msg := <-hubCh:
			var env struct {
				Event struct {
					Type   string `json:"type"`
					TaskID string `json:"taskId"`
				} `json:"event"`
			}
			if err := json.Unmarshal(msg, &env); err != nil {
				continue
			}
			if env.Event.Type == "task_delivered" {
				delivered = true
			}
		case <-time.After(100 * time.Millisecond):
		}
	}
	if !delivered {
		t.Fatal("task_delivered never observed")
	}

	// Store reflects delivery + audit events.
	tasks, _ := st.ListTasks(agent.ID, 10)
	if len(tasks) != 1 || tasks[0].Status != store.TaskDelivered {
		t.Fatalf("task after delivery = %+v", tasks)
	}
	evts, _ := st.AgentEvents(agent.ID, 10)
	sawEnq, sawDel := false, false
	for _, e := range evts {
		switch e.Type {
		case "task.enqueued":
			sawEnq = true
		case "task.delivered":
			sawDel = true
		}
	}
	if !sawEnq || !sawDel {
		t.Errorf("audit events missing (enq=%v del=%v): %+v", sawEnq, sawDel, evts)
	}

	// Stop is idempotent and unregisters.
	if !rt.Stop(agent.ID) {
		t.Error("first Stop = false")
	}
	if rt.Stop(agent.ID) {
		t.Error("second Stop = true")
	}
	if rt.Get(agent.ID) != nil {
		t.Error("agent still registered after Stop")
	}
}

func waitHub(t *testing.T, ch <-chan []byte, typ string, d time.Duration) map[string]any {
	t.Helper()
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		select {
		case msg := <-ch:
			var env struct {
				Event map[string]any `json:"event"`
			}
			if err := json.Unmarshal(msg, &env); err != nil {
				continue
			}
			got, _ := env.Event["type"].(string)
			if got == typ {
				return env.Event
			}
		case <-time.After(50 * time.Millisecond):
		}
	}
	t.Fatalf("hub never saw %s", typ)
	return nil
}

func TestWaitingDialog(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "picode.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	w, agent, err := addWorkspaceWithAgent(st, "Ask", t.TempDir())
	if err != nil {
		t.Fatalf("AddWorkspace: %v", err)
	}
	rt := startRuntime(t, st)
	if err := rt.Start(agent.ID, w.Path); err != nil {
		t.Fatalf("Start: %v", err)
	}
	ma := rt.Get(agent.ID)
	if ma == nil {
		t.Fatal("managed agent not registered")
	}
	hubCh, unsub := ma.hub.Subscribe()
	defer unsub()

	// Row 1: managed, no dialog → not waiting.
	if ma.Snapshot().Waiting {
		t.Fatal("idle snapshot waiting")
	}

	// Row 6: notify is not waiting.
	if _, err := st.EnqueueTask(agent.ID, store.TaskPrompt, "ASK:notify", "user"); err != nil {
		t.Fatalf("enqueue notify: %v", err)
	}
	_ = waitHub(t, hubCh, "extension_ui_request", 5*time.Second)
	time.Sleep(30 * time.Millisecond)
	if ma.Snapshot().Waiting {
		t.Fatal("notify set waiting")
	}

	// Row 2: confirm → waiting.
	if _, err := st.EnqueueTask(agent.ID, store.TaskPrompt, "ASK:confirm", "user"); err != nil {
		t.Fatalf("enqueue confirm: %v", err)
	}
	ev := waitHub(t, hubCh, "extension_ui_request", 5*time.Second)
	snap := ma.Snapshot()
	if !snap.Waiting || snap.Dialog == nil || snap.Dialog.Method != "confirm" {
		t.Fatalf("confirm snapshot = %+v event=%v", snap, ev)
	}

	// Row 3: answer ends waiting.
	yes := true
	if err := ma.ReplyUI("ui-ask", "", &yes, false); err != nil {
		t.Fatalf("ReplyUI yes: %v", err)
	}
	time.Sleep(30 * time.Millisecond)
	if ma.Snapshot().Waiting {
		t.Fatal("still waiting after yes")
	}

	// Row 4: cancel.
	if _, err := st.EnqueueTask(agent.ID, store.TaskPrompt, "ASK:confirm", "user"); err != nil {
		t.Fatalf("enqueue confirm2: %v", err)
	}
	_ = waitHub(t, hubCh, "extension_ui_request", 5*time.Second)
	if err := ma.ReplyUI("ui-ask", "", nil, true); err != nil {
		t.Fatalf("ReplyUI cancel: %v", err)
	}
	time.Sleep(30 * time.Millisecond)
	if ma.Snapshot().Waiting {
		t.Fatal("still waiting after cancel")
	}

	// Row 5: timeout clears waiting; reply is rejected.
	if _, err := st.EnqueueTask(agent.ID, store.TaskPrompt, "ASK:timeout", "user"); err != nil {
		t.Fatalf("enqueue timeout: %v", err)
	}
	_ = waitHub(t, hubCh, "extension_ui_request", 5*time.Second)
	if !ma.Snapshot().Waiting {
		t.Fatal("timeout dialog not waiting")
	}
	_ = waitHub(t, hubCh, "extension_ui_timeout", 2*time.Second)
	if ma.Snapshot().Waiting {
		t.Fatal("still waiting after timeout")
	}
	if err := ma.ReplyUI("ui-to", "", nil, true); err == nil {
		t.Fatal("ReplyUI after timeout succeeded")
	}
	// Unblock the fake process.
	_ = ma.client.SendRaw(map[string]any{"type": "extension_ui_response", "id": "ui-to", "cancelled": true})
}

func TestEffectiveTurnKind(t *testing.T) {
	rows := []struct {
		kind string
		busy bool
		want string
	}{
		{store.TaskPrompt, false, store.TaskPrompt},
		{store.TaskPrompt, true, store.TaskFollowUp},
		{"", true, store.TaskFollowUp},
		{store.TaskSteer, true, store.TaskSteer},
		{store.TaskFollowUp, true, store.TaskFollowUp},
		{store.TaskSteer, false, store.TaskSteer},
	}
	for _, r := range rows {
		got := EffectiveTurnKind(r.kind, r.busy)
		if got != r.want {
			t.Errorf("kind=%q busy=%v → %q, want %q", r.kind, r.busy, got, r.want)
		}
	}
}

func TestQueueWhileWaiting(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "picode.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	w, agent, err := addWorkspaceWithAgent(st, "Queue", t.TempDir())
	if err != nil {
		t.Fatalf("AddWorkspace: %v", err)
	}
	rt := startRuntime(t, st)
	if err := rt.Start(agent.ID, w.Path); err != nil {
		t.Fatalf("Start: %v", err)
	}
	ma := rt.Get(agent.ID)
	hubCh, unsub := ma.hub.Subscribe()
	defer unsub()

	if _, err := st.EnqueueTask(agent.ID, store.TaskPrompt, "ASK:confirm", "user"); err != nil {
		t.Fatalf("enqueue confirm: %v", err)
	}
	_ = waitHub(t, hubCh, "extension_ui_request", 5*time.Second)
	if !ma.Snapshot().Waiting {
		t.Fatal("not waiting")
	}

	// Row 4: follow_up while waiting returns now; dialog stays.
	done := make(chan error, 1)
	go func() { done <- ma.SendTurn(store.TaskFollowUp, "later", nil) }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("follow_up: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("follow_up blocked on waiting")
	}
	if !ma.Snapshot().Waiting {
		t.Fatal("follow_up cleared waiting")
	}

	// Row 5: prompt while waiting becomes follow_up and does not error.
	go func() { done <- ma.SendTurn(store.TaskPrompt, "also", nil) }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("busy prompt: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("busy prompt blocked on waiting")
	}

	if err := ma.ReplyUI("ui-ask", "", nil, true); err != nil {
		t.Fatalf("cancel: %v", err)
	}
}

func TestAbortCancelsWaitingDialog(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "picode.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	w, agent, err := addWorkspaceWithAgent(st, "AbortWait", t.TempDir())
	if err != nil {
		t.Fatalf("AddWorkspace: %v", err)
	}
	rt := startRuntime(t, st)
	if err := rt.Start(agent.ID, w.Path); err != nil {
		t.Fatalf("Start: %v", err)
	}
	ma := rt.Get(agent.ID)
	hubCh, unsub := ma.hub.Subscribe()
	defer unsub()
	if _, err := st.EnqueueTask(agent.ID, store.TaskPrompt, "ASK:confirm", "user"); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	_ = waitHub(t, hubCh, "extension_ui_request", 5*time.Second)
	if !ma.Snapshot().Waiting {
		t.Fatal("not waiting")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := ma.Abort(ctx); err != nil {
		t.Fatalf("Abort: %v", err)
	}
	if ma.Snapshot().Waiting {
		t.Fatal("abort left waiting")
	}
}

// A slow human in an extension picker must not fail the task: the prompt
// response only arrives when the whole /roles flow ends, so a pending
// dialog at the delivery deadline counts as delivered.
func TestSlowDialogIsDeliveredNotFailed(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "picode.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	w, agent, err := addWorkspaceWithAgent(st, "SlowDialog", t.TempDir())
	if err != nil {
		t.Fatalf("AddWorkspace: %v", err)
	}
	old := deliverTimeout
	deliverTimeout = 300 * time.Millisecond
	t.Cleanup(func() { deliverTimeout = old })

	rt := startRuntime(t, st)
	if err := rt.Start(agent.ID, w.Path); err != nil {
		t.Fatalf("Start: %v", err)
	}
	ma := rt.Get(agent.ID)
	hubCh, unsub := ma.hub.Subscribe()
	defer unsub()
	if _, err := st.EnqueueTask(agent.ID, store.TaskPrompt, "ASK:confirm", "user"); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	_ = waitHub(t, hubCh, "extension_ui_request", 5*time.Second)

	// The fake answers the prompt only after the dialog reply, so the
	// deadline passes while waiting — the task must still be delivered.
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case msg := <-hubCh:
			var env struct {
				Event struct {
					Type  string `json:"type"`
					Error string `json:"error"`
				} `json:"event"`
			}
			if err := json.Unmarshal(msg, &env); err != nil {
				continue
			}
			if env.Event.Type == "task_failed" {
				t.Fatalf("task_failed while a dialog was pending: %s", env.Event.Error)
			}
			if env.Event.Type == "task_delivered" {
				if err := ma.ReplyUI("ui-ask", "", nil, true); err != nil {
					t.Fatalf("cancel: %v", err)
				}
				return
			}
		case <-time.After(200 * time.Millisecond):
		}
	}
	t.Fatal("no task_delivered")
}

func TestDoubleStartRejected(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "picode.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	w, agent, err := addWorkspaceWithAgent(st, "Dup", t.TempDir())
	if err != nil {
		t.Fatalf("AddWorkspace: %v", err)
	}

	rt := startRuntime(t, st)
	if err := rt.Start(agent.ID, w.Path); err != nil {
		t.Fatalf("first Start: %v", err)
	}
	if err := rt.Start(agent.ID, w.Path); err == nil {
		t.Fatal("double Start accepted")
	}
}
