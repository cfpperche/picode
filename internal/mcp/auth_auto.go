package mcp

import (
	_ "embed"
	"os"
	"path/filepath"
)

//go:embed auth-auto.js
var authExt []byte

// EnsureAuthExt writes the headless OAuth -e script into dataDir.
func EnsureAuthExt(dataDir string) (string, error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dataDir, "mcp-auth-auto.js")
	if err := os.WriteFile(path, authExt, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// AdapterDir is the installed pi-mcp-adapter package (source of authenticate).
func AdapterDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".pi", "agent", "npm", "node_modules", "pi-mcp-adapter")
}

// AuthOutPath is the result file for one headless sign-in.
func AuthOutPath(dataDir, id string) string {
	if dataDir == "" || id == "" {
		return ""
	}
	return filepath.Join(dataDir, "mcp-auth", id+".json")
}
