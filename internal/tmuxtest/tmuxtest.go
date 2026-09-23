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
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/tmux"
)

// Main scrubs the test process onto a private tmux namespace, runs the
// suite, and tears the namespace down. Errors are best-effort: a missing
// tmux binary must not fail the suite, and a cleanup failure must not
// replace the suite's own verdict.
func Main(m *testing.M) int {
	dir, err := os.MkdirTemp(namespaceBase(), "picode-tmuxtest-")
	if err != nil {
		return m.Run() // no isolation available; run as before
	}
	_ = os.Unsetenv("TMUX")
	_ = os.Unsetenv("TMUX_PANE")
	// Marked before the env points at it: the kill on the way out must not
	// read this private directory as the user's (it did, and every suite
	// leaked its server — 2026-09-23).
	tmux.MarkIsolated(dir)
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

// namespaceBase is TMPDIR, unless a socket under it would not fit.
//
// tmux binds <namespace>/tmux-<uid>/default, and the kernel's sun_path is
// 108 bytes on Linux and 104 on macOS/BSD. On a macOS runner TMPDIR is
// /var/folders/<2>/<28>/T/ — 49 characters — and the namespace socket
// lands at 92, which fits; the tests that failed there built their own
// deeper socket paths and are fixed in the package's own fixtures. So
// this stays on TMPDIR wherever it already works, and only falls back
// when it genuinely does not: forcing /tmp unconditionally changed the
// namespace on runners that never had a problem, and internal/tmux went
// red on ubuntu for it.
func namespaceBase() string {
	base := os.TempDir()
	if len(filepath.Join(base, "picode-tmuxtest-1234567890", "tmux-"+strconv.Itoa(os.Getuid()), "default")) <= sunPathLimit {
		return base
	}
	if runtime.GOOS != "windows" {
		if st, err := os.Stat("/tmp"); err == nil && st.IsDir() {
			return "/tmp"
		}
	}
	return base
}

// sunPathLimit is the smaller of the two platform ceilings, so the check
// above is conservative everywhere.
const sunPathLimit = 104
