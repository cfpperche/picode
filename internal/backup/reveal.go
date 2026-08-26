package backup

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// WSLToWin maps "/mnt/c/Users/x" to `C:\Users\x`.
func WSLToWin(p string) (string, bool) {
	slash := filepath.ToSlash(p)
	const prefix = "/mnt/"
	if !strings.HasPrefix(slash, prefix) || len(slash) < 6 {
		return "", false
	}
	rest := slash[len(prefix):]
	if rest[0] < 'a' || rest[0] > 'z' {
		return "", false
	}
	if len(rest) > 1 && rest[1] != '/' {
		return "", false
	}
	drive := strings.ToUpper(rest[:1]) + ":"
	if len(rest) == 1 {
		return drive + `\`, true
	}
	return drive + strings.ReplaceAll(rest[1:], "/", `\`), true
}

func runningWSL() bool {
	if os.Getenv("WSL_DISTRO_NAME") != "" || os.Getenv("WSL_INTEROP") != "" {
		return true
	}
	b, err := os.ReadFile("/proc/version")
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(string(b)), "microsoft")
}

// Reveal opens path in the OS file manager (Explorer on Windows/WSL).
func Reveal(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("backup: no path")
	}
	st, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !st.IsDir() {
		path = filepath.Dir(path)
	}
	if runningWSL() {
		if win, ok := WSLToWin(path); ok {
			return exec.Command("explorer.exe", win).Start()
		}
	}
	switch runtime.GOOS {
	case "windows":
		return exec.Command("explorer.exe", path).Start()
	case "darwin":
		return exec.Command("open", path).Start()
	default:
		return exec.Command("xdg-open", path).Start()
	}
}
