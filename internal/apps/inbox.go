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
	if path != "" {
		return View{}, fmt.Errorf("inbox: no view at %q", path)
	}
	return a.rootView(h)
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

func (inboxApp) rootView(h Host) (View, error) {
	blocks, err := listPanes(h, "")
	if err != nil {
		return View{}, err
	}
	v := View{APIVersion: APIVersion, Title: "Inbox", Layout: "split", Blocks: blocks}
	if len(blocks) == 0 {
		// Inbox zero: the host draws its blankslate, no split chrome
		// wrapped around nothing.
		v.Layout = ""
		v.Empty = "Inbox zero — Agents and terminals file questions and results here. Nothing needs you right now."
	}
	return v, nil
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
	// The list travels with every view so the split never blanks.
	blocks, err := listPanes(h, it.ID)
	if err != nil {
		return View{}, err
	}
	v := View{APIVersion: APIVersion, Title: it.Title, Layout: "split", Blocks: blocks}

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
	if it.State == store.InboxDone && it.Response != nil {
		detail = append(detail, Block{Type: "detail", Pane: "detail", Title: "Answered", Markdown: "> " + *it.Response})
	}

	if it.State != store.InboxDone {
		allowed := map[string]bool{}
		for _, verb := range it.Allowed {
			allowed[verb] = true
		}
		if (it.Kind == store.InboxQuestion || it.Kind == store.InboxApproval) && allowed[store.VerbRespond] {
			detail = append(detail, Block{Type: "form", Pane: "detail", Form: &Form{
				ID:     "respond",
				Submit: "Send reply",
				Fields: []Field{{
					Name: "reply", Method: "editor", Title: "Your reply",
					Placeholder: "Type your answer…",
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
