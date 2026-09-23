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
	// The test binary doubles as a fake `codex app-server` (codex_login_test.go):
	// a script on PATH re-executes it with this variable set.
	if os.Getenv("PICODE_FAKE_CODEX") == "1" {
		fakeCodexAppServer()
		os.Exit(0)
	}
	if os.Getenv("PICODE_FAKE_GROK") == "1" {
		os.Exit(fakeGrok(os.Args[1:]))
	}
	if os.Getenv("PICODE_FAKE_MUSE") == "1" {
		os.Exit(fakeMuse(os.Args[1:]))
	}
	if os.Getenv("PICODE_FAKE_OPENCODE") == "1" {
		os.Exit(fakeOpencode(os.Args[1:]))
	}
	if os.Getenv("PICODE_FAKE_AGY") == "1" {
		os.Exit(fakeAgy(os.Args[1:]))
	}
	if os.Getenv("PICODE_FAKE_HERMES") == "1" {
		os.Exit(fakeHermes(os.Args[1:]))
	}
	home, err := os.MkdirTemp("", "picode-test-home")
	if err != nil {
		panic(err)
	}
	os.Setenv("HOME", home)
	os.Setenv("USERPROFILE", home)
	os.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	os.Setenv("XDG_DATA_HOME", filepath.Join(home, ".local", "share"))
	os.Setenv("PICODE_TEST_HOME", home)
	// The vault follows PICODE_DATA before HOME (internal/credentials
	// defaultDir) and the agent runtime hands it to every PiCode terminal
	// (internal/rpc/runtime.go). A gate run from one therefore resolved the
	// live vault: 30 test fixtures landed in the owner's real vault on
	// 2026-09-21. Same class as the 2026-09-16 HOME leak below, one variable
	// further along — so it gets the same sandbox and its own guardrail.
	os.Setenv("PICODE_DATA", filepath.Join(home, ".picode"))
	// Omp's roster appends the providers of the omp on PATH (clicreds'
	// omp_catalog.go): a developer machine with Omp installed would get a
	// different roster than CI. Point the reader at nothing.
	os.Setenv("PICODE_OMP_RULES", filepath.Join(home, "no-omp-rules.json"))
	os.Setenv("PICODE_HERMES_CATALOG", filepath.Join(home, "no-hermes-catalog.json"))
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

func TestSuiteIsVaultIsolated(t *testing.T) {
	dir := os.Getenv("PICODE_DATA")
	if dir == "" {
		t.Fatal("PICODE_DATA is unset — the suite would resolve the live vault; keep the TestMain sandbox")
	}
	if !strings.HasPrefix(dir, os.TempDir()) {
		t.Fatalf("PICODE_DATA = %s — outside the temp sandbox; keep the TestMain sandbox", dir)
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
