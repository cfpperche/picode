package store

import (
	"errors"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/cfpperche/picode/internal/session"
)

func TestCLIFlagsForSpawn(t *testing.T) {
	fresh := Agent{}
	if got, want := fresh.CLIFlagsForSpawn("new-id"), []string{"--session-id", "new-id"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("fresh start = %v, want %v", got, want)
	}
	if got := fresh.CLIFlagsForSpawn(""); len(got) != 0 {
		t.Fatalf("empty sessionID should be a no-op, got %v", got)
	}

	path := "/some/path.jsonl"
	resuming := Agent{SessionPath: &path}
	got := resuming.CLIFlagsForSpawn("should-be-ignored")
	if want := resuming.CLIFlags(); !reflect.DeepEqual(got, want) {
		t.Fatalf("resuming agent = %v, want CLIFlags() = %v", got, want)
	}
	for _, a := range got {
		if a == "--session-id" {
			t.Fatalf("resuming agent must not also get --session-id: %v", got)
		}
	}
}

// TestCLIFlagsSessionDir locks in ADR-0040: every agent with a persisted
// id gets a private --session-dir, unconditionally (fresh or resuming);
// an empty-id Agent{} (never persisted) gets none, matching SpawnEnv's
// existing guard.
func TestCLIFlagsSessionDir(t *testing.T) {
	withID := Agent{ID: "agent-xyz"}
	flags := withID.CLIFlags()
	want := session.AgentDir("agent-xyz")
	found := false
	for i, f := range flags {
		if f == "--session-dir" {
			found = true
			if i+1 >= len(flags) || flags[i+1] != want {
				t.Fatalf("--session-dir value in %v, want %q", flags, want)
			}
		}
	}
	if !found {
		t.Fatalf("CLIFlags = %v, missing --session-dir", flags)
	}

	if got := (Agent{}).CLIFlags(); len(got) != 0 {
		t.Fatalf("empty-id agent must get no flags at all, got %v", got)
	}
}

func TestAddAgentCLIDefaultPi(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "picode.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	a, err := s.AddAgent(FreeWorkspaceID, "default", "")
	if err != nil {
		t.Fatal(err)
	}
	if a.CLI != CLIPi || !a.IsPi() {
		t.Fatalf("cli=%q IsPi=%v", a.CLI, a.IsPi())
	}
	got, err := s.GetAgent(a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.CLI != CLIPi {
		t.Fatalf("reload cli=%q", got.CLI)
	}
}

func TestAddAgentWithCLI(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "picode.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	a, err := s.AddAgentWithCLI(FreeWorkspaceID, "claude-code", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if a.CLI != "claude-code" || a.IsPi() {
		t.Fatalf("cli=%q IsPi=%v", a.CLI, a.IsPi())
	}
	if a.Name != "Claude Code" {
		t.Fatalf("name=%q, want catalog name", a.Name)
	}
	if _, err := s.AddAgentWithCLI(FreeWorkspaceID, "not-a-cli", "x", ""); !errors.Is(err, ErrInvalid) {
		t.Fatalf("unknown cli err=%v", err)
	}
}

func TestAgentTerminalBind(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "picode.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	w, err := s.AddWorkspace("App", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	a, err := s.AddAgentWithCLI(w.ID, "claude-code", "Claude", "")
	if err != nil {
		t.Fatal(err)
	}
	if a.TerminalID != nil {
		t.Fatalf("fresh CLI agent has terminal %v", a.TerminalID)
	}
	tm, err := s.CreateTerminalIn(w.ID, "Claude", w.Path)
	if err != nil {
		t.Fatal(err)
	}
	tid := tm.ID
	got, err := s.UpdateAgent(a.ID, AgentPatch{TerminalID: &tid})
	if err != nil {
		t.Fatal(err)
	}
	if got.TerminalID == nil || *got.TerminalID != tid {
		t.Fatalf("terminalId=%v", got.TerminalID)
	}
	b, err := s.AddAgentWithCLI(w.ID, "codex", "Codex", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.UpdateAgent(b.ID, AgentPatch{TerminalID: &tid}); !errors.Is(err, ErrConflict) {
		t.Fatalf("second bind err=%v", err)
	}
}
