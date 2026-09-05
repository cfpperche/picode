//go:build unix

package rpc

import "syscall"

// processAlive reports whether the pid still corresponds to a live process.
func processAlive(pid int) bool {
	return syscall.Kill(pid, syscall.Signal(0)) == nil
}
