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
// waitExited has no unreaped wait here; the group kill is a plain kill.
func waitExited(cmd *exec.Cmd) (reap func() error) {
	err := cmd.Wait()
	return func() error { return err }
}
func inspectExecutable(ctx context.Context, dir, flag string) ([]byte, error) {
	return nil, errors.New("Local services require Linux or WSL.")
}
