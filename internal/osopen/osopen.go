// Package osopen opens paths and URLs in the host operating system's own
// UI — the file manager or the default browser. It knows the WSL bridge
// (explorer.exe for folders, PowerShell Start-Process for links, the URL
// handed over through an environment variable so no shell ever parses it);
// plain Linux, macOS and Windows are one exec each.
package osopen

import (
	"fmt"
	"net/url"
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

func RunningWSL() bool {
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
		return fmt.Errorf("osopen: no path")
	}
	st, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !st.IsDir() {
		path = filepath.Dir(path)
	}
	if RunningWSL() {
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

// ValidOpenURL applies the same allowlist the desktop shell applies before
// handing a target to the OS (desktop-shell/src/external.rs): a trimmed
// http(s) URL, bounded length, no whitespace or characters that could read
// as a second command. '&' stays legal — a link is one argv element here,
// and OAuth login URLs carry query strings.
func ValidOpenURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || len(raw) > 2048 {
		return "", fmt.Errorf("osopen: no openable URL")
	}
	if strings.IndexFunc(raw, func(r rune) bool {
		return r <= 0x20 || r == 0x7f || strings.ContainsRune("\"'`<>^", r)
	}) >= 0 {
		return "", fmt.Errorf("osopen: unsafe character in URL")
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "", fmt.Errorf("osopen: only http(s) URLs open")
	}
	return raw, nil
}

// windowsPowerShell finds the bridge Start-Process runs on from WSL.
func windowsPowerShell() string {
	if p, err := exec.LookPath("powershell.exe"); err == nil {
		return p
	}
	for _, p := range []string{
		"/mnt/c/Windows/System32/WindowsPowerShell/v1.0/powershell.exe",
	} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// OpenURL opens url in the host's default browser:
// WSL → the Windows browser via Start-Process (the URL travels in an
// environment variable: Powershell reads $Env:PICODE_OPEN_URL, so the link
// is never parsed as a command line), macOS → open, Linux → xdg-open,
// Windows → rundll32's file-protocol handler. All forms pass the URL as
// one argv element; no shell sees it.
func OpenURL(raw string) error {
	url, err := ValidOpenURL(raw)
	if err != nil {
		return err
	}
	if RunningWSL() {
		exe := windowsPowerShell()
		if exe == "" {
			return fmt.Errorf("can't find powershell.exe")
		}
		cmd := exec.Command(exe, "-NoProfile", "-NonInteractive", "-Command", "Start-Process -FilePath $Env:PICODE_OPEN_URL")
		cmd.Env = append(os.Environ(), "PICODE_OPEN_URL="+url)
		return cmd.Start()
	}
	switch runtime.GOOS {
	case "windows":
		return exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		return exec.Command("open", url).Start()
	default:
		return exec.Command("xdg-open", url).Start()
	}
}
