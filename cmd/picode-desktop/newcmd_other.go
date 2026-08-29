//go:build !windows

package main

import "os/exec"

// Off Windows there is no console to suppress. This build exists so `doctor`
// can be run from inside the distro while developing, where wsl.exe is
// reachable through /mnt/c — the tray itself stays Windows-only.
func newCmd(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}
