package tmux

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// The guard is the safety property of this file: a harness may end the server
// it made, never the one the human is working in. Both refusals are asserted,
// because the failure they prevent is not recoverable — an unguarded
// `kill-server` on 2026-09-13 took 140 sessions with it.
func TestKillIsolatedServerRefusesTheUsersSocketDir(t *testing.T) {
	for _, tc := range []struct{ name, dir string }{
		{"empty", ""},
		{"the user's default", DefaultSocketDir()},
		{"the default with a trailing slash", DefaultSocketDir() + "/"},
		{"the default undone by ..", filepath.Join(DefaultSocketDir(), "..", filepath.Base(DefaultSocketDir()))},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := KillIsolatedServer(context.Background(), tc.dir)
			if err == nil {
				t.Fatalf("KillIsolatedServer(%q) was allowed", tc.dir)
			}
			if !strings.Contains(err.Error(), "refusing") {
				t.Fatalf("error does not say it refused: %v", err)
			}
		})
	}
}

// DefaultSocketDir follows tmux's own order ($TMUX_TMPDIR, then the OS temp
// dir) so the refusal above tracks where tmux actually keeps the socket.
func TestDefaultSocketDirFollowsTmuxsOwnOrder(t *testing.T) {
	t.Setenv(SocketDirEnv, "")
	if got, want := DefaultSocketDir(), filepath.Join(os.TempDir(), "tmux-"+itoa(os.Getuid())); got != want {
		t.Fatalf("default = %q, want %q", got, want)
	}
	t.Setenv(SocketDirEnv, "/tmp/somewhere-else")
	if got, want := DefaultSocketDir(), "/tmp/somewhere-else/tmux-"+itoa(os.Getuid()); got != want {
		t.Fatalf("with TMUX_TMPDIR set, default = %q, want %q", got, want)
	}
}

// A private directory is accepted, and killing the server there ends it: the
// session list comes back empty. The fixture addresses its server by -S
// rather than by rebinding the process-wide TMUX_TMPDIR — this test ends a
// server, and the env shape pointed every other tmux call in the binary at
// the one being dismantled (2026-09-15). KillIsolatedServer still takes the
// *directory*: it derives tmux's socket path from it, which is the same path
// the manager below carries, so the two halves agree without an env.
func TestKillIsolatedServerEndsAServerInItsOwnDirectory(t *testing.T) {
	if !New().Available() {
		t.Skip("tmux not installed")
	}
	dir := t.TempDir()
	m := NewWithSocket(filepath.Join(dir, "tmux-"+itoa(os.Getuid()), "default"))
	if err := m.NewSession(context.Background(), SessionName("guard-test"), dir, "sleep", "30"); err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	if got, err := m.ListSessions(context.Background()); err != nil || len(got) != 1 {
		t.Fatalf("private server should hold the session: %v, %v", got, err)
	}
	if err := KillIsolatedServer(context.Background(), dir); err != nil {
		t.Fatalf("KillIsolatedServer: %v", err)
	}
	if got, err := m.ListSessions(context.Background()); err != nil || len(got) != 0 {
		t.Fatalf("server survived the kill: %v, %v", got, err)
	}
}

// Nothing to kill is not a failure: every harness calls this on the way out,
// including a run whose tests never opened a session.
func TestKillIsolatedServerWithoutAServerIsFine(t *testing.T) {
	if !New().Available() {
		t.Skip("tmux not installed")
	}
	if err := KillIsolatedServer(context.Background(), t.TempDir()); err != nil {
		t.Fatalf("KillIsolatedServer on an empty dir: %v", err)
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

// A socket inside a directory that does not exist yet is the case that made
// the failure above loud: tmux creates the socket, not its directory, and it
// exits 0 while printing `error creating …`. NewWithSocket makes the
// directory, so the session really exists and the list shows it.
func TestNewWithSocketMakesItsDirectory(t *testing.T) {
	if !New().Available() {
		t.Skip("tmux not installed")
	}
	dir := t.TempDir()
	m := NewWithSocket(filepath.Join(dir, "fresh", "deeper", "tmux-"+itoa(os.Getuid()), "default"))
	name := SessionName("socket-dir-" + time.Now().Format("150405-000000000"))
	if err := m.NewSession(context.Background(), name, dir, "sleep", "30"); err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	t.Cleanup(func() { _ = m.KillSession(context.Background(), name) })
	got, err := m.ListSessions(context.Background())
	if err != nil || len(got) != 1 || got[0].Name != name {
		t.Fatalf("sessions = %+v, %v; want just %q", got, err, name)
	}
}

// And a path that cannot be a socket directory fails loudly instead of
// pretending: the manager's answer to a silent no-op.
func TestNewWithSocketReportsAnUnusablePath(t *testing.T) {
	if !New().Available() {
		t.Skip("tmux not installed")
	}
	dir := t.TempDir()
	blocker := filepath.Join(dir, "not-a-directory")
	if err := os.WriteFile(blocker, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	m := NewWithSocket(filepath.Join(blocker, "default"))
	err := m.NewSession(context.Background(), SessionName("unusable"), dir, "sleep", "5")
	if err == nil {
		t.Fatal("a session on an unusable socket path must be an error, not silence")
	}
	if !strings.Contains(err.Error(), "error creating") && !strings.Contains(err.Error(), "error connecting") {
		t.Fatalf("error should name tmux's complaint: %v", err)
	}
}
