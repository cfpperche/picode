//go:build !windows

package server

import (
	"errors"
	"syscall"
	"time"
)

// reapPaneGroup ends what a killed tmux pane left running. A PiCode launch
// root ignores SIGHUP by design (ADR-0085), so `kill-session` alone can leave
// the pane's process tree alive; it then kept writing under the test's data
// dir while t.TempDir removed it ("directory not empty"), and some pairs
// outlived their test by hours (measured 2026-09-23). tmux makes each pane
// root its own process-group leader, so the group is the pane's whole tree.
// A pid that is no longer a group leader is left alone: its number may have
// been reused by an unrelated process.
func reapPaneGroup(pid int, wait time.Duration) {
	if pid <= 0 {
		return
	}
	if pgid, err := syscall.Getpgid(pid); err != nil || pgid != pid {
		return
	}
	_ = syscall.Kill(-pid, syscall.SIGTERM)
	if groupGone(pid, wait) {
		return
	}
	_ = syscall.Kill(-pid, syscall.SIGKILL)
	groupGone(pid, 2*time.Second)
}

func groupGone(pgid int, wait time.Duration) bool {
	deadline := time.Now().Add(wait)
	for {
		if err := syscall.Kill(-pgid, 0); errors.Is(err, syscall.ESRCH) {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(20 * time.Millisecond)
	}
}
