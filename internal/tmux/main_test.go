// External test package on purpose: tmuxtest imports internal/tmux, so a
// TestMain in package tmux would be an import cycle. Both packages compile
// into the same test binary, so this TestMain governs the whole suite.
package tmux_test

import (
	"os"
	"testing"

	"github.com/cfpperche/picode/internal/tmuxtest"
)

// These are integration tests against a real tmux binary; the namespace is
// private so a run never touches the user's server (a 2026-09-13 harness
// killed 140 of its sessions).
func TestMain(m *testing.M) { os.Exit(tmuxtest.Main(m)) }
