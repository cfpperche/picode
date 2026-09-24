//go:build !windows

package server

import (
	"os/exec"
	"syscall"
)

// setLoginPgroup puts a GUI sign-in's process in its own group, so the whole
// tree (the pty wrapper and what it runs) can be killed at once instead of
// leaving orphans behind. killLoginTree kills that group; killLoginProcess
// kills the leader alone, for the callers that need both.
func setLoginPgroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func killLoginTree(pid int) {
	_ = syscall.Kill(-pid, syscall.SIGKILL)
}

func killLoginProcess(pid int) {
	_ = syscall.Kill(pid, syscall.SIGKILL)
}
