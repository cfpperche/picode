//go:build linux

package server

import (
	"context"
	"errors"
	"os"
	"syscall"
	"time"
)

// Stop only descendants captured from this exact pane; a reused PID is never
// signalled. Refuse a replacement while any captured writer still lives.
func stopPeerPane(ctx context.Context, deps Deps, name, id string, pane int, rt TermRuntime) error {
	if pane <= 0 || rt.PID <= 0 {
		return errors.New("Could not verify the previous process.")
	}
	procs := readProcSnapshot()
	owned := map[int]string{}
	for pid := range procs.ppid {
		for n, current := 0, pid; n < 256 && current > 0; n, current = n+1, procs.ppid[current] {
			if current == pane {
				if token := processStartToken(pid); token != "" {
					owned[pid] = token
				}
				break
			}
		}
	}
	if owned[rt.PID] == "" || owned[rt.PID] != rt.ProcStart {
		return errors.New("The terminal process changed.")
	}
	if e := savePeerStop(deps, id, owned); e != nil {
		return e
	}
	if e := deps.Tmux.KillSession(ctx, name); e != nil {
		return e
	}
	for pid, token := range owned {
		if processStartToken(pid) == token {
			if p, e := os.FindProcess(pid); e == nil {
				_ = p.Signal(syscall.SIGTERM)
			}
		}
	}
	tick := time.NewTicker(50 * time.Millisecond)
	defer tick.Stop()
	for {
		alive := false
		for pid, token := range owned {
			if processAlive(TermRuntime{PID: pid, ProcStart: token}) {
				alive = true
				break
			}
		}
		if !alive {
			_, e := peerStopPending(deps, id)
			return e
		}
		select {
		case <-ctx.Done():
			return errors.New("The previous process is still closing. Try again after it exits.")
		case <-tick.C:
		}
	}
}
