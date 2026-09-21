package tmux

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// maxSocketPath is the practical ceiling for a unix socket path. The
// kernel struct is `sun_path[108]` on Linux and `sun_path[104]` on
// macOS/BSD, and tmux appends its own name under the directory it is
// given, so the margin below is what a test may build on.
const maxSocketPath = 90

// socketDir returns a private directory for a tmux socket whose path is
// short enough to bind.
//
// t.TempDir() cannot be used for this. It honours TMPDIR, and on a macOS
// runner TMPDIR is already something like
// /var/folders/36/tjdph2t965j8snz9_vkdnw0r0000gn/T/ — 50 characters
// before Go appends the test name and a counter. Every tmux test in this
// package failed on macOS CI with `error connecting to … (File name too
// long)`, which reads like a tmux problem and is really a path-length
// one; the same failure reproduces on Linux by exporting a long TMPDIR.
//
// The cwd a session is created in can stay in t.TempDir(): only the
// socket path is bound.
func socketDir(t *testing.T) string {
	t.Helper()
	base := "/tmp"
	if runtime.GOOS == "windows" {
		base = os.TempDir()
	}
	if _, err := os.Stat(base); err != nil {
		base = os.TempDir()
	}
	dir, err := os.MkdirTemp(base, "pxs")
	if err != nil {
		t.Fatalf("socket dir: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	// A helper that silently returned something unbindable would move the
	// failure, not remove it.
	if n := len(filepath.Join(dir, "tmux.sock")); n > maxSocketPath {
		t.Fatalf("socket dir %s leaves %d bytes of path, over the %d the kernel binds", dir, n, maxSocketPath)
	}
	return dir
}

// socketPath is socketDir plus one name, for the common case.
func socketPath(t *testing.T, name string) string {
	t.Helper()
	return filepath.Join(socketDir(t), name)
}

// samePath compares two paths the way the kernel sees them, not the way
// the test wrote them.
//
// A pane reports the directory it is actually in, which is the resolved
// one. On macOS the test's own fixture is under /var/folders/… and /var
// is a symlink to /private/var, so a literal comparison fails with two
// spellings of one directory — the same aliasing internal/server's
// cleanup.go already resolves for its own paths. It reproduces on Linux
// by pointing TMPDIR at a symlink.
func samePath(t *testing.T, a, b string) bool {
	t.Helper()
	resolve := func(p string) string {
		if r, err := filepath.EvalSymlinks(p); err == nil {
			return filepath.Clean(r)
		}
		return filepath.Clean(p)
	}
	return resolve(a) == resolve(b)
}
