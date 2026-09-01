package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cfpperche/picode/internal/browserhost"
	"github.com/cfpperche/picode/internal/desktop"
)

func runBrowserHost(distro, user string) error {
	client := browserhost.NewClient()
	client.Resolve = func() (string, error) {
		a, err := resolve(distro, user)
		if err != nil {
			return "", err
		}
		return desktop.ServerURL(a.runner, a.distro, a.user)
	}
	return browserhost.Serve(os.Stdin, os.Stdout, client.Handle)
}

func runExtensionInstall() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	exe, err = filepath.Abs(exe)
	if err != nil {
		return err
	}
	nmh, err := browserhost.WindowsHostPath(exe)
	if err != nil {
		return err
	}
	dir := windowsManifestDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	dest := filepath.Join(dir, browserhost.WindowsHostExe)
	if err := browserhost.CopyFile(nmh, dest); err != nil {
		return err
	}
	path := filepath.Join(dir, browserhost.HostName+".json")
	raw, err := json.MarshalIndent(browserhost.NewManifest(dest), "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, append(raw, '\n'), 0o644); err != nil {
		return err
	}
	if err := (osRunner{}).Run("reg", browserhost.WindowsRegistryAddArgs(path)...); err != nil {
		return fmt.Errorf("register Chrome native host: %w (manifest at %s)", err, path)
	}
	fmt.Println("Chrome native host installed:")
	fmt.Println("  " + path)
	fmt.Println("Load the unpacked extension from ext/ in the PiCode repo (inside WSL).")
	return nil
}

func runExtensionUninstall() error {
	_ = (osRunner{}).Run("reg", browserhost.WindowsRegistryDeleteArgs()...)
	dir := windowsManifestDir()
	_ = os.Remove(filepath.Join(dir, browserhost.HostName+".json"))
	fmt.Println("Chrome native host removed.")
	return nil
}

func windowsManifestDir() string {
	if d := os.Getenv("LOCALAPPDATA"); d != "" {
		return filepath.Join(d, "PiCode")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".picode")
}
