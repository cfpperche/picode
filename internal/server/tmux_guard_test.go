package server

// The tmux guard's decision table (ADR-0138), tested on two layers:
//
//  1. Policy tests against a fake real tmux that logs its argv and echoes
//     canned marker answers — hermetic, no tmux binary needed.
//  2. One integration test against a real tmux server isolated by
//     TMUX_TMPDIR (a fresh default socket namespace, so the wrapper's
//     un-prefixed probes and the fixture agree on the server). Skips where
//     tmux is absent — accepted debt, same as the internal/tmux fixtures.
//
// Every row of the ADR-0138 table has a case here; a row without a test
// would have to be named as debt in docs/handoff/open/<topic>.md.

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type guardFixture struct {
	t       *testing.T
	dataDir string
	realLog string
	real    string // path of the fake real tmux
}

func newGuardFixture(t *testing.T) *guardFixture {
	t.Helper()
	root := t.TempDir()
	dataDir := filepath.Join(root, "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatal(err)
	}
	f := &guardFixture{t: t, dataDir: dataDir, realLog: filepath.Join(root, "real.log")}
	// The fake is named `tmux` so the wrapper's find-real walk (skip the bin
	// dir, first other PATH dir wins) picks it over the system binary. A
	// fake `date` keeps refusal-log lines deterministic; nothing else on the
	// wrapper's path resolves to the system — a leak here must fail the
	// test, not touch the live server.
	f.real = filepath.Join(root, "tmux")
	body := "#!/bin/sh\n" +
		"printf '%s\\n' \"$*\" >> \"" + f.realLog + "\"\n" +
		"if [ \"$1\" = marker ]; then printf '%s\\n' \"$2\"; exit 0; fi\n" +
		"if [ \"$1\" = list-sessions ]; then printf 'owned\\nforeign\\nuser\\n'; exit 0; fi\n" +
		"exit 0\n"
	if err := writeExecutable(f.real, body); err != nil {
		t.Fatal(err)
	}
	if err := writeExecutable(filepath.Join(filepath.Dir(f.real), "date"), "#!/bin/sh\nprintf '1970-01-01T00:00:00\\n"+"'\n"); err != nil {
		t.Fatal(err)
	}
	if err := installTmuxGuard(dataDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = uninstallTmuxGuard(dataDir) })
	return f
}

// run invokes the wrapper with the fake real tmux first on PATH. marker is
// what `show-environment` answers for the target; empty means "unknown
// variable" (rc 1), the real command's answer for an unmarked session.
// Every exec carries a deadline: a wrapper that loops (the 2026-09-15
// self-exec under a minimal PATH) must fail the test, never hang it.
func (f *guardFixture) run(termID string, args ...string) (string, error) {
	f.t.Helper()
	ctx, cancel := context.WithTimeout(f.t.Context(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, wrapperPath(f.dataDir, "tmux"), args...)
	// PATH: guard bin first, then ONLY the fake dir — no system fallback.
	env := []string{
		"PATH=" + interceptBinDir(f.dataDir) + string(os.PathListSeparator) + filepath.Dir(f.real),
	}
	if termID != "" {
		env = append(env, "PICODE_TERM_ID="+termID)
	}
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func (f *guardFixture) realCalls(t *testing.T) []string {
	t.Helper()
	raw, err := os.ReadFile(f.realLog)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		t.Fatal(err)
	}
	var out []string
	for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

func (f *guardFixture) wantRefusal(t *testing.T, args ...string) {
	t.Helper()
	out, err := f.run("T1", args...)
	if err == nil {
		t.Fatalf("tmux %v: expected refusal, exited 0 (out %q)", args, out)
	}
	if !strings.Contains(out, "picode tmux guard: refused") {
		t.Fatalf("tmux %v: refusal copy missing from %q", args, out)
	}
	for _, call := range f.realCalls(t) {
		if strings.Join(args, " ") == call {
			t.Fatalf("tmux %v: refused command reached the real tmux (%q)", args, call)
		}
	}
	lines := f.guardLogLines(t)
	if len(lines) == 0 {
		t.Fatalf("tmux %v: refusal was not logged", args)
	}
	last := lines[len(lines)-1]
	if !strings.Contains(last, "term=T1") || !strings.Contains(last, strings.Join(args, " ")) {
		t.Fatalf("refusal log line %q lacks the term id and argv", last)
	}
}

func (f *guardFixture) guardLogLines(t *testing.T) []string {
	t.Helper()
	raw, err := os.ReadFile(tmuxGuardLog(f.dataDir))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		t.Fatal(err)
	}
	var out []string
	for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

func TestTmuxGuardRefusesWholeServerKills(t *testing.T) {
	f := newGuardFixture(t)
	for _, args := range [][]string{
		{"kill-server"},
		{"kill-window"},
		{"kill-pane"},
		{"kill-pane", "-t", "picode-sh-owned"},
		{"-L", "other", "kill-server"}, // server options do not bypass the gate
	} {
		f.wantRefusal(t, args...)
	}
}

func TestTmuxGuardRefusesPatternAndMassSessionKills(t *testing.T) {
	f := newGuardFixture(t)
	for _, args := range [][]string{
		{"kill-session", "-t", "picode-*"},
		{"kill-session", "-t", "picode-sh-[a-z]*"},
		{"kill-session", "-a"},
		{"kill-session", "-a", "-t", "owned"},
		{"kill-session", "-at", "owned"}, // combined cluster: all-but + target
	} {
		f.wantRefusal(t, args...)
	}
}

func TestTmuxGuardAllowsExactOwnedKill(t *testing.T) {
	f := newGuardFixture(t)
	// The allow path is exercised in TestTmuxGuardKillSessionOwnership
	// (owned-session case) and end to end against a real server below; this
	// row only pins that the fake echoes the marker it is asked for.
	out, err := f.run("T1", "marker", "PICODE_TERM_ID=T1")
	if err != nil {
		t.Fatalf("marker echo: %v", err)
	}
	if !strings.Contains(out, "PICODE_TERM_ID=T1") {
		t.Fatalf("fake real tmux marker echo broken: %q", out)
	}
}

func TestTmuxGuardKillSessionOwnership(t *testing.T) {
	// The marker answers come from the fake real tmux each subtest writes;
	// the wrapper shells out to the real binary for show-environment, so
	// the ownership rows are driven through PATH-shadowed truth, not mocks
	// inside the wrapper.
	t.Run("owned session is allowed through untouched", func(t *testing.T) {
		f2 := newGuardFixture(t)
		// replace the fake with one that answers the probe for "owned"
		body := "#!/bin/sh\n" +
			"printf '%s\\n' \"$*\" >> \"" + f2.realLog + "\"\n" +
			"if [ \"$1\" = show-environment ] && [ \"$3\" = owned ]; then printf 'PICODE_TERM_ID=T1\\n'; exit 0; fi\n" +
			"if [ \"$1\" = show-environment ]; then printf 'unknown variable\\n' >&2; exit 1; fi\n" +
			"exit 0\n"
		if err := writeExecutable(f2.real, body); err != nil {
			t.Fatal(err)
		}
		if _, err := f2.run("T1", "kill-session", "-t", "owned"); err != nil {
			t.Fatalf("owned exact kill refused: %v", err)
		}
		calls := f2.realCalls(t)
		if len(calls) == 0 {
			t.Fatal("real tmux was never invoked")
		}
		last := calls[len(calls)-1]
		if last != "kill-session -t owned" {
			t.Fatalf("real tmux saw %q, want the original kill-session argv", last)
		}
	})
	t.Run("foreign marker is refused", func(t *testing.T) {
		f3 := newGuardFixture(t)
		body := "#!/bin/sh\n" +
			"if [ \"$1\" = show-environment ] && [ \"$3\" = foreign ]; then printf 'PICODE_TERM_ID=T2\\n'; exit 0; fi\n" +
			"exit 1\n"
		if err := writeExecutable(f3.real, body); err != nil {
			t.Fatal(err)
		}
		f3.wantRefusal(t, "kill-session", "-t", "foreign")
	})
	t.Run("unmarked (user) session is refused", func(t *testing.T) {
		f4 := newGuardFixture(t)
		body := "#!/bin/sh\nif [ \"$1\" = show-environment ]; then printf 'unknown variable\\n' >&2; exit 1; fi\nexit 1\n"
		if err := writeExecutable(f4.real, body); err != nil {
			t.Fatal(err)
		}
		f4.wantRefusal(t, "kill-session", "-t", "user-own")
	})
}

func TestTmuxGuardSendKeysPayload(t *testing.T) {
	f := newGuardFixture(t)
	f.wantRefusal(t, "send-keys", "-t", "picode-sh-owned", "tmux kill-server", "Enter")
	f.wantRefusal(t, "send-keys", "-t", "picode-sh-owned", "pkill -f picode-", "Enter")
	f.wantRefusal(t, "send-keys", "-t", "picode-sh-owned", "killall -q picode", "Enter")
	if _, err := f.run("T1", "send-keys", "-t", "picode-sh-owned", "echo hello", "Enter"); err != nil {
		t.Fatalf("ordinary send-keys refused: %v", err)
	}
}

func TestTmuxGuardNewSessionStampsOwner(t *testing.T) {
	f := newGuardFixture(t)
	if _, err := f.run("T1", "new-session", "-d", "-s", "picode-sh-scratch"); err != nil {
		t.Fatalf("new-session refused: %v", err)
	}
	calls := f.realCalls(t)
	if len(calls) == 0 {
		t.Fatal("real tmux was never invoked")
	}
	last := calls[len(calls)-1]
	if !strings.HasPrefix(last, "new-session -e PICODE_TERM_ID=T1 ") {
		t.Fatalf("stamped argv = %q, want new-session -e PICODE_TERM_ID=T1 …", last)
	}
	if !strings.Contains(last, "-d -s picode-sh-scratch") {
		t.Fatalf("stamped argv = %q, the original arguments must survive", last)
	}
	// An explicit marker is never overwritten.
	if _, err := f.run("T1", "new-session", "-d", "-s", "x", "-e", "PICODE_TERM_ID=T9"); err != nil {
		t.Fatalf("new-session with explicit marker refused: %v", err)
	}
	calls = f.realCalls(t)
	if len(calls) == 0 {
		t.Fatal("real tmux was never invoked")
	}
	last = calls[len(calls)-1]
	if strings.Contains(last, "-e PICODE_TERM_ID=T1") {
		t.Fatalf("explicit marker was overwritten: %q", last)
	}
}

func TestTmuxGuardMineListsOwnSessions(t *testing.T) {
	f := newGuardFixture(t)
	// mine pipes list-sessions output through show-environment probes; the
	// fake answers "owned" with our marker and refuses the rest.
	body := "#!/bin/sh\n" +
		"if [ \"$1\" = list-sessions ]; then printf 'owned\\nforeign\\nuser\\n'; exit 0; fi\n" +
		"if [ \"$1\" = show-environment ] && [ \"$3\" = owned ]; then printf 'PICODE_TERM_ID=T1\\n'; exit 0; fi\n" +
		"exit 1\n"
	if err := writeExecutable(f.real, body); err != nil {
		t.Fatal(err)
	}
	out, err := f.run("T1", "mine")
	if err != nil {
		t.Fatalf("mine refused: %v", err)
	}
	if strings.TrimSpace(out) != "owned" {
		t.Fatalf("mine = %q, want only the owned session", out)
	}
}

func TestTmuxGuardPassesUnrelatedCommandsThrough(t *testing.T) {
	f := newGuardFixture(t)
	if _, err := f.run("T1", "ls"); err != nil {
		t.Fatalf("ls refused: %v", err)
	}
	if _, err := f.run("T1", "capture-pane", "-t", "picode-sh-owned", "-p"); err != nil {
		t.Fatalf("capture-pane refused: %v", err)
	}
	calls := f.realCalls(t)
	if len(calls) == 0 {
		t.Fatal("real tmux was never invoked")
	}
	if calls[len(calls)-1] != "capture-pane -t picode-sh-owned -p" {
		t.Fatalf("passthrough argv = %q", calls[len(calls)-1])
	}
}

func TestTmuxGuardOutsideManagedTerminalPassesThrough(t *testing.T) {
	f := newGuardFixture(t)
	// No PICODE_TERM_ID: the wrapper must not judge — kill-server reaches the
	// real binary (which here is the fake; the integration test proves the
	// same shape against a throwaway server).
	out, err := f.run("", "kill-server")
	if err != nil {
		t.Fatalf("unguarded kill-server failed: %v (out %q)", err, out)
	}
	if calls := f.realCalls(t); len(calls) == 0 || calls[len(calls)-1] != "kill-server" {
		t.Fatalf("real tmux did not receive the passthrough argv: %v", calls)
	}
	if lines := f.guardLogLines(t); len(lines) != 0 {
		t.Fatalf("passthrough was logged as a refusal: %v", lines)
	}
}

func TestTmuxGuardNoTargetInsideTmuxResolvesCurrentSession(t *testing.T) {
	f := newGuardFixture(t)
	body := "#!/bin/sh\n" +
		"if [ \"$1\" = display-message ]; then printf 'picode-sh-current\\n'; exit 0; fi\n" +
		"if [ \"$1\" = show-environment ] && [ \"$3\" = picode-sh-current ]; then printf 'PICODE_TERM_ID=T1\\n'; exit 0; fi\n" +
		"printf '%s\\n' \"$*\" >> \"" + f.realLog + "\"\n" +
		"exit 0\n"
	if err := writeExecutable(f.real, body); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(wrapperPath(f.dataDir, "tmux"), "kill-session")
	cmd.Env = []string{
		"PATH=" + interceptBinDir(f.dataDir) + string(os.PathListSeparator) + filepath.Dir(f.real),
		"PICODE_TERM_ID=T1",
		"TMUX=/tmp/tmux-test/default,1,0", // inside tmux, so no -t resolves to the current session
	}
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("kill-session without -t refused inside own session: %v (%s)", err, out)
	}
}

func TestTmuxGuardDefaultsOnAndSurvivesCleanup(t *testing.T) {
	root := t.TempDir()
	dataDir := filepath.Join(root, "data")
	// Default on: enabled.json never mentions the guard, yet the wrapper
	// installs and interceptOn answers true.
	if interceptOn(dataDir, TmuxGuardID) != true {
		t.Fatal("tmux guard must default on")
	}
	ensureTmuxGuard(dataDir)
	if _, err := os.Stat(wrapperPath(dataDir, "tmux")); err != nil {
		t.Fatalf("ensureTmuxGuard did not install the wrapper: %v", err)
	}
	// A missing wrapper is reinstalled at session creation (upgrade clean-up).
	if err := os.Remove(wrapperPath(dataDir, "tmux")); err != nil {
		t.Fatal(err)
	}
	ensureTmuxGuard(dataDir)
	if _, err := os.Stat(wrapperPath(dataDir, "tmux")); err != nil {
		t.Fatalf("guard was not restored: %v", err)
	}
	// Explicit opt-out wins: no wrapper, no PATH entry, ensure stays idle.
	if err := uninstallTmuxGuard(dataDir); err != nil {
		t.Fatal(err)
	}
	if interceptOn(dataDir, TmuxGuardID) {
		t.Fatal("explicit opt-out lost")
	}
	ensureTmuxGuard(dataDir)
	if _, err := os.Stat(wrapperPath(dataDir, "tmux")); !os.IsNotExist(err) {
		t.Fatal("guard came back after an explicit opt-out")
	}
	m := loadInterceptEnabled(dataDir)
	if m[TmuxGuardID] {
		t.Fatal("opt-out must persist in enabled.json")
	}
}

func TestTmuxGuardSyntax(t *testing.T) {
	f := newGuardFixture(t)
	if out, err := exec.Command("sh", "-n", wrapperPath(f.dataDir, "tmux")).CombinedOutput(); err != nil {
		t.Fatalf("wrapper syntax: %v: %s", err, out)
	}
}

// TestTmuxGuardMinimalPathDoesNotSelfExec pins the 2026-09-15 incident: the
// guard's find-real must not depend on dirname(1). Under a minimal PATH the
// shared walk computed a wrong `here`, found the guard itself in the bin dir
// and exec'd in an endless loop (the test hung and orphaned a process). The
// fixture PATH is exactly that minimal shape, and run()'s deadline turns a
// regression into a failure instead of a hang.
func TestTmuxGuardMinimalPathDoesNotSelfExec(t *testing.T) {
	f := newGuardFixture(t)
	if _, err := os.Stat(f.real); err != nil {
		t.Fatalf("fake real tmux missing: %v", err)
	}
	out, err := f.run("T1", "marker", "PICODE_TERM_ID=T1")
	if err != nil {
		t.Fatalf("minimal PATH run failed (self-exec loop?): %v (%s)", err, out)
	}
	calls := f.realCalls(t)
	if len(calls) == 0 || calls[len(calls)-1] != "marker PICODE_TERM_ID=T1" {
		t.Fatalf("fake real tmux did not receive the passthrough: %v", calls)
	}
}

// --- integration: real tmux, isolated default socket ---

func requireRealTmux(t *testing.T) string {
	t.Helper()
	// LookPath from inside a guarded PiCode pane finds the intercept wrapper
	// first (~/.picode/bin/tmux leads the injected PATH). Building the
	// fixture PATH around a wrapper makes two guard generations resolve each
	// other as "$real" and exec in a loop — TestTmuxGuardAgainstRealServer
	// hung 8m56s, twice (2026-09-17). A real tmux is one that is not a
	// PiCode wrapper: check the bytes, not the directory.
	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		if dir == "" {
			continue
		}
		p := filepath.Join(dir, "tmux")
		info, err := os.Stat(p)
		if err != nil || info.IsDir() || info.Mode()&0o111 == 0 {
			continue
		}
		f, err := os.Open(p)
		if err != nil {
			continue
		}
		head := make([]byte, 512)
		n, _ := f.Read(head)
		_ = f.Close()
		if bytes.Contains(head[:n], []byte("PiCode intercept")) {
			continue
		}
		return p
	}
	t.Skip("no unwrapped tmux on PATH — integration test skipped (accepted debt, docs/handoff/open/terminal.md)")
	return ""
}

// scrubbedTmuxEnv is os.Environ with every variable that could retarget a
// tmux client stripped, plus a fresh TMUX_TMPDIR. $TMUX beats TMUX_TMPDIR:
// a test binary inheriting TMUX from the developer's pane talks to THAT
// server no matter what TMUX_TMPDIR says (measured 2026-09-15 — a fixture
// "isolated" this way ran kill-server on the live server twice). The
// duplicate-key case matters too: a pre-existing TMUX_TMPDIR left earlier
// in the slice would win the client's first-match getenv.
func scrubbedTmuxEnv(tmpDir string) []string {
	out := make([]string, 0, len(os.Environ())+1)
	for _, kv := range os.Environ() {
		switch {
		case strings.HasPrefix(kv, "TMUX="), strings.HasPrefix(kv, "TMUX_PANE="), strings.HasPrefix(kv, "TMUX_TMPDIR="):
			continue
		}
		out = append(out, kv)
	}
	return append(out, "TMUX_TMPDIR="+tmpDir)
}

func TestTmuxGuardAgainstRealServer(t *testing.T) {
	// Incident 2026-09-15: an early draft of this fixture trusted TMUX_TMPDIR
	// without proving it and ran kill-server in cleanup — when the fake
	// binary was misnamed, its PATH leaked to the system tmux and left stray
	// sessions on the live server. Rules since, enforced here:
	//
	//  1. isolation is ASSERTED (#{socket_path} must land under the temp
	//     dir) — the test skips rather than run against the live server;
	//  2. cleanup kills each fixture session BY EXACT NAME — no test in this
	//     repository may run kill-server, not even against a fixture;
	//  3. the wrapper under test is found before the system one, and a
	//     PATH leak would fail the fixture setup below, loudly.
	real := requireRealTmux(t)
	root := t.TempDir()
	// TMUX_TMPDIR: a private default-socket namespace, kept short enough
	// to bind (see shortSocketDir).
	tmp := filepath.Join(shortSocketDir(t), "tmp")
	dataDir := filepath.Join(root, "data")
	for _, d := range []string{tmp, dataDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := installTmuxGuard(dataDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = uninstallTmuxGuard(dataDir) })

	base := scrubbedTmuxEnv(tmp)
	tmux := func(args ...string) *exec.Cmd {
		c := exec.Command(real, args...)
		c.Env = base
		return c
	}

	// Probe BEFORE anything else is created. The probe session is the first
	// client of the fixture server; its socket_path is the proof that
	// isolation holds. If it does not land under the temp dir the test
	// skips — it must never operate on the live server it aims to observe.
	if out, err := tmux("new-session", "-d", "-s", "picode-guard-probe", "-x", "120", "-y", "30").CombinedOutput(); err != nil {
		t.Fatalf("fixture probe session: %v (%s)", err, out)
	}
	t.Cleanup(func() { _ = tmux("kill-session", "-t", "picode-guard-probe").Run() })
	out, err := tmux("display", "-p", "#{socket_path}").CombinedOutput()
	if err != nil {
		t.Skipf("cannot read socket_path on the fixture server: %v (%s)", err, out)
	}
	if !strings.HasPrefix(strings.TrimSpace(string(out)), tmp) {
		t.Skipf("fixture socket %q escaped the temp namespace — refusing to run against the live server", strings.TrimSpace(string(out)))
	}

	// Fixture sessions, exact names — cleanup is per name, never kill-server.
	fixtures := []struct{ name, marker string }{
		{"picode-guard-owned", "T1"},
		{"picode-guard-foreign", "T2"},
		{"user-guard-own", ""},
	}
	for _, fx := range fixtures {
		args := []string{"new-session", "-d", "-s", fx.name, "-x", "120", "-y", "30"}
		if fx.marker != "" {
			args = append(args, "-e", "PICODE_TERM_ID="+fx.marker)
		}
		if out, err := tmux(args...).CombinedOutput(); err != nil {
			t.Fatalf("fixture %s: %v (%s)", fx.name, err, out)
		}
		t.Cleanup(func() { _ = tmux("kill-session", "-t", fx.name).Run() })
	}

	wrapper := wrapperPath(dataDir, "tmux")
	// The wrapper's PATH: the guard bin dir first, then ONLY the real tmux's
	// directory — never a broad /usr/bin:/bin fallback that could silently
	// resolve some other system binary.
	run := func(termID string, args ...string) (string, error) {
		cmd := exec.Command(wrapper, args...)
		cmd.Env = append(base, "PATH="+interceptBinDir(dataDir)+string(os.PathListSeparator)+filepath.Dir(real))
		if termID != "" {
			cmd.Env = append(cmd.Env, "PICODE_TERM_ID="+termID)
		}
		out, err := cmd.CombinedOutput()
		return string(out), err
	}

	// kill-server refused, server stays up.
	if out, err := run("T1", "kill-server"); err == nil {
		t.Fatalf("kill-server was not refused (out %q)", out)
	}
	if err := tmux("has-session", "-t", "picode-guard-owned").Run(); err != nil {
		t.Fatal("server died behind the guard")
	}

	// Owned exact kill works end to end.
	if out, err := run("T1", "kill-session", "-t", "picode-guard-owned"); err != nil {
		t.Fatalf("owned kill refused: %v (%s)", err, out)
	}
	if err := tmux("has-session", "-t", "picode-guard-owned").Run(); err == nil {
		t.Fatal("owned session survived its owner's kill")
	}

	// Foreign and user sessions are refused.
	if out, err := run("T1", "kill-session", "-t", "picode-guard-foreign"); err == nil {
		t.Fatalf("foreign kill was allowed (%s)", out)
	}
	if out, err := run("T1", "kill-session", "-t", "user-guard-own"); err == nil {
		t.Fatalf("user session kill was allowed (%s)", out)
	}
	if err := tmux("has-session", "-t", "picode-guard-foreign").Run(); err != nil {
		t.Fatal("foreign session died")
	}

	// new-session through the wrapper is stamped with the creator's marker.
	if out, err := run("T1", "new-session", "-d", "-s", "picode-guard-scratch", "-x", "120", "-y", "30"); err != nil {
		t.Fatalf("stamped new-session failed: %v (%s)", err, out)
	}
	out, err = tmux("show-environment", "-t", "picode-guard-scratch", "PICODE_TERM_ID").CombinedOutput()
	if err != nil || !strings.Contains(string(out), "PICODE_TERM_ID=T1") {
		t.Fatalf("scratch session not stamped: %v (%s)", err, out)
	}
	// …and its owner can clean it up by exact name.
	if out, err := run("T1", "kill-session", "-t", "picode-guard-scratch"); err != nil {
		t.Fatalf("owner could not clean up its scratch session: %v (%s)", err, out)
	}

	// ls passes through.
	if out, err := run("T1", "ls"); err != nil {
		t.Fatalf("ls refused: %v (%s)", err, out)
	}
}
