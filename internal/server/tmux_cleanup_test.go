package server

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/tmux"
)

// A test that creates a tmux session must kill it with a context that is
// still alive when Cleanup runs. t.Context() is not: the testing package
// cancels it just before Cleanup functions, so the kill command never
// spawns. This test pins both halves — the helper works, and the trap it
// exists for is real — because the failure mode is silent: the test passes
// while a shell survives it forever.
func TestKillTmuxOnCleanupOutlivesTheTestContext(t *testing.T) {
	tm := tmux.New()
	if !tm.Available() {
		t.Skip("tmux not installed")
	}
	stamp := time.Now().UnixNano()
	helped := tmux.ShellSessionName(fmt.Sprintf("cleanup-helper-%d", stamp))
	trapped := tmux.ShellSessionName(fmt.Sprintf("cleanup-trap-%d", stamp))
	// Whatever the subtest leaves behind, this test does not add to the leak.
	t.Cleanup(func() {
		_ = tm.KillSession(context.Background(), helped)
		_ = tm.KillSession(context.Background(), trapped)
	})

	t.Run("sessions", func(t *testing.T) {
		for _, name := range []string{helped, trapped} {
			if err := tm.NewSession(context.Background(), name, t.TempDir(), "sleep", "60"); err != nil {
				t.Fatalf("create %q: %v", name, err)
			}
		}
		killTmuxOnCleanup(t, helped)
		ctx := t.Context()
		t.Cleanup(func() { _ = tm.KillSession(ctx, trapped) }) // the mistake, on purpose
	})

	if alive, err := tm.HasSession(context.Background(), helped); err != nil {
		t.Fatalf("HasSession(%q): %v", helped, err)
	} else if alive {
		t.Errorf("killTmuxOnCleanup left %q running: the cleanup did not reach tmux", helped)
	}
	// The trap is Go's documented behaviour, not ours to guarantee: assert
	// nothing, report what happened, and never leak either way.
	if alive, err := tm.HasSession(context.Background(), trapped); err == nil && alive {
		t.Logf("as expected, a kill issued with t.Context() left %q running", trapped)
	} else if err == nil {
		t.Logf("note: a kill issued with t.Context() reached tmux — Go's cleanup semantics may have changed")
	}
}
