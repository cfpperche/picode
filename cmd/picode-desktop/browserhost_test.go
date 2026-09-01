package main

import (
	"os"
	"testing"

	"github.com/cfpperche/picode/internal/browserhost"
)

func TestCommandTreatsChromeLaunchAsHost(t *testing.T) {
	old := os.Args
	t.Cleanup(func() { os.Args = old })
	os.Args = []string{
		"picode-desktop",
		"chrome-extension://" + browserhost.ExtensionID + "/",
		"--parent-window=12345",
	}
	if !browserhost.IsHostArg(command()) {
		t.Fatalf("command() = %q, want chrome origin", command())
	}
}
