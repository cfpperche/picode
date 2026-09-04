package apps

import (
	"context"
	"fmt"
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
	// Active-empty drops the split and hands the host a blankslate line
	// instead of faking content with a markdown block. Copy is scoped to
	// Active specifically ("nothing NEEDS you"), not "Inbox zero" — Done
	// may hold real history even when the queue is clear.
	if v.Layout != "" || len(v.Blocks) != 0 || !strings.Contains(v.Empty, "Nothing needs you") {
		t.Fatalf("empty root = %q %q %+v", v.Layout, v.Empty, v.Blocks)
	}
	// Tabs travel with every view, even an empty one.
	if len(v.Tabs) != 3 || v.Tabs[0].ID != "active" || v.Tabs[1].ID != "done" || v.Tabs[2].ID != "all" {
		t.Fatalf("tabs on empty root = %+v", v.Tabs)
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

	// ADR-0060: there is no Open terminal escape hatch on the card — the
	// reply itself goes into the agent's terminal.
	detail, err := app.View(ctx, h, "item/"+q3.ID)
	if err != nil {
		t.Fatalf("interactive detail: %v", err)
	}
	for _, b := range detail.Blocks {
		if b.Type != "actions" {
			continue
		}
		for _, act := range b.Actions {
			if act.ID == "open-terminal" {
				t.Fatalf("interactive item still offers Open terminal: %+v", detail.Blocks)
			}
		}
	}
}

func TestInboxDoneView(t *testing.T) {
	h := inboxHost(t)
	app := inboxApp{}

	// Empty Done: blankslate, tabs still present with a zero (omitted) badge.
	v, err := app.View(context.Background(), h, "done")
	if err != nil {
		t.Fatalf("empty done: %v", err)
	}
	if err := v.Validate(); err != nil {
		t.Fatalf("empty done invalid: %v", err)
	}
	if v.Layout != "" || !strings.Contains(v.Empty, "No answered items") {
		t.Fatalf("empty done = %q %q", v.Layout, v.Empty)
	}
	if v.Tabs[1].Badge != "" {
		t.Fatalf("done badge on empty = %q, want omitted", v.Tabs[1].Badge)
	}

	q := mustItem(t, h, store.InboxItemParams{Kind: store.InboxQuestion, SourceKind: store.InboxFromSystem, Reason: "r", Title: "answered one", Body: "?"})
	if _, err := h.Store.RespondInboxItem(q.ID, store.VerbRespond, "8445"); err != nil {
		t.Fatalf("respond: %v", err)
	}
	active := mustItem(t, h, store.InboxItemParams{Kind: store.InboxFYI, SourceKind: store.InboxFromSystem, Reason: "r", Title: "still open"})

	v, err = app.View(context.Background(), h, "done")
	if err != nil {
		t.Fatalf("done view: %v", err)
	}
	if err := v.Validate(); err != nil {
		t.Fatalf("done view invalid: %v", err)
	}
	if v.Tabs[1].Badge != "1" {
		t.Fatalf("done badge = %q, want 1", v.Tabs[1].Badge)
	}
	// No title on this block (ADR-0036 amendment): the Done tab pill
	// already says so, so a second "Done" line right under it was
	// pure duplication.
	if len(v.Blocks) != 2 || v.Blocks[0].Type != "list" || v.Blocks[0].Title != "" {
		t.Fatalf("done blocks = %+v", v.Blocks)
	}
	row := v.Blocks[0].Items[0]
	if row.ID != q.ID {
		t.Fatalf("done row = %+v, want %s", row, q.ID)
	}
	// The response itself is visible in the row without opening it.
	found := false
	for _, m := range row.Meta {
		if strings.Contains(m, "8445") {
			found = true
		}
	}
	if !found {
		t.Fatalf("response not visible in row meta: %+v", row.Meta)
	}
	if len(row.Actions) != 1 || row.Actions[0].ID != "delete" {
		t.Fatalf("done row actions = %+v", row.Actions)
	}
	// The still-open item never appears in Done.
	for _, it := range v.Blocks[0].Items {
		if it.ID == active.ID {
			t.Fatalf("active item leaked into done view")
		}
	}
	// Bulk-clear action block, with the count baked into copy.
	acts := v.Blocks[1]
	if acts.Type != "actions" || acts.Pane != "list" || len(acts.Actions) != 1 {
		t.Fatalf("clear-done block = %+v", acts)
	}
	clear := acts.Actions[0]
	if clear.ID != "clear-done" || !clear.Danger || !strings.Contains(clear.Confirm, "1") {
		t.Fatalf("clear-done action = %+v", clear)
	}
}

func TestInboxAllView(t *testing.T) {
	h := inboxHost(t)
	app := inboxApp{}

	q := mustItem(t, h, store.InboxItemParams{Kind: store.InboxQuestion, SourceKind: store.InboxFromSystem, Reason: "r", Title: "needs you", Body: "?"})
	f := mustItem(t, h, store.InboxItemParams{Kind: store.InboxFYI, SourceKind: store.InboxFromSystem, Reason: "r", Title: "feed item"})
	d := mustItem(t, h, store.InboxItemParams{Kind: store.InboxFYI, SourceKind: store.InboxFromSystem, Reason: "r", Title: "history item"})
	if _, err := h.Store.SetInboxItemState(d.ID, store.InboxDone, nil); err != nil {
		t.Fatalf("mark done: %v", err)
	}

	v, err := app.View(context.Background(), h, "all")
	if err != nil {
		t.Fatalf("all view: %v", err)
	}
	if err := v.Validate(); err != nil {
		t.Fatalf("all view invalid: %v", err)
	}
	// Needs you + Feed + Done, same three-section shape as root+done combined.
	if len(v.Blocks) != 3 {
		t.Fatalf("all blocks = %+v", v.Blocks)
	}
	titles := []string{v.Blocks[0].Title, v.Blocks[1].Title, v.Blocks[2].Title}
	if titles[0] != "Needs you" || titles[1] != "Feed" || titles[2] != "Done" {
		t.Fatalf("all block titles = %v", titles)
	}
	ids := map[string]bool{}
	for _, b := range v.Blocks {
		for _, it := range b.Items {
			ids[it.ID] = true
		}
	}
	if !ids[q.ID] || !ids[f.ID] || !ids[d.ID] {
		t.Fatalf("all view missing an item: %+v", ids)
	}
}

func TestInboxItemViewMirrorsDoneList(t *testing.T) {
	h := inboxHost(t)
	app := inboxApp{}

	active := mustItem(t, h, store.InboxItemParams{Kind: store.InboxFYI, SourceKind: store.InboxFromSystem, Reason: "r", Title: "active"})
	done := mustItem(t, h, store.InboxItemParams{Kind: store.InboxFYI, SourceKind: store.InboxFromSystem, Reason: "r", Title: "done"})
	if _, err := h.Store.SetInboxItemState(done.ID, store.InboxDone, nil); err != nil {
		t.Fatalf("mark done: %v", err)
	}

	// Opening a done item's detail must show the Done list beside it, not
	// snap back to Active — the exact inconsistency this plan fixes.
	v, err := app.View(context.Background(), h, "item/"+done.ID)
	if err != nil {
		t.Fatalf("done item view: %v", err)
	}
	if err := v.Validate(); err != nil {
		t.Fatalf("done item view invalid: %v", err)
	}
	// The done pane carries no title (ADR-0036 amendment: it would just
	// repeat the Done tab pill), so identify it by its own row instead.
	sawDoneList, sawActiveItem := false, false
	for _, b := range v.Blocks {
		if b.Pane != "list" {
			continue
		}
		for _, it := range b.Items {
			if it.ID == done.ID {
				sawDoneList = true
			}
			if it.ID == active.ID {
				sawActiveItem = true
			}
		}
	}
	if !sawDoneList {
		t.Fatalf("done item's detail did not show the Done list: %+v", v.Blocks)
	}
	if sawActiveItem {
		t.Fatalf("done item's detail leaked the Active item into its list pane")
	}
	// Tabs travel to the item detail too (best-effort chrome).
	if len(v.Tabs) != 3 {
		t.Fatalf("item view tabs = %+v", v.Tabs)
	}
	// The done item's own detail now offers delete.
	var hasDelete bool
	for _, b := range v.Blocks {
		if b.Pane != "detail" || b.Type != "actions" {
			continue
		}
		for _, a := range b.Actions {
			if a.ID == "delete" {
				hasDelete = true
			}
		}
	}
	if !hasDelete {
		t.Fatalf("done item detail has no delete action: %+v", v.Blocks)
	}

	// An active item's detail keeps showing the Active lists, unchanged.
	v2, err := app.View(context.Background(), h, "item/"+active.ID)
	if err != nil {
		t.Fatalf("active item view: %v", err)
	}
	for _, b := range v2.Blocks {
		for _, it := range b.Items {
			if it.ID == done.ID {
				t.Fatalf("active item's detail pulled in the Done list unexpectedly")
			}
		}
	}
}

func TestInboxDeleteAction(t *testing.T) {
	h := inboxHost(t)
	app := inboxApp{}
	ctx := context.Background()

	done := mustItem(t, h, store.InboxItemParams{Kind: store.InboxFYI, SourceKind: store.InboxFromSystem, Reason: "r", Title: "gone"})
	if _, err := h.Store.SetInboxItemState(done.ID, store.InboxDone, nil); err != nil {
		t.Fatalf("mark done: %v", err)
	}

	// Fired from the Done tab (row action): stays on Done afterward.
	res, err := app.Action(ctx, h, ActionRequest{Action: "delete", Path: "done", Args: map[string]string{"item": done.ID}})
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	if res.Toast != "Item deleted" {
		t.Fatalf("toast = %q", res.Toast)
	}
	if res.View == nil || len(res.View.Blocks) != 0 {
		t.Fatalf("expected an empty Done view back, got %+v", res.View)
	}
	if _, err := h.Store.GetInboxItem(done.ID); err != store.ErrNotFound {
		t.Fatalf("item survived delete: %v", err)
	}

	// Fired from the item's own detail: collapses to Active root, same
	// as every other item-detail action (unchanged existing behavior).
	done2 := mustItem(t, h, store.InboxItemParams{Kind: store.InboxFYI, SourceKind: store.InboxFromSystem, Reason: "r", Title: "gone2"})
	if _, err := h.Store.SetInboxItemState(done2.ID, store.InboxDone, nil); err != nil {
		t.Fatalf("mark done2: %v", err)
	}
	res, err = app.Action(ctx, h, ActionRequest{Action: "delete", Path: "item/" + done2.ID, Args: map[string]string{"item": done2.ID}})
	if err != nil {
		t.Fatalf("delete from detail: %v", err)
	}
	if res.View == nil || res.View.Layout != "" {
		t.Fatalf("delete from detail should return Active root: %+v", res.View)
	}
}

// TestInboxDeleteSiblingRowWhileDetailOpen is the regression for a bug
// caught live: deleting a DIFFERENT item's row (a sibling in the list
// pane) while an item's own detail is open used to collapse to Active
// root — path alone ("item/<open-id>") looked identical to "acting on
// the open item itself", so the fix that special-cased item/ paths was
// too broad. It must stay on the open item's detail instead.
func TestInboxDeleteSiblingRowWhileDetailOpen(t *testing.T) {
	h := inboxHost(t)
	app := inboxApp{}
	ctx := context.Background()

	viewing := mustItem(t, h, store.InboxItemParams{Kind: store.InboxFYI, SourceKind: store.InboxFromSystem, Reason: "r", Title: "currently open"})
	sibling := mustItem(t, h, store.InboxItemParams{Kind: store.InboxFYI, SourceKind: store.InboxFromSystem, Reason: "r", Title: "sibling row"})
	if _, err := h.Store.SetInboxItemState(viewing.ID, store.InboxDone, nil); err != nil {
		t.Fatalf("mark viewing done: %v", err)
	}
	if _, err := h.Store.SetInboxItemState(sibling.ID, store.InboxDone, nil); err != nil {
		t.Fatalf("mark sibling done: %v", err)
	}

	// Path is "item/<viewing>" (as AppSurface would send it — the current
	// top-level path — while the row action targets the SIBLING id).
	res, err := app.Action(ctx, h, ActionRequest{
		Action: "delete", Path: "item/" + viewing.ID, Args: map[string]string{"item": sibling.ID},
	})
	if err != nil {
		t.Fatalf("delete sibling: %v", err)
	}
	if _, err := h.Store.GetInboxItem(sibling.ID); err != store.ErrNotFound {
		t.Fatalf("sibling survived: %v", err)
	}
	if _, err := h.Store.GetInboxItem(viewing.ID); err != nil {
		t.Fatalf("the item being viewed was affected: %v", err)
	}
	// Must still be looking at `viewing`'s own detail, not root.
	if res.View == nil || res.View.Title != "currently open" {
		t.Fatalf("deleting a sibling navigated away from the open item: %+v", res.View)
	}
	var listedSibling bool
	for _, b := range res.View.Blocks {
		for _, it := range b.Items {
			if it.ID == sibling.ID {
				listedSibling = true
			}
		}
	}
	if listedSibling {
		t.Fatalf("deleted sibling still listed beside the open item")
	}
}

func TestInboxClearDoneAction(t *testing.T) {
	h := inboxHost(t)
	app := inboxApp{}
	ctx := context.Background()

	d1 := mustItem(t, h, store.InboxItemParams{Kind: store.InboxFYI, SourceKind: store.InboxFromSystem, Reason: "r", Title: "d1"})
	d2 := mustItem(t, h, store.InboxItemParams{Kind: store.InboxFYI, SourceKind: store.InboxFromSystem, Reason: "r", Title: "d2"})
	active := mustItem(t, h, store.InboxItemParams{Kind: store.InboxFYI, SourceKind: store.InboxFromSystem, Reason: "r", Title: "still open"})
	if _, err := h.Store.SetInboxItemState(d1.ID, store.InboxDone, nil); err != nil {
		t.Fatalf("mark d1: %v", err)
	}
	if _, err := h.Store.SetInboxItemState(d2.ID, store.InboxDone, nil); err != nil {
		t.Fatalf("mark d2: %v", err)
	}

	// clear-done carries no item arg — must not trip the "no item" guard.
	res, err := app.Action(ctx, h, ActionRequest{Action: "clear-done", Path: "done"})
	if err != nil {
		t.Fatalf("clear-done: %v", err)
	}
	if res.Toast != "Cleared 2 item(s)" {
		t.Fatalf("toast = %q", res.Toast)
	}
	if _, err := h.Store.GetInboxItem(d1.ID); err != store.ErrNotFound {
		t.Fatalf("d1 survived: %v", err)
	}
	if _, err := h.Store.GetInboxItem(d2.ID); err != store.ErrNotFound {
		t.Fatalf("d2 survived: %v", err)
	}
	if _, err := h.Store.GetInboxItem(active.ID); err != nil {
		t.Fatalf("active item was cleared: %v", err)
	}
	// Stays on the Done tab (now empty) rather than jumping to Active.
	if res.View == nil || !strings.Contains(res.View.Empty, "No answered items") {
		t.Fatalf("clear-done did not stay on Done: %+v", res.View)
	}
}

func TestInboxRecoveredReplyIsPrefilled(t *testing.T) {
	h := inboxHost(t)
	app := inboxApp{}
	q := mustItem(t, h, store.InboxItemParams{Kind: store.InboxQuestion, SourceKind: store.InboxFromSystem, Reason: "r", Title: "q", Body: "?"})
	if _, err := h.Store.RespondInboxItem(q.ID, store.VerbRespond, "keep this answer"); err != nil {
		t.Fatal(err)
	}
	if _, err := h.Store.SetInboxItemState(q.ID, store.InboxUnread, nil); err != nil {
		t.Fatal(err)
	}
	view, err := app.View(context.Background(), h, "item/"+q.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, block := range view.Blocks {
		if block.Form != nil && len(block.Form.Fields) > 0 {
			if got := block.Form.Fields[0].Prefill; got != "keep this answer" {
				t.Fatalf("recovered prefill = %q", got)
			}
			return
		}
	}
	t.Fatal("recovered item has no reply form")
}

// An interactive TUI agent's reply goes straight to the terminal through the
// host's DeliverReply; deliverable agents keep the ordinary durable forward.
func TestInboxDeliverReply(t *testing.T) {
	h := inboxHost(t)
	app := inboxApp{}
	ctx := context.Background()
	ws, _ := h.Store.AddWorkspace("wsx", t.TempDir())
	ag, _ := h.Store.AddAgent(ws.ID, "tui", "")
	h.AgentDeliverable = func(string) bool { return false }
	var calls []string
	h.DeliverReply = func(itemID, verb, text string) (string, error) {
		calls = append(calls, itemID+":"+verb+":"+text)
		if _, _, err := h.Store.RespondAndPark(itemID, verb, text); err != nil {
			return "", err
		}
		return ag.ID, nil
	}
	q := mustItem(t, h, store.InboxItemParams{Kind: store.InboxQuestion, SourceKind: store.InboxFromAgent, SourceID: ag.ID, Reason: "r", Title: "q", Body: "?"})

	res, err := app.Action(ctx, h, ActionRequest{Action: "respond", Path: "item/" + q.ID, Args: map[string]string{"reply": "go"}})
	if err != nil {
		t.Fatalf("respond-terminal: %v", err)
	}
	if res.Goto != "" && strings.Contains(res.Goto, "chat") {
		t.Fatalf("goto = %q, want no chat navigation", res.Goto)
	}
	if len(calls) != 1 || calls[0] != q.ID+":respond:go" {
		t.Fatalf("deliver calls = %v", calls)
	}
	if got, _ := h.Store.GetInboxItem(q.ID); got.State != store.InboxDone {
		t.Fatalf("item not done after delivery")
	}
	tasks, _ := h.Store.ListTasks(ag.ID, 5)
	if len(tasks) != 1 || tasks[0].Kind != store.TaskFollowUp || tasks[0].Source != "inbox-tui:"+q.ID {
		t.Fatalf("parked task = %+v", tasks)
	}

	// An approval's primary Accept button uses the same terminal delivery.
	approval := mustItem(t, h, store.InboxItemParams{Kind: store.InboxApproval, SourceKind: store.InboxFromAgent, SourceID: ag.ID, Reason: "r", Title: "approve", Body: "?"})
	if _, err := app.Action(ctx, h, ActionRequest{Action: "accept", Path: "item/" + approval.ID}); err != nil {
		t.Fatalf("approval delivery: %v", err)
	}
	if got := calls[len(calls)-1]; got != approval.ID+":accept:" {
		t.Fatalf("approval deliver call = %q", got)
	}

	// A delivery failure reaches the form verbatim.
	q2 := mustItem(t, h, store.InboxItemParams{Kind: store.InboxQuestion, SourceKind: store.InboxFromAgent, SourceID: ag.ID, Reason: "r", Title: "q2", Body: "?"})
	h.DeliverReply = func(string, string, string) (string, error) {
		return "", fmt.Errorf("the terminal did not pick up the reply")
	}
	if _, err := app.Action(ctx, h, ActionRequest{Action: "respond", Path: "item/" + q2.ID, Args: map[string]string{"reply": "go"}}); err == nil {
		t.Fatalf("delivery failure must reach the form")
	}

	// A deliverable agent uses the ordinary forward path.
	h.AgentDeliverable = func(string) bool { return true }
	before := len(calls)
	q3 := mustItem(t, h, store.InboxItemParams{Kind: store.InboxQuestion, SourceKind: store.InboxFromAgent, SourceID: ag.ID, Reason: "r", Title: "q3", Body: "?"})
	if _, err := app.Action(ctx, h, ActionRequest{Action: "respond", Path: "item/" + q3.ID, Args: map[string]string{"reply": "go"}}); err != nil {
		t.Fatalf("deliverable respond: %v", err)
	}
	if len(calls) != before {
		t.Fatalf("deliverable agent triggered a terminal delivery")
	}

	// Without a host delivery channel the interactive refusal stands.
	h.DeliverReply = nil
	h.AgentDeliverable = func(string) bool { return false }
	q4 := mustItem(t, h, store.InboxItemParams{Kind: store.InboxQuestion, SourceKind: store.InboxFromAgent, SourceID: ag.ID, Reason: "r", Title: "q4", Body: "?"})
	if _, err := app.Action(ctx, h, ActionRequest{Action: "respond", Path: "item/" + q4.ID, Args: map[string]string{"reply": "again"}}); err == nil || !strings.Contains(err.Error(), "interactive terminal") {
		t.Fatalf("plain respond on interactive agent = %v, want the interactive refusal", err)
	}
}
