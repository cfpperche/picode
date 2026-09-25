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

// A supervisor killed outright (ADR-0090 amendment 2026-09-25): the router
// dies of its parent-death signal, its child does not — until the daemon,
// which learned the router's group from the supervisor, clears the group.
func TestDaemonClearsTheGroupOfAKilledSupervisor(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux process group acceptance; see docs/plans/llama-manager.md.")
	}
	self, _ := os.Executable()
	pidFile := filepath.Join(t.TempDir(), "child")
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	groupR, groupW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(self, "--internal-llama-supervisor", "/bin/sh", "-c", `sleep 60 & echo $! > "$1"; wait`, "qa", pidFile)
	cmd.Stdin = reader
	cmd.ExtraFiles = []*os.File{groupW}
	cmd.Env = append(os.Environ(), routerFDEnv+"=3")
	if err = cmd.Start(); err != nil {
		t.Fatal(err)
	}
	_ = reader.Close()
	_ = groupW.Close()
	t.Cleanup(func() { _ = writer.Close(); _ = cmd.Process.Kill() })
	group := readRouterGroup(groupR)
	if group <= 1 {
		t.Fatal("the supervisor did not report its router")
	}
	var child string
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		raw, _ := os.ReadFile(pidFile)
		if child = strings.TrimSpace(string(raw)); child != "" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if child == "" {
		t.Fatal("the router's child did not start")
	}
	t.Cleanup(func() {
		if pid, _ := strconv.Atoi(child); pid > 1 {
			if p, err := os.FindProcess(pid); err == nil {
				_ = p.Kill()
			}
		}
	})
	alive := func() bool {
		stat, e := os.ReadFile("/proc/" + child + "/stat")
		return e == nil && !strings.Contains(string(stat), ") Z ")
	}
	if err := cmd.Process.Kill(); err != nil { // the supervisor, outright
		t.Fatal(err)
	}
	_ = cmd.Wait()
	time.Sleep(300 * time.Millisecond)
	if !alive() {
		t.Fatal("the child died without the daemon: this test no longer shows the gap")
	}
	clearOrphanedGroup(group)
	deadline = time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if !alive() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("the router's child outlived the daemon's group clear")
}

// While a process holds the group's number as its PID the group is not
// ours to clear — a reused number, or a leader still alive — and nothing is
// sent to it.
func TestGroupClearLeavesAGroupWithALiveLeader(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux process group acceptance; see docs/plans/llama-manager.md.")
	}
	leader := exec.Command("sleep", "60")
	configureProcess(leader) // its own group, whose number is its PID
	if err := leader.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = leader.Process.Kill(); _ = leader.Wait() })
	clearOrphanedGroup(leader.Process.Pid)
	stat, err := os.ReadFile("/proc/" + strconv.Itoa(leader.Process.Pid) + "/stat")
	if err != nil || strings.Contains(string(stat), ") Z ") {
		t.Fatal("a group whose leader is alive was killed")
	}
}

// The report descriptor and its variable stay with the supervisor: the router
// (and so its model servers) sees neither.
func TestRouterDoesNotInheritTheReportDescriptor(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux process acceptance; see docs/plans/llama-manager.md.")
	}
	self, _ := os.Executable()
	out := filepath.Join(t.TempDir(), "seen")
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	groupR, groupW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	script := `{ [ -e /proc/$$/fd/3 ] && echo fd3; echo "env=$` + routerFDEnv + `"; } > "$1"; sleep 60`
	cmd := exec.Command(self, "--internal-llama-supervisor", "/bin/sh", "-c", script, "qa", out)
	cmd.Stdin = reader
	cmd.ExtraFiles = []*os.File{groupW}
	cmd.Env = append(os.Environ(), routerFDEnv+"=3")
	if err = cmd.Start(); err != nil {
		t.Fatal(err)
	}
	_ = reader.Close()
	_ = groupW.Close()
	t.Cleanup(func() { _ = writer.Close(); _ = cmd.Wait() })
	if readRouterGroup(groupR) <= 1 {
		t.Fatal("the supervisor did not report its router")
	}
	var seen string
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		raw, _ := os.ReadFile(out)
		if seen = string(raw); strings.Contains(seen, "env=") {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if strings.Contains(seen, "fd3") || strings.TrimSpace(seen) != "env=" {
		t.Fatalf("the router inherited the report: %q", seen)
	}
}
