//go:build linux

package llamaservice

import (
	"context"
	"os/exec"
	"syscall"

	"golang.org/x/sys/unix"
)

func configureProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Pdeathsig: syscall.SIGKILL, Setpgid: true}
}
func killProcessGroup(cmd *exec.Cmd) {
	if cmd.Process != nil {
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
}
// waitExited returns once the process has exited, leaving it unreaped
// (WNOWAIT): its PID stays reserved until the returned reap collects it.
func waitExited(cmd *exec.Cmd) (reap func() error) {
	var info unix.Siginfo
	for {
		err := unix.Waitid(unix.P_PID, cmd.Process.Pid, &info, unix.WEXITED|unix.WNOWAIT, nil)
		if err != unix.EINTR {
			return cmd.Wait
		}
	}
}
func inspectExecutable(ctx context.Context, dir, flag string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, dir+"/llama-server", flag)
	cmd.Dir = dir
	cmd.Env = []string{"PATH=/usr/bin:/bin", "LD_LIBRARY_PATH=" + dir}
	return cmd.CombinedOutput()
}
