package store

import (
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
	got, err := s.RespondAndForward(q.ID, VerbRespond, "8445")
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
	if _, err := s.RespondAndForward(q2.ID, VerbIgnore, ""); err != nil {
		t.Fatalf("ignore: %v", err)
	}
	if tasks, _ := s.ListTasks(ag.ID, 10); len(tasks) != 1 {
		t.Fatalf("ignore forwarded a task")
	}

	// Dead agent: annotated, still open, error surfaces.
	q3, _ := s.CreateInboxItem(InboxItemParams{Kind: InboxQuestion, SourceKind: InboxFromAgent, SourceID: "ghost-agent", Reason: "r", Title: "q3", Body: "?"})
	if _, err := s.RespondAndForward(q3.ID, VerbRespond, "hello"); err == nil {
		t.Fatalf("dead agent forward succeeded")
	}
	after, _ := s.GetInboxItem(q3.ID)
	if after.State == InboxDone || !strings.Contains(after.Body, "agent no longer exists") {
		t.Fatalf("dead-agent item = %+v", after)
	}
}
