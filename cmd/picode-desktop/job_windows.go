//go:build windows

package main

import (
	"fmt"
	"os/exec"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

// The keepalive is a wsl.exe, and killing that wsl.exe is what ends the
// `sleep infinity` inside the distro — verified by killing one and watching
// the Linux process go with it. So the child's lifetime has to be tied to the
// tray's, including the ways the tray can die without running any Go code:
// taskkill /F, a crash, the process being ended from Task Manager.
//
// A job object does exactly that. Windows terminates a job's processes when
// the last handle to the job closes, and the handle closes when the process
// holding it dies, however it dies. Deferred cleanup cannot make that promise;
// the kernel can.

var (
	jobOnce sync.Once
	jobH    windows.Handle
	jobErr  error
)

// job creates the process group on first use and keeps the handle open for the
// life of this process. Closing it early would kill the children immediately,
// so it is deliberately never closed.
func job() (windows.Handle, error) {
	jobOnce.Do(func() {
		h, err := windows.CreateJobObject(nil, nil)
		if err != nil {
			jobErr = fmt.Errorf("create job object: %w", err)
			return
		}
		info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{
			BasicLimitInformation: windows.JOBOBJECT_BASIC_LIMIT_INFORMATION{
				LimitFlags: windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE,
			},
		}
		if _, err := windows.SetInformationJobObject(
			h,
			windows.JobObjectExtendedLimitInformation,
			uintptr(unsafe.Pointer(&info)),
			uint32(unsafe.Sizeof(info)),
		); err != nil {
			_ = windows.CloseHandle(h)
			jobErr = fmt.Errorf("set kill-on-close: %w", err)
			return
		}
		jobH = h
	})
	return jobH, jobErr
}

// superviseChild puts an already-started child in that job.
func superviseChild(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return fmt.Errorf("child has not started")
	}
	h, err := job()
	if err != nil {
		return err
	}
	// os.Process keeps its handle unexported, so the child is reopened by pid.
	// Between Start and here the pid cannot have been reused: the process is
	// still running and this program holds a handle to it.
	proc, err := windows.OpenProcess(
		windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(cmd.Process.Pid))
	if err != nil {
		return fmt.Errorf("open child %d: %w", cmd.Process.Pid, err)
	}
	defer windows.CloseHandle(proc)

	if err := windows.AssignProcessToJobObject(h, proc); err != nil {
		return fmt.Errorf("assign child %d to the job: %w", cmd.Process.Pid, err)
	}
	return nil
}
