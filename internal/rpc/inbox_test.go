package rpc

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/store"
)

func inboxTestSetup(t *testing.T) (*store.Store, *Runtime, store.Agent, string) {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "picode.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	w, agent, err := addWorkspaceWithAgent(st, "Inbox", t.TempDir())
	if err != nil {
		t.Fatalf("workspace: %v", err)
	}
	rt := startRuntime(t, st)
	return st, rt, agent, w.Path
}

func waitInbox(t *testing.T, st *store.Store, want int, d time.Duration) []store.InboxItem {
	t.Helper()
	deadline := time.Now().Add(d)
	for {
		items, err := st.ListInboxItems(store.InboxFilter{IncludeSnoozed: true})
		if err != nil {
			t.Fatalf("list inbox: %v", err)
		}
		if len(items) == want {
			return items
		}
		if time.Now().After(deadline) {
			t.Fatalf("inbox has %d items, want %d: %+v", len(items), want, items)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func waitTaskDone(t *testing.T, st *store.Store, agentID string, d time.Duration) {
	t.Helper()
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		tasks, _ := st.ListTasks(agentID, 5)
		if len(tasks) > 0 && tasks[0].Status == store.TaskDelivered {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("task never delivered")
}

func TestSettleUnobservedFilesResult(t *testing.T) {
	st, rt, agent, path := inboxTestSetup(t)
	if err := rt.Start(agent.ID, path); err != nil {
		t.Fatalf("Start: %v", err)
	}
	// No hub subscriber: the run is unobserved.
	if _, err := st.EnqueueTask(agent.ID, store.TaskPrompt, "hello", "user"); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	waitTaskDone(t, st, agent.ID, 10*time.Second)
	items := waitInbox(t, st, 1, 5*time.Second)
	it := items[0]
	if it.Kind != store.InboxResult || it.SourceID != agent.ID || it.State != store.InboxUnread {
		t.Fatalf("item = %+v", it)
	}
	if !strings.Contains(it.Body, "hello from fake") {
		t.Fatalf("result body lost the final message: %q", it.Body)
	}
	if !strings.Contains(it.Title, "finished") {
		t.Fatalf("title = %q", it.Title)
	}

	// A second unobserved run supersedes the unread item — still one row.
	if _, err := st.EnqueueTask(agent.ID, store.TaskPrompt, "again", "user"); err != nil {
		t.Fatalf("enqueue 2: %v", err)
	}
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		tasks, _ := st.ListTasks(agent.ID, 5)
		if len(tasks) == 2 && tasks[0].Status == store.TaskDelivered {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	time.Sleep(200 * time.Millisecond) // let the settle hook run
	items = waitInbox(t, st, 1, 3*time.Second)
	if items[0].ID != it.ID {
		t.Fatalf("unread result was not superseded")
	}
}

func TestSettleObservedFilesNothing(t *testing.T) {
	st, rt, agent, path := inboxTestSetup(t)
	if err := rt.Start(agent.ID, path); err != nil {
		t.Fatalf("Start: %v", err)
	}
	ma := rt.Get(agent.ID)
	hubCh, unsub := ma.hub.Subscribe() // someone is watching
	defer unsub()
	if _, err := st.EnqueueTask(agent.ID, store.TaskPrompt, "hello", "user"); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	waitHub(t, hubCh, "agent_settled", 10*time.Second)
	time.Sleep(200 * time.Millisecond)
	waitInbox(t, st, 0, time.Second)
}

func TestStopFilesNoFyi(t *testing.T) {
	st, rt, agent, path := inboxTestSetup(t)
	if err := rt.Start(agent.ID, path); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if !rt.Stop(agent.ID) {
		t.Fatalf("Stop = false")
	}
	waitInbox(t, st, 0, time.Second)
}

func TestUnexpectedExitFilesFyi(t *testing.T) {
	st, rt, agent, path := inboxTestSetup(t)
	if err := rt.Start(agent.ID, path); err != nil {
		t.Fatalf("Start: %v", err)
	}
	ma := rt.Get(agent.ID)
	// Close the client without Runtime.Stop: an unexpected death.
	ma.client.Close()
	items := waitInbox(t, st, 1, 5*time.Second)
	it := items[0]
	if it.Kind != store.InboxFYI || it.Reason != "process exited" || it.SourceID != agent.ID {
		t.Fatalf("fyi item = %+v", it)
	}
}
