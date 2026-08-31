package apps

import (
	"context"
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
	if path != "" {
		return View{}, fmt.Errorf("inbox: no view at %q", path)
	}
	return a.rootView(h)
}

func (inboxApp) rootView(h Host) (View, error) {
	items, err := h.Store.ListInboxItems(store.InboxFilter{})
	if err != nil {
		return View{}, err
	}
	v := View{APIVersion: APIVersion, Title: "Inbox"}
	var needsMe, feed []store.InboxItem
	for _, it := range items {
		if it.Blocking {
			needsMe = append(needsMe, it)
		} else {
			feed = append(feed, it)
		}
	}
	if len(items) == 0 {
		v.Blocks = append(v.Blocks, Block{Type: "detail", Markdown: "Inbox zero — nothing needs you."})
		return v, nil
	}
	if len(needsMe) > 0 {
		v.Blocks = append(v.Blocks, Block{Type: "detail", Markdown: "### Needs you"})
		v.Blocks = append(v.Blocks, Block{Type: "list", Items: inboxRows(h, needsMe, false)})
	}
	if len(feed) > 0 {
		v.Blocks = append(v.Blocks, Block{Type: "detail", Markdown: "### Feed"})
		v.Blocks = append(v.Blocks, Block{Type: "list", Items: inboxRows(h, feed, true)})
	}
	return v, nil
}

// inboxRows renders items as list rows. Feed rows carry Done/Snooze;
// blocking rows demand a decision on their detail view instead.
func inboxRows(h Host, items []store.InboxItem, triage bool) []ListItem {
	rows := make([]ListItem, 0, len(items))
	for _, it := range items {
		row := ListItem{
			ID:       it.ID,
			Title:    it.Title,
			Subtitle: sourceLabel(h, it) + " · " + it.Reason,
			Badge:    rowBadge(it),
			Path:     "item/" + it.ID,
		}
		if triage {
			row.Actions = []Action{
				{ID: "done", Label: "Done", Args: map[string]string{"item": it.ID}},
				{ID: "snooze", Label: "Snooze 1h", Args: map[string]string{"item": it.ID}},
			}
		}
		rows = append(rows, row)
	}
	return rows
}

func rowBadge(it store.InboxItem) string {
	if it.State == store.InboxUnread {
		return it.Kind
	}
	return ""
}

func (inboxApp) itemView(h Host, id string) (View, error) {
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
	v := View{APIVersion: APIVersion, Title: it.Title}
	head := "**" + it.Reason + "** · " + sourceLabel(h, it)
	if it.State == store.InboxDone && it.Response != nil {
		head += " · answered: " + *it.Response
	}
	md := head
	if strings.TrimSpace(it.Body) != "" {
		md += "\n\n" + it.Body // markdown, never HTML (ADR mitigation)
	}
	v.Blocks = append(v.Blocks, Block{Type: "detail", Markdown: md})
	if it.State == store.InboxDone {
		return v, nil
	}

	allowed := map[string]bool{}
	for _, verb := range it.Allowed {
		allowed[verb] = true
	}
	if (it.Kind == store.InboxQuestion || it.Kind == store.InboxApproval) && allowed[store.VerbRespond] {
		v.Blocks = append(v.Blocks, Block{Type: "form", Form: &Form{
			ID:     "respond",
			Submit: "Send reply",
			Fields: []Field{{Name: "reply", Method: "editor", Title: "Your reply", Placeholder: "The agent picks this up as a follow-up message."}},
		}})
	}
	var acts []Action
	if allowed[store.VerbAccept] {
		acts = append(acts, Action{ID: "accept", Label: "Accept", Args: map[string]string{"item": it.ID}})
	}
	if allowed[store.VerbIgnore] {
		ig := Action{ID: "ignore", Label: "Ignore", Args: map[string]string{"item": it.ID}}
		if it.Blocking {
			ig.Confirm = "Ignore this? The agent gets no reply."
			ig.Danger = true
		}
		acts = append(acts, ig)
	}
	if !it.Blocking {
		acts = append(acts, Action{ID: "done", Label: "Done", Args: map[string]string{"item": it.ID}})
	}
	if len(acts) > 0 {
		v.Blocks = append(v.Blocks, Block{Type: "actions", Actions: acts})
	}
	return v, nil
}

func (a inboxApp) Action(_ context.Context, h Host, req ActionRequest) (ActionResult, error) {
	id := req.Args["item"]
	if id == "" && strings.HasPrefix(req.Path, "item/") {
		id = strings.TrimPrefix(req.Path, "item/")
	}
	if id == "" {
		return ActionResult{}, fmt.Errorf("inbox: no item in request")
	}
	switch req.Action {
	case "done":
		if _, err := h.Store.SetInboxItemState(id, store.InboxDone, nil); err != nil {
			return ActionResult{}, err
		}
		return a.backToRoot(h, "Done")
	case "snooze":
		until := time.Now().UTC().Add(time.Hour).Format(time.RFC3339)
		if _, err := h.Store.SetInboxItemState(id, "", &until); err != nil {
			return ActionResult{}, err
		}
		return a.backToRoot(h, "Snoozed for 1 hour")
	case "respond", "accept", "ignore":
		verb := req.Action
		if verb == "respond" {
			verb = store.VerbRespond
		}
		text := req.Args["reply"]
		if verb == store.VerbRespond && strings.TrimSpace(text) == "" {
			return ActionResult{}, fmt.Errorf("write a reply first")
		}
		if _, err := h.Store.RespondAndForward(id, verb, text); err != nil {
			if strings.Contains(err.Error(), "agent no longer exists") {
				return ActionResult{}, fmt.Errorf("reply not delivered — the agent no longer exists; the item stays open")
			}
			return ActionResult{}, err
		}
		toast := "Done"
		if verb != store.VerbIgnore {
			toast = "Reply sent — the agent will pick it up."
		}
		return a.backToRoot(h, toast)
	}
	return ActionResult{}, fmt.Errorf("inbox: unknown action %q", req.Action)
}

// backToRoot returns a fresh root view so row actions update in place.
func (a inboxApp) backToRoot(h Host, toast string) (ActionResult, error) {
	v, err := a.rootView(h)
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
	}
	return "PiCode"
}
