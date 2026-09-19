package server

import (
	"log"

	"github.com/cfpperche/picode/internal/store"
)

// Reason on Inbox items filed when a managed CLI principal reports needs-you
// (ADR-0159 Fatia 1). Dedup and auto-done key off this, not the title.
const cliNeedsYouReason = "cli-needs-you"

// syncManagedCLIInbox files one Inbox item when a bound CLI asks for the
// human, and marks it done when the CLI is no longer waiting. Unbound
// terminals stay chips-only (ADR-0056 tier 1). Push rides inbox.created
// (blocking FYI). No composer, no Runtime.Start.
func syncManagedCLIInbox(deps Deps, termID, state string) {
	if deps.Store == nil || termID == "" {
		return
	}
	m, err := deps.Store.ManagedCLIByTerminal(termID)
	if err != nil {
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
		name := m.Name
		if name == "" {
			name = m.CLI
		}
		_, err := deps.Store.CreateInboxItem(store.InboxItemParams{
			Kind:        store.InboxFYI,
			SourceKind:  store.InboxFromTerminal,
			SourceID:    termID,
			WorkspaceID: m.WorkspaceID,
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
