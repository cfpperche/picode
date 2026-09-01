package rpc

import (
	"strings"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/store"
)

// An attached RunObserver receives the settle and owns the Inbox: the
// default unobserved-result item is not filed (ADR-0046 on ADR-0037).
func TestObserverReceivesSettleAndSuppressesResult(t *testing.T) {
	st, rt, agent, path := inboxTestSetup(t)
	if err := rt.Start(agent.ID, path); err != nil {
		t.Fatalf("Start: %v", err)
	}
	settled := make(chan string, 1)
	exited := make(chan bool, 1)
	rt.Get(agent.ID).Observe(&RunObserver{
		OnSettled: func(final string) { settled <- final },
		OnExit:    func(expected bool) { exited <- expected },
	})
	if _, err := st.EnqueueTask(agent.ID, store.TaskPrompt, "hello", "user"); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	select {
	case final := <-settled:
		if !strings.Contains(final, "hello from fake") {
			t.Fatalf("final = %q", final)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("observer never saw the settle")
	}
	time.Sleep(100 * time.Millisecond)
	if items, _ := st.ListInboxItems(store.InboxFilter{IncludeSnoozed: true}); len(items) != 0 {
		t.Fatalf("default result item filed despite observer: %+v", items)
	}
	rt.Stop(agent.ID)
	select {
	case expected := <-exited:
		if !expected {
			t.Fatal("Stop must report an expected exit")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("observer never saw the exit")
	}
	if items, _ := st.ListInboxItems(store.InboxFilter{IncludeSnoozed: true}); len(items) != 0 {
		t.Fatalf("exit item filed despite observer: %+v", items)
	}
}
