package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/tmux"
)

// Text reaches a real Omp TUI through the unattended prompt door
// (ADR-0217's reader), wide and at a phone's width. Live only:
// PICODE_OMP_LIVE=1, PICODE_OMP_BIN naming the real omp, PICODE_OMP_HOME the
// real home (the package isolates HOME) and PICODE_OMP_CWD a folder Omp knows.
// The tmux server is this test's own socket, never the user's. The task is
// submitted, so each width costs one short model turn.
func TestLiveOmpDoorWideAndNarrow(t *testing.T) {
	bin := os.Getenv("PICODE_OMP_BIN")
	if os.Getenv("PICODE_OMP_LIVE") != "1" || bin == "" {
		t.Skip("PICODE_OMP_LIVE=1 and PICODE_OMP_BIN run this against a real omp")
	}
	for _, width := range []int{100, 44} {
		t.Run(fmt.Sprintf("%dcols", width), func(t *testing.T) {
			deps, _, launch := guestLaunchFixture(t, "omp")
			// A folder Omp already knows: a fresh one opens its setup wizard,
			// which the door rightly refuses as "not at a prompt".
			cwd := os.Getenv("PICODE_OMP_CWD")
			if cwd == "" {
				cwd, _ = os.Getwd()
			}
			deps.Tmux = tmux.NewWithSocket(filepath.Join(t.TempDir(), "tmux.sock"))
			deps.TermStates = NewTermStates()
			name := tmux.ShellSessionName(launch.TerminalID)
			ctx := context.Background()
			// The package's tests run with an isolated HOME; Omp needs the
			// real one, or it opens its first-run setup instead of a prompt.
			var env []string
			if h := os.Getenv("PICODE_OMP_HOME"); h != "" {
				env = append(env, "HOME="+h)
			}
			if err := deps.Tmux.NewSessionEnvSize(ctx, name, cwd, width, 40, env, bin); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = deps.Tmux.KillSession(context.Background(), name) })
			term, err := deps.Store.GetTerminal(launch.TerminalID)
			if err != nil {
				t.Fatal(err)
			}
			task := "first line\nsecond line"
			var last map[string]any
			deadline := time.Now().Add(60 * time.Second)
			for time.Now().Before(deadline) {
				time.Sleep(2 * time.Second)
				c, cancel := context.WithTimeout(ctx, 20*time.Second)
				status, res := doorDeliverUnattended(deps, c, term, task)
				cancel()
				last = res
				if status == http.StatusOK {
					t.Logf("width %d: delivered %v", width, res)
					time.Sleep(3 * time.Second)
					snap, _ := deps.Tmux.InputSnapshot(ctx, name)
					for i, l := range snap.Lines {
						if c := strings.TrimSpace(terminalSGR.ReplaceAllString(l, "")); c != "" && i >= len(snap.Lines)-14 {
							t.Logf("  %2d %s", i, c)
						}
					}
					return
				}
			}
			snap, _ := deps.Tmux.InputSnapshot(ctx, name)
			var shown []string
			for i, l := range snap.Lines {
				if strings.TrimSpace(l) != "" {
					shown = append(shown, fmt.Sprintf("%2d %q", i, l))
				}
			}
			t.Fatalf("width %d: not delivered: %v\ncursor %d,%d width %d\n%s", width, last, snap.CursorX, snap.CursorY, snap.Width, strings.Join(shown, "\n"))
		})
	}
}
