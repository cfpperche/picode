package llamaservice

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestSupervisorClosesDescendants(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux process group acceptance; see docs/plans/llama-manager.md.")
	}
	self, _ := os.Executable()
	dir := t.TempDir()
	pidFile := filepath.Join(dir, "child")
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(self, "--internal-llama-supervisor", "/bin/sh", "-c", `sleep 60 & echo $! > "$1"; wait`, "qa", pidFile)
	cmd.Stdin = reader
	if err = cmd.Start(); err != nil {
		t.Fatal(err)
	}
	_ = reader.Close()
	t.Cleanup(func() { _ = writer.Close(); _ = cmd.Process.Kill() })
	var raw []byte
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		raw, err = os.ReadFile(pidFile)
		if err == nil && len(raw) > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if len(raw) == 0 {
		t.Fatal("descendant did not start")
	}
	_ = writer.Close()
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err = <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("supervisor did not exit on parent disconnect")
	}
	// SIGKILL delivery to descendants is asynchronous. Wait for their exit too,
	// rather than assuming the shell's Wait reaped unrelated descendants.
	deadline = time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		stat, e := os.ReadFile("/proc/" + strings.TrimSpace(string(raw)) + "/stat")
		if os.IsNotExist(e) || (e == nil && strings.Contains(string(stat), ") Z ")) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("descendant did not exit after process-group termination")
}

// A router that dies on its own (the OOM killer, a crash) takes its children
// with it. This guards the behavior, not the 2026-09-25 ordering change: a
// live child keeps the group's number reserved, so the old reap-then-kill
// passed too. What the kill-before-reap closes — every member already dead
// and the router's PID reused by a new group leader — is not reproducible
// here.
func TestSupervisorClosesDescendantsWhenTheRouterDies(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux process group acceptance; see docs/plans/llama-manager.md.")
	}
	self, _ := os.Executable()
	dir := t.TempDir()
	pidFile := filepath.Join(dir, "pids")
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	// The shell stands in for the router: it starts a child, writes both
	// pids, and waits.
	cmd := exec.Command(self, "--internal-llama-supervisor", "/bin/sh", "-c", `sleep 60 & echo "$$ $!" > "$1"; wait`, "qa", pidFile)
	cmd.Stdin = reader
	if err = cmd.Start(); err != nil {
		t.Fatal(err)
	}
	_ = reader.Close()
	t.Cleanup(func() { _ = writer.Close(); _ = cmd.Process.Kill() })
	var fields []string
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		raw, _ := os.ReadFile(pidFile)
		if fields = strings.Fields(string(raw)); len(fields) == 2 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if len(fields) != 2 {
		t.Fatal("router and child did not start")
	}
	router, _ := strconv.Atoi(fields[0])
	proc, err := os.FindProcess(router)
	if err != nil {
		t.Fatal(err)
	}
	if err := proc.Kill(); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case <-done: // exit 1: the router was killed
	case <-time.After(3 * time.Second):
		t.Fatal("supervisor did not exit after its router died")
	}
	deadline = time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		stat, e := os.ReadFile("/proc/" + fields[1] + "/stat")
		if os.IsNotExist(e) || (e == nil && strings.Contains(string(stat), ") Z ")) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("the router's child outlived the router")
}
