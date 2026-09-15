package tmux

// The machine-socket discovery (ADR-0139 follow-up): every socket in the
// user's tmux directory plus this instance's own, with honest liveness.
// Isolated via TMUX_TMPDIR so nothing of the real machine is read, and the
// liveness probe must never resurrect a dead socket.

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMachineSocketsDiscoverTheMachine(t *testing.T) {
	requireDrainTmux(t)
	ctx := context.Background()
	base := t.TempDir()
	home := filepath.Join(base, "tmuxdir")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv(SocketDirEnv, home) // this test's "user tmux directory"
	sockDir := DefaultSocketDir()
	if err := os.MkdirAll(sockDir, 0o700); err != nil {
		t.Fatal(err)
	}

	// A live server in the user's directory, holding one PiCode session.
	name := SessionName("sockets-" + time.Now().Format("150405-000000000"))
	live := NewWithSocket(filepath.Join(sockDir, "live"))
	if err := live.NewSession(ctx, name, base, "sleep", "30"); err != nil {
		t.Fatalf("live session: %v", err)
	}
	t.Cleanup(func() { _ = live.KillSession(context.Background(), name) })

	// A stale socket file: present, nothing listening.
	stale := filepath.Join(sockDir, "stale")
	if err := os.WriteFile(stale, nil, 0o600); err != nil {
		t.Fatal(err)
	}

	// This instance's own socket, not started yet.
	own := filepath.Join(base, "instance", "tmux.sock")
	m := NewWithSocket(own)

	byName := map[string]MachineSocket{}
	for _, s := range m.MachineSockets(ctx) {
		byName[s.Name] = s
	}
	ours, ok := byName["tmux.sock"]
	if !ok || !ours.Ours || ours.Running {
		t.Fatalf("own socket = %+v (found %v), want ours and not running", ours, ok)
	}
	got, ok := byName["live"]
	if !ok || !got.Running || got.Sessions != 1 || got.PicodeSessions != 1 {
		t.Fatalf("live socket = %+v (found %v), want running with 1 picode session", got, ok)
	}
	dead, ok := byName["stale"]
	if !ok || dead.Running || dead.Sessions != 0 {
		t.Fatalf("stale socket = %+v (found %v), want a dead socket file", dead, ok)
	}

	// Probing must be inert: a second pass still reports the stale socket as
	// dead — the dial must not have started a server on it.
	if again := m.MachineSockets(ctx); func() bool {
		for _, s := range again {
			if s.Name == "stale" && s.Running {
				return true
			}
		}
		return false
	}() {
		t.Fatal("the liveness probe resurrected the stale socket")
	}
	if os.Getenv("TMUX") != "" || os.Getenv("TMUX_PANE") != "" {
		// The test binary's own environment is scrub-clean via tmuxtest;
		// this is a guard for running the test alone.
		t.Log("note: TMUX/TMUX_PANE present in the test process; -S keeps the calls isolated")
	}
}
