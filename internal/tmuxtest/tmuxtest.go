// Package tmuxtest isolates a test binary's tmux activity onto a private
// server, so a suite that creates sessions can never touch the user's own:
// on 2026-09-15 `make ci` left twelve orphan shells on production
// (`picode-sh-feed-fixture-*`, `picode-sh-pi-from-claude-code-race-fix-*`),
// and a stalled branch carried an unguarded `kill-server` from a 2026-09-13
// run that took 140 sessions with it.
//
// The scheme is TMUX_TMPDIR, not `tmux -L`: -L has to ride every argv, and a
// test that shells out to `tmux` itself would miss it, while TMUX_TMPDIR is
// inherited by every child the harness spawns. What TMUX_TMPDIR alone does
// NOT do (measured 2026-09-15) is beat `$TMUX`: a process started inside a
// session talks to the server named in `$TMUX` no matter what TMUX_TMPDIR
// says. Main therefore scrubs TMUX and TMUX_PANE for the whole test binary,
// which is the only place that fix can live for grandchildren the tests
// spawn.
//
// Usage, once per package that can start a tmux server:
//
//	func TestMain(m *testing.M) { os.Exit(tmuxtest.Main(m)) }
//
// Cleanup kills the private server by its own socket directory (through
// tmux.KillIsolatedServer, which refuses the user's directory) and removes
// the directory; leftovers from a flaky test die with it, which is why no
// name-matching sweep exists anywhere in this repository.
package tmuxtest

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/tmux"
)

// Main scrubs the test process onto a private tmux namespace, runs the
// suite, and tears the namespace down. Errors are best-effort: a missing
// tmux binary must not fail the suite, and a cleanup failure must not
// replace the suite's own verdict.
func Main(m *testing.M) int {
	dir, err := os.MkdirTemp("", "picode-tmuxtest-")
	if err != nil {
		return m.Run() // no isolation available; run as before
	}
	_ = os.Unsetenv("TMUX")
	_ = os.Unsetenv("TMUX_PANE")
	_ = os.Setenv(tmux.SocketDirEnv, dir)

	code := m.Run()

	// The environment is NOT restored on the way out, deliberately: a test
	// server's watchers can still be finishing a tick when m.Run returns,
	// and restoring TMUX_TMPDIR would let that straggler re-arm the leak
	// (observed 2026-09-15: two sessions landed on the live server in the
	// seconds around cleanup). The process is about to exit; the isolated
	// environment outliving the suite costs nothing and a straggler's tmux
	// client just starts an empty server in a directory that is about to
	// disappear. Two kills with a grace between them catch a straggler
	// mid-flight.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = tmux.KillIsolatedServer(ctx, dir)
	time.Sleep(300 * time.Millisecond)
	_ = tmux.KillIsolatedServer(ctx, dir)
	_ = os.RemoveAll(dir)
	return code
}
