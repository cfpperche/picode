package store

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCreateInboxItemDefaults(t *testing.T) {
	s := openTest(t)
	cases := []struct {
		kind     string
		body     string
		verbs    []string
		blocking bool
	}{
		{InboxFYI, "", []string{VerbIgnore}, false},
		{InboxResult, "done it", []string{VerbIgnore}, false},
		{InboxQuestion, "which db?", []string{VerbRespond, VerbIgnore}, true},
		{InboxApproval, "may I push?", []string{VerbAccept, VerbRespond, VerbIgnore}, true},
	}
	for _, tc := range cases {
		it, err := s.CreateInboxItem(InboxItemParams{
			Kind: tc.kind, SourceKind: InboxFromSystem, Reason: "test", Title: "t-" + tc.kind, Body: tc.body,
		})
		if err != nil {
			t.Fatalf("%s: %v", tc.kind, err)
		}
		if it.State != InboxUnread || it.Blocking != tc.blocking {
			t.Fatalf("%s: state=%s blocking=%v", tc.kind, it.State, it.Blocking)
		}
		if strings.Join(it.Allowed, ",") != strings.Join(tc.verbs, ",") {
			t.Fatalf("%s: verbs %v, want %v", tc.kind, it.Allowed, tc.verbs)
		}
	}
}

func TestInboxSessionPathRoundTrip(t *testing.T) {
	s := openTest(t)
	path := "/tmp/exact-session.jsonl"
	it, err := s.CreateInboxItem(InboxItemParams{
		Kind: InboxQuestion, SourceKind: InboxFromAgent, SourceID: "agent-a",
		Reason: "agent needs input", Title: "question", Body: "?", SessionPath: path,
	})
	if err != nil {
		t.Fatal(err)
	}
	if it.SessionPath != path {
		t.Fatalf("created sessionPath = %q", it.SessionPath)
	}
	got, err := s.GetInboxItem(it.ID)
	if err != nil || got.SessionPath != path {
		t.Fatalf("loaded item = %+v, %v", got, err)
	}
	list, err := s.ListInboxItems(InboxFilter{IncludeSnoozed: true})
	if err != nil || len(list) != 1 || list[0].SessionPath != path {
		t.Fatalf("listed items = %+v, %v", list, err)
	}
}

func TestCreateInboxItemRejects(t *testing.T) {
	s := openTest(t)
	bad := []InboxItemParams{
		{Kind: "video", SourceKind: InboxFromSystem, Reason: "r", Title: "t"},
		{Kind: InboxFYI, SourceKind: "ghost", Reason: "r", Title: "t"},
		{Kind: InboxFYI, SourceKind: InboxFromSystem, Reason: "", Title: "t"},
		{Kind: InboxFYI, SourceKind: InboxFromSystem, Reason: "r", Title: " "},
		{Kind: InboxQuestion, SourceKind: InboxFromAgent, Reason: "r", Title: "t", Body: " "},
		{Kind: InboxFYI, SourceKind: InboxFromSystem, Reason: "r", Title: "t", Allowed: []string{"maybe"}},
	}
	for i, p := range bad {
		if _, err := s.CreateInboxItem(p); err == nil {
			t.Fatalf("case %d: expected error, got nil", i)
		}
	}
}

func TestRespondInboxItem(t *testing.T) {
	s := openTest(t)
	it, _ := s.CreateInboxItem(InboxItemParams{Kind: InboxQuestion, SourceKind: InboxFromSystem, Reason: "r", Title: "q", Body: "?"})

	if _, err := s.RespondInboxItem(it.ID, VerbAccept, ""); err == nil {
		t.Fatalf("disallowed verb accepted")
	}
	got, err := s.RespondInboxItem(it.ID, VerbRespond, "use sqlite")
	if err != nil {
		t.Fatalf("respond: %v", err)
	}
	if got.State != InboxDone || got.Response == nil || !strings.Contains(*got.Response, "use sqlite") {
		t.Fatalf("responded item = %+v", got)
	}
	if _, err := s.RespondInboxItem(it.ID, VerbRespond, "again"); err == nil {
		t.Fatalf("double respond accepted")
	}
}

func TestInboxSnooze(t *testing.T) {
	s := openTest(t)
	it, _ := s.CreateInboxItem(InboxItemParams{Kind: InboxFYI, SourceKind: InboxFromSystem, Reason: "r", Title: "zzz"})

	future := time.Now().UTC().Add(time.Hour).Format(time.RFC3339)
	if _, err := s.SetInboxItemState(it.ID, "", &future); err != nil {
		t.Fatalf("snooze: %v", err)
	}
	list, _ := s.ListInboxItems(InboxFilter{})
	if len(list) != 0 {
		t.Fatalf("snoozed item still listed: %v", list)
	}
	all, _ := s.ListInboxItems(InboxFilter{IncludeSnoozed: true})
	if len(all) != 1 {
		t.Fatalf("IncludeSnoozed missed it")
	}
	// A due snooze (sub-second boundary: stored at second precision).
	past := time.Now().UTC().Add(-time.Second).Format(time.RFC3339)
	if _, err := s.SetInboxItemState(it.ID, "", &past); err != nil {
		t.Fatalf("re-snooze: %v", err)
	}
	list, _ = s.ListInboxItems(InboxFilter{})
	if len(list) != 1 {
		t.Fatalf("due item hidden")
	}
}

func TestInboxBadge(t *testing.T) {
	s := openTest(t)
	blocking, other, err := s.CountInboxBadge()
	if err != nil || blocking != 0 || other {
		t.Fatalf("empty badge = %d %v %v", blocking, other, err)
	}
	q, _ := s.CreateInboxItem(InboxItemParams{Kind: InboxQuestion, SourceKind: InboxFromSystem, Reason: "r", Title: "q", Body: "?"})
	_, _ = s.CreateInboxItem(InboxItemParams{Kind: InboxFYI, SourceKind: InboxFromSystem, Reason: "r", Title: "n"})
	blocking, other, _ = s.CountInboxBadge()
	if blocking != 1 || !other {
		t.Fatalf("badge = %d %v, want 1 true", blocking, other)
	}
	// Responding clears the blocking count.
	if _, err := s.RespondInboxItem(q.ID, VerbRespond, "ok"); err != nil {
		t.Fatalf("respond: %v", err)
	}
	blocking, other, _ = s.CountInboxBadge()
	if blocking != 0 || !other {
		t.Fatalf("badge after respond = %d %v", blocking, other)
	}
}

func TestFileAgentResultSupersedes(t *testing.T) {
	s := openTest(t)
	a, err := s.FileAgentResult("ag1", "ws1", "run 1 done", "first", "run finished unobserved")
	if err != nil {
		t.Fatalf("file: %v", err)
	}
	b, err := s.FileAgentResult("ag1", "ws1", "run 2 done", "second", "run finished unobserved")
	if err != nil {
		t.Fatalf("file 2: %v", err)
	}
	if a.ID != b.ID || b.Title != "run 2 done" || b.Body != "second" {
		t.Fatalf("unread result not superseded: %+v vs %+v", a, b)
	}
	// A read item is history — the next run files fresh.
	if _, err := s.SetInboxItemState(a.ID, InboxRead, nil); err != nil {
		t.Fatalf("read: %v", err)
	}
	c, _ := s.FileAgentResult("ag1", "ws1", "run 3 done", "third", "run finished unobserved")
	if c.ID == a.ID {
		t.Fatalf("read item was overwritten")
	}
	// Different agent never collides.
	d, _ := s.FileAgentResult("ag2", "", "other", "x", "run finished unobserved")
	if d.ID == c.ID {
		t.Fatalf("cross-agent supersede")
	}
}

func TestAnnotateInboxItem(t *testing.T) {
	s := openTest(t)
	it, _ := s.CreateInboxItem(InboxItemParams{Kind: InboxQuestion, SourceKind: InboxFromAgent, SourceID: "gone", Reason: "r", Title: "q", Body: "?"})
	if err := s.AnnotateInboxItem(it.ID, "Reply could not be delivered."); err != nil {
		t.Fatalf("annotate: %v", err)
	}
	got, _ := s.GetInboxItem(it.ID)
	if !strings.Contains(got.Body, "> Reply could not be delivered.") {
		t.Fatalf("annotation missing: %q", got.Body)
	}
	if err := s.AnnotateInboxItem("nope", "x"); err != ErrNotFound {
		t.Fatalf("annotate missing = %v", err)
	}
}

func TestRespondAndForward(t *testing.T) {
	s := openTest(t)
	// Real agent: reply becomes a queued follow_up task with source inbox.
	ws, err := s.AddWorkspace("wsx", t.TempDir())
	if err != nil {
		t.Fatalf("ws: %v", err)
	}
	ag, err := s.AddAgent(ws.ID, "helper", "")
	if err != nil {
		t.Fatalf("agent: %v", err)
	}
	q, _ := s.CreateInboxItem(InboxItemParams{Kind: InboxQuestion, SourceKind: InboxFromAgent, SourceID: ag.ID, Reason: "agent needs your input", Title: "which port?", Body: "8080 or 8445?"})
	got, err := s.RespondAndForward(q.ID, VerbRespond, "8445", nil)
	if err != nil {
		t.Fatalf("respond+forward: %v", err)
	}
	if got.State != InboxDone {
		t.Fatalf("item not done")
	}
	tasks, _ := s.ListTasks(ag.ID, 10)
	if len(tasks) != 1 || tasks[0].Kind != TaskFollowUp || tasks[0].Source != "inbox" || !strings.Contains(tasks[0].Payload, "8445") {
		t.Fatalf("task = %+v", tasks)
	}

	// Ignore never forwards.
	q2, _ := s.CreateInboxItem(InboxItemParams{Kind: InboxQuestion, SourceKind: InboxFromAgent, SourceID: ag.ID, Reason: "r", Title: "q2", Body: "?"})
	if _, err := s.RespondAndForward(q2.ID, VerbIgnore, "", nil); err != nil {
		t.Fatalf("ignore: %v", err)
	}
	if tasks, _ := s.ListTasks(ag.ID, 10); len(tasks) != 1 {
		t.Fatalf("ignore forwarded a task")
	}

	// Dead agent: annotated, still open, error surfaces.
	q3, _ := s.CreateInboxItem(InboxItemParams{Kind: InboxQuestion, SourceKind: InboxFromAgent, SourceID: "ghost-agent", Reason: "r", Title: "q3", Body: "?"})
	if _, err := s.RespondAndForward(q3.ID, VerbRespond, "hello", nil); err == nil {
		t.Fatalf("dead agent forward succeeded")
	}
	after, _ := s.GetInboxItem(q3.ID)
	if after.State == InboxDone || !strings.Contains(after.Body, "agent no longer exists") {
		t.Fatalf("dead-agent item = %+v", after)
	}
}

// TestRespondAndForwardInteractiveAgent covers the park-and-wake gap found
// live: a follow_up queued for an agent running in a TUI/tmux session is
// never drained (deliverLoop only exists for the RPC runtime). The
// deliverable callback must refuse before enqueueing — never a silent
// task that sits forever — and leave the item open with an actionable note.
func TestRespondAndForwardInteractiveAgent(t *testing.T) {
	s := openTest(t)
	ws, _ := s.AddWorkspace("wsx", t.TempDir())
	ag, err := s.AddAgent(ws.ID, "helper", "")
	if err != nil {
		t.Fatalf("agent: %v", err)
	}
	q, _ := s.CreateInboxItem(InboxItemParams{Kind: InboxQuestion, SourceKind: InboxFromAgent, SourceID: ag.ID, Reason: "r", Title: "q", Body: "?"})

	interactive := func(id string) bool { return false } // "not deliverable"
	if _, err := s.RespondAndForward(q.ID, VerbRespond, "hi", interactive); !errors.Is(err, ErrAgentInteractive) {
		t.Fatalf("respond to interactive agent = %v, want ErrAgentInteractive", err)
	}
	after, _ := s.GetInboxItem(q.ID)
	if after.State == InboxDone {
		t.Fatalf("interactive-agent item marked done")
	}
	if !strings.Contains(after.Body, "interactive terminal") {
		t.Fatalf("no actionable annotation: %q", after.Body)
	}
	if tasks, _ := s.ListTasks(ag.ID, 10); len(tasks) != 0 {
		t.Fatalf("a task was queued despite the agent being undeliverable: %+v", tasks)
	}

	// Ignore never forwards, so it must not be gated by deliverability.
	q2, _ := s.CreateInboxItem(InboxItemParams{Kind: InboxQuestion, SourceKind: InboxFromAgent, SourceID: ag.ID, Reason: "r", Title: "q2", Body: "?"})
	if _, err := s.RespondAndForward(q2.ID, VerbIgnore, "", interactive); err != nil {
		t.Fatalf("ignore gated by deliverability: %v", err)
	}

	// The same agent, once reachable, delivers normally.
	deliverable := func(id string) bool { return true }
	q3, _ := s.CreateInboxItem(InboxItemParams{Kind: InboxQuestion, SourceKind: InboxFromAgent, SourceID: ag.ID, Reason: "r", Title: "q3", Body: "?"})
	if _, err := s.RespondAndForward(q3.ID, VerbRespond, "hi", deliverable); err != nil {
		t.Fatalf("respond to deliverable agent: %v", err)
	}
	if tasks, _ := s.ListTasks(ag.ID, 10); len(tasks) != 1 {
		t.Fatalf("reachable agent did not get its task: %+v", tasks)
	}
}

func TestRespondAndParkReturnsAndClaimsOnlyExactTask(t *testing.T) {
	s := openTest(t)
	ag, _ := s.AddAgent(FreeWorkspaceID, "helper", t.TempDir())
	unrelated, _ := s.EnqueueTask(ag.ID, TaskPrompt, "older work", "user")
	q, _ := s.CreateInboxItem(InboxItemParams{Kind: InboxQuestion, SourceKind: InboxFromAgent, SourceID: ag.ID, Reason: "r", Title: "q", Body: "?"})
	item, parked, err := s.RespondAndPark(q.ID, VerbRespond, "exact reply")
	if err != nil {
		t.Fatal(err)
	}
	if item.State != InboxDone || parked.ID == "" || parked.Kind != TaskFollowUp || parked.AgentID != ag.ID {
		t.Fatalf("respond-and-park = item %+v task %+v", item, parked)
	}
	claimed, err := s.ClaimTask(ag.ID, parked.ID)
	if err != nil || claimed.ID != parked.ID || claimed.Status != TaskDelivering || claimed.Attempts != 1 {
		t.Fatalf("exact claim = %+v, %v", claimed, err)
	}
	tasks, _ := s.ListTasks(ag.ID, 10)
	for _, task := range tasks {
		if task.ID == unrelated.ID && task.Status != TaskQueued {
			t.Fatalf("unrelated task was drained: %+v", task)
		}
	}
	if err := s.FinishTask(claimed.ID, TaskQueued, "retry"); err != nil {
		t.Fatal(err)
	}
	retry, err := s.ClaimTask(ag.ID, parked.ID)
	if err != nil || retry.Attempts != 2 {
		t.Fatalf("retry claim = %+v, %v", retry, err)
	}
	if err := s.FinishTask(retry.ID, TaskFailed, "test cleanup"); err != nil {
		t.Fatal(err)
	}
	approval, _ := s.CreateInboxItem(InboxItemParams{Kind: InboxApproval, SourceKind: InboxFromAgent, SourceID: ag.ID, Reason: "r", Title: "ship this", Body: "?"})
	_, declined, err := s.RespondAndPark(approval.ID, VerbRespond, "Declined.")
	if err != nil || declined.Payload != `Human reply to your question "ship this": Declined.` {
		t.Fatalf("decline payload = %q, %v", declined.Payload, err)
	}
}

func TestInterruptedBurstRecoveryReopensExactInboxItem(t *testing.T) {
	for _, claim := range []bool{false, true} {
		name := "queued"
		if claim {
			name = "delivering"
		}
		t.Run(name, func(t *testing.T) {
			path := t.TempDir() + "/picode.db"
			s, err := Open(path)
			if err != nil {
				t.Fatal(err)
			}
			ag, _ := s.AddAgent(FreeWorkspaceID, "helper", t.TempDir())
			q, _ := s.CreateInboxItem(InboxItemParams{Kind: InboxQuestion, SourceKind: InboxFromAgent, SourceID: ag.ID, Reason: "r", Title: "q", Body: "?"})
			_, task, err := s.RespondAndPark(q.ID, VerbRespond, "please continue")
			if err != nil {
				t.Fatal(err)
			}
			if claim {
				if _, err := s.ClaimTask(ag.ID, task.ID); err != nil {
					t.Fatal(err)
				}
			}
			if err := s.Close(); err != nil {
				t.Fatal(err)
			}
			s, err = Open(path)
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			item, _ := s.GetInboxItem(q.ID)
			if item.State != InboxUnread || item.Response == nil || !strings.Contains(*item.Response, "please continue") || !strings.Contains(item.Body, "interrupted") {
				t.Fatalf("recovered item = %+v", item)
			}
			tasks, _ := s.ListTasks(ag.ID, 10)
			if len(tasks) != 1 || tasks[0].Status != TaskFailed || tasks[0].LastError == nil || !strings.Contains(*tasks[0].LastError, "interrupted") {
				t.Fatalf("recovered task = %+v", tasks)
			}
		})
	}
}

func TestInterruptedBurstRecoveryRecognizesMaterializedDeliveringReply(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "picode.db")
	sessionPath := filepath.Join(dir, "session.jsonl")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	ag, _ := s.AddAgent(FreeWorkspaceID, "helper", t.TempDir())
	q, _ := s.CreateInboxItem(InboxItemParams{
		Kind: InboxQuestion, SourceKind: InboxFromAgent, SourceID: ag.ID,
		Reason: "r", Title: "q", Body: "?", SessionPath: sessionPath,
	})
	_, task, err := s.RespondAndPark(q.ID, VerbRespond, "materialized before crash")
	if err != nil {
		t.Fatal(err)
	}
	claimed, err := s.ClaimTask(ag.ID, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	created, _ := time.Parse(time.RFC3339Nano, claimed.CreatedAt)
	row, _ := json.Marshal(map[string]any{
		"type": "message", "timestamp": created.Add(time.Millisecond).Format(time.RFC3339Nano),
		"message": map[string]any{"role": "user", "content": []map[string]string{{"type": "text", "text": claimed.Payload}}},
	})
	if err := os.WriteFile(sessionPath, append(row, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}

	s, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	item, _ := s.GetInboxItem(q.ID)
	if item.State != InboxDone || strings.Contains(item.Body, "interrupted") {
		t.Fatalf("materialized reply was reopened: %+v", item)
	}
	tasks, _ := s.ListTasks(ag.ID, 10)
	if len(tasks) != 1 || tasks[0].Status != TaskDelivered {
		t.Fatalf("materialized task was not recovered as delivered: %+v", tasks)
	}
}

func TestBurstTaskMaterializedRejectsMatchingRowOlderThanTask(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.jsonl")
	created := time.Now().UTC()
	payload := "Human reply to your question: repeated"
	row, _ := json.Marshal(map[string]any{
		"type": "message", "timestamp": created.Truncate(time.Millisecond).Add(-time.Millisecond).Format(time.RFC3339Nano),
		"message": map[string]any{"role": "user", "content": []map[string]string{{"type": "text", "text": payload}}},
	})
	if err := os.WriteFile(path, append(row, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	if replyTaskMaterialized(path, payload, created.Format(time.RFC3339Nano)) {
		t.Fatal("an older identical reply proved crash-time delivery")
	}
	newerLonger, _ := json.Marshal(map[string]any{
		"type": "message", "timestamp": created.Add(time.Second).Format(time.RFC3339Nano),
		"message": map[string]any{"role": "user", "content": []map[string]string{{"type": "text", "text": payload + " with unrelated suffix"}}},
	})
	if err := os.WriteFile(path, append(append(row, '\n'), append(newerLonger, '\n')...), 0o600); err != nil {
		t.Fatal(err)
	}
	if replyTaskMaterialized(path, payload, created.Format(time.RFC3339Nano)) {
		t.Fatal("a longer user message was mistaken for the exact reply")
	}
}

func TestBurstTaskMaterializedContinuesPastLargeJSONLRow(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.jsonl")
	created := time.Now().UTC()
	payload := "exact reply after a large tool result"
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(`{"type":"message","message":{"role":"assistant","content":[{"type":"text","text":"`); err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(strings.Repeat("x", 17*1024*1024)); err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(`"}]}}` + "\n"); err != nil {
		t.Fatal(err)
	}
	row, _ := json.Marshal(map[string]any{
		"type": "message", "timestamp": created.Add(time.Millisecond).Format(time.RFC3339Nano),
		"message": map[string]any{"role": "user", "content": []map[string]string{{"type": "text", "text": payload}}},
	})
	if _, err := f.Write(append(row, '\n')); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if !replyTaskMaterialized(path, payload, created.Format(time.RFC3339Nano)) {
		t.Fatal("large preceding JSONL row hid the durable reply")
	}
}

func TestInterruptedBurstRecoveryDoesNotReopenDeliveredReply(t *testing.T) {
	path := t.TempDir() + "/picode.db"
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	ag, _ := s.AddAgent(FreeWorkspaceID, "helper", t.TempDir())
	q, _ := s.CreateInboxItem(InboxItemParams{Kind: InboxQuestion, SourceKind: InboxFromAgent, SourceID: ag.ID, Reason: "r", Title: "q", Body: "?"})
	_, task, err := s.RespondAndPark(q.ID, VerbRespond, "already materialized")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.ClaimTask(ag.ID, task.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.FinishTask(task.ID, TaskDelivered, ""); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}

	s, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	item, _ := s.GetInboxItem(q.ID)
	if item.State != InboxDone || strings.Contains(item.Body, "interrupted") {
		t.Fatalf("delivered reply was reopened after restart: %+v", item)
	}
	tasks, _ := s.ListTasks(ag.ID, 10)
	if len(tasks) != 1 || tasks[0].Status != TaskDelivered {
		t.Fatalf("delivered task changed during recovery: %+v", tasks)
	}
}

func TestDeleteInboxItem(t *testing.T) {
	s := openTest(t)
	it, _ := s.CreateInboxItem(InboxItemParams{Kind: InboxFYI, SourceKind: InboxFromSystem, Reason: "r", Title: "gone soon"})
	if err := s.DeleteInboxItem(it.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := s.GetInboxItem(it.ID); err != ErrNotFound {
		t.Fatalf("get after delete = %v, want ErrNotFound", err)
	}
	if err := s.DeleteInboxItem(it.ID); err != ErrNotFound {
		t.Fatalf("delete missing = %v, want ErrNotFound", err)
	}
	// Deleting works regardless of state — the store doesn't gate this,
	// the UI decides when to expose the action.
	active, _ := s.CreateInboxItem(InboxItemParams{Kind: InboxQuestion, SourceKind: InboxFromSystem, Reason: "r", Title: "still open", Body: "?"})
	if err := s.DeleteInboxItem(active.ID); err != nil {
		t.Fatalf("delete active item: %v", err)
	}
}

func TestDeleteDoneInboxItems(t *testing.T) {
	s := openTest(t)
	done1, _ := s.CreateInboxItem(InboxItemParams{Kind: InboxFYI, SourceKind: InboxFromSystem, Reason: "r", Title: "d1"})
	done2, _ := s.CreateInboxItem(InboxItemParams{Kind: InboxFYI, SourceKind: InboxFromSystem, Reason: "r", Title: "d2"})
	active, _ := s.CreateInboxItem(InboxItemParams{Kind: InboxQuestion, SourceKind: InboxFromSystem, Reason: "r", Title: "still open", Body: "?"})
	if _, err := s.SetInboxItemState(done1.ID, InboxDone, nil); err != nil {
		t.Fatalf("mark done1: %v", err)
	}
	if _, err := s.SetInboxItemState(done2.ID, InboxDone, nil); err != nil {
		t.Fatalf("mark done2: %v", err)
	}

	n, err := s.DeleteDoneInboxItems()
	if err != nil {
		t.Fatalf("clear done: %v", err)
	}
	if n != 2 {
		t.Fatalf("cleared %d, want 2", n)
	}
	if _, err := s.GetInboxItem(done1.ID); err != ErrNotFound {
		t.Fatalf("done1 survived: %v", err)
	}
	if _, err := s.GetInboxItem(done2.ID); err != ErrNotFound {
		t.Fatalf("done2 survived: %v", err)
	}
	if _, err := s.GetInboxItem(active.ID); err != nil {
		t.Fatalf("active item was deleted: %v", err)
	}
	// Idempotent: nothing left to clear.
	if n, err := s.DeleteDoneInboxItems(); err != nil || n != 0 {
		t.Fatalf("second clear = %d, %v, want 0, nil", n, err)
	}
}

func TestCountInboxItems(t *testing.T) {
	s := openTest(t)
	unread, _ := s.CreateInboxItem(InboxItemParams{Kind: InboxFYI, SourceKind: InboxFromSystem, Reason: "r", Title: "u"})
	done, _ := s.CreateInboxItem(InboxItemParams{Kind: InboxFYI, SourceKind: InboxFromSystem, Reason: "r", Title: "d"})
	if _, err := s.SetInboxItemState(done.ID, InboxDone, nil); err != nil {
		t.Fatalf("mark done: %v", err)
	}
	if _, err := s.SetInboxItemState(unread.ID, InboxRead, nil); err != nil {
		t.Fatalf("mark read: %v", err)
	}

	if n, err := s.CountInboxItems(InboxDone); err != nil || n != 1 {
		t.Fatalf("done count = %d, %v, want 1", n, err)
	}
	if n, err := s.CountInboxItems(InboxRead); err != nil || n != 1 {
		t.Fatalf("read count = %d, %v, want 1", n, err)
	}
	if n, err := s.CountInboxItems(InboxUnread); err != nil || n != 0 {
		t.Fatalf("unread count = %d, %v, want 0", n, err)
	}
}

func TestCountAllInboxItems(t *testing.T) {
	s := openTest(t)
	if n, err := s.CountAllInboxItems(); err != nil || n != 0 {
		t.Fatalf("empty count = %d, %v, want 0", n, err)
	}
	a, _ := s.CreateInboxItem(InboxItemParams{Kind: InboxFYI, SourceKind: InboxFromSystem, Reason: "r", Title: "a"})
	_, _ = s.CreateInboxItem(InboxItemParams{Kind: InboxFYI, SourceKind: InboxFromSystem, Reason: "r", Title: "b"})
	if _, err := s.SetInboxItemState(a.ID, InboxDone, nil); err != nil {
		t.Fatalf("mark done: %v", err)
	}
	// Counts everything regardless of state.
	if n, err := s.CountAllInboxItems(); err != nil || n != 2 {
		t.Fatalf("count = %d, %v, want 2", n, err)
	}
}
