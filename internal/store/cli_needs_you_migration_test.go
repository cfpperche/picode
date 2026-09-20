package store

import (
	"testing"

	"github.com/cfpperche/picode/internal/clilaunch"
)

// ADR-0160 Fatia E: open "needs you" items rekey from the terminal onto the
// CLI agent; items whose terminal has no bound agent close — the process
// they were about is gone.
func TestMigrateCLINeedsYouOntoAgents(t *testing.T) {
	s := openTest(t)
	proj := t.TempDir()
	w, err := s.AddWorkspace("App", proj)
	if err != nil {
		t.Fatal(err)
	}
	tm, err := s.CreateTerminalIn(w.ID, "Claude", proj)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SetTerminalLaunch(tm.ID, "claude-code", clilaunch.Overrides{}); err != nil {
		t.Fatal(err)
	}
	a, err := s.AddAgentWithCLI(w.ID, "claude-code", "Claude", "")
	if err != nil {
		t.Fatal(err)
	}
	tid := tm.ID
	if _, err := s.UpdateAgent(a.ID, AgentPatch{TerminalID: &tid}); err != nil {
		t.Fatal(err)
	}
	orphan, err := s.CreateTerminalIn(w.ID, "Gone", proj)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateInboxItem(InboxItemParams{
		Kind:       InboxFYI,
		SourceKind: InboxFromTerminal,
		SourceID:   tm.ID,
		Reason:     "cli-needs-you",
		Title:      "Claude needs you",
		Blocking:   true,
		Allowed:    []string{VerbIgnore},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateInboxItem(InboxItemParams{
		Kind:       InboxFYI,
		SourceKind: InboxFromTerminal,
		SourceID:   orphan.ID,
		Reason:     "cli-needs-you",
		Title:      "Gone needs you",
		Blocking:   true,
		Allowed:    []string{VerbIgnore},
	}); err != nil {
		t.Fatal(err)
	}

	if _, err := s.db.Exec(`DELETE FROM schema_migrations WHERE version = 62`); err != nil {
		t.Fatal(err)
	}
	if err := s.migrate(); err != nil {
		t.Fatalf("migrate 062: %v", err)
	}

	open, err := s.ActiveInboxBySourceReason(InboxFromAgent, a.ID, "cli-needs-you")
	if err != nil || len(open) != 1 {
		t.Fatalf("rekeyed item: %v %#v", err, open)
	}
	if open[0].Title != "Claude needs you" {
		t.Fatalf("rekeyed = %+v", open[0])
	}
	done, err := s.ActiveInboxBySourceReason(InboxFromTerminal, orphan.ID, "cli-needs-you")
	if err != nil || len(done) != 0 {
		t.Fatalf("orphan item survived: %v %#v", err, done)
	}
	var state string
	if err := s.db.QueryRow(`SELECT state FROM inbox_items WHERE source_kind = ? AND source_id = ?`, InboxFromTerminal, orphan.ID).Scan(&state); err != nil {
		t.Fatal(err)
	}
	if state != InboxDone {
		t.Fatalf("orphan state = %q", state)
	}
}
