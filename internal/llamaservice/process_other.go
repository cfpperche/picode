//go:build !linux

package llamaservice

import (
	"context"
	"errors"
	"os/exec"
)

func configureProcess(cmd *exec.Cmd) {}
func killProcessGroup(cmd *exec.Cmd) {
	if cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
}
func inspectExecutable(ctx context.Context, dir, flag string) ([]byte, error) {
	return nil, errors.New("Local services require Linux or WSL.")
}
