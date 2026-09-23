package tmux

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeScript(t *testing.T, dir, body string) string {
	t.Helper()
	p := filepath.Join(dir, "tmux")
	if err := os.WriteFile(p, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

// PiCode's own tmux calls skip the guard wrapper on a terminal's PATH and
// reach the next tmux — a fake one in a test is still found.
func TestBinarySkipsTheInterceptWrapper(t *testing.T) {
	guard, fake := t.TempDir(), t.TempDir()
	writeScript(t, guard, "#!/bin/sh\n# PiCode intercept — tmux guard (ADR-0138).\nexit 1\n")
	want := writeScript(t, fake, "#!/bin/sh\nexit 0\n")
	t.Setenv("PATH", guard+string(os.PathListSeparator)+fake)
	if got := Binary(); got != want {
		t.Fatalf("Binary() = %q, want %q", got, want)
	}
	t.Setenv("PATH", guard)
	if got := Binary(); got != "tmux" {
		t.Fatalf("only a wrapper on PATH: Binary() = %q, want the plain name", got)
	}
}

// The case that leaked the docs fixture's servers: a guard that refuses
// kill-server ahead of the real tmux. A private server is still ended.
func TestKillServerThroughAGuardedPath(t *testing.T) {
	real := Binary()
	if real == "tmux" {
		t.Skip("tmux not installed")
	}
	guard := t.TempDir()
	writeScript(t, guard, "#!/bin/sh\n# PiCode intercept — tmux guard (ADR-0138).\necho 'picode tmux guard: refused.' >&2\nexit 1\n")
	t.Setenv("PATH", guard+string(os.PathListSeparator)+filepath.Dir(real))
	t.Setenv("TMUX", "")
	sock := socketPath(t, "tmux.sock")
	m := NewWithSocket(sock)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := m.NewSession(ctx, "picode-sh-guarded-kill", t.TempDir(), "sh"); err != nil {
		t.Fatalf("new session: %v", err)
	}
	if err := m.KillServer(ctx); err != nil {
		t.Fatalf("kill-server through a guarded PATH: %v", err)
	}
	if alive, _ := m.HasSession(ctx, "picode-sh-guarded-kill"); alive {
		t.Fatal("the private server survived")
	}
}

// The instance's own server is never PiCode's to kill from a harness.
func TestKillServerRefusesTheInstanceSocket(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home")
	}
	if err := refuseUserServer(filepath.Join(home, ".picode", "tmux.sock")); err == nil {
		t.Fatal("the production instance's socket was not refused")
	}
	if err := refuseUserServer(socketPath(t, "tmux.sock")); err != nil {
		t.Fatalf("a private socket was refused: %v", err)
	}
}
