package server

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/clilaunch"
	"github.com/cfpperche/picode/internal/store"
)

func TestManagedCLINeedsYouFilesInboxOnce(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "picode.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	proj := t.TempDir()
	w, err := st.AddWorkspace("App", proj)
	if err != nil {
		t.Fatal(err)
	}
	tm, err := st.CreateTerminalIn(w.ID, "Claude", proj)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.SetTerminalLaunch(tm.ID, "claude-code", clilaunch.Overrides{}); err != nil {
		t.Fatal(err)
	}
	a, err := st.AddAgentWithCLI(w.ID, "claude-code", "Claude", "")
	if err != nil {
		t.Fatal(err)
	}
	tid := tm.ID
	if _, err := st.UpdateAgent(a.ID, store.AgentPatch{TerminalID: &tid}); err != nil {
		t.Fatal(err)
	}
	deps := Deps{Store: st, TermStates: NewTermStates()}
	now := time.Now()
	reportTermState(deps, tm.ID, TermNeedsYou, "claude-code", now)
	open, err := st.ActiveInboxBySourceReason(store.InboxFromTerminal, tm.ID, cliNeedsYouReason)
	if err != nil || len(open) != 1 {
		t.Fatalf("inbox after needs-you: %v %#v", err, open)
	}
	if !open[0].Blocking || open[0].Kind != store.InboxFYI {
		t.Fatalf("want blocking FYI, got %+v", open[0])
	}
	reportTermState(deps, tm.ID, TermNeedsYou, "claude-code", now.Add(time.Second))
	open, _ = st.ActiveInboxBySourceReason(store.InboxFromTerminal, tm.ID, cliNeedsYouReason)
	if len(open) != 1 {
		t.Fatalf("duplicate inbox: %d", len(open))
	}
	reportTermState(deps, tm.ID, TermIdle, "claude-code", now.Add(2*time.Second))
	open, _ = st.ActiveInboxBySourceReason(store.InboxFromTerminal, tm.ID, cliNeedsYouReason)
	if len(open) != 0 {
		t.Fatalf("still open after idle: %+v", open)
	}
}

func TestUnboundTerminalNeedsYouDoesNotFileInbox(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "picode.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	proj := t.TempDir()
	w, _ := st.AddWorkspace("App", proj)
	tm, _ := st.CreateTerminalIn(w.ID, "Shell", proj)
	deps := Deps{Store: st, TermStates: NewTermStates()}
	reportTermState(deps, tm.ID, TermNeedsYou, "claude-code", time.Now())
	open, err := st.ActiveInboxBySourceReason(store.InboxFromTerminal, tm.ID, cliNeedsYouReason)
	if err != nil || len(open) != 0 {
		t.Fatalf("unbound terminal filed inbox: %v %#v", err, open)
	}
}
