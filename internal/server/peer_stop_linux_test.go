//go:build linux

package server

// Row 12 of the onboarding table: a native child that ignores SIGTERM must
// leave the shutdown receipt pending — stop never reports success and a
// replacement stays refused until the writer is actually gone.
import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/tmux"
)

func TestPeerStopStubbornChildStaysPending(t *testing.T) {
	m := tmux.New()
	if !m.Available() {
		t.Skip("tmux not installed")
	}
	if processStartToken(os.Getpid()) == "" {
		t.Skip("process ownership tokens unavailable")
	}
	ctx := context.Background()
	sess := "picode-stopchild-" + strconv.Itoa(os.Getpid())
	marker := filepath.Join(t.TempDir(), "ready")
	script := "trap '' TERM HUP; : > " + marker + "; while :; do :; done"
	// The test may itself run inside tmux (a PiCode terminal). A nested client
	// resolves `-t` against the attached session, so every command here runs
	// without the ambient TMUX environment and targets the session exactly.
	env := []string{}
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(kv, "TMUX=") {
			env = append(env, kv)
		}
	}
	run := func(args ...string) ([]byte, error) {
		cmd := exec.Command("tmux", args...)
		cmd.Env = env
		return cmd.CombinedOutput()
	}
	if out, err := run("new-session", "-d", "-x", "80", "-y", "24", "-s", sess, "sh", "-c", script); err != nil {
		t.Fatalf("new-session: %v: %s", err, out)
	}
	t.Cleanup(func() { _, _ = run("kill-session", "-t", "="+sess) })
	// The pane process must have installed its traps before the stop signals it,
	// otherwise a startup race decides the test.
	ready := time.Now().Add(3 * time.Second)
	for {
		if _, err := os.Stat(marker); err == nil {
			break
		}
		if time.Now().After(ready) {
			t.Fatal("pane never installed its traps")
		}
		time.Sleep(10 * time.Millisecond)
	}
	out, err := run("list-panes", "-t", "="+sess, "-F", "#{pane_pid}")
	if err != nil {
		t.Fatalf("pane pid: %v: %s", err, out)
	}
	pane, err := strconv.Atoi(strings.TrimSpace(strings.Split(string(out), "\n")[0]))
	if err != nil || pane <= 0 {
		t.Fatalf("pane pid %q", out)
	}
	t.Cleanup(func() {
		// The pane process is the writer; killing it ends the session even
		// when the signals above were trapped.
		_ = syscall.Kill(pane, syscall.SIGKILL)
	})
	token := processStartToken(pane)
	if token == "" {
		t.Skip("process ownership tokens unavailable")
	}

	deps := Deps{DataDir: t.TempDir(), Tmux: m}
	rt := TermRuntime{PID: pane, ProcStart: token}
	runCtx, cancel := context.WithTimeout(ctx, 400*time.Millisecond)
	defer cancel()
	stopErr := stopPeerPane(runCtx, deps, sess, "fixture", pane, rt)
	if stopErr == nil || !strings.Contains(stopErr.Error(), "still closing") {
		t.Fatalf("stubborn child reported stopped: %v", stopErr)
	}
	pending, e := peerStopPending(deps, "fixture")
	if !pending || e != nil {
		t.Fatalf("surviving writer must keep the receipt pending: %v %v", pending, e)
	}
	// Once the writer actually dies, the same receipt clears by itself and a
	// replacement may start again.
	if err := syscall.Kill(pane, syscall.SIGKILL); err != nil {
		t.Fatal(err)
	}
	for deadline := time.Now().Add(3 * time.Second); time.Now().Before(deadline); {
		if pending, e := peerStopPending(deps, "fixture"); !pending && e == nil {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatal("receipt never cleared after the writer died")
}
