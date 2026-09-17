//go:build linux

package server

import (
	"os"
	"syscall"
)

// signalDevServerProcess asks one process to end. SIGTERM first — a dev
// server that flushed its state is a dev server that did not lose it — and
// SIGKILL only when the panel's Stop already did not land (the human is told
// which one happened). The caller has re-read /proc and matched the process's
// start token, so this never signals a process that reused the id.
func signalDevServerProcess(pid int, force bool) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	if force {
		return process.Signal(syscall.SIGKILL)
	}
	return process.Signal(syscall.SIGTERM)
}
