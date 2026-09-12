//go:build linux

package install

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
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

// ADR-0086: a deploy while anyone is mid-turn is refused before the binary
// is touched; --force goes through; a silent daemon never blocks.
func TestDeployRefusesWhileAgentsWork(t *testing.T) {
	// The suite may itself run inside a PiCode pane; the guard drops the
	// caller's own id, so clear it to keep this table about other people.
	t.Setenv("PICODE_TERM_ID", "")
	t.Setenv("PICODE_AGENT_ID", "")
	home := t.TempDir()
	p := ForHome(home)
	for _, d := range []string{filepath.Dir(p.Unit), filepath.Dir(p.Bin)} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(p.Unit, []byte("[Unit]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p.Bin, []byte("OLD BINARY"), 0o755); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(t.TempDir(), "picode-new")
	if err := os.WriteFile(src, []byte("NEW BINARY"), 0o755); err != nil {
		t.Fatal(err)
	}

	prevCheck, prevRun, prevReady := EnsureUserSession, Run, Readiness
	t.Cleanup(func() { EnsureUserSession, Run, Readiness = prevCheck, prevRun, prevReady })
	EnsureUserSession = func() error { return nil }
	ran := 0
	Run = func(string, ...string) error { ran++; return nil }
	Readiness = func(string) ([]Busy, error) {
		return []Busy{{Kind: "terminal", ID: "t1", Name: "codex", Why: "working"}}, nil
	}

	err := DeployForce(src, home, "/usr/bin", false)
	if !errors.Is(err, ErrDeployBusy) {
		t.Fatalf("got %v, want ErrDeployBusy", err)
	}
	if !strings.Contains(err.Error(), `terminal "codex" is working`) {
		t.Fatalf("refusal must name who is busy: %v", err)
	}
	if ran != 0 {
		t.Fatalf("nothing should have been run, got %d calls", ran)
	}
	if body, _ := os.ReadFile(p.Bin); string(body) != "OLD BINARY" {
		t.Fatalf("the installed binary was replaced anyway: %q", body)
	}

	if err := DeployForce(src, home, "/usr/bin", true); err != nil {
		t.Fatalf("--force must deploy: %v", err)
	}
	if ran == 0 {
		t.Fatal("--force must restart the unit")
	}
	if body, _ := os.ReadFile(p.Bin); string(body) != "NEW BINARY" {
		t.Fatalf("--force must replace the binary, got %q", body)
	}

	Readiness = func(string) ([]Busy, error) { return nil, nil }
	if err := DeployForce(src, home, "/usr/bin", false); err != nil {
		t.Fatalf("an idle fleet must deploy: %v", err)
	}
}
