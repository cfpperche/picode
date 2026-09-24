// Package pimission supplies Pi's native Missions tool without a workspace install.
package pimission

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

//go:embed extension.ts
var extension []byte

// Path returns the stable private path used by Pi's -e launch argument.
func Path(dataDir string) string { return filepath.Join(dataDir, "intercept", "pi-mission.ts") }

// Ensure writes the embedded extension into PiCode's private data directory.
func Ensure(dataDir string) (string, error) {
	dataDir = strings.TrimSpace(dataDir)
	if dataDir == "" {
		home, err := os.UserHomeDir()
		if err != nil || home == "" {
			return "", fmt.Errorf("PiCode data directory is unavailable")
		}
		dataDir = filepath.Join(home, ".picode")
	}
	dir := filepath.Join(dataDir, "intercept")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	path := Path(dataDir)
	if err := os.WriteFile(path, extension, 0o600); err != nil {
		return "", err
	}
	return path, nil
}
