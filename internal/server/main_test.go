package server

import (
	"os"
	"testing"

	"github.com/cfpperche/picode/internal/tmux"
	"github.com/cfpperche/picode/internal/tmuxtest"
)

// The suite starts real tmux servers (terminal feed and CLI fixtures). It
// runs them on a private namespace so `make ci` cannot leave orphan shells
// on the user's server; see internal/tmuxtest for the measured reason
// TMUX_TMPDIR alone is not enough.
func TestMain(m *testing.M) { os.Exit(tmuxtest.Main(m)) }

// TestSuiteIsTmuxIsolated is the guardrail's guardrail: removing TestMain
// above would quietly put the suite back on the user's server (it left
// twelve orphan shells on 2026-09-15), so the contract is asserted, not
// assumed.
func TestSuiteIsTmuxIsolated(t *testing.T) {
	if os.Getenv(tmux.SocketDirEnv) == "" {
		t.Fatalf("%s is unset — the suite is not isolated; keep TestMain(tmuxtest.Main)", tmux.SocketDirEnv)
	}
	if os.Getenv("TMUX") != "" {
		t.Fatal("TMUX is set in the test process — it outranks TMUX_TMPDIR and would retarget tmux calls at the user's server")
	}
}
