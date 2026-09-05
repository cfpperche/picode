//go:build linux

package server

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

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
