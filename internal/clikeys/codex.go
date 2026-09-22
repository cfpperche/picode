package clikeys

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/cfpperche/picode/internal/clisettings"
)

// CodexMap declares Codex's key map: one file, one table per context, and every
// row at `[tui.keymap.<context>.<action>]` — a *key* inside its context's table,
// not a table of its own (the plan's `[tui.keymap.<context>.<action>]` would be a
// TOML type error; corrected against the vendor's schema, 2026-09-21).
//
// The file is the same one the settings editor writes (ADR-0163): the two panes
// touch different keys of it, both through the settings package's document
// primitives, so each save re-reads and re-parses what the other left behind.
var CodexMap = FlatMap{
	CLI:     "codex",
	Catalog: CodexCatalog,
	File:    codexKeyMapFile,
	// A row's ID is `<context>.<action>`: the table is the context, the key
	// inside it is the vendor's own action name.
	Path: func(a Action) []string {
		return []string{"tui", "keymap", a.Group, strings.TrimPrefix(a.ID, a.Group+".")}
	},
	Normalize: codexChord,
}

// codexChord renders a chord the pane captured (`ctrl+alt+k`, `ctrl+pageup`) in
// the spelling codex's own parser accepts (`ctrl-alt-k`, `ctrl-page-up`) — the
// vocabulary read out of codex-rs/tui/src/keymap.rs, parse_keybinding.
//
// A chord it cannot express is refused by name rather than written: codex
// validates its keymap at startup and refuses to run on one it cannot parse, so
// a chord in the wrong spelling would leave the user with a CLI that does not
// start.
func codexChord(chord string) (string, error) {
	if chord == "" {
		return "", errors.New("an empty chord is not a binding")
	}
	if !strings.Contains(chord, "+") {
		// One stroke with no modifier. The pane's own name for a key comes
		// first (`escape` -> `esc`, `pageup` -> `page-up`); a chord already in
		// the file's spelling — a value the pane read back and is writing again
		// — is checked and kept as it is.
		if name, ok := codexKeyName(chord); ok {
			return name, nil
		}
		if codexSpells(chord) {
			return chord, nil
		}
		return "", fmt.Errorf("codex has no chord called %q", chord)
	}
	parts := strings.Split(chord, "+")
	key := parts[len(parts)-1]
	mods := parts[:len(parts)-1]
	for _, mod := range mods {
		switch mod {
		case "ctrl", "alt", "shift":
		default:
			return "", fmt.Errorf("codex has no %q modifier; its chords are ctrl, alt and shift", mod)
		}
	}
	name, ok := codexKeyName(key)
	if !ok {
		return "", fmt.Errorf("codex has no key called %q", key)
	}
	return strings.Join(append(append([]string{}, mods...), name), "-"), nil
}

// codexKeyName is the pane's name for a key in codex's own vocabulary. A single
// character passes through — codex takes any one-character key — except `-`,
// which is its separator and has a word of its own.
func codexKeyName(key string) (string, bool) {
	if name, ok := codexKeys[key]; ok {
		return name, true
	}
	if len([]rune(key)) == 1 && key != "-" && key != "+" && key != " " {
		return key, true
	}
	if isFunctionKey(key) {
		return key, true
	}
	return "", false
}

// codexSpells reports whether a chord is already written the way codex parses
// one: modifiers first, in its own order, then a key it knows.
func codexSpells(chord string) bool {
	parts := strings.Split(chord, "-")
	order := []string{"ctrl", "alt", "shift"}
	last, i := -1, 0
	for ; i < len(parts); i++ {
		at := indexOf(order, parts[i])
		if at < 0 {
			break
		}
		if at <= last {
			return false
		}
		last = at
	}
	if i == len(parts) {
		return false
	}
	key := strings.Join(parts[i:], "-")
	if _, ok := codexNames[key]; ok {
		return true
	}
	return len([]rune(key)) == 1 || isFunctionKey(key)
}

func indexOf(list []string, want string) int {
	for i, item := range list {
		if item == want {
			return i
		}
	}
	return -1
}

// isFunctionKey reports codex's f1..f24 range (MAX_FUNCTION_KEY in its parser).
func isFunctionKey(key string) bool {
	if len(key) < 2 || key[0] != 'f' {
		return false
	}
	n, err := strconv.Atoi(key[1:])
	return err == nil && n >= 1 && n <= 24
}

// codexKeys is the pane's key names in codex's vocabulary, read out of
// parse_keybinding (2026-09-21).
var codexKeys = map[string]string{
	"escape": "esc", "return": "enter", "enter": "enter", "tab": "tab",
	"backspace": "backspace", "delete": "delete", "space": "space", " ": "space",
	"up": "up", "down": "down", "left": "left", "right": "right",
	"home": "home", "end": "end", "pageup": "page-up", "pagedown": "page-down",
	"-": "minus",
}

// codexNames is codex's own spelling of a key, for a chord that is already
// written the way the file wants it.
var codexNames = map[string]bool{
	"esc": true, "enter": true, "tab": true, "backspace": true, "delete": true,
	"space": true, "up": true, "down": true, "left": true, "right": true,
	"home": true, "end": true, "page-up": true, "page-down": true, "minus": true,
}

// codexKeyMapFile is `~/.codex/config.toml`. Codex resolves its config from the
// home directory only — the installed build (0.155.1) has no environment
// override, unlike omp's PI_CONFIG_DIR — and this is the same file the settings
// pane edits.
func codexKeyMapFile() (string, clisettings.Format, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", "", err
	}
	return filepath.Join(home, ".codex", "config.toml"), clisettings.FormatTOML, nil
}
