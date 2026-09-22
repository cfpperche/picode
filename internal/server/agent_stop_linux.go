//go:build linux

package server

import (
	"context"
	"os"
	"strconv"
	"time"
)

func platformProcessStartToken(pid int) string { return "" }

func stopInteractivePane(ctx context.Context, deps Deps, name, id string) error {
	live, err := deps.Tmux.HasSession(ctx, name)
	if err != nil {
		return err
	}
	if !live {
		if blocked, e := peerStopPending(deps, id); e != nil {
			return e
		} else if blocked {
			return errAgentTUIInFlight
		}
		return nil
	}
	pid, err := deps.Tmux.PanePID(ctx, name)
	if err != nil {
		return err
	}
	stopCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	return stopPeerPane(stopCtx, deps, name, id, pid, TermRuntime{PID: pid, ProcStart: processStartToken(pid)})
}

// processZombie reads the state field of /proc/<pid>/stat — field 3, which
// sits right after the comm field's final ')'. An unreadable file means the
// process is gone: on Linux /proc is always there, so its absence for one
// pid is the answer, not a gap.
func processZombie(pid int) bool {
	if pid <= 0 {
		return true
	}
	data, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
	if err != nil {
		return true
	}
	raw := string(data)
	end := -1
	for i := len(raw) - 1; i >= 0; i-- {
		if raw[i] == ')' {
			end = i
			break
		}
	}
	if end < 0 || end+2 >= len(raw) {
		return false
	}
	return raw[end+2] == 'Z'
}
