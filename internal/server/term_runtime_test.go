package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/feed"
	"github.com/cfpperche/picode/internal/store"
)

func TestTerminalCLIFromCommand(t *testing.T) {
	cases := []struct {
		command string
		want    string
	}{
		{"pi", "pi"},
		{"/usr/local/bin/claude", "claude-code"},
		{"codex", "codex"},
		{"grok", "grok"},
		{"bash", ""},
		{"", ""},
	}
	for _, tc := range cases {
		t.Run(tc.command, func(t *testing.T) {
			if got := terminalCLIFromCommand(tc.command); got != tc.want {
				t.Fatalf("terminalCLIFromCommand(%q) = %q, want %q", tc.command, got, tc.want)
			}
		})
	}
}

func TestTermRuntimeRunIDDecisionTable(t *testing.T) {
	now := time.Date(2026, 9, 4, 11, 0, 0, 0, time.UTC)
	runtimes := NewTermRuntimes()
	first := TermRuntime{CLI: "pi", Source: "wrapper", RunID: "run-1", PID: 10, ProcStart: "100", StartedAt: now}
	if got, changed := runtimes.Start("t1", first); !changed || got.RunID != "run-1" {
		t.Fatalf("first start = %+v changed=%v", got, changed)
	}
	if _, changed := runtimes.Start("t1", TermRuntime{CLI: "pi", Source: "wrapper", RunID: "run-1", PID: 10, ProcStart: "100", StartedAt: now.Add(time.Minute)}); changed {
		t.Fatal("same run identity must not publish a duplicate start")
	}
	if _, removed := runtimes.End("t1", "old-run"); removed {
		t.Fatal("stale end must not remove the current runtime")
	}
	second := TermRuntime{CLI: "codex", Source: "wrapper", RunID: "run-2", PID: 11, ProcStart: "101", StartedAt: now.Add(2 * time.Minute)}
	if _, changed := runtimes.Start("t1", second); !changed {
		t.Fatal("a new run must replace the old runtime")
	}
	if got, ok := runtimes.Get("t1"); !ok || got.RunID != "run-2" || got.CLI != "codex" {
		t.Fatalf("current runtime = %+v ok=%v", got, ok)
	}
	if _, removed := runtimes.End("t1", "run-1"); removed {
		t.Fatal("an old run cannot end a newer runtime")
	}
	if _, removed := runtimes.End("t1", "run-2"); !removed {
		t.Fatal("current run must be removable")
	}
}

func TestRuntimePresenceRejectsStaleActivityProjection(t *testing.T) {
	states := NewTermStates()
	runtimes := NewTermRuntimes()
	deps := Deps{TermStates: states, TermRuntimes: runtimes}
	runtimes.Start("t1", TermRuntime{CLI: "codex", Source: "wrapper", RunID: "new", PID: 10})
	states.SetForRun("t1", TermWorking, "pi", "old", time.Now())
	view := map[string]any{}
	applyTermRuntime(deps, view, "t1")
	applyTermState(deps, view, "t1")
	if view["cli"] != "codex" {
		t.Fatalf("runtime identity missing: %+v", view)
	}
	if _, ok := view["state"]; ok {
		t.Fatalf("stale state projected: %+v", view)
	}
}

func TestRuntimePresenceDoesNotInventFallbackActivity(t *testing.T) {
	states := NewTermStates()
	runtimes := NewTermRuntimes()
	deps := Deps{TermStates: states, TermRuntimes: runtimes}
	states.SetForRun("t1", TermWorking, "pi", "", time.Now())

	registerTermRuntime(deps, "t1", TermRuntime{CLI: "pi", Source: "tmux-fallback", RunID: "tmux-t1-9", PID: 9})
	if got, ok := states.Get("t1"); !ok || got.State != TermWorking {
		t.Fatalf("fallback removed existing activity: %+v ok=%v", got, ok)
	}

	registerTermRuntime(deps, "t1", TermRuntime{CLI: "pi", Source: "wrapper", RunID: "run-2", PID: 10})
	if _, ok := states.Get("t1"); ok {
		t.Fatal("new wrapper run retained unscoped activity from the previous process")
	}
}

func TestTerminalRuntimeHTTPRoundTrip(t *testing.T) {
	root := t.TempDir()
	st, err := store.Open(filepath.Join(root, "picode.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	term, err := st.CreateTerminal("Shell", root)
	if err != nil {
		t.Fatal(err)
	}
	f := &feed.Feed{}
	deps := Deps{Store: st, Feed: f, TermStates: NewTermStates(), TermRuntimes: NewTermRuntimes()}
	ts := httptest.NewServer(New("127.0.0.1:0", deps).Handler)
	t.Cleanup(ts.Close)

	blankStart := postJSON(t, ts, "/api/terminals/"+term.ID+"/runtime", map[string]any{"action": "start", "cli": "pi", "runId": "   ", "pid": os.Getpid()})
	if blankStart.StatusCode != http.StatusBadRequest {
		t.Fatalf("blank start run id = %d, want 400", blankStart.StatusCode)
	}
	blankEnd := postJSON(t, ts, "/api/terminals/"+term.ID+"/runtime", map[string]any{"action": "end", "runId": "   "})
	if blankEnd.StatusCode != http.StatusBadRequest {
		t.Fatalf("blank end run id = %d, want 400", blankEnd.StatusCode)
	}

	start := postJSON(t, ts, "/api/terminals/"+term.ID+"/runtime", map[string]any{
		"action": "start", "cli": "claude", "runId": "run-current", "pid": os.Getpid(),
	})
	if start.StatusCode != http.StatusOK {
		t.Fatalf("start = %d", start.StatusCode)
	}
	var started map[string]any
	if err := json.NewDecoder(start.Body).Decode(&started); err != nil {
		t.Fatal(err)
	}
	if started["cli"] != "claude-code" || started["runId"] != "run-current" {
		t.Fatalf("start body = %+v", started)
	}

	listed := do(t, ts.Client(), mustGet(t, ts.URL+"/api/terminals"))
	var page struct {
		Terminals []map[string]any `json:"terminals"`
	}
	if err := json.NewDecoder(listed.Body).Decode(&page); err != nil {
		t.Fatal(err)
	}
	if len(page.Terminals) != 1 || page.Terminals[0]["cli"] != "claude-code" {
		t.Fatalf("list = %+v", page.Terminals)
	}
	if _, ok := page.Terminals[0]["tui"]; !ok {
		t.Fatalf("list lacks authoritative tui presence: %+v", page.Terminals[0])
	}

	stale := postJSON(t, ts, "/api/terminals/"+term.ID+"/runtime", map[string]any{"action": "end", "runId": "old"})
	if stale.StatusCode != http.StatusOK {
		t.Fatalf("stale end = %d", stale.StatusCode)
	}
	still := do(t, ts.Client(), mustGet(t, ts.URL+"/api/terminals"))
	page.Terminals = nil
	_ = json.NewDecoder(still.Body).Decode(&page)
	if _, ok := page.Terminals[0]["tui"]; !ok {
		t.Fatal("stale end removed current runtime")
	}

	ended := postJSON(t, ts, "/api/terminals/"+term.ID+"/runtime", map[string]any{"action": "end", "runId": "run-current"})
	if ended.StatusCode != http.StatusOK {
		t.Fatalf("end = %d", ended.StatusCode)
	}
	final := do(t, ts.Client(), mustGet(t, ts.URL+"/api/terminals"))
	page.Terminals = nil
	_ = json.NewDecoder(final.Body).Decode(&page)
	if _, ok := page.Terminals[0]["tui"]; ok || page.Terminals[0]["cli"] != nil {
		t.Fatalf("runtime still present after end: %+v", page.Terminals[0])
	}
}

// spawnWrappedCLI builds the wrapper shape ADR-0062 produces in a pane: a
// shell wrapper named name that stays alive as the real CLI's parent. It
// returns the wrapper's process (killed with its group on cleanup).
func spawnWrappedCLI(t *testing.T, name, innerScript string) *exec.Cmd {
	t.Helper()
	dir := t.TempDir()
	wrapper := filepath.Join(dir, name)
	body := "#!/bin/sh\n" + innerScript + "\n"
	if err := os.WriteFile(wrapper, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(wrapper)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		_ = cmd.Wait()
	})
	return cmd
}

func waitUntil(want func() bool) {
	for i := 0; i < 100; i++ {
		if want() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestIdentifyPaneCLIProcsFindsWrappedCLI(t *testing.T) {
	if _, err := os.Stat("/proc/self/stat"); err != nil {
		t.Skip("no /proc on this platform")
	}
	inner := filepath.Join(t.TempDir(), "claude")
	if err := os.WriteFile(inner, []byte("#!/bin/sh\nsleep 5\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	// The wrapper shell stays the parent while the inner "claude" runs.
	cmd := spawnWrappedCLI(t, "claude", exec.Command(inner).Path+" &\nwait")
	wrapperPID := cmd.Process.Pid
	waitUntil(func() bool {
		snap := readProcSnapshot()
		cli, pid := identifyPaneCLIProcs(wrapperPID, snap)
		return cli == "claude-code" && pid > 0
	})
	snap := readProcSnapshot()
	cli, pid := identifyPaneCLIProcs(wrapperPID, snap)
	if cli != "claude-code" {
		t.Fatalf("cli=%q want claude-code", cli)
	}
	if pid <= 0 {
		t.Fatalf("pid=%d want a live process", pid)
	}
	// The wrapper shell matches first (sh + script basename) and only lives
	// while the wrapped CLI runs, so its identity is honest presence.
	if pid != wrapperPID {
		t.Logf("matched deeper process %d instead of wrapper %d — still valid", pid, wrapperPID)
	}
	if token := processStartToken(pid); token == "" {
		t.Fatalf("no start token for %d", pid)
	}
}

func TestIdentifyPaneCLIProcsIgnoresUnknownTree(t *testing.T) {
	if _, err := os.Stat("/proc/self/stat"); err != nil {
		t.Skip("no /proc on this platform")
	}
	inner := filepath.Join(t.TempDir(), "vite")
	if err := os.WriteFile(inner, []byte("#!/bin/sh\nsleep 5\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := spawnWrappedCLI(t, "vite", exec.Command(inner).Path+" &\nwait")
	wrapperPID := cmd.Process.Pid
	waitUntil(func() bool {
		return len(readProcSnapshot().ppid) > 0
	})
	if cli, _ := identifyPaneCLIProcs(wrapperPID, readProcSnapshot()); cli != "" {
		t.Fatalf("cli=%q want empty for an unknown tree", cli)
	}
	if cli, _ := identifyPaneCLIProcs(0, readProcSnapshot()); cli != "" {
		t.Fatalf("cli=%q want empty for pid 0", cli)
	}
}
