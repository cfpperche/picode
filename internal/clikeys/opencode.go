package clikeys

import (
	"os"
	"path/filepath"

	"github.com/cfpperche/picode/internal/clisettings"
)

// OpenCodeMap declares OpenCode's key map: a flat `keybinds` object of action →
// chord, in a TUI config file of its own.
var OpenCodeMap = FlatMap{
	CLI:     "opencode",
	Catalog: OpenCodeCatalog,
	File:    opencodeKeyMapFile,
}

// opencodeKeyMapFile resolves the user-scope TUI config: `tui.jsonc` if the user
// has one (a .jsonc file wins over a .json in the same directory, and comments
// are legal there), else `tui.json`, which PiCode creates on first write.
// Never the main opencode.json: writing keybinds there is the legacy shape, and
// the CLI's own startup migration would move them into a tui.json behind
// PiCode's back (packages/opencode/src/config/tui-migrate.ts, v1.18.32).
//
// Scope note: opencode deep-merges global → project files per action, so a
// project's tui.json overrides the user's global rows. PiCode edits the user
// file only; the pane names it, and the registry's Source says a project file
// wins (docs/plans/keyboard-pane.md, P4).
func opencodeKeyMapFile() (string, clisettings.Format, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", "", err
	}
	dir := filepath.Join(home, ".config", "opencode")
	if _, err := os.Stat(filepath.Join(dir, "tui.jsonc")); err == nil {
		return filepath.Join(dir, "tui.jsonc"), clisettings.FormatJSONC, nil
	}
	return filepath.Join(dir, "tui.json"), clisettings.FormatJSON, nil
}
