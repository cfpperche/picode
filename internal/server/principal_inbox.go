package server

import (
	"log"

	"github.com/cfpperche/picode/internal/store"
)

// Reason on Inbox items filed when a guest agent reports needs-you
// (ADR-0159 Fatia 1; lookup is agents.terminal_id since ADR-0160 Fatia C).
// Dedup and auto-done key off this, not the title. Rekey onto the agent id
// is Fatia E.
const cliNeedsYouReason = "cli-needs-you"

// syncManagedCLIInbox files one Inbox item when a guest agent's TUI asks
// for the human, and marks it done when the CLI is no longer waiting.
// Unbound terminals stay chips-only (ADR-0056 tier 1). Push rides
// inbox.created (blocking FYI). No composer, no Runtime.Start.
func syncManagedCLIInbox(deps Deps, termID, state string) {
	if deps.Store == nil || termID == "" {
		return
	}
	a, err := deps.Store.AgentByTerminal(termID)
	if err != nil || a.IsPi() {
		return
	}
	open, err := deps.Store.ActiveInboxBySourceReason(store.InboxFromTerminal, termID, cliNeedsYouReason)
	if err != nil {
		log.Printf("managed CLI inbox: list: %v", err)
		return
	}
	if state == TermNeedsYou {
		if len(open) > 0 {
			return
		}
		name := a.Name
		if name == "" {
			name = a.CLI
		}
		_, err := deps.Store.CreateInboxItem(store.InboxItemParams{
			Kind:        store.InboxFYI,
			SourceKind:  store.InboxFromTerminal,
			SourceID:    termID,
			WorkspaceID: a.WorkspaceID,
			Reason:      cliNeedsYouReason,
			Title:       name + " needs you",
			Body:        "Open the terminal to continue. PiCode cannot answer inside this CLI's own prompt.",
			Blocking:    true,
			Allowed:     []string{store.VerbIgnore},
		})
		if err != nil {
			log.Printf("managed CLI inbox: create: %v", err)
		}
		return
	}
	for _, it := range open {
		if _, err := deps.Store.SetInboxItemState(it.ID, store.InboxDone, nil); err != nil {
			log.Printf("managed CLI inbox: done %s: %v", it.ID, err)
		}
	}
}
