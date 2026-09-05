//go:build windows

package rpc

// processAlive is only consulted from unix-verified assertions; the test
// skips before reaching it on windows.
func processAlive(pid int) bool { return true }
