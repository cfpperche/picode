package tmux

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
// socket file disappears and a plain `tmux ls` (which inherits the same
// environment) sees no server.
func TestKillIsolatedServerEndsAServerInItsOwnDirectory(t *testing.T) {
	if !New().Available() {
		t.Skip("tmux not installed")
	}
	dir := t.TempDir()
	t.Setenv(SocketDirEnv, dir) // every tmux call in this test now goes there
	if err := New().NewSession(context.Background(), SessionName("guard-test"), dir, "sleep", "30"); err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	if got, err := New().ListSessions(context.Background()); err != nil || len(got) != 1 {
		t.Fatalf("private server should hold the session: %v, %v", got, err)
	}
	if err := KillIsolatedServer(context.Background(), dir); err != nil {
		t.Fatalf("KillIsolatedServer: %v", err)
	}
	if got, err := New().ListSessions(context.Background()); err != nil || len(got) != 0 {
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
