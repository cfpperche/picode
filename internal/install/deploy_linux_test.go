//go:build linux

package install

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// The whole point: an update that cannot restart must not copy. Before this,
// Deploy left the new binary on disk with the old one running — an update that
// looked done and was not. This is Linux-only because Deploy uses a systemd
// user unit there; Windows has a separate desktop-host update path.
func TestDeployRefusesBeforeCopyingWhenThereIsNoSession(t *testing.T) {
	home := t.TempDir()
	p := ForHome(home)
	if err := os.MkdirAll(filepath.Dir(p.Unit), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p.Unit, []byte("[Unit]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(p.Bin), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p.Bin, []byte("OLD BINARY"), 0o755); err != nil {
		t.Fatal(err)
	}

	src := filepath.Join(t.TempDir(), "picode-new")
	if err := os.WriteFile(src, []byte("NEW BINARY"), 0o755); err != nil {
		t.Fatal(err)
	}

	prevCheck, prevRun := EnsureUserSession, Run
	t.Cleanup(func() { EnsureUserSession, Run = prevCheck, prevRun })
	EnsureUserSession = func() error { return ErrNoUserSession }
	ran := 0
	Run = func(string, ...string) error { ran++; return nil }

	err := Deploy(src, home, "/usr/bin")
	if !errors.Is(err, ErrNoUserSession) {
		t.Fatalf("got %v, want ErrNoUserSession", err)
	}
	if ran != 0 {
		t.Fatalf("nothing should have been run, got %d calls", ran)
	}
	body, readErr := os.ReadFile(p.Bin)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(body) != "OLD BINARY" {
		t.Fatalf("the installed binary was replaced anyway: %q", body)
	}
}
