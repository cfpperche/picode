// Package install enables PiCode as a systemd user service (ADR-0018).
package install

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// Run is systemctl (and friends). Tests replace it.
var Run = func(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	// Fill in the session variables when this process was not started from a
	// login shell; without them `systemctl --user` cannot find its own manager.
	cmd.Env = append(os.Environ(), sessionEnv(os.Getenv, os.Getuid(), pathExists)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Paths for a user home.
type Paths struct {
	Home string
	Bin  string
	Unit string
	Lock string
	Data string
}

func ForHome(home string) Paths {
	return Paths{
		Home: home,
		Bin:  filepath.Join(home, ".local", "bin", "picode"),
		Unit: filepath.Join(home, ".config", "systemd", "user", UnitName),
		Data: filepath.Join(home, ".picode"),
		Lock: filepath.Join(home, ".picode", "picode.lock"),
	}
}

func withLocalBin(path, binDir string) string {
	if path == "" {
		return binDir + ":/usr/bin:/bin"
	}
	if strings.Contains(path, binDir) {
		return path
	}
	return binDir + ":" + path
}

// CopyExe writes src to dest (0755). Same path is a no-op.
func CopyExe(src, dest string) error {
	src, err := filepath.EvalSymlinks(src)
	if err != nil {
		src, err = filepath.Abs(src)
		if err != nil {
			return err
		}
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	if same, _ := sameFile(src, dest); same {
		return nil
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	tmp := dest + ".tmp"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, dest)
}

func sameFile(a, b string) (bool, error) {
	sa, err := os.Stat(a)
	if err != nil {
		return false, err
	}
	sb, err := os.Stat(b)
	if err != nil {
		return false, err
	}
	return os.SameFile(sa, sb), nil
}

func writeUnit(p Paths, pathEnv string) error {
	if err := os.MkdirAll(filepath.Dir(p.Unit), 0o755); err != nil {
		return err
	}
	body := UnitFile(p.Bin, pathEnv, p.Home)
	return os.WriteFile(p.Unit, []byte(body), 0o644)
}

func lockPID(path string) int {
	b, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	n, err := strconv.Atoi(strings.TrimSpace(string(b)))
	if err != nil || n <= 0 {
		return 0
	}
	return n
}

func systemdAvailable() bool {
	_, err := exec.LookPath("systemctl")
	if err != nil {
		return false
	}
	if _, err := os.Stat("/run/systemd/system"); err != nil {
		return false
	}
	return true
}
