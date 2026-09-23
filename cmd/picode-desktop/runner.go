package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

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

// timedRunner is osRunner with a deadline per call, for the calls a person
// waits on in the Management window: a hung WSL, or a UAC prompt nobody
// answers, must end in an error rather than hold the window (and the
// shell's one-job lock) forever. WaitDelay closes the pipes shortly after
// the kill, so a grandchild holding them cannot stretch the bound.
type timedRunner struct{ d time.Duration }

func (t timedRunner) Output(name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), t.d)
	defer cancel()
	cmd := newCmdContext(ctx, name, args...)
	cmd.WaitDelay = 2 * time.Second
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return out, fmt.Errorf("%s did not finish within %s", name, t.d)
	}
	if err != nil && stderr.Len() > 0 {
		err = fmt.Errorf("%w: %s", err, strings.TrimSpace(desktop.DecodeWindows(stderr.Bytes())))
	}
	return out, err
}

func (t timedRunner) Run(name string, args ...string) error {
	_, err := t.Output(name, args...)
	return err
}
