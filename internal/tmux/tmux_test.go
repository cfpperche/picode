package tmux

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// These are integration tests: they require a tmux binary. They run on
// dev machines and on CI (ubuntu installs tmux — see ci.yml); elsewhere
// they skip, which is recorded as accepted debt in docs/handoff/open/terminal.md.
func requireTmux(t *testing.T) *Manager {
	t.Helper()
	m := New()
	if !m.Available() {
		t.Skip("tmux not installed — integration test skipped (accepted, see docs/handoff/open/terminal.md)")
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

func TestServerVersionAsksTheServerItsOwnVersion(t *testing.T) {
	sock := socketPath(t, "version.sock")
	m := NewWithSocket(sock)
	var got [][]string
	m.exec = func(_ context.Context, _ string, args ...string) ([]byte, error) {
		got = append(got, append([]string(nil), args...))
		return []byte("3.6\n"), nil
	}
	v, err := m.ServerVersion(context.Background())
	if err != nil {
		t.Fatalf("ServerVersion: %v", err)
	}
	if v != "3.6" {
		t.Errorf("ServerVersion() = %q, want 3.6", v)
	}
	want := "-S " + sock + " display-message -p #{version}"
	if len(got) != 1 || strings.Join(got[0], " ") != want {
		t.Errorf("argv = %v, want [%s]", got, want)
	}
}

// The skew case ADR-0164 is about: with a server behind the socket a report
// names the server's version — the installed binary may be newer.
func TestVersionInUseFollowsTheAnsweringServer(t *testing.T) {
	requireTmux(t)
	dir := t.TempDir()
	iso := NewWithSocket(socketPath(t, "tmux.sock"))
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	installed, err := iso.Version()
	if err != nil {
		t.Fatalf("Version: %v", err)
	}
	if got := iso.VersionInUse(ctx); got != installed {
		t.Fatalf("VersionInUse() with no server = %q, want the installed %q", got, installed)
	}

	name := SessionName("ver-" + time.Now().Format("150405-000000000"))
	if err := iso.NewSessionEnvSize(ctx, name, dir, 100, 30, nil, "sleep", "30"); err != nil {
		t.Fatalf("NewSessionEnvSize: %v", err)
	}
	// Its own context: the test's is canceled before cleanups run, and a tmux
	// call made with it fails silently and leaks the session (2026-09-15).
	t.Cleanup(func() {
		killCtx, killCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer killCancel()
		_ = iso.KillSession(killCtx, name)
	})

	server, err := iso.ServerVersion(ctx)
	if err != nil {
		t.Fatalf("ServerVersion: %v", err)
	}
	if server == "" || strings.Contains(server, "tmux") {
		t.Errorf("ServerVersion() = %q, want a bare version like 3.7c", server)
	}
	if got := iso.VersionInUse(ctx); got != server {
		t.Errorf("VersionInUse() = %q, want the server's %q", got, server)
	}
}

func TestNewSessionPreservesGeometryWithoutBrowser(t *testing.T) {
	m := requireTmux(t)
	ctx := context.Background()
	name := SessionName("size-" + time.Now().Format("150405-000000000"))
	if err := m.NewSessionEnvSize(ctx, name, t.TempDir(), 151, 43, nil, "sleep", "30"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = m.KillSession(ctx, name) })
	s, err := m.InputSnapshot(ctx, name)
	if err != nil || s.Width != 151 || len(s.Lines) != 43 {
		t.Fatalf("detached geometry = %dx%d: %v", s.Width, len(s.Lines), err)
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

func TestRespawnPanePreservesSessionAndQuotesArgs(t *testing.T) {
	m := requireTmux(t)
	ctx := context.Background()
	name := SessionName("respawn-" + time.Now().Format("150405-000000000"))
	cwd := t.TempDir()
	if err := m.NewSession(ctx, name, cwd, "sleep", "30"); err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	t.Cleanup(func() { _ = m.KillSession(ctx, name) })
	before, err := m.run(ctx, "display-message", "-p", "-t", name+":0.0", "#{session_id}")
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(cwd, "quoted.txt")
	value := "space and ' quote"
	if err := m.RespawnPaneEnv(ctx, name, cwd, []string{"BURST_VALUE=" + value}, "/bin/sh", "-c", `printf '%s' "$BURST_VALUE" > "$1"; sleep 30`, "holder", out); err != nil {
		t.Fatalf("RespawnPaneEnv: %v", err)
	}
	after, err := m.run(ctx, "display-message", "-p", "-t", name+":0.0", "#{session_id}")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(before) != strings.TrimSpace(after) {
		t.Fatalf("session changed across respawn: %q -> %q", before, after)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		body, err := os.ReadFile(out)
		if err == nil {
			if string(body) != value {
				t.Fatalf("quoted env = %q, want %q", body, value)
			}
			// macOS's /bin/sh is a bash binary and reports itself as such.
			if command, err := m.PaneCommand(ctx, name); err != nil || (command != "sh" && command != "bash") {
				t.Fatalf("PaneCommand = %q, %v", command, err)
			}
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatal("respawned command did not write its output")
}

// tmux reports pane paths as the process sees them, resolved through
// symlinks; macOS keeps TempDir under /var → /private/var, so cwd
// expectations must be resolved the same way.
func resolvedTempDir(t *testing.T) string {
	t.Helper()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return dir
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
			if err == nil && samePath(t, got, want) {
				return
			}
			time.Sleep(50 * time.Millisecond)
		}
		t.Fatalf("PaneCwd = %q (err %v), want %q", got, err, want)
	}

	start := resolvedTempDir(t)
	name := SessionName("cwd-" + time.Now().Format("150405-000000000"))
	if err := m.NewSession(ctx, name, start, "sleep", "30"); err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	t.Cleanup(func() { _ = m.KillSession(ctx, name) })
	// A detached tmux session briefly reports the server process and its cwd
	// before the pane child has exec'd. The API contract is the live process
	// cwd, so assert convergence rather than scheduler timing.
	waitCwd(name, start)

	live := resolvedTempDir(t)
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

// A prepared command nobody submitted must not take the next one onto its
// end: ClearLine empties the prompt first (ADR-0096).
func TestClearLineEmptiesTheTypedLine(t *testing.T) {
	m := requireTmux(t)
	ctx := context.Background()
	name := SessionName("clearline-" + time.Now().Format("150405-000000000"))
	if err := m.NewSession(ctx, name, t.TempDir(), "bash", "--norc", "--noprofile", "-i"); err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	t.Cleanup(func() { _ = m.KillSession(ctx, name) })
	// Wait for a prompt: keystrokes that arrive before readline is up are
	// echoed by the tty, not edited, and the test would be a lie. A shell
	// that never draws one is a failure of the fixture, not a skip.
	promptBy := time.Now().Add(5 * time.Second)
	for {
		tail, _ := m.CaptureTail(ctx, name, 4)
		if strings.Contains(tail, "$") {
			break
		}
		if time.Now().After(promptBy) {
			t.Fatalf("bash never drew a prompt:\n%s", tail)
		}
		time.Sleep(100 * time.Millisecond)
	}

	if err := m.TypeText(ctx, name, "echo first-never-submitted"); err != nil {
		t.Fatalf("TypeText: %v", err)
	}
	if err := m.ClearLine(ctx, name); err != nil {
		t.Fatalf("ClearLine: %v", err)
	}
	if err := m.TypeText(ctx, name, "echo second-only"); err != nil {
		t.Fatalf("TypeText: %v", err)
	}
	if err := m.SendKeys(ctx, name, "Enter"); err != nil {
		t.Fatalf("SendKeys: %v", err)
	}
	deadline := time.Now().Add(6 * time.Second)
	tail := ""
	for time.Now().Before(deadline) {
		tail, _ = m.CaptureTail(ctx, name, 12)
		if strings.Contains(tail, "second-only") {
			break
		}
		time.Sleep(150 * time.Millisecond)
	}
	if !strings.Contains(tail, "second-only") {
		t.Fatalf("the second command never ran:\n%s", tail)
	}
	if strings.Contains(tail, "first-never-submitted") {
		t.Errorf("the abandoned command survived into the run:\n%s", tail)
	}
}

func TestClearLineRefusesAnEmptyName(t *testing.T) {
	if err := requireTmux(t).ClearLine(context.Background(), " "); err == nil {
		t.Error("an empty session name must be refused")
	}
}

// scriptedExec answers tmux invocations from a queue — output plus whether
// the client failed — and succeeds for anything past it. It reproduces the
// first-server race, which a live server cannot produce on demand.
type scriptedExec struct {
	calls []string
	queue []scriptedReply
}

type scriptedReply struct {
	out  string
	fail bool
}

func (s *scriptedExec) run(_ context.Context, _ string, args ...string) ([]byte, error) {
	s.calls = append(s.calls, args[0])
	if len(s.queue) == 0 {
		return nil, nil
	}
	r := s.queue[0]
	s.queue = s.queue[1:]
	if r.fail {
		return []byte(r.out), errors.New("exit status 1")
	}
	return []byte(r.out), nil
}

// The startup commands retry the client that lost the race to start the
// first tmux server; every other failure surfaces at once.
func TestStartupCommandsRetryTheServerStartRace(t *testing.T) {
	lost := scriptedReply{"server exited unexpectedly\n", true}
	absent := scriptedReply{"can't find session: x\n", true}
	present := scriptedReply{"", false}
	ctx := context.Background()
	cases := []struct {
		name      string
		queue     []scriptedReply
		call      func(*Manager) (bool, error)
		wantHas   bool
		wantErr   bool
		wantCalls []string // prefix of the tmux subcommands issued
		exact     bool     // the prefix is the whole call list
	}{
		{"has-session: lost once, then present", []scriptedReply{lost, present},
			func(m *Manager) (bool, error) { return m.HasSession(ctx, "x") },
			true, false, []string{"has-session", "has-session"}, true},
		{"has-session: lost twice, then absent", []scriptedReply{lost, lost, absent},
			func(m *Manager) (bool, error) { return m.HasSession(ctx, "x") },
			false, false, []string{"has-session", "has-session", "has-session"}, true},
		{"has-session: lost three times gives up", []scriptedReply{lost, lost, lost},
			func(m *Manager) (bool, error) { return m.HasSession(ctx, "x") },
			false, true, []string{"has-session", "has-session", "has-session"}, true},
		{"has-session: a real failure is not retried", []scriptedReply{{"unknown command: nope\n", true}},
			func(m *Manager) (bool, error) { return m.HasSession(ctx, "x") },
			false, true, []string{"has-session"}, true},
		{"has-session: a live server holding no sessions is absent", []scriptedReply{{"no current target\n", true}},
			func(m *Manager) (bool, error) { return m.HasSession(ctx, "x") },
			false, false, []string{"has-session"}, true},
		{"new-session: lost once after the lookup", []scriptedReply{absent, lost, present},
			func(m *Manager) (bool, error) { return false, m.NewSession(ctx, "x", "/tmp", "cat") },
			false, false, []string{"has-session", "new-session", "new-session"}, false},
		{"new-session: a duplicate is not retried", []scriptedReply{absent, {"duplicate session: x\n", true}},
			func(m *Manager) (bool, error) { return false, m.NewSession(ctx, "x", "/tmp", "cat") },
			false, true, []string{"has-session", "new-session"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := &scriptedExec{queue: append([]scriptedReply(nil), tc.queue...)}
			m := &Manager{exec: s.run}
			has, err := tc.call(m)
			if has != tc.wantHas || (err != nil) != tc.wantErr {
				t.Fatalf("got has=%v err=%v, want has=%v err=%v", has, err, tc.wantHas, tc.wantErr)
			}
			got := strings.Join(s.calls, " ")
			want := strings.Join(tc.wantCalls, " ")
			if (tc.exact && got != want) || (!tc.exact && !strings.HasPrefix(got, want)) {
				t.Fatalf("tmux calls = %q, want %q", got, want)
			}
		})
	}
}

// TestPaneCwdIsRightFromTheFirstInstant is the 2026-09-15 race as a test:
// #{pane_current_path} for a pane tmux has not polled yet is the *server's*
// directory (this process's cwd), so a read taken in the same instant as
// new-session answered with the wrong folder — 33 of 40 creations in a loop.
// The Manager remembers what it created, so the first read is right.
func TestPaneCwdIsRightFromTheFirstInstant(t *testing.T) {
	m := requireTmux(t)
	ctx := context.Background()
	root := t.TempDir()
	here, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Clean(root) == filepath.Clean(here) {
		t.Fatal("fixture must differ from the process cwd to be meaningful")
	}
	for i := 0; i < 12; i++ {
		dir := filepath.Join(root, "shell-"+strconv.Itoa(i))
		if err := os.Mkdir(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		name := SessionName("born-" + strconv.Itoa(i))
		if err := m.NewSession(ctx, name, dir, "sleep", "30"); err != nil {
			t.Fatalf("NewSession: %v", err)
		}
		got, err := m.PaneCwd(ctx, name)
		if err != nil {
			t.Fatalf("iteration %d: PaneCwd: %v", i, err)
		}
		if !samePath(t, got, dir) {
			t.Fatalf("iteration %d: PaneCwd = %q, want %q (the folder it was created in)", i, got, dir)
		}
		if err := m.KillSession(ctx, name); err != nil {
			t.Fatalf("iteration %d: KillSession: %v", i, err)
		}
	}
}

// And the creation record never outlives its usefulness: a shell that has
// moved reports its own folder immediately.
func TestPaneCwdFollowsAShellThatMovedRightAfterCreation(t *testing.T) {
	m := requireTmux(t)
	ctx := context.Background()
	root := t.TempDir()
	start, moved := filepath.Join(root, "start"), filepath.Join(root, "moved")
	for _, dir := range []string{start, moved} {
		if err := os.Mkdir(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	name := SessionName("born-cd")
	if err := m.NewSession(ctx, name, start, "/bin/sh"); err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	t.Cleanup(func() { _ = m.KillSession(context.Background(), name) })
	if got, err := m.PaneCwd(ctx, name); err != nil || !samePath(t, got, start) {
		t.Fatalf("first read = %q (err %v), want %q", got, err, start)
	}
	if err := m.TypeText(ctx, name, "cd "+moved); err != nil {
		t.Fatalf("TypeText: %v", err)
	}
	if err := m.SendKeys(ctx, name, "Enter"); err != nil {
		t.Fatalf("SendKeys: %v", err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for {
		got, err := m.PaneCwd(ctx, name)
		if err == nil && samePath(t, got, moved) {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("after cd, PaneCwd = %q (err %v), want %q", got, err, moved)
		}
		time.Sleep(50 * time.Millisecond)
	}
}
