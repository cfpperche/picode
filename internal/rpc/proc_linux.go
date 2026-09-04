//go:build linux

package rpc

import (
	"os/exec"
	"syscall"
)

// The daemon owns transient RPC workers. Interactive TUI processes live under
// tmux and intentionally survive deploys; a leased writer must do the opposite
// or it can overlap the crash-safe TUI restore and corrupt one session JSONL.
func configureChild(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Pdeathsig: syscall.SIGKILL}
}
