package rpc

import (
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

func TestManagedDeliveryEngine(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "picode.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	w, agent, err := st.AddWorkspace("Managed", t.TempDir())
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
		case "task_enqueued":
			sawEnq = true
		case "task_delivered":
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

func TestDoubleStartRejected(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "picode.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	w, agent, err := st.AddWorkspace("Dup", t.TempDir())
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
