//go:build windows

package install

import "fmt"

func Install(exe, home, pathEnv string) error {
	return fmt.Errorf("picode install uses systemd (Linux / WSL)")
}

func Uninstall(home string, purge bool) error {
	return fmt.Errorf("picode install uses systemd (Linux / WSL)")
}

func Deploy(exe, home, pathEnv string) error {
	return fmt.Errorf("picode deploy uses systemd (Linux / WSL)")
}
