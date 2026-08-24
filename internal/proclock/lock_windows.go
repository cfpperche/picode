//go:build windows

package proclock

import (
	"fmt"
	"os"
)

// Acquire on Windows uses exclusive file create. Best-effort: a crashed
// process may leave the file; delete and retry once.
func Acquire(path string) (func(), error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o600)
	if err != nil {
		_ = os.Remove(path)
		f, err = os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o600)
		if err != nil {
			return nil, fmt.Errorf("another picode is already running (lock %s)", path)
		}
	}
	_, _ = f.WriteString(fmt.Sprintf("%d\n", os.Getpid()))
	return func() {
		_ = f.Close()
		_ = os.Remove(path)
	}, nil
}
