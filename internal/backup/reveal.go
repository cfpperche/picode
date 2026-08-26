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

func windowsExplorer() string {
	if p, err := exec.LookPath("explorer.exe"); err == nil {
		return p
	}
	for _, p := range []string{
		"/mnt/c/Windows/explorer.exe",
		"/mnt/c/WINDOWS/explorer.exe",
	} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

func wslUNC(linuxPath string) string {
	distro := os.Getenv("WSL_DISTRO_NAME")
	if distro == "" {
		distro = "Ubuntu"
	}
	p := strings.TrimPrefix(filepath.ToSlash(linuxPath), "/")
	return `\\wsl.localhost\` + distro + `\` + strings.ReplaceAll(p, "/", `\`)
}

// Reveal opens path in the host file manager:
// WSL → Windows Explorer, macOS → Finder, Linux → xdg-open.
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
		exe := windowsExplorer()
		if exe == "" {
			return fmt.Errorf("can't find Windows Explorer")
		}
		target, ok := WSLToWin(path)
		if !ok {
			target = wslUNC(path)
		}
		return exec.Command(exe, target).Start()
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
