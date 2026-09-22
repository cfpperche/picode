//go:build !linux && !windows

package server

import (
	"errors"
	"syscall"
)

// processZombie answers "has this process ended?" where there is no /proc.
//
// The Linux version treats an unreadable /proc/<pid>/stat as "gone", which
// is right there and wrong here: on macOS the file never exists, so every
// live process looked dead. devServerProcessGone feeds Stop and the hide
// sweep, so a macOS user got `stopped: true` for a server still running and
// lost every hide on the next read.
//
// Signal 0 is the portable liveness test: ESRCH means the pid is gone, and
// anything else — including EPERM for a process this user does not own —
// means it is there. A zombie cannot be told apart this way, so a process
// that exists is reported as not-a-zombie: the caller has already matched
// the start token, so "the same process is still here" is the honest answer
// available.
func processZombie(pid int) bool {
	if pid <= 0 {
		return true
	}
	return errors.Is(syscall.Kill(pid, 0), syscall.ESRCH)
}
