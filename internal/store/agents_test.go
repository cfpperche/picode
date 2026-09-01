package store

import (
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
