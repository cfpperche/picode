package server

import (
	"errors"
	"os"
	"syscall"
)

// The recorder's exclusive lock covers both publications. A shared, nonblocking
// read distinguishes an in-flight write from permanent corruption without
// waiting in an API handler or mistaking truncated writer bytes for a failure.
func openNativeObservationFence(path string) (*os.File, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Mode().Perm() != 0600 || info.Size() > 16*1024 {
		return nil, errNativeObservationBlocked
	}
	f, err := os.OpenFile(path, os.O_RDWR|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, errNativeObservationBlocked
	}
	info, err = f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0600 || info.Size() > 16*1024 {
		f.Close()
		return nil, errNativeObservationBlocked
	}
	if err = syscall.Flock(int(f.Fd()), syscall.LOCK_SH|syscall.LOCK_NB); err != nil {
		f.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, errNativeObservationUpdating
		}
		return nil, errNativeObservationBlocked
	}
	return f, nil
}
