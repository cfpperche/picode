//go:build !windows

package main

import "os/exec"

// Off Windows there is no tray, so nothing starts a keepalive to supervise.
func superviseChild(*exec.Cmd) error { return nil }
