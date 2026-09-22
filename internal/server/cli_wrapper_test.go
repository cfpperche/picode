package server

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/clilaunch"
)

// fakeWrapperBin drops a stub real binary that reports whether the wrapper
// granted it a presence lease (PICODE_TUI_RUN_ID exported) and echoes argv.
func fakeWrapperBin(t *testing.T, name string) string {
	t.Helper()
	bin := t.TempDir()
	script := "#!/bin/sh\necho \"RUN_ID=${PICODE_TUI_RUN_ID-unset}\"\nprintf 'ARGV:'\nprintf ' <%s>' \"$@\"\necho\n"
	if err := os.WriteFile(filepath.Join(bin, name), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return bin
}

// runWrapper executes the installed dataDir wrapper the way a terminal
// would: TERM_ID set, the fake real binary first on PATH.
func runWrapper(t *testing.T, dataDir, name, fakebin string, args ...string) string {
	t.Helper()
	cmd := exec.Command(wrapperPath(dataDir, name), args...)
	// Scrubbed ambient TUI vars: the lease verdict must come from the
	// wrapper's own export, never from the invoking shell (the agent
	// terminal running this suite carries real ones).
	cmd.Env = append(os.Environ(), "PICODE_TERM_ID=t-lease-1", "PICODE_TUI_RUN_ID=", "PICODE_TUI_PID=", "PATH="+fakebin+string(os.PathListSeparator)+os.Getenv("PATH"))
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("%s %v: %v\n%s", name, args, err, out)
	}
	return string(out)
}

// Decision table: interactive invocations keep the presence lease (the
// stub sees RUN_ID set); maintenance subcommands and headless flags do
// not. A wrong lease either spams runtimes or drops presence.
func TestWrapperPresenceLease(t *testing.T) {
	agyTestHome(t)
	dataDir := t.TempDir()
	if _, err := ensureHookScript(dataDir); err != nil {
		t.Fatal(err)
	}
	if err := installIntercept(dataDir, "muse"); err != nil {
		t.Fatal(err)
	}
	if err := installIntercept(dataDir, "agy"); err != nil {
		t.Fatal(err)
	}
	if err := installIntercept(dataDir, "omp"); err != nil {
		t.Fatal(err)
	}
	museBin := fakeWrapperBin(t, "muse")
	agyBin := fakeWrapperBin(t, "agy")
	ompBin := fakeWrapperBin(t, "omp")
	cases := []struct {
		cli, wantLease string
		args           []string
	}{
		{"muse", "lease", nil},
		{"muse", "lease", []string{"a starting prompt"}},
		{"muse", "lease", []string{"resume", "--last"}},
		{"muse", "lease", []string{"--model", "x"}},
		{"muse", "none", []string{"exec", "hi"}},
		{"muse", "none", []string{"export", "--session", "s"}},
		{"muse", "none", []string{"--version"}},
		{"muse", "none", []string{"serve"}},
		{"agy", "lease", nil},
		{"agy", "lease", []string{"-i"}},
		{"agy", "lease", []string{"--prompt-interactive", "hi"}},
		{"agy", "lease", []string{"--conversation", "c9"}},
		{"agy", "lease", []string{"--model", "m"}},
		{"agy", "none", []string{"-p", "hi"}},
		{"agy", "none", []string{"--print", "hi"}},
		{"agy", "none", []string{"update"}},
		{"agy", "none", []string{"mcp", "list"}},
		{"omp", "lease", nil},
		{"omp", "lease", []string{"a starting prompt"}},
		{"omp", "lease", []string{"-c"}},
		{"omp", "lease", []string{"--resume", "01a0b00a"}},
		{"omp", "lease", []string{"--model", "gemini-3.6-flash", "fix the tests"}},
		{"omp", "none", []string{"update"}},
		{"omp", "none", []string{"config", "path"}},
		{"omp", "none", []string{"acp"}},
		{"omp", "none", []string{"--mode", "rpc"}},
		{"omp", "none", []string{"--mode=rpc"}},
		{"omp", "none", []string{"--export", "session.html"}},
		{"omp", "none", []string{"--export=session.html"}},
		{"omp", "none", []string{"-p", "hi"}},
		{"omp", "none", []string{"--version"}},
	}
	for _, c := range cases {
		bin := museBin
		if c.cli == "agy" {
			bin = agyBin
		}
		if c.cli == "omp" {
			bin = ompBin
		}
		out := runWrapper(t, dataDir, c.cli, bin, c.args...)
		leased := !strings.Contains(out, "RUN_ID=unset") && !strings.Contains(out, "RUN_ID=\n")
		if (c.wantLease == "lease") != leased {
			t.Errorf("%s %v: lease = %v, want %s\n%s", c.cli, c.args, leased, c.wantLease, out)
		}
		if !strings.Contains(out, "ARGV:") {
			t.Errorf("%s %v: real binary did not run\n%s", c.cli, c.args, out)
		}
	}
}

// installIntercept for the wrapper CLIs is executable shell the terminal
// can run, carrying the runtime-start call the presence lease needs.
func TestWrapperInstallShape(t *testing.T) {
	agyTestHome(t)
	dataDir := t.TempDir()
	if _, err := ensureHookScript(dataDir); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"muse", "agy", "omp"} {
		if err := installIntercept(dataDir, name); err != nil {
			t.Fatal(err)
		}
		raw, err := os.ReadFile(wrapperPath(dataDir, name))
		if err != nil {
			t.Fatal(err)
		}
		body := string(raw)
		for _, want := range []string{"name=" + name, "runtime-start", "\"$real\"", "runtime-end"} {
			if !strings.Contains(body, want) {
				t.Errorf("%s wrapper lacks %q", name, want)
			}
		}
		if name == "omp" {
			// The extensions ride only session runs; the wrapper must name
			// both injected files and keep maintenance runs extension-free.
			for _, want := range []string{"omp-terminal-state.ts", "omp-checklist.ts"} {
				if !strings.Contains(body, want) {
					t.Errorf("omp wrapper lacks %s", want)
				}
			}
			ext, err := os.ReadFile(filepath.Join(dataDir, "intercept", "omp-terminal-state.ts"))
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(ext), `"omp"`) || !strings.Contains(string(ext), "session_start") {
				t.Error("omp extension not shaped for omp or not named omp")
			}
			// omp's measured event set: agent_end settles (only when the run is
			// really over — willContinue marks a mid-run turn), approvals arrive
			// as tool_approval_requested/resolved and the ask card as
			// tool_execution_start/tool_result; agent_settled and ui_prompt_*
			// never fire.
			for _, want := range []string{`"agent_end"`, "willContinue", `"tool_approval_requested"`, `"tool_approval_resolved"`, `"tool_execution_start"`, `"tool_result"`} {
				if !strings.Contains(string(ext), want) {
					t.Errorf("omp extension lacks %s", want)
				}
			}
			if strings.Contains(string(ext), "agent_settled") || strings.Contains(string(ext), "ui_prompt_start") {
				t.Error("omp extension listens to pi events omp never fires")
			}
		}
	}
}

// Uninstalling omp removes the wrapper and both extensions — the mirror file
// must not outlive the toggle that injected it.
func TestInterceptOmpUninstallRemovesExtensions(t *testing.T) {
	agyTestHome(t)
	dataDir := t.TempDir()
	if err := installIntercept(dataDir, "omp"); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{wrapperPath(dataDir, "omp"), ompTerminalStateExtensionFile(dataDir), ompChecklistExtensionFile(dataDir)} {
		if _, err := os.Stat(f); err != nil {
			t.Fatalf("install left %s: %v", f, err)
		}
	}
	if err := uninstallIntercept(dataDir, "omp"); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{wrapperPath(dataDir, "omp"), ompTerminalStateExtensionFile(dataDir), ompChecklistExtensionFile(dataDir)} {
		if _, err := os.Stat(f); !os.IsNotExist(err) {
			t.Fatalf("uninstall left %s behind: %v", f, err)
		}
	}
}

// Decision table for the omp adapter's one measured foot-gun: omp refuses
// a run outright when a --trusted-extension launch argument meets PiCode's
// injected -e. Preview names the conflict; prepare refuses; integration
// off stays clean.
func TestOmpTrustedExtensionConflictTable(t *testing.T) {
	home := agyTestHome(t)
	// Preview checks arguments without launching the CLI. Use a local fixture
	// so this decision table does not depend on an installed Omp executable.
	tool := filepath.Join(home, "omp-preview")
	if err := os.WriteFile(tool, []byte("#!/bin/sh\nexit 1\n"), 0700); err != nil {
		t.Fatal(err)
	}
	ts, _, _ := cleanupServer(t)
	for _, tc := range []struct {
		name        string
		integration bool
		args        []string
		wantProblem string
	}{
		{"integration off, trusted extension", false, []string{"--trusted-extension", "/x.ts"}, ""},
		{"integration on, clean args", true, nil, ""},
		{"integration on, trusted extension", true, []string{"--trusted-extension", "/x.ts"}, "conflicts with"},
		{"integration on, trusted extension inline", true, []string{"--trusted-extension=/x.ts"}, "conflicts with"},
		{"integration on, lookalike flag", true, []string{"--trusted-extension-something"}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := clilaunch.Config{Executable: tool, Integration: tc.integration, Args: tc.args}
			v := cliRequest(t, ts, "POST", "/api/clis/omp/preview", map[string]any{"config": c}, 200)
			plan, _ := v["plan"].(map[string]any)
			problem, _ := plan["problem"].(string)
			if tc.wantProblem == "" && problem != "" {
				t.Errorf("problem = %q, want none", problem)
			}
			if tc.wantProblem != "" && !strings.Contains(problem, tc.wantProblem) {
				t.Errorf("problem = %q, want it to contain %q", problem, tc.wantProblem)
			}
		})
	}
	if !ompTrustedExtensionConflict(clilaunch.Config{Args: []string{"--trusted-extension"}}) {
		t.Error("bare --trusted-extension not detected")
	}
}
