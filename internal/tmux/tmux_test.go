package tmux

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// These are integration tests: they require a tmux binary. They run on
// dev machines and on CI (ubuntu installs tmux — see ci.yml); elsewhere
// they skip, which is recorded as accepted debt in docs/handoff.md.
func requireTmux(t *testing.T) *Manager {
	t.Helper()
	m := New()
	if !m.Available() {
		t.Skip("tmux not installed — integration test skipped (accepted, see docs/handoff.md)")
	}
	return m
}

func TestVersionParses(t *testing.T) {
	m := requireTmux(t)
	v, err := m.Version()
	if err != nil {
		t.Fatalf("Version: %v", err)
	}
	if v == "" || strings.Contains(v, "tmux") {
		t.Errorf("Version() = %q, want bare version like 3.6", v)
	}
}

func TestNewHasListKillSession(t *testing.T) {
	m := requireTmux(t)
	ctx := context.Background()
	name := SessionName("test-" + time.Now().Format("150405-000000000"))

	if err := m.NewSession(ctx, name, t.TempDir(), "sleep", "10"); err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	t.Cleanup(func() { _ = m.KillSession(ctx, name) })

	has, err := m.HasSession(ctx, name)
	if err != nil || !has {
		t.Fatalf("HasSession after create = %v, %v; want true, nil", has, err)
	}

	if err := m.NewSession(ctx, name, t.TempDir(), "sleep", "10"); err == nil {
		t.Error("NewSession on existing name: want error, got nil")
	}

	sessions, err := m.ListSessions(ctx)
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	found := false
	for _, s := range sessions {
		if s.Name == name {
			found = true
			if s.Windows < 1 {
				t.Errorf("session %s windows = %d, want >= 1", name, s.Windows)
			}
		}
	}
	if !found {
		t.Errorf("ListSessions: session %q not found among %d sessions", name, len(sessions))
	}

	if err := m.KillSession(ctx, name); err != nil {
		t.Fatalf("KillSession: %v", err)
	}
	has, err = m.HasSession(ctx, name)
	if err != nil || has {
		t.Fatalf("HasSession after kill = %v, %v; want false, nil", has, err)
	}

	// Killing a missing session is a no-op, not an error.
	if err := m.KillSession(ctx, name); err != nil {
		t.Errorf("KillSession on missing session: want nil, got %v", err)
	}
}

func TestHasSessionMissing(t *testing.T) {
	m := requireTmux(t)
	has, err := m.HasSession(context.Background(), "picode-definitely-missing-1234")
	if err != nil {
		t.Fatalf("HasSession missing: %v", err)
	}
	if has {
		t.Error("HasSession on random name = true, want false")
	}
}

func TestSendKeysReachesSession(t *testing.T) {
	m := requireTmux(t)
	ctx := context.Background()
	name := SessionName("sendkeys-" + time.Now().Format("150405-000000000"))
	if err := m.NewSession(ctx, name, t.TempDir(), "sleep", "10"); err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	t.Cleanup(func() { _ = m.KillSession(ctx, name) })
	if err := m.SendKeys(ctx, name, "true", "Enter"); err != nil {
		t.Fatalf("SendKeys: %v", err)
	}
}

func TestEnsureExtendedKeysXterm(t *testing.T) {
	m := requireTmux(t)
	ctx := context.Background()
	name := SessionName("extkeys-" + time.Now().Format("150405-000000000"))
	if err := m.NewSession(ctx, name, t.TempDir(), "sleep", "10"); err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	t.Cleanup(func() { _ = m.KillSession(ctx, name) })
	f, err := m.ExtendedKeysFormat(ctx)
	if err != nil {
		t.Fatalf("ExtendedKeysFormat: %v", err)
	}
	if f != "xterm" {
		t.Fatalf("extended-keys-format = %q, want xterm", f)
	}
}

func TestSessionNamePrefix(t *testing.T) {
	if got := SessionName("abc"); got != "picode-abc" {
		t.Errorf("SessionName(abc) = %q, want picode-abc", got)
	}
	if got := ShellSessionName("abc"); got != "picode-sh-abc" {
		t.Errorf("ShellSessionName(abc) = %q, want picode-sh-abc", got)
	}
	if !IsShellSession("picode-sh-abc") {
		t.Error("IsShellSession(picode-sh-abc) = false")
	}
	if IsShellSession(SessionName("abc")) {
		t.Error("IsShellSession on agent session = true")
	}
}

func TestSessionNameSanitizes(t *testing.T) {
	cases := map[string]string{
		"My Project":  "picode-my-project",
		"a.b:c":       "picode-a-b-c",
		"UPPER_case":  "picode-upper-case",
		"sp/ac/es":    "picode-sp-ac-es",
		"trailing---": "picode-trailing",
	}
	for in, want := range cases {
		if got := SessionName(in); got != want {
			t.Errorf("SessionName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestPaneCwdFollowsProcess(t *testing.T) {
	m := requireTmux(t)
	ctx := context.Background()
	start := t.TempDir()
	name := SessionName("cwd-" + time.Now().Format("150405-000000000"))
	if err := m.NewSession(ctx, name, start, "sleep", "30"); err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	t.Cleanup(func() { _ = m.KillSession(ctx, name) })
	got, err := m.PaneCwd(ctx, name)
	if err != nil {
		t.Fatalf("PaneCwd: %v", err)
	}
	if filepath.Clean(got) != filepath.Clean(start) {
		t.Fatalf("PaneCwd = %q, want %q", got, start)
	}

	live := t.TempDir()
	name2 := SessionName("cwd2-" + time.Now().Format("150405-000000000"))
	if err := m.NewSession(ctx, name2, start, "sh", "-c", "cd "+live+" && sleep 30"); err != nil {
		t.Fatalf("NewSession cd: %v", err)
	}
	t.Cleanup(func() { _ = m.KillSession(ctx, name2) })
	deadline := time.Now().Add(2 * time.Second)
	var got2 string
	for time.Now().Before(deadline) {
		got2, err = m.PaneCwd(ctx, name2)
		if err == nil && filepath.Clean(got2) == filepath.Clean(live) {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("PaneCwd after cd = %q (err %v), want %q", got2, err, live)
}
