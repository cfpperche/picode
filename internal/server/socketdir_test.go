package server

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// shortSocketDir returns a private directory for a tmux socket whose path
// is short enough for the kernel to bind.
//
// t.TempDir() is not usable for this. It honours TMPDIR, and a macOS runner
// sets TMPDIR to /var/folders/<2>/<28>/T/ — 49 characters before Go appends
// the test name — while tmux then binds <dir>/tmux-<uid>/default. The total
// passes the sun_path ceiling (108 bytes on Linux, 104 on macOS/BSD) and the
// failure reads `error connecting to … (File name too long)`, which looks
// like a tmux fault and is a path-length one. internal/tmux has the same
// helper for the same reason; these fixtures build their own sockets, so
// they need it too.
//
// Only the socket goes here. A session's working directory can stay in
// t.TempDir(), since nothing binds it.
func shortSocketDir(t *testing.T) string {
	t.Helper()
	base := "/tmp"
	if runtime.GOOS == "windows" {
		base = os.TempDir()
	}
	if st, err := os.Stat(base); err != nil || !st.IsDir() {
		base = os.TempDir()
	}
	dir, err := os.MkdirTemp(base, "pcs")
	if err != nil {
		t.Fatalf("socket dir: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	// A helper that quietly returned something unbindable would move the
	// failure rather than remove it.
	if n := len(filepath.Join(dir, "tmux-000000", "default")); n > 100 {
		t.Fatalf("socket dir %s leaves %d bytes of path, past what the kernel binds", dir, n)
	}
	return dir
}
