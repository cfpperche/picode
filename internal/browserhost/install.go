package browserhost

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
)

// Manifest is the JSON file Chrome reads to launch the native host.
type Manifest struct {
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	Path           string   `json:"path"`
	Type           string   `json:"type"`
	AllowedOrigins []string `json:"allowed_origins"`
}

// NewManifest points Chrome at exePath (must be absolute).
func NewManifest(exePath string) Manifest {
	return Manifest{
		Name:           HostName,
		Description:    "PiCode browser host",
		Path:           exePath,
		Type:           "stdio",
		AllowedOrigins: []string{ExtensionOrigin},
	}
}

// ChromeHostDir is where Chrome looks for native-host manifests on this OS.
// Windows uses the registry instead; this is the Linux/macOS path.
func ChromeHostDir(home string) string {
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "Google", "Chrome", "NativeMessagingHosts")
	default:
		return filepath.Join(home, ".config", "google-chrome", "NativeMessagingHosts")
	}
}

// ChromiumHostDir is the Chromium (not Google Chrome) equivalent, written
// too so Linux dogfood against chromium still finds the host.
func ChromiumHostDir(home string) string {
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "Chromium", "NativeMessagingHosts")
	default:
		return filepath.Join(home, ".config", "chromium", "NativeMessagingHosts")
	}
}

// WriteHostManifest writes the Chrome (and Chromium, when that config dir
// already exists) native-host file. Returns the Chrome path.
func WriteHostManifest(home, exePath string) (string, error) {
	if home == "" {
		return "", fmt.Errorf("browserhost: no home directory")
	}
	if !filepath.IsAbs(exePath) {
		return "", fmt.Errorf("browserhost: host path must be absolute")
	}
	raw, err := json.MarshalIndent(NewManifest(exePath), "", "  ")
	if err != nil {
		return "", err
	}
	raw = append(raw, '\n')

	chromeDir := ChromeHostDir(home)
	if err := os.MkdirAll(chromeDir, 0o755); err != nil {
		return "", err
	}
	chromePath := filepath.Join(chromeDir, HostName+".json")
	if err := os.WriteFile(chromePath, raw, 0o644); err != nil {
		return "", err
	}

	// Only drop a Chromium copy when the user already has that browser —
	// do not create ~/.config/chromium just for us.
	chromiumRoot := filepath.Dir(ChromiumHostDir(home))
	if st, err := os.Stat(chromiumRoot); err == nil && st.IsDir() {
		dir := ChromiumHostDir(home)
		if err := os.MkdirAll(dir, 0o755); err == nil {
			_ = os.WriteFile(filepath.Join(dir, HostName+".json"), raw, 0o644)
		}
	}
	return chromePath, nil
}

// WindowsRegistryAddArgs registers the native host for the current user.
// Chrome on Windows does not read NativeMessagingHosts from the profile.
func WindowsRegistryAddArgs(manifestPath string) []string {
	return []string{
		"add", `HKCU\Software\Google\Chrome\NativeMessagingHosts\` + HostName,
		"/ve", "/t", "REG_SZ", "/d", manifestPath, "/f",
	}
}

// WindowsRegistryDeleteArgs undoes WindowsRegistryAddArgs.
func WindowsRegistryDeleteArgs() []string {
	return []string{
		"delete", `HKCU\Software\Google\Chrome\NativeMessagingHosts\` + HostName,
		"/f",
	}
}

// WindowsHostPath is the console host Chrome should launch. desktopExe is
// picode-desktop.exe; the nmh sibling must sit next to it.
func WindowsHostPath(desktopExe string) (string, error) {
	if desktopExe == "" {
		return "", fmt.Errorf("browserhost: no desktop executable")
	}
	dir := filepath.Dir(desktopExe)
	nmh := filepath.Join(dir, WindowsHostExe)
	if st, err := os.Stat(nmh); err != nil || st.IsDir() {
		return "", fmt.Errorf("picode-nmh.exe is missing next to picode-desktop — run make desktop")
	}
	return nmh, nil
}

// CopyFile writes src onto dst (0755). Same path is a no-op.
func CopyFile(src, dst string) error {
	src, err := filepath.Abs(src)
	if err != nil {
		return err
	}
	dst, err = filepath.Abs(dst)
	if err != nil {
		return err
	}
	if src == dst {
		return nil
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

// RemoveHostManifest deletes the Chrome and Chromium host files if present.
func RemoveHostManifest(home string) error {
	var first error
	for _, dir := range []string{ChromeHostDir(home), ChromiumHostDir(home)} {
		p := filepath.Join(dir, HostName+".json")
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) && first == nil {
			first = err
		}
	}
	return first
}
