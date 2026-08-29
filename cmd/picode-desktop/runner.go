package main

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/cfpperche/picode/internal/desktop"
)

// osRunner is the real process boundary. Output returns stdout only, so the
// JSON contract with `picode provision` is never polluted by a shell banner on
// stderr; stderr is folded into the error instead, where it is the diagnosis.
type osRunner struct{}

func (osRunner) Output(name string, args ...string) ([]byte, error) {
	cmd := newCmd(name, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil && stderr.Len() > 0 {
		err = fmt.Errorf("%w: %s", err, strings.TrimSpace(desktop.DecodeWindows(stderr.Bytes())))
	}
	return out, err
}

func (osRunner) Run(name string, args ...string) error {
	cmd := newCmd(name, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return fmt.Errorf("%w: %s", err, strings.TrimSpace(desktop.DecodeWindows(stderr.Bytes())))
		}
		return err
	}
	return nil
}

// startSupervised launches a child meant to outlive the call — the keepalive
// that holds the distro open — and ties its lifetime to this process, so the
// child cannot survive a tray that was force-killed.
func startSupervised(name string, args ...string) (*exec.Cmd, error) {
	cmd := newCmd(name, args...)
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	if err := superviseChild(cmd); err != nil {
		// The keepalive still works; it just outlives a forced kill. Better a
		// stray sleep than no keepalive at all.
		return cmd, err
	}
	return cmd, nil
}
