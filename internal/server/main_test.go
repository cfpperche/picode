package server

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/tmux"
	"github.com/cfpperche/picode/internal/tmuxtest"
)

// The suite starts real tmux servers (terminal feed and CLI fixtures). It
// runs them on a private namespace so `make ci` cannot leave orphan shells
// on the user's server; see internal/tmuxtest for the measured reason
// TMUX_TMPDIR alone is not enough.
//
// The suite also boots real servers, and boot seeds Activity on and syncs
// integration files — including the two CLIs whose install path is the
// user's own settings (agy title reporter, muse hooks). Without an
// isolated HOME those writes land in the developer's real config files
// (measured 2026-09-16: TestBackupAPI merged dead /tmp hook paths into
// ~/.config/muse/settings.json and the agy settings). Per-test Setenv
// cannot cover the 30+ files that boot servers, so the whole process gets
// a throwaway HOME here; tests that need a second one still Setenv their
// own on top.
func TestMain(m *testing.M) {
	home, err := os.MkdirTemp("", "picode-test-home")
	if err != nil {
		panic(err)
	}
	os.Setenv("HOME", home)
	os.Setenv("USERPROFILE", home)
	os.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	os.Setenv("XDG_DATA_HOME", filepath.Join(home, ".local", "share"))
	os.Setenv("PICODE_TEST_HOME", home)
	os.Exit(tmuxtest.Main(m))
}

// TestSuiteIsTmuxIsolated is the guardrail's guardrail: removing TestMain
// above would quietly put the suite back on the user's server (it left
// twelve orphan shells on 2026-09-15), so the contract is asserted, not
// assumed.
func TestSuiteIsHomeIsolated(t *testing.T) {
	if os.Getenv("PICODE_TEST_HOME") == "" {
		t.Fatal("PICODE_TEST_HOME is unset — the suite is not isolated; keep the TestMain HOME sandbox")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	// The sandbox lives under the system temp dir; the developer's real
	// HOME never does. A server booted by any test in this process may
	// merge integration files into $HOME ($XDG_CONFIG_HOME for muse), so
	// this must stay a temp dir or owner configs get polluted again.
	if !strings.HasPrefix(home, os.TempDir()) {
		t.Fatalf("UserHomeDir = %s — outside the temp sandbox; keep the TestMain HOME sandbox", home)
	}
}

func TestSuiteIsTmuxIsolated(t *testing.T) {
	if os.Getenv(tmux.SocketDirEnv) == "" {
		t.Fatalf("%s is unset — the suite is not isolated; keep TestMain(tmuxtest.Main)", tmux.SocketDirEnv)
	}
	if os.Getenv("TMUX") != "" {
		t.Fatal("TMUX is set in the test process — it outranks TMUX_TMPDIR and would retarget tmux calls at the user's server")
	}
}
