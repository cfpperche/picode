package apps

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/store"
)

func inboxHost(t *testing.T) Host {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "picode.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return Host{Store: st}
}

func mustItem(t *testing.T, h Host, p store.InboxItemParams) store.InboxItem {
	t.Helper()
	it, err := h.Store.CreateInboxItem(p)
	if err != nil {
		t.Fatalf("create item: %v", err)
	}
	return it
}

func TestInboxBadgeApp(t *testing.T) {
	h := inboxHost(t)
	app := inboxApp{}
	b, err := app.Badge(context.Background(), h)
	if err != nil || b.Count != 0 || b.Dot {
		t.Fatalf("empty badge = %+v %v", b, err)
	}
	mustItem(t, h, store.InboxItemParams{Kind: store.InboxQuestion, SourceKind: store.InboxFromSystem, Reason: "r", Title: "q", Body: "?"})
	mustItem(t, h, store.InboxItemParams{Kind: store.InboxFYI, SourceKind: store.InboxFromSystem, Reason: "r", Title: "n"})
	b, _ = app.Badge(context.Background(), h)
	if b.Count != 1 || !b.Dot {
		t.Fatalf("badge = %+v", b)
	}
}

func TestInboxRootView(t *testing.T) {
	h := inboxHost(t)
	app := inboxApp{}

	v, err := app.View(context.Background(), h, "")
	if err != nil {
		t.Fatalf("empty root: %v", err)
	}
	if err := v.Validate(); err != nil {
		t.Fatalf("empty root invalid: %v", err)
	}
	// Inbox zero drops the split and hands the host a blankslate line
	// instead of faking content with a markdown block.
	if v.Layout != "" || len(v.Blocks) != 0 || !strings.Contains(v.Empty, "Inbox zero") {
		t.Fatalf("empty root = %q %q %+v", v.Layout, v.Empty, v.Blocks)
	}

	q := mustItem(t, h, store.InboxItemParams{Kind: store.InboxQuestion, SourceKind: store.InboxFromSystem, Reason: "needs input", Title: "q", Body: "?"})
	f := mustItem(t, h, store.InboxItemParams{Kind: store.InboxFYI, SourceKind: store.InboxFromSystem, Reason: "heads up", Title: "n"})
	v, _ = app.View(context.Background(), h, "")
	if err := v.Validate(); err != nil {
		t.Fatalf("root invalid: %v", err)
	}
	// Split: two list panes titled by section, no markdown headers.
	if v.Layout != "split" || len(v.Blocks) != 2 {
		t.Fatalf("root blocks = %q %+v", v.Layout, v.Blocks)
	}
	needs, feed := v.Blocks[0], v.Blocks[1]
	if needs.Type != "list" || needs.Pane != "list" || needs.Title != "Needs you" {
		t.Fatalf("needs-me block = %+v", needs)
	}
	if feed.Title != "Feed" || feed.Pane != "list" {
		t.Fatalf("feed block = %+v", feed)
	}
	row := needs.Items[0]
	if row.ID != q.ID || row.Path != "item/"+q.ID {
		t.Fatalf("needs-me row = %+v", row)
	}
	// The row now carries what the host needs to draw a real row.
	if !row.Unread || row.Tone != "info" || row.Badge != store.InboxQuestion || row.At == "" || len(row.Meta) != 2 {
		t.Fatalf("row meta lost: %+v", row)
	}
	feedRow := feed.Items[0]
	if feedRow.ID != f.ID || len(feedRow.Actions) != 2 || feedRow.Actions[0].Icon == "" {
		t.Fatalf("feed row = %+v", feedRow)
	}
}

func TestInboxItemViews(t *testing.T) {
	h := inboxHost(t)
	app := inboxApp{}
	q := mustItem(t, h, store.InboxItemParams{Kind: store.InboxQuestion, SourceKind: store.InboxFromSystem, Reason: "needs input", Title: "q", Body: "pick a db"})
	n := mustItem(t, h, store.InboxItemParams{Kind: store.InboxFYI, SourceKind: store.InboxFromSystem, Reason: "heads up", Title: "note", Body: "text"})

	qv, err := app.View(context.Background(), h, "item/"+q.ID)
	if err != nil {
		t.Fatalf("question view: %v", err)
	}
	if err := qv.Validate(); err != nil {
		t.Fatalf("question view invalid: %v", err)
	}
	var hasForm, hasIgnore, hasAccept bool
	for _, b := range qv.Blocks {
		if b.Type == "form" && b.Form.ID == "respond" {
			hasForm = true
		}
		if b.Type == "actions" {
			for _, a := range b.Actions {
				if a.ID == "ignore" {
					hasIgnore = true
					if a.Confirm == "" || !a.Danger {
						t.Fatalf("blocking ignore should confirm+danger: %+v", a)
					}
				}
				if a.ID == "accept" {
					hasAccept = true
				}
			}
		}
	}
	if !hasForm || !hasIgnore || hasAccept {
		t.Fatalf("question view form=%v ignore=%v accept=%v", hasForm, hasIgnore, hasAccept)
	}
	// Opening marked it read.
	if got, _ := h.Store.GetInboxItem(q.ID); got.State != store.InboxRead {
		t.Fatalf("question not marked read: %s", got.State)
	}

	nv, _ := app.View(context.Background(), h, "item/"+n.ID)
	for _, b := range nv.Blocks {
		if b.Type == "form" {
			t.Fatalf("fyi has a form")
		}
	}
	if _, err := app.View(context.Background(), h, "item/nope"); err == nil {
		t.Fatalf("missing item view = nil error")
	}
	if _, err := app.View(context.Background(), h, "weird"); err == nil {
		t.Fatalf("weird path = nil error")
	}
}

func TestInboxActions(t *testing.T) {
	h := inboxHost(t)
	app := inboxApp{}
	ctx := context.Background()

	n := mustItem(t, h, store.InboxItemParams{Kind: store.InboxFYI, SourceKind: store.InboxFromSystem, Reason: "r", Title: "note"})
	res, err := app.Action(ctx, h, ActionRequest{Action: "done", Args: map[string]string{"item": n.ID}})
	if err != nil || res.View == nil {
		t.Fatalf("done = %+v %v", res, err)
	}
	if got, _ := h.Store.GetInboxItem(n.ID); got.State != store.InboxDone {
		t.Fatalf("done did not stick")
	}

	z := mustItem(t, h, store.InboxItemParams{Kind: store.InboxFYI, SourceKind: store.InboxFromSystem, Reason: "r", Title: "later"})
	if _, err := app.Action(ctx, h, ActionRequest{Action: "snooze", Args: map[string]string{"item": z.ID}}); err != nil {
		t.Fatalf("snooze: %v", err)
	}
	if list, _ := h.Store.ListInboxItems(store.InboxFilter{}); len(list) != 0 {
		t.Fatalf("snoozed item listed: %+v", list)
	}

	// Respond via form path: item id comes from req.Path.
	ws, _ := h.Store.AddWorkspace("wsx", t.TempDir())
	ag, _ := h.Store.AddAgent(ws.ID, "helper", "")
	q := mustItem(t, h, store.InboxItemParams{Kind: store.InboxQuestion, SourceKind: store.InboxFromAgent, SourceID: ag.ID, Reason: "r", Title: "port?", Body: "?"})
	res, err = app.Action(ctx, h, ActionRequest{Action: "respond", Path: "item/" + q.ID, Args: map[string]string{"reply": "8445"}})
	if err != nil || !strings.Contains(res.Toast, "Reply sent") {
		t.Fatalf("respond = %+v %v", res, err)
	}
	tasks, _ := h.Store.ListTasks(ag.ID, 5)
	if len(tasks) != 1 || tasks[0].Source != "inbox" {
		t.Fatalf("task = %+v", tasks)
	}

	// Empty reply refused; no item id refused; dead agent stays open.
	q2 := mustItem(t, h, store.InboxItemParams{Kind: store.InboxQuestion, SourceKind: store.InboxFromAgent, SourceID: "ghost", Reason: "r", Title: "q2", Body: "?"})
	if _, err := app.Action(ctx, h, ActionRequest{Action: "respond", Path: "item/" + q2.ID, Args: map[string]string{"reply": " "}}); err == nil {
		t.Fatalf("empty reply accepted")
	}
	if _, err := app.Action(ctx, h, ActionRequest{Action: "respond", Args: map[string]string{"reply": "x"}}); err == nil {
		t.Fatalf("missing item accepted")
	}
	if _, err := app.Action(ctx, h, ActionRequest{Action: "respond", Path: "item/" + q2.ID, Args: map[string]string{"reply": "hi"}}); err == nil || !strings.Contains(err.Error(), "no longer exists") {
		t.Fatalf("dead agent error = %v", err)
	}
	if got, _ := h.Store.GetInboxItem(q2.ID); got.State == store.InboxDone {
		t.Fatalf("dead-agent item closed")
	}

	// An agent running in a TUI has no delivery loop draining follow_up —
	// the Action handler must refuse via h.AgentDeliverable, not enqueue
	// a reply that sits forever (the gap found live with grok-a7f396).
	h.AgentDeliverable = func(string) bool { return false }
	q3 := mustItem(t, h, store.InboxItemParams{Kind: store.InboxQuestion, SourceKind: store.InboxFromAgent, SourceID: ag.ID, Reason: "r", Title: "q3", Body: "?"})
	if _, err := app.Action(ctx, h, ActionRequest{Action: "respond", Path: "item/" + q3.ID, Args: map[string]string{"reply": "hi"}}); err == nil || !strings.Contains(err.Error(), "interactive terminal") {
		t.Fatalf("interactive-agent error = %v", err)
	}
	if got, _ := h.Store.GetInboxItem(q3.ID); got.State == store.InboxDone {
		t.Fatalf("interactive-agent item closed")
	}
	if tasks, _ := h.Store.ListTasks(ag.ID, 10); len(tasks) != 1 { // still just the earlier successful one
		t.Fatalf("interactive agent got a queued task: %+v", tasks)
	}
}
