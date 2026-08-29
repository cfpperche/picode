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

// startDetached launches a child meant to outlive the call — the keepalive
// that holds the distro open.
func startDetached(name string, args ...string) (*exec.Cmd, error) {
	cmd := newCmd(name, args...)
	return cmd, cmd.Start()
}
