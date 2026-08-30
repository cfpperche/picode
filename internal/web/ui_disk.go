//go:build !embedui

package web

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// DirEnv overrides where the UI is read from. `make dev` runs from the repo
// root, where the default is already right.
const DirEnv = "PICODE_UI_DIR"

const defaultDir = "internal/web/public"

// UI reads the UI from disk. Nothing is embedded in this build, so the binary
// is not portable — `make build` is the contract that produces one that is.
func UI() fs.FS {
	return os.DirFS(Dir())
}

// Dir is the directory UI reads from.
func Dir() string {
	if d := strings.TrimSpace(os.Getenv(DirEnv)); d != "" {
		return d
	}
	return filepath.FromSlash(defaultDir)
}

// Embedded reports how this binary was built, so the server can explain
// itself when the UI is missing.
func Embedded() bool { return false }
