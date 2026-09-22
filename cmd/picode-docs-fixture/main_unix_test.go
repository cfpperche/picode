//go:build !windows

// The fixture's shutdown test is Unix-only: it watches the pane's own
// process with syscall.Kill(pid, 0), which Go does not define on Windows.
// Splitting it out keeps TestPortSuffix compiling in the Windows job —
// that job exists to compile every package and test, and it went red on
// `undefined: syscall.Kill` rather than on anything the daemon does.
// The daemon itself runs on Linux/macOS, and inside WSL on Windows
// (ADR-0020), so there is nothing here for a native Windows run to check.

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
