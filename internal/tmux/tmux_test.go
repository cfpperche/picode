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
	waitCwd := func(name, want string) {
		t.Helper()
		deadline := time.Now().Add(2 * time.Second)
		var got string
		var err error
		for time.Now().Before(deadline) {
			got, err = m.PaneCwd(ctx, name)
			if err == nil && filepath.Clean(got) == filepath.Clean(want) {
				return
			}
			time.Sleep(50 * time.Millisecond)
		}
		t.Fatalf("PaneCwd = %q (err %v), want %q", got, err, want)
	}

	start := t.TempDir()
	name := SessionName("cwd-" + time.Now().Format("150405-000000000"))
	if err := m.NewSession(ctx, name, start, "sleep", "30"); err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	t.Cleanup(func() { _ = m.KillSession(ctx, name) })
	// A detached tmux session briefly reports the server process and its cwd
	// before the pane child has exec'd. The API contract is the live process
	// cwd, so assert convergence rather than scheduler timing.
	waitCwd(name, start)

	live := t.TempDir()
	name2 := SessionName("cwd2-" + time.Now().Format("150405-000000000"))
	if err := m.NewSession(ctx, name2, start, "sh", "-c", "cd "+live+" && sleep 30"); err != nil {
		t.Fatalf("NewSession cd: %v", err)
	}
	t.Cleanup(func() { _ = m.KillSession(ctx, name2) })
	waitCwd(name2, live)
}

// NewSession must create surfaces without the tmux status line (it renders
// as a green bar at the bottom of the web terminal) — parity for agent TUI
// sessions and first-class terminals.
func TestNewSessionNoStatusBar(t *testing.T) {
	m := New()
	if !m.Available() {
		t.Skip("tmux not installed")
	}
	ctx := context.Background()
	name := SessionName("nostatus-" + time.Now().Format("150405-000000000"))
	if err := m.NewSession(ctx, name, t.TempDir(), "sleep", "30"); err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	t.Cleanup(func() { _ = m.KillSession(ctx, name) })
	// Session-scoped option (-v -t name:), not the global (-g) one.
	out, err := m.run(ctx, "show-options", "-v", "-t", name+":", "status")
	if err != nil {
		t.Fatalf("show-options status: %v", err)
	}
	if got := strings.TrimSpace(out); got != "off" {
		t.Errorf("session status = %q, want off", got)
	}
}

// Real pane tails captured on the owner's machine (agente-auto, pi
// 0.84.4, zai/glm-5.3-flash), trimmed to the shape that matters. The
// borders are pi's input-box drawing; the last two lines are the footer.
const (
	tailWorking = " ⠹ Working...\n" +
		"\n" +
		"────────\n" +
		"\n" +
		"────────\n" +
		"~/.picode/work/agente-auto\n" +
		"↑17k ↓799 R57k CH0.0% $0.002 0.9%/1.0M (auto)   (zai) glm-5.3-flash · high\n" +
		"🔌 MCP: 1 server enabled\n"

	tailIdle = "────────\n" +
		"\n" +
		"────────\n" +
		"~/.picode/work/agente-auto\n" +
		"↑8.0k ↓780 R57k CH99.1% $0.002 0.9%/1.0M (auto)   (zai) glm-5.3-flash · high\n" +
		"🔌 MCP: 1 server enabled\n"

	// The 2026-09-02 false positive: an idle agent whose last pane lines
	// were its own reply about this very feature — the old substring
	// match flagged it busy for as long as the prose stayed on screen.
	tailProseMentioningWorking = "the composer now shows a \"Working in the terminal\" row with\n" +
		"an Open button that docks the TUI; no fake streaming, no Stop. UI\n" +
		"only; the server pieces shipped with ADR-0048 (watch tick 3s).\n" +
		"Deployed and verified live on agente-auto: send-keys → row appears\n" +
		"within one tick with the spinner, Open docks the TUI mid-work, row\n" +
		"clears when the pane's working line ends; overlayAudit ok.\n" +
		"visual-review: PASS (qa4-row-live.png + qa4-open-docked.png,\n" +
		"card 5/5).\n"
)

// Decision table for LooksWorking: the pane's working indicator is the
// spinner line and nothing else. Prose that merely contains the word
// "working" — the 2026-09-02 false positive — must never count.
func TestLooksWorkingDecisionTable(t *testing.T) {
	cases := []struct {
		name string
		tail string
		want bool
	}{
		{"real working tail", tailWorking, true},
		{"spinner frame without leading space", "⠸ Working...\n" + tailIdle, true},
		{"renamed message still matches (frames anchor)", " ⠸ Thinking hard...\n" + tailIdle, true},
		{"idle footer only", tailIdle, false},
		{"idle tail ending with prose about working", tailIdle + tailProseMentioningWorking, false},
		{"prose-only tail (the reported false positive)", tailProseMentioningWorking, false},
		{"braille mid-line is not the indicator", "run: pi --frames ⠹ now\n" + tailIdle, false},
		{"empty capture", "", false},
	}
	for _, tc := range cases {
		if got := LooksWorking(tc.tail); got != tc.want {
			t.Errorf("%s: LooksWorking = %v, want %v", tc.name, got, tc.want)
		}
	}
}
