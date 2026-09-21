package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/tmux"
)

// The directory name must be stable for a given address (captures compare
// pixel for pixel) and a single safe path element whatever the flag holds.
func TestPortSuffix(t *testing.T) {
	cases := []struct{ addr, want string }{
		{"127.0.0.1:18740", "18740"},
		{":18741", "18741"},
		{"[::1]:18742", "18742"},
		{"localhost", "localhost"},
		{"", "default"},
		{"host:odd chars", "odd_chars"},                // the port part wins when the address parses
		{"host/with odd chars", "host_with_odd_chars"}, // no port: the whole address, reduced
	}
	for _, tc := range cases {
		if got := portSuffix(tc.addr); got != tc.want {
			t.Errorf("portSuffix(%q) = %q, want %q", tc.addr, got, tc.want)
		}
	}
}

// The captured world's terminals run on the fixture's private server, and
// shutdown is what ends them: the capture pipeline kills the process with
// SIGTERM, and a cleanup that only removed the directory would leave that
// server running invisibly, still holding every pane's shell — seven of them
// had piled up by 2026-09-21. The observable is the pane's own process, not
// the socket: an unlinked socket cannot answer either way.
func TestShutdownEndsTheServerBeforeTheDirectory(t *testing.T) {
	if !tmux.New().Available() {
		t.Skip("tmux not installed")
	}
	// The socket path is what the kernel binds, and t.TempDir() honours TMPDIR
	// (50 characters on a macOS runner before Go appends the test name), so the
	// socket gets a short directory of its own — internal/tmux/socketdir_test.go
	// measures the same hazard for the package that owns the manager.
	dir, err := os.MkdirTemp("/tmp", "pxs")
	if err != nil {
		t.Fatalf("socket dir: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	m := tmux.NewWithSocket(filepath.Join(dir, "tmux.sock"))
	ctx := context.Background()
	name := tmux.SessionName("fixture-shutdown-test")
	if err := m.NewSession(ctx, name, dir, "sleep", "300"); err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	pid, err := m.PanePID(ctx, name)
	if err != nil || pid <= 0 {
		t.Fatalf("PanePID = %d, %v", pid, err)
	}

	shutdown(dir, m)

	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("data directory survived shutdown: %v", err)
	}
	// A killed pane can sit unreaped for a moment; the server that leaked
	// never lets go of it at all.
	deadline := time.Now().Add(3 * time.Second)
	for {
		err := syscall.Kill(pid, 0)
		if errors.Is(err, syscall.ESRCH) {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("pane %d still alive after shutdown (%v) — the private server leaked", pid, err)
		}
		time.Sleep(100 * time.Millisecond)
	}
}
