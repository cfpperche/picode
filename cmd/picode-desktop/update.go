package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/cfpperche/picode/internal/desktop"
	"github.com/cfpperche/picode/internal/install"
	"github.com/cfpperche/picode/internal/version"
)

// DesktopAsset is this program's name on a GitHub release. It is a separate
// asset from picode's own, because the two are different binaries on
// different sides of the WSL boundary — but they ship in the same release, so
// their versions never disagree.
const DesktopAsset = "picode-desktop-windows-amd64.exe"

// ShellAsset is the resident's name on a GitHub release (ADR-0142): the tool
// and the shell ship in the same tag, and `update` refreshes both or
// neither, so the pair never disagrees about the version.
const ShellAsset = "picode-shell-windows-amd64.exe"

// runUpdate replaces this executable — and the shell resident beside it —
// with a newer release. Both downloads verify against SHA256SUMS before
// either is swapped: an unverified binary is never installed (the daemon's
// `picode update` refuses the same way).
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
	shellURL := rel.URLs[ShellAsset]
	if shellURL == "" {
		return fmt.Errorf("release %s has no %s — download it from %s", rel.Tag, ShellAsset, rel.URL)
	}
	if rel.SumsURL == "" {
		return fmt.Errorf("release %s has no %s — refusing to install an unverified binary", rel.Tag, install.SumsAsset)
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

	shell, err := desktop.ShellExe(exe, os.Getenv("LOCALAPPDATA"))
	if err != nil {
		// A machine without the shell is not migrated yet; installing the
		// resident is the retarget's job (ADR-0142), not the updater's
		// surprise — the tool updates alone.
		fmt.Println("No shell installed — updating the tool only.")
		shell = ""
	}

	sums, err := install.Fetch(rel.SumsURL)
	if err != nil {
		return fmt.Errorf("%s: %w", install.SumsAsset, err)
	}

	toolNext := exe + ".new"
	if err := install.Download(rel.AssetURL, toolNext); err != nil {
		return fmt.Errorf("download %s: %w", rel.Asset, err)
	}
	if err := install.VerifySHA256(toolNext, sums, rel.Asset); err != nil {
		_ = os.Remove(toolNext)
		return err
	}
	shellNext := ""
	if shell != "" {
		shellNext = shell + ".new"
		if err := install.Download(shellURL, shellNext); err != nil {
			_ = os.Remove(toolNext)
			return fmt.Errorf("download %s: %w", ShellAsset, err)
		}
		if err := install.VerifySHA256(shellNext, sums, ShellAsset); err != nil {
			_ = os.Remove(toolNext)
			_ = os.Remove(shellNext)
			return err
		}
	}

	if err := swapExe(exe, toolNext); err != nil {
		if shellNext != "" {
			_ = os.Remove(shellNext)
		}
		return err
	}
	if shell != "" {
		if err := swapExe(shell, shellNext); err != nil {
			// The pair updates together or not at all: the shell failed,
			// so the tool goes back rather than stranding a mixed pair.
			unswapExe(exe)
			return err
		}
		fmt.Printf("Updated to %s (tool and shell). Restart PiCode Desktop to run it.\n", rel.Tag)
		return nil
	}
	fmt.Printf("Updated to %s. Restart PiCode Desktop to run it.\n", rel.Tag)
	return nil
}

// swapExe moves next into exe's place. Windows will not let a running program
// be overwritten, so the old one is renamed aside first — the same move an
// updater has to make on every Windows app, running or not.
func swapExe(exe, next string) error {
	old := exe + ".old"
	_ = os.Remove(old)
	if err := os.Rename(exe, old); err != nil {
		_ = os.Remove(next)
		return fmt.Errorf("set %s aside: %w", filepath.Base(exe), err)
	}
	if err := os.Rename(next, exe); err != nil {
		// Put it back rather than leaving the machine with no binary at all.
		_ = os.Rename(old, exe)
		_ = os.Remove(next)
		return fmt.Errorf("install the new %s: %w", filepath.Base(exe), err)
	}
	// The renamed original stays until the next update: Windows still has it
	// open, so deleting it now would fail anyway.
	return nil
}

// unswapExe undoes a swap: the new file goes away and the set-aside original
// comes back. Only the two-binary update calls it, when the second swap fails.
func unswapExe(exe string) {
	_ = os.Remove(exe)
	_ = os.Rename(exe+".old", exe)
}
