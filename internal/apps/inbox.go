package apps

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/store"
)

// inboxApp is the Inbox (ADR-0037) — the first production app on the
// host. Two zones from the frozen primitives: a needs-me queue (blocking
// items, each demanding a decision) and the archivable feed. Responding
// is the done; replies to agent questions ride the durable task queue.
type inboxApp struct{}

func (inboxApp) Manifest() Manifest {
	return Manifest{ID: "inbox", Name: "Inbox", Icon: "inbox", APIVersion: APIVersion}
}

func (inboxApp) Badge(_ context.Context, h Host) (Badge, error) {
	blocking, other, err := h.Store.CountInboxBadge()
	if err != nil {
		return Badge{}, err
	}
	return Badge{Count: blocking, Dot: other}, nil
}

func (a inboxApp) View(_ context.Context, h Host, path string) (View, error) {
	if strings.HasPrefix(path, "item/") {
		return a.itemView(h, strings.TrimPrefix(path, "item/"))
	}
	switch path {
	case "":
		return a.rootView(h)
	case "done":
		return a.doneView(h)
	case "all":
		return a.allView(h)
	}
	return View{}, fmt.Errorf("inbox: no view at %q", path)
}

// tabs builds the segmented Active/Done/All strip shared by every view
// (best-effort on the detail view, so a count query failing there
// doesn't take the item down with it). Active carries no badge — its
// sections already show counts the same way Needs-you/Feed do.
func (inboxApp) tabs(h Host) ([]Tab, error) {
	doneN, err := h.Store.CountInboxItems(store.InboxDone)
	if err != nil {
		return nil, err
	}
	allN, err := h.Store.CountAllInboxItems()
	if err != nil {
		return nil, err
	}
	doneBadge, allBadge := "", ""
	if doneN > 0 {
		doneBadge = fmt.Sprintf("%d", doneN)
	}
	if allN > 0 {
		allBadge = fmt.Sprintf("%d", allN)
	}
	return []Tab{
		{ID: "active", Label: "Active", Path: ""},
		{ID: "done", Label: "Done", Path: "done", Badge: doneBadge},
		{ID: "all", Label: "All", Path: "all", Badge: allBadge},
	}, nil
}

// listPanes is the left column of the split: the needs-me queue over the
// archivable feed. Every view emits it so selecting an item never blanks
// the list.
func listPanes(h Host, selected string) ([]Block, error) {
	items, err := h.Store.ListInboxItems(store.InboxFilter{})
	if err != nil {
		return nil, err
	}
	var needsMe, feed []store.InboxItem
	for _, it := range items {
		if it.Blocking {
			needsMe = append(needsMe, it)
		} else {
			feed = append(feed, it)
		}
	}
	blocks := []Block{}
	if len(needsMe) > 0 {
		blocks = append(blocks, Block{Type: "list", Pane: "list", Title: "Needs you", Items: inboxRows(h, needsMe, false, selected)})
	}
	if len(feed) > 0 {
		blocks = append(blocks, Block{Type: "list", Pane: "list", Title: "Feed", Items: inboxRows(h, feed, true, selected)})
	}
	return blocks, nil
}

func (a inboxApp) rootView(h Host) (View, error) {
	tabs, err := a.tabs(h)
	if err != nil {
		return View{}, err
	}
	blocks, err := listPanes(h, "")
	if err != nil {
		return View{}, err
	}
	v := View{APIVersion: APIVersion, Title: "Inbox", Layout: "split", Tabs: tabs, Blocks: blocks}
	if len(blocks) == 0 {
		// An empty Active queue is common once history is visible in Done
		// (ADR-0037 follow-up) — "Inbox zero" would wrongly imply the
		// whole mailbox is empty when Done might hold real history.
		v.Layout = ""
		v.Empty = "Nothing needs you right now."
	}
	return v, nil
}

// doneItems fetches every done item, newest first. IncludeSnoozed is
// mandatory here: an item answered before an earlier snooze expired must
// never hide behind that stale timestamp.
func doneItems(h Host) ([]store.InboxItem, error) {
	return h.Store.ListInboxItems(store.InboxFilter{State: store.InboxDone, IncludeSnoozed: true})
}

// donePane is itemView's list column when the open item is itself done:
// the list beside the detail mirrors what you navigated from, instead of
// snapping back to Active under you.
func donePane(h Host, selected string) ([]Block, error) {
	items, err := doneItems(h)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return []Block{}, nil
	}
	// No title: the enclosing Done tab already says so (ADR-0036 amendment
	// — a second "Done" line directly under the "Done" tab pill was pure
	// duplication, unlike "Needs you"/"Feed" on Active or the trailing
	// "Done" block on All, which each name a real subset of that tab).
	return []Block{{Type: "list", Pane: "list", Items: doneRows(h, items)}}, nil
}

func (a inboxApp) doneView(h Host) (View, error) {
	tabs, err := a.tabs(h)
	if err != nil {
		return View{}, err
	}
	items, err := doneItems(h)
	if err != nil {
		return View{}, err
	}
	v := View{APIVersion: APIVersion, Title: "Inbox", Layout: "split", Tabs: tabs}
	if len(items) == 0 {
		v.Layout = ""
		v.Empty = "No answered items yet."
		return v, nil
	}
	v.Blocks = []Block{
		// Same reasoning as donePane: this list IS the Done tab's content.
		{Type: "list", Pane: "list", Items: doneRows(h, items)},
		{Type: "actions", Pane: "list", Actions: []Action{{
			ID: "clear-done", Label: fmt.Sprintf("Clear all done (%d)", len(items)), Danger: true,
			Confirm: fmt.Sprintf("Delete all %d done item(s)? This cannot be undone.", len(items)),
		}}},
	}
	return v, nil
}

func (a inboxApp) allView(h Host) (View, error) {
	tabs, err := a.tabs(h)
	if err != nil {
		return View{}, err
	}
	blocks, err := listPanes(h, "")
	if err != nil {
		return View{}, err
	}
	done, err := doneItems(h)
	if err != nil {
		return View{}, err
	}
	if len(done) > 0 {
		blocks = append(blocks, Block{Type: "list", Pane: "list", Title: "Done", Items: doneRows(h, done)})
	}
	v := View{APIVersion: APIVersion, Title: "Inbox", Layout: "split", Tabs: tabs, Blocks: blocks}
	if len(blocks) == 0 {
		v.Layout = ""
		v.Empty = "Inbox zero — nothing has come through yet."
	}
	return v, nil
}

// doneRows renders answered/ignored items for history: the response
// itself lands in Meta, so "what was decided" is visible without opening
// the item — the literal ask behind this view. Delete is the only row
// action; Done/Snooze make no sense on an item that's already settled.
func doneRows(h Host, items []store.InboxItem) []ListItem {
	rows := make([]ListItem, 0, len(items))
	for _, it := range items {
		meta := []string{sourceLabel(h, it), it.Reason}
		if it.Response != nil {
			meta = append(meta, truncate(*it.Response, 60))
		}
		rows = append(rows, ListItem{
			ID: it.ID, Title: it.Title, Meta: meta, At: it.CreatedAt,
			Badge: it.Kind, Tone: kindTone(it.Kind), Path: "item/" + it.ID,
			Actions: []Action{{
				ID: "delete", Label: "Delete", Icon: "trash", Danger: true,
				Confirm: "Delete this item? This cannot be undone.",
				Args:    map[string]string{"item": it.ID},
			}},
		})
	}
	return rows
}

func priorInboxReply(it store.InboxItem) string {
	if it.Response == nil {
		return ""
	}
	const prefix = store.VerbRespond + ": "
	if strings.HasPrefix(*it.Response, prefix) {
		return strings.TrimSpace(strings.TrimPrefix(*it.Response, prefix))
	}
	return ""
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// inboxRows renders items as list rows. Feed rows carry Done/Snooze as
// quiet glyphs; blocking rows demand a decision on their detail instead.
func inboxRows(h Host, items []store.InboxItem, triage bool, selected string) []ListItem {
	rows := make([]ListItem, 0, len(items))
	for _, it := range items {
		row := ListItem{
			ID:     it.ID,
			Title:  it.Title,
			Meta:   []string{sourceLabel(h, it), it.Reason},
			At:     it.CreatedAt,
			Badge:  it.Kind,
			Tone:   kindTone(it.Kind),
			Unread: it.State == store.InboxUnread,
			Path:   "item/" + it.ID,
		}
		_ = selected // selection is derived from the open path, host-side
		if triage {
			row.Actions = []Action{
				{ID: "done", Label: "Done", Icon: "check", Args: map[string]string{"item": it.ID}},
				{ID: "snooze", Label: "Snooze 1h", Icon: "clock", Args: map[string]string{"item": it.ID}},
			}
		}
		rows = append(rows, row)
	}
	return rows
}

// kindTone maps an item kind to the host's tone vocabulary. Red is for
// destruction only, so an approval — which merely *asks* about something
// destructive — reads as warn, not danger.
func kindTone(kind string) string {
	switch kind {
	case store.InboxQuestion:
		return "info"
	case store.InboxApproval:
		return "warn"
	case store.InboxResult:
		return "ok"
	}
	return ""
}

func (a inboxApp) itemView(h Host, id string) (View, error) {
	it, err := h.Store.GetInboxItem(id)
	if err != nil {
		return View{}, err
	}
	// Opening is the read (ADR: read is explicit or on-open; done stays explicit).
	if it.State == store.InboxUnread {
		if updated, err := h.Store.SetInboxItemState(id, store.InboxRead, nil); err == nil {
			it = updated
		}
	}
	// The list travels with every view so the split never blanks — and it
	// mirrors the item's own state, so opening a Done row doesn't snap
	// the left column back to Active under you.
	var blocks []Block
	if it.State == store.InboxDone {
		blocks, err = donePane(h, it.ID)
	} else {
		blocks, err = listPanes(h, it.ID)
	}
	if err != nil {
		return View{}, err
	}
	v := View{APIVersion: APIVersion, Title: it.Title, Layout: "split", Blocks: blocks}
	// Best-effort: the tab strip is chrome, not content — a count-query
	// failure here shouldn't take the item's own detail down with it.
	if tabs, err := a.tabs(h); err == nil {
		v.Tabs = tabs
	}

	detail := []Block{}
	body := strings.TrimSpace(it.Body)
	if body == "" {
		body = "_No details._"
	}
	detail = append(detail, Block{
		Type: "detail", Pane: "detail", Title: it.Title,
		Meta: []string{sourceLabel(h, it), it.Reason}, At: it.CreatedAt,
		Markdown: body,
	})
	if it.State == store.InboxDone {
		if it.Response != nil {
			detail = append(detail, Block{Type: "detail", Pane: "detail", Title: "Answered", Markdown: "> " + *it.Response})
		}
		detail = append(detail, Block{Type: "actions", Pane: "detail", Actions: []Action{{
			ID: "delete", Label: "Delete", Icon: "trash", Danger: true,
			Confirm: "Delete this item? This cannot be undone.",
			Args:    map[string]string{"item": it.ID},
		}}})
	}

	if it.State != store.InboxDone {
		allowed := map[string]bool{}
		for _, verb := range it.Allowed {
			allowed[verb] = true
		}
		if (it.Kind == store.InboxQuestion || it.Kind == store.InboxApproval) && allowed[store.VerbRespond] {
			detail = append(detail, Block{Type: "form", Pane: "detail", Form: &Form{
				ID:       "respond",
				Submit:   "Send reply",
				Terminal: h.AgentDeliverable != nil && !h.AgentDeliverable(it.SourceID),
				Fields: []Field{{
					Name: "reply", Method: "editor", Title: "Your reply",
					Placeholder: "Type your answer…", Prefill: priorInboxReply(it),
				}},
			}})
		}
		var acts []Action
		if allowed[store.VerbAccept] {
			// An approval's decision is the primary act; replying is the
			// side channel, so the fill goes here.
			acts = append(acts, Action{ID: "accept", Label: "Accept", Icon: "check", Primary: true, Args: map[string]string{"item": it.ID}})
			// Accept without a decline leaves no way to answer "no" — only
			// to abandon. Declining forwards the refusal to the agent.
			acts = append(acts, Action{ID: "decline", Label: "Decline", Args: map[string]string{"item": it.ID}})
		}
		if it.Blocking {
			// Ignore is the destructive out on a blocking item: the agent
			// gets no reply. It goes last and asks first.
			if allowed[store.VerbIgnore] {
				acts = append(acts, Action{
					ID: "ignore", Label: "Ignore", Danger: true,
					Confirm: "Ignore this? The agent gets no reply.",
					Args:    map[string]string{"item": it.ID},
				})
			}
		} else {
			// News: Done is the whole triage; Ignore would say the same thing twice.
			acts = append(acts, Action{ID: "done", Label: "Done", Icon: "check", Args: map[string]string{"item": it.ID}})
			acts = append(acts, Action{ID: "snooze", Label: "Snooze 1h", Icon: "clock", Args: map[string]string{"item": it.ID}})
		}
		if len(acts) > 0 {
			detail = append(detail, Block{Type: "actions", Pane: "detail", Actions: acts})
		}
	}
	v.Blocks = append(v.Blocks, detail...)
	return v, nil
}

func (a inboxApp) Action(_ context.Context, h Host, req ActionRequest) (ActionResult, error) {
	// The only action with no item — checked first so the "no item in
	// request" guard below doesn't have to special-case it.
	if req.Action == "clear-done" {
		n, err := h.Store.DeleteDoneInboxItems()
		if err != nil {
			return ActionResult{}, err
		}
		return a.backTo(h, req.Path, fmt.Sprintf("Cleared %d item(s)", n))
	}

	id := req.Args["item"]
	if id == "" && strings.HasPrefix(req.Path, "item/") {
		id = strings.TrimPrefix(req.Path, "item/")
	}
	if id == "" {
		return ActionResult{}, fmt.Errorf("inbox: no item in request")
	}
	// path alone can't tell "acting on the item you're viewing" from
	// "acting on a sibling row while that item's detail happens to be
	// open" — both carry path="item/<open-id>". Only the FORMER should
	// leave the detail (existing behavior for respond/accept/ignore/
	// decline, and necessary for delete: re-rendering a just-deleted
	// item's own path would 404). A sibling's row action — e.g. deleting
	// a different Done item while reading this one — must stay put:
	// re-rendering the same item/<open-id> just reflects the sibling's
	// removal in the list beside it, without navigating anywhere.
	returnPath := req.Path
	if strings.HasPrefix(returnPath, "item/") && strings.TrimPrefix(returnPath, "item/") == id {
		returnPath = ""
	}
	switch req.Action {
	case "done":
		if _, err := h.Store.SetInboxItemState(id, store.InboxDone, nil); err != nil {
			return ActionResult{}, err
		}
		return a.backTo(h, returnPath, "Done")
	case "snooze":
		until := time.Now().UTC().Add(time.Hour).Format(time.RFC3339)
		if _, err := h.Store.SetInboxItemState(id, "", &until); err != nil {
			return ActionResult{}, err
		}
		return a.backTo(h, returnPath, "Snoozed for 1 hour")
	case "delete":
		if err := h.Store.DeleteInboxItem(id); err != nil {
			return ActionResult{}, err
		}
		return a.backTo(h, returnPath, "Item deleted")
	case "respond", "accept", "ignore", "decline":
		verb := req.Action
		text := req.Args["reply"]
		switch req.Action {
		case "respond":
			verb = store.VerbRespond
		case "decline":
			// A decline is a real answer: it rides the reply channel so the
			// agent hears "no" instead of silence.
			verb = store.VerbRespond
			if strings.TrimSpace(text) == "" {
				text = "Declined."
			}
		}
		if verb == store.VerbRespond && strings.TrimSpace(text) == "" {
			return ActionResult{}, fmt.Errorf("write a reply first")
		}
		// ADR-0060: a TUI agent receives the reply in its own terminal —
		// receiver extension or tmux paste — and the host owns delivery
		// proof. Anything else falls through to the ordinary durable gate.
		it, err := h.Store.GetInboxItem(id)
		if err != nil {
			return ActionResult{}, err
		}
		interactive := h.AgentDeliverable != nil && !h.AgentDeliverable(it.SourceID) &&
			it.SourceKind == store.InboxFromAgent
		if interactive && h.DeliverReply != nil && it.State != store.InboxDone &&
			(it.Kind == store.InboxQuestion || it.Kind == store.InboxApproval) {
			if _, err := h.DeliverReply(id, verb, text); err != nil {
				if strings.Contains(err.Error(), "agent no longer exists") {
					return ActionResult{}, fmt.Errorf("Reply not delivered — the agent no longer exists; the item stays open")
				}
				return ActionResult{}, err
			}
			return a.backTo(h, returnPath, "Reply sent to the terminal.")
		}
		if _, err := h.Store.RespondAndForward(id, verb, text, h.AgentDeliverable); err != nil {
			// These surface verbatim as a toast (handleAppAction writes
			// err.Error() straight through), so they read as UI copy —
			// sentence case, not the lowercase Go error-string convention.
			if errors.Is(err, store.ErrAgentInteractive) {
				return ActionResult{}, fmt.Errorf("Reply not delivered — the agent is running in an interactive terminal; the item stays open")
			}
			if strings.Contains(err.Error(), "agent no longer exists") {
				return ActionResult{}, fmt.Errorf("Reply not delivered — the agent no longer exists; the item stays open")
			}
			return ActionResult{}, err
		}
		toast := "Done"
		switch req.Action {
		case "respond":
			toast = "Reply sent — the agent will pick it up."
		case "accept":
			toast = "Accepted — the agent can go ahead."
		case "decline":
			toast = "Declined — the agent was told no."
		}
		return a.backTo(h, returnPath, toast)
	}
	return ActionResult{}, fmt.Errorf("inbox: unknown action %q", req.Action)
}

// backTo re-renders the view at path so an action updates the screen the
// user was actually looking at — deleting a Done row keeps you on Done,
// clearing history from All keeps you on All — instead of always
// snapping back to Active regardless of where the action fired from.
func (a inboxApp) backTo(h Host, path, toast string) (ActionResult, error) {
	v, err := a.View(context.Background(), h, path)
	if err != nil {
		return ActionResult{Toast: toast}, nil
	}
	return ActionResult{Toast: toast, View: &v}, nil
}

// sourceLabel names the item's origin for row subtitles and headers.
func sourceLabel(h Host, it store.InboxItem) string {
	switch it.SourceKind {
	case store.InboxFromAgent:
		if h.Store != nil {
			if ag, err := h.Store.GetAgent(it.SourceID); err == nil && ag.Name != "" {
				return ag.Name
			}
		}
		if it.SourceID != "" {
			return it.SourceID
		}
		return "agent"
	case store.InboxFromTerminal:
		if it.SourceID != "" {
			return "terminal " + it.SourceID
		}
		return "terminal"
	case store.InboxFromAutomation:
		if h.Store != nil {
			if a, err := h.Store.GetAutomation(it.SourceID); err == nil && a.Name != "" {
				return a.Name
			}
		}
		return "automation"
	}
	return "PiCode"
}
