//go:build windows

package main

import (
	"os/exec"
	"syscall"
)

// createNoWindow keeps children from flashing a console window. Linking with
// -H=windowsgui hides this program's own console but does nothing for the
// wsl.exe and powershell.exe processes it spawns, so both are needed.
const createNoWindow = 0x08000000

func newCmd(name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: createNoWindow}
	return cmd
}
