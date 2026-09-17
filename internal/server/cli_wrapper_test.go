package server

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
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
	museBin := fakeWrapperBin(t, "muse")
	agyBin := fakeWrapperBin(t, "agy")
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
	}
	for _, c := range cases {
		bin := museBin
		if c.cli == "agy" {
			bin = agyBin
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
	for _, name := range []string{"muse", "agy"} {
		if err := installIntercept(dataDir, name); err != nil {
			t.Fatal(err)
		}
		raw, err := os.ReadFile(wrapperPath(dataDir, name))
		if err != nil {
			t.Fatal(err)
		}
		body := string(raw)
		for _, want := range []string{"name=" + name, "runtime-start", "\"$real\" \"$@\"", "runtime-end"} {
			if !strings.Contains(body, want) {
				t.Errorf("%s wrapper lacks %q", name, want)
			}
		}
	}
}
