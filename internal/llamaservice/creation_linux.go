//go:build linux

package llamaservice

import "golang.org/x/sys/unix"

// os.Rename can replace an empty directory belonging to someone else.
func publishCreation(from, to string) error {
	return unix.Renameat2(unix.AT_FDCWD, from, unix.AT_FDCWD, to, unix.RENAME_NOREPLACE)
}
