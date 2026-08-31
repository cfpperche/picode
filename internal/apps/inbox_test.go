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
	if len(v.Blocks) != 1 || !strings.Contains(v.Blocks[0].Markdown, "Inbox zero") {
		t.Fatalf("empty root = %+v", v.Blocks)
	}

	q := mustItem(t, h, store.InboxItemParams{Kind: store.InboxQuestion, SourceKind: store.InboxFromSystem, Reason: "needs input", Title: "q", Body: "?"})
	f := mustItem(t, h, store.InboxItemParams{Kind: store.InboxFYI, SourceKind: store.InboxFromSystem, Reason: "heads up", Title: "n"})
	v, _ = app.View(context.Background(), h, "")
	if err := v.Validate(); err != nil {
		t.Fatalf("root invalid: %v", err)
	}
	// detail(Needs you) + list + detail(Feed) + list
	if len(v.Blocks) != 4 || v.Blocks[1].Type != "list" || v.Blocks[3].Type != "list" {
		t.Fatalf("root blocks = %+v", v.Blocks)
	}
	if v.Blocks[1].Items[0].ID != q.ID || v.Blocks[1].Items[0].Path != "item/"+q.ID {
		t.Fatalf("needs-me row = %+v", v.Blocks[1].Items)
	}
	feedRow := v.Blocks[3].Items[0]
	if feedRow.ID != f.ID || len(feedRow.Actions) != 2 {
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
}
