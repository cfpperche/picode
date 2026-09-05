//go:build windows

package rpc

import (
	"os"
	"os/exec"
)

// setPgroup is a no-op on windows: process groups are a posix concept.
// The managed-agent server does not run on windows; if it ever does, Job
// Objects are the faithful equivalent of killProcessTree below.
func setPgroup(cmd *exec.Cmd) {}

// killProcessTree kills the direct child. Windows lacks posix process
// groups here, so a grandchild that inherits the stdout pipe would keep
// it open; Close compensates by closing the pipe itself (see client.go).
func killProcessTree(p *os.Process) {
	if p != nil {
		_ = p.Kill()
	}
}
