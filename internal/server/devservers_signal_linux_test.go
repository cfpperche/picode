//go:build linux

package server

import (
	"context"
	"os/exec"
	"testing"
	"time"
)

// The real signal path, with real children: an assertion about Stop that
// never ended a process would be an assertion about the seam, not about the
// feature. Everything here starts its own child and reaps it.

// startDevServerChild starts a shell child and answers its pid, waiting until
// /proc can identify it (the start token is what the guards compare).
func startDevServerChild(t *testing.T, script string) (int, string) {
	t.Helper()
	cmd := exec.Command("/bin/sh", "-c", script)
	if err := cmd.Start(); err != nil {
		t.Skipf("cannot start a child here: %v", err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	})
	pid := cmd.Process.Pid
	deadline := time.Now().Add(2 * time.Second)
	for {
		if token := processStartToken(pid); token != "" {
			return pid, token
		}
		if time.Now().After(deadline) {
			t.Fatalf("pid %d never appeared in /proc", pid)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func waitGone(t *testing.T, pid int, token string) bool {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for {
		if devServerProcessGone(pid, token) {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestSignalDevServerProcessEndsAChild(t *testing.T) {
	pid, token := startDevServerChild(t, "exec sleep 30")
	if err := signalDevServerProcess(pid, false); err != nil {
		t.Fatalf("SIGTERM: %v", err)
	}
	if !waitGone(t, pid, token) {
		t.Fatal("a child that does not trap SIGTERM survived Stop")
	}

	// A process that ignores SIGTERM needs the force path — and that is the
	// reason the panel offers it at all.
	pid, token = startDevServerChild(t, "trap '' TERM; exec sleep 30")
	if err := signalDevServerProcess(pid, false); err != nil {
		t.Fatalf("SIGTERM: %v", err)
	}
	time.Sleep(200 * time.Millisecond)
	if devServerProcessGone(pid, token) {
		t.Skip("this shell does not honour the trap: force was not needed")
	}
	if err := signalDevServerProcess(pid, true); err != nil {
		t.Fatalf("SIGKILL: %v", err)
	}
	if !waitGone(t, pid, token) {
		t.Fatal("a child that ignores SIGTERM survived Force")
	}
}

// TestDevServerStopEndsARealChild is the whole route against a real process:
// the listener is a child this test started, the signal is the platform's own,
// and the answer is the truth about a process that actually ended.
func TestDevServerStopEndsARealChild(t *testing.T) {
	pid, token := startDevServerChild(t, "exec sleep 30")
	owners := map[int]devServerOwner{
		5173: {kind: "term", id: "t1", name: "web", workspace: "PiCode", tool: "sleep", pid: pid, startKey: token},
	}
	deps, _ := actionDeps(t, owners)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	code, out := postDevServerAction(t, handleDevServerStop(deps), map[string]any{"port": 5173, "pid": pid, "startKey": token}, ctx)
	if code != 200 {
		t.Fatalf("status=%d body=%v", code, out)
	}
	if out["stopped"] != true {
		t.Fatalf("a child that ended answered stopped=%v", out["stopped"])
	}
}
