//go:build linux

package server

// Row 12 of the onboarding table: a native child that ignores SIGTERM must
// leave the shutdown receipt pending — stop never reports success and a
// replacement stays refused until the writer is actually gone.
import (
	"context"
	"os"
	"os/exec"
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
	t.Cleanup(func() {
		_ = m.KillSession(ctx, sess)
	})
	// The pane itself is the stubborn writer: it ignores both signals a stop
	// uses (the launcher's SIGHUP ignore and the SIGTERM escalation).
	script := "trap '' TERM HUP; while :; do :; done"
	env := os.Environ()[:0]
	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, "TMUX=") {
			continue
		}
		env = append(env, kv)
	}
	created := exec.Command("tmux", "new-session", "-d", "-x", "80", "-y", "24", "-s", sess, "sh", "-c", script)
	created.Env = env
	if out, err := created.CombinedOutput(); err != nil {
		t.Fatalf("new-session: %v: %s", err, out)
	}
	out, err := exec.Command("tmux", "display-message", "-p", "-t", sess, "#{pane_pid}").Output()
	if err != nil {
		t.Fatalf("pane pid: %v: %s", err, out)
	}
	pane, err := strconv.Atoi(strings.TrimSpace(string(out)))
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
