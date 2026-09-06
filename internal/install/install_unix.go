//go:build unix

package install

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/cfpperche/picode/internal/version"
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
	AppendDeployRecord(p.Data)
	return nil
}

// deployRecord is one line of <data>/var/deploy-log.jsonl (ADR-0085): who
// deployed is the first question every session-loss post-mortem asks.
type deployRecord struct {
	At      string `json:"at"`
	Version string `json:"version"`
	TermID  string `json:"termId,omitempty"`
	Cwd     string `json:"cwd,omitempty"`
}

// AppendDeployRecord appends the current deploy to the deploy log. Best
// effort — a failed append never fails the deploy. The terminal id rides
// the environment when the deploy ran inside a PiCode terminal, which is
// exactly the common case (agents deploying from their own panes).
func AppendDeployRecord(dataDir string) {
	if dataDir == "" {
		return
	}
	rec := deployRecord{
		At:      time.Now().UTC().Format(time.RFC3339),
		Version: version.Build(),
		TermID:  os.Getenv("PICODE_TERM_ID"),
	}
	if cwd, err := os.Getwd(); err == nil {
		rec.Cwd = cwd
	}
	raw, err := json.Marshal(rec)
	if err != nil {
		return
	}
	dir := filepath.Join(dataDir, "var")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return
	}
	f, err := os.OpenFile(filepath.Join(dir, "deploy-log.jsonl"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.Write(append(raw, '\n'))
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
