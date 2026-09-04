//go:build !linux

package rpc

import "os/exec"

// Non-Linux platforms use the burst holder's recorded PID as the crash
// fallback. Linux additionally has a kernel parent-death guarantee.
func configureChild(_ *exec.Cmd) {}
