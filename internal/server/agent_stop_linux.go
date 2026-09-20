//go:build linux

package server

import (
	"context"
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
