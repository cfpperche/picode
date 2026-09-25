package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cfpperche/picode/internal/version"
)

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
