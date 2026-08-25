//go:build unix

package binwatch

import (
	"log"
	"os"
	"syscall"
)

// Reexec replaces this process with exe. Call after releasing the lock.
func Reexec(exe string) {
	err := syscall.Exec(exe, os.Args, os.Environ())
	log.Fatalf("picode: reexec %s: %v", exe, err)
}
