//go:build !windows

package main

import "fmt"

// The notification area is a Windows concept. On Linux, PiCode already starts
// with the distro through systemd — there is nothing for a tray to add.
func runTray(string, string) error {
	return fmt.Errorf("the tray is Windows-only; on Linux use `picode provision`")
}
