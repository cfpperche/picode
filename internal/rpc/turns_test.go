package rpc

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/store"
)

// ADR-0194: a managed turn (pi's agent_start) counts on the agent, so a
// person's removal can tell whether the agent ever worked.
func TestManagedTurnIsCounted(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "picode.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	w, agent, err := addWorkspaceWithAgent(st, "Turns", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	rt := startRuntime(t, st)
	if err := rt.Start(agent.ID, w.Path); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if _, err := st.EnqueueTask(agent.ID, store.TaskPrompt, "hello", "user"); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		act, err := st.AgentActivityOf(agent.ID)
		if err != nil {
			t.Fatal(err)
		}
		if act.Turns != nil && *act.Turns >= 1 && act.FirstWorkedAt != nil {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("agent_start did not count a turn")
}
