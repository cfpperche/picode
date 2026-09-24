//go:build windows

package server

import (
	"os"
	"os/exec"
)

// The daemon does not run on Windows (ADR-0020) — this file exists so the
// package compiles for it, which is what the Windows leg of CI checks. There
// are no process groups here, so a "tree" kill degrades to the direct child;
// that is honest rather than silent, and the code that calls it never runs on
// this platform.
func setLoginPgroup(*exec.Cmd) {}

func killLoginTree(pid int) { killLoginProcess(pid) }

func killLoginProcess(pid int) {
	if p, err := os.FindProcess(pid); err == nil {
		_ = p.Kill()
	}
}
