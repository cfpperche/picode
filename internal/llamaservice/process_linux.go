//go:build linux

package llamaservice

import (
	"context"
	"os"
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

// changeTime is the inode's ctime: any write or metadata change moves it, and
// no user can set it back (hash_memo.go).
func changeTime(info os.FileInfo) int64 {
	if st, ok := info.Sys().(*syscall.Stat_t); ok {
		return st.Ctim.Nano()
	}
	return 0
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
