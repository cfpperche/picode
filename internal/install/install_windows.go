//go:build windows

package install

import "fmt"

func Install(exe, home, pathEnv string) error {
	return fmt.Errorf("picode install uses systemd (Linux / WSL)")
}

func Uninstall(home string, purge bool) error {
	return fmt.Errorf("picode install uses systemd (Linux / WSL)")
}

func Update(exe, home, pathEnv string) error {
	return fmt.Errorf("picode update uses systemd (Linux / WSL)")
}
