package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/version"
)

// A pre-migration logon task still launches the tool with --tray. That must
// fail loud, on stderr, with the repair named — never as a silent exit zero
// the scheduler would call success, and never as a second resident.
func TestRunRetiredTrayNamesTheRepair(t *testing.T) {
	err := runRetiredTray()
	if err == nil {
		t.Fatal("the retired tray launch succeeded")
	}
	for _, want := range []string{"retired", "startup-repair --retarget-shell"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("retired message %q misses %q", err, want)
		}
	}
}

func TestShellReleaseTagTracksThisBuild(t *testing.T) {
	oldVersion, oldStamped := version.Version, version.Stamped
	defer func() { version.Version, version.Stamped = oldVersion, oldStamped }()
	version.Version, version.Stamped = "0.7.0", "release"
	if got := shellReleaseTag(); got != "v0.7.0" {
		t.Fatalf("stamped tag = %q", got)
	}
	version.Stamped = ""
	if got := shellReleaseTag(); got != "" {
		t.Fatalf("unstamped tag = %q, want latest", got)
	}
}

func TestStageShellExePrefersTheSibling(t *testing.T) {
	dir := t.TempDir()
	shell := filepath.Join(dir, "picode-shell.exe")
	if err := os.WriteFile(shell, []byte("shell"), 0o755); err != nil {
		t.Fatal(err)
	}
	// No network, no LOCALAPPDATA: the sibling beside the tool wins.
	t.Setenv("LOCALAPPDATA", t.TempDir())
	got, err := stageShellExe(filepath.Join(dir, "picode-desktop.exe"))
	if err != nil || got != shell {
		t.Fatalf("stage = %q, %v", got, err)
	}
}
