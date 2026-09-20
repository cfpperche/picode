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
	// The item belongs to the agent (ADR-0160 Fatia E), not the terminal.
	open, err := st.ActiveInboxBySourceReason(store.InboxFromAgent, a.ID, store.InboxNeedsYouReason)
	if err != nil || len(open) != 1 {
		t.Fatalf("inbox after needs-you: %v %#v", err, open)
	}
	if open[0].SourceKind != store.InboxFromAgent || open[0].SourceID != a.ID {
		t.Fatalf("item keyed on the terminal: %+v", open[0])
	}
	if !open[0].Blocking || open[0].Kind != store.InboxFYI {
		t.Fatalf("want blocking FYI, got %+v", open[0])
	}
	reportTermState(deps, tm.ID, TermNeedsYou, "claude-code", now.Add(time.Second))
	open, _ = st.ActiveInboxBySourceReason(store.InboxFromAgent, a.ID, store.InboxNeedsYouReason)
	if len(open) != 1 {
		t.Fatalf("duplicate inbox: %d", len(open))
	}
	reportTermState(deps, tm.ID, TermIdle, "claude-code", now.Add(2*time.Second))
	open, _ = st.ActiveInboxBySourceReason(store.InboxFromAgent, a.ID, store.InboxNeedsYouReason)
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
	open, err := st.ActiveInboxBySourceReason(store.InboxFromTerminal, tm.ID, store.InboxNeedsYouReason)
	if err != nil || len(open) != 0 {
		t.Fatalf("unbound terminal filed inbox: %v %#v", err, open)
	}
}

// The launch env names the bound agent (ADR-0160 Fatia E): picode mcp then
// resolves the agent principal, so grants given to the agent in Settings
// match the CLI's calls.
func TestLaunchIdentityEnvCarriesTheBoundAgent(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "picode.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	proj := t.TempDir()
	w, _ := st.AddWorkspace("App", proj)

	deps := Deps{Store: st}
	shell, _ := st.CreateTerminalIn(w.ID, "Shell", proj)
	if got := launchIdentityEnv(deps, shell.ID); len(got) != 1 || got[0] != "PICODE_TERM_ID="+shell.ID {
		t.Fatalf("unbound terminal env = %q", got)
	}

	tm, _ := st.CreateTerminalIn(w.ID, "Claude", proj)
	a, err := st.AddAgentWithCLI(w.ID, "claude-code", "Claude", "")
	if err != nil {
		t.Fatal(err)
	}
	tid := tm.ID
	if _, err := st.UpdateAgent(a.ID, store.AgentPatch{TerminalID: &tid}); err != nil {
		t.Fatal(err)
	}
	want := []string{"PICODE_TERM_ID=" + tm.ID, "PICODE_AGENT_ID=" + a.ID}
	if got := launchIdentityEnv(deps, tm.ID); len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("guest env = %q, want %q", got, want)
	}

	// A Pi agent bound to a terminal gets the same treatment: the agent id
	// wins the grant key wherever it can be named.
	pi, err := st.AddAgent(w.ID, "Pi", "")
	if err != nil {
		t.Fatal(err)
	}
	tm2, _ := st.CreateTerminalIn(w.ID, "pi", proj)
	pid := tm2.ID
	if _, err := st.UpdateAgent(pi.ID, store.AgentPatch{TerminalID: &pid}); err != nil {
		t.Fatal(err)
	}
	if got := launchIdentityEnv(deps, tm2.ID); len(got) != 2 || got[1] != "PICODE_AGENT_ID="+pi.ID {
		t.Fatalf("pi-bound env = %q", got)
	}
}
