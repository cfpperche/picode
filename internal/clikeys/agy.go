package clikeys

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/cfpperche/picode/internal/clisettings"
)

// AgyMap declares Antigravity's key map: a flat JSON file whose rows are the
// vendor's override layer — removing one returns that action to the binding
// built into the binary, so a reset is "hand the key back", never a hole.
var AgyMap = FlatMap{
	CLI:     "agy",
	Catalog: AgyCatalog,
	File:    agyKeyMapFile,
	Normalize: func(chord string) (string, error) {
		// The pane captures `escape`; the file spells it `esc`. Everything else
		// the pane produces (single characters, `ctrl+k`, `shift+enter`, `f4`)
		// is already the file's own vocabulary — `pgup`/`pgdown` cannot be
		// captured in a browser tab at all, so they only ever arrive read back
		// from the file, which passes through unchanged.
		return strings.Replace(chord, "escape", "esc", 1), nil
	},
}

// agyKeyMapFile is `~/.gemini/antigravity-cli/keybindings.json`: one override
// file, no environment override and no second layer (read 2026-09-21).
func agyKeyMapFile() (string, clisettings.Format, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", "", err
	}
	return filepath.Join(home, ".gemini", "antigravity-cli", "keybindings.json"), clisettings.FormatJSON, nil
}
