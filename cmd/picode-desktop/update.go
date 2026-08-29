package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/cfpperche/picode/internal/install"
	"github.com/cfpperche/picode/internal/version"
)

// DesktopAsset is this program's name on a GitHub release. It is a separate
// asset from picode's own, because the two are different binaries on
// different sides of the WSL boundary — but they ship in the same release, so
// their versions never disagree.
const DesktopAsset = "picode-desktop-windows-amd64.exe"

// runUpdate replaces this executable with a newer release. Windows will not
// let a running program be overwritten, so the old one is renamed aside first
// — the same move an updater has to make on every Windows app.
func runUpdate() error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("picode-desktop is a Windows program")
	}

	rel, err := install.LatestReleaseFor(DesktopAsset)
	if err != nil {
		return err
	}
	if rel.Tag == "" {
		return fmt.Errorf("no published release yet")
	}
	if !install.Newer(version.Version, rel.Tag) {
		fmt.Printf("Already up to date (%s).\n", version.Version)
		return nil
	}
	if rel.AssetURL == "" {
		return fmt.Errorf("release %s has no %s — download it from %s", rel.Tag, DesktopAsset, rel.URL)
	}

	fmt.Printf("Version %s is out (you have %s).\n", rel.Tag, version.Version)

	exe, err := os.Executable()
	if err != nil {
		return err
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return err
	}

	next := exe + ".new"
	if err := install.Download(rel.AssetURL, next); err != nil {
		return fmt.Errorf("download %s: %w", rel.Asset, err)
	}

	old := exe + ".old"
	_ = os.Remove(old)
	if err := os.Rename(exe, old); err != nil {
		_ = os.Remove(next)
		return fmt.Errorf("set the running program aside: %w", err)
	}
	if err := os.Rename(next, exe); err != nil {
		// Put it back rather than leaving the machine with no picode-desktop.
		_ = os.Rename(old, exe)
		_ = os.Remove(next)
		return fmt.Errorf("install the new version: %w", err)
	}
	// The renamed original stays until the next update: Windows still has it
	// open, so deleting it now would fail anyway.
	fmt.Printf("Updated to %s. Restart PiCode Desktop to run it.\n", rel.Tag)
	return nil
}
