//go:build unix

package rpc

import (
	"os"
	"os/exec"
	"syscall"
)

// setPgroup puts the spawned process in its own process group, so Close
// can signal the whole tree with one kill.
func setPgroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// killProcessTree SIGKILLs the child's whole process group. The intercept
// wrapper (~/.picode/bin/pi) runs the real pi as a child, and that child
// inherits the client's stdout pipe — killing only the direct child left
// the pipe write end open, pump never saw EOF, and Close waited on <-done
// forever, wedging every later stop and open (the 2026-09-05 pi-diff
// hang: Stop did nothing, Open terminal never switched to the TUI). A
// dedicated group makes the grandchild addressable in one signal.
func killProcessTree(p *os.Process) {
	if p == nil {
		return
	}
	_ = syscall.Kill(-p.Pid, syscall.SIGKILL) // the group: children and grandchildren
	_ = p.Kill()                              // belt and braces for the leader itself
}
