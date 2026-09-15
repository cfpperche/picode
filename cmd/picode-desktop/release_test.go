package main

import (
	"os"
	"strings"
	"testing"
)

// The asset name is a contract between this program and the release workflow.
// Nothing fails loudly if they drift: `update` would simply never find a
// download, on every machine, forever.
func TestReleaseWorkflowPublishesTheAssetUpdateLooksFor(t *testing.T) {
	b, err := os.ReadFile("../../.github/workflows/release.yml")
	if err != nil {
		t.Fatal(err)
	}
	workflow := string(b)

	if !strings.Contains(workflow, DesktopAsset) {
		t.Errorf("the release workflow does not build %q", DesktopAsset)
	}
	// The shell ships in the same tag (ADR-0142); `update` refuses a
	// release without it, so a workflow that drops the asset breaks every
	// update instead of half of one.
	if !strings.Contains(workflow, ShellAsset) {
		t.Errorf("the release workflow does not build %q", ShellAsset)
	}
	// internal/install.assetName() builds this shape for picode itself.
	if !strings.Contains(workflow, "picode-${goos}-${goarch}") {
		t.Error("the release workflow does not build picode-<goos>-<goarch>, which `picode update` looks for")
	}
	// A release that does not stamp the version ships a binary claiming to be
	// the source default, so `update` would offer the same upgrade forever.
	if !strings.Contains(workflow, "internal/version.Version=") {
		t.Error("the release workflow does not stamp the version")
	}
}
