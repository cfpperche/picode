//go:build linux

package llamaservice

import (
	"context"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

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

func closeOnExec(fd int) { syscall.CloseOnExec(fd) }

// clearOrphanedGroup kills what is left of the router's process group after
// its supervisor exited (ADR-0090 amendment 2026-09-25). The router got its
// parent-death signal; the per-model servers it started did not. The group is
// killed only while its leader is gone and members remain: a live process
// whose PID is the group's number means the number is someone else's now (or
// the router survived, which the supervisor's own kill would have prevented),
// so nothing is sent.
func clearOrphanedGroup(group int) {
	if group <= 1 {
		return
	}
	// The router's parent-death kill is asynchronous, and init reaps it a
	// moment later: wait for its PID to disappear before judging the group.
	deadline := time.Now().Add(3 * time.Second)
	for {
		if _, err := os.Stat("/proc/" + strconv.Itoa(group)); err != nil {
			break
		}
		if time.Now().After(deadline) {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	if len(groupMembers(group)) == 0 {
		return
	}
	_ = syscall.Kill(-group, syscall.SIGKILL)
}

// groupMembers lists the live processes whose process group is group, read
// from /proc/<pid>/stat (field 5, after the parenthesized command name).
func groupMembers(group int) []int {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil
	}
	var out []int
	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}
		raw, err := os.ReadFile("/proc/" + e.Name() + "/stat")
		if err != nil {
			continue
		}
		stat := string(raw)
		end := strings.LastIndexByte(stat, ')')
		if end < 0 {
			continue
		}
		fields := strings.Fields(stat[end+1:])
		// fields[0] state, [1] ppid, [2] pgrp
		if len(fields) > 2 && fields[0] != "Z" && fields[2] == strconv.Itoa(group) {
			out = append(out, pid)
		}
	}
	return out
}
