//go:build windows

package binwatch

import (
	"log"
	"os"
	"os/exec"
)

// Reexec starts exe and exits. Windows cannot Exec over a locked binary.
func Reexec(exe string) {
	cmd := exec.Command(exe, os.Args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	if err := cmd.Start(); err != nil {
		log.Fatalf("picode: reexec %s: %v", exe, err)
	}
	os.Exit(0)
}
