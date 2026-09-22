//go:build !linux

package server

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func platformProcessStartToken(pid int) string {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "ps", "-p", strconv.Itoa(pid), "-o", "lstart=").Output()
	if err != nil {
		return ""
	}
	if token := strings.TrimSpace(string(out)); token != "" {
		return "ps:" + token
	}
	return ""
}

// Unix hosts without /proc use ps to capture the exact pane's descendants.
// Keep the shutdown receipt so a timeout cannot authorize another writer.
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
	call, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	out, err := exec.CommandContext(call, "ps", "-axo", "pid=,ppid=").Output()
	if err != nil {
		return err
	}
	parents := map[int]int{}
	for _, line := range strings.Split(string(out), "\n") {
		f := strings.Fields(line)
		if len(f) != 2 {
			continue
		}
		child, _ := strconv.Atoi(f[0])
		parent, _ := strconv.Atoi(f[1])
		parents[child] = parent
	}
	owned := map[int]string{}
	for child := range parents {
		for n, p := 0, child; n < 256 && p > 0; n, p = n+1, parents[p] {
			if p == pid {
				if token := processStartToken(child); token != "" {
					owned[child] = token
				}
				break
			}
		}
	}
	if owned[pid] == "" {
		return errors.New("Could not verify the terminal process.")
	}
	if err = savePeerStop(deps, id, owned); err != nil {
		return err
	}
	if err = deps.Tmux.KillSession(call, name); err != nil {
		return err
	}
	for child, token := range owned {
		if processStartToken(child) == token {
			if p, e := os.FindProcess(child); e == nil {
				_ = p.Signal(syscall.SIGTERM)
			}
		}
	}
	tick := time.NewTicker(50 * time.Millisecond)
	defer tick.Stop()
	for {
		blocked, e := peerStopPending(deps, id)
		if e != nil {
			return e
		}
		if !blocked {
			return nil
		}
		select {
		case <-call.Done():
			return errors.New("The previous process is still closing. Try again after it exits.")
		case <-tick.C:
		}
	}
}
