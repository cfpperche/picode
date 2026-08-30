//go:build unix

package install

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// Install copies the current binary, writes the user unit, and starts it.
func Install(exe, home, pathEnv string) error {
	if !systemdAvailable() {
		return fmt.Errorf("need systemd (user). In WSL set systemd=true in /etc/wsl.conf")
	}
	if home == "" {
		h, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		home = h
	}
	p := ForHome(home)
	// Before touching anything: an install that copies and then cannot enable
	// the unit is worse than one that never started.
	if err := EnsureUserSession(); err != nil {
		return err
	}
	pathEnv = withLocalBin(pathEnv, filepath.Dir(p.Bin))
	if err := CopyExe(exe, p.Bin); err != nil {
		return fmt.Errorf("copy binary: %w", err)
	}
	if err := writeUnit(p, pathEnv); err != nil {
		return fmt.Errorf("write unit: %w", err)
	}
	stopStray(p)
	if err := Run("systemctl", "--user", "daemon-reload"); err != nil {
		return fmt.Errorf("systemctl daemon-reload: %w", err)
	}
	if err := Run("systemctl", "--user", "enable", "--now", UnitName); err != nil {
		return fmt.Errorf("systemctl enable: %w", err)
	}
	return nil
}

// Deploy copies this binary over the installed one and restarts the unit.
func Deploy(exe, home, pathEnv string) error {
	if !systemdAvailable() {
		return fmt.Errorf("need systemd (user). In WSL set systemd=true in /etc/wsl.conf")
	}
	if home == "" {
		h, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		home = h
	}
	p := ForHome(home)
	if _, err := os.Stat(p.Unit); err != nil {
		return fmt.Errorf("not installed — run picode install first")
	}
	// Same reason as Install: copying the new binary and failing to restart
	// leaves the old one running and looks like a successful deploy.
	if err := EnsureUserSession(); err != nil {
		return err
	}
	pathEnv = withLocalBin(pathEnv, filepath.Dir(p.Bin))
	if err := CopyExe(exe, p.Bin); err != nil {
		return fmt.Errorf("copy binary: %w", err)
	}
	if err := writeUnit(p, pathEnv); err != nil {
		return fmt.Errorf("write unit: %w", err)
	}
	if err := Run("systemctl", "--user", "daemon-reload"); err != nil {
		return fmt.Errorf("systemctl daemon-reload: %w", err)
	}
	if err := Run("systemctl", "--user", "restart", UnitName); err != nil {
		return fmt.Errorf("systemctl restart: %w", err)
	}
	return nil
}

// Uninstall stops the unit and removes it. purge deletes ~/.picode.
func Uninstall(home string, purge bool) error {
	if home == "" {
		h, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		home = h
	}
	p := ForHome(home)
	if systemdAvailable() {
		_ = Run("systemctl", "--user", "disable", "--now", UnitName)
		_ = Run("systemctl", "--user", "daemon-reload")
	}
	_ = os.Remove(p.Unit)
	_ = os.Remove(p.Bin)
	if purge {
		if err := os.RemoveAll(p.Data); err != nil {
			return fmt.Errorf("purge data: %w", err)
		}
	}
	return nil
}

func stopStray(p Paths) {
	if systemdAvailable() {
		_ = Run("systemctl", "--user", "stop", UnitName)
	}
	pid := lockPID(p.Lock)
	if pid <= 0 || pid == os.Getpid() {
		return
	}
	_ = syscall.Kill(pid, syscall.SIGTERM)
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if err := syscall.Kill(pid, 0); err != nil {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	_ = syscall.Kill(pid, syscall.SIGKILL)
}
