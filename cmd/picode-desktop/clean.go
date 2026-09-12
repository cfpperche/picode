package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/cfpperche/picode/internal/desktop"
)

// runClean passes `picode clean` through to the distro, the same way disk
// measurement asks the binary inside. The in-distro command owns the rules
// (its table, its refusals); this side only carries the bytes, so progress
// lines and the outcome reach the shell untouched. Stdout is wired straight
// through: nothing here buffers a minutes-long run.
func runClean(distroFlag, userFlag, apply string, listOnly, yes bool) error {
	a, err := resolve(distroFlag, userFlag)
	if err != nil {
		return err
	}
	exe, err := desktop.PicodePath(a.runner, a.distro, a.user)
	if err != nil {
		return fmt.Errorf("picode is not installed in %s: %w", a.distro, err)
	}

	args := []string{"clean", "--json"}
	switch {
	case listOnly:
		args = append(args, "--list")
	case apply != "":
		args = append(args, "--apply", apply)
		if yes {
			args = append(args, "--yes")
		}
	default:
		args = append(args, "--list")
	}

	argv := append([]string{exe}, args...)
	cmd := exec.Command(desktop.WSLExe, desktop.WSLArgs(a.distro, a.user, argv...)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
