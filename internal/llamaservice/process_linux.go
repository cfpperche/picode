//go:build linux

package llamaservice

import (
	"context"
	"os/exec"
	"syscall"
)

func configureProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Pdeathsig: syscall.SIGKILL, Setpgid: true}
}
func killProcessGroup(cmd *exec.Cmd) {
	if cmd.Process != nil {
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
}
func inspectExecutable(ctx context.Context, dir, flag string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, dir+"/llama-server", flag)
	cmd.Dir = dir
	cmd.Env = []string{"PATH=/usr/bin:/bin", "LD_LIBRARY_PATH=" + dir}
	return cmd.CombinedOutput()
}
