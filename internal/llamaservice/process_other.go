//go:build !linux

package llamaservice

import (
	"context"
	"errors"
	"os"
	"os/exec"
)

func configureProcess(cmd *exec.Cmd) {}
func killProcessGroup(cmd *exec.Cmd) {
	if cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
}

// changeTime is not read here: the local service is Linux-only, and the
// memo's other checks still apply.
func changeTime(os.FileInfo) int64 { return 0 }

func closeOnExec(int) {}

// clearOrphanedGroup has nothing to clear: the local service is Linux-only.
func clearOrphanedGroup(int) {}

// waitExited has no unreaped wait here; the group kill is a plain kill.
func waitExited(cmd *exec.Cmd) (reap func() error) {
	err := cmd.Wait()
	return func() error { return err }
}
func inspectExecutable(ctx context.Context, dir, flag string) ([]byte, error) {
	return nil, errors.New("Local services require Linux or WSL.")
}
