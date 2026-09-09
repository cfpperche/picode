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

// DeployForce mirrors the unix signature so cmd/picode compiles on Windows
// (the CI contract: every package and test builds there); deploying still
// goes through systemd, so force changes nothing.
func DeployForce(exe, home, pathEnv string, force bool) error {
	return Deploy(exe, home, pathEnv)
}
