package tmux

// -S rides every command of a socket-bound Manager (ADR-0139), and the
// default Manager keeps the plain shape. These tests use explicit socket
// paths, so they never touch the default socket: -S beats $TMUX and
// TMUX_TMPDIR (the trap class of 2026-09-15).

import (
	"context"
	"os/exec"
	"testing"
)

func TestSocketFlagIsCarried(t *testing.T) {
	sock := socketPath(t, "socket.sock")
	m := NewWithSocket(sock)
	var got [][]string
	m.exec = func(_ context.Context, _ string, args ...string) ([]byte, error) {
		got = append(got, append([]string(nil), args...))
		return []byte("x\n"), nil
	}
	if _, err := m.HasSession(context.Background(), "x"); err != nil {
		t.Fatalf("has-session: %v", err)
	}
	if len(got) == 0 {
		t.Fatal("no tmux call recorded")
	}
	if got[0][0] != "-S" || got[0][1] != sock {
		t.Fatalf("argv = %v, want it to start with -S <path>", got[0])
	}
	if got[0][2] != "has-session" {
		t.Fatalf("argv = %v, want the subcommand after the socket flag", got[0])
	}
}

func TestDefaultManagerKeepsThePlainShape(t *testing.T) {
	m := New()
	var got [][]string
	m.exec = func(_ context.Context, _ string, args ...string) ([]byte, error) {
		got = append(got, append([]string(nil), args...))
		return []byte("x\n"), nil
	}
	if _, err := m.HasSession(context.Background(), "x"); err != nil {
		t.Fatal(err)
	}
	if len(got) == 0 || got[0][0] != "has-session" {
		t.Fatalf("argv = %v, want the plain subcommand (no socket)", got)
	}
}

func requireTmuxBinary(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not installed")
	}
}
