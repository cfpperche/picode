// Package pikeys reads and patches ~/.pi/agent/keybindings.json (Pi's file).
package pikeys

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/cfpperche/picode/internal/pipkg"
)

// File is ~/.pi/agent/keybindings.json.
func File() string {
	dir := pipkg.UserDir()
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, "keybindings.json")
}

// LoadUser returns user overrides. Missing file is empty, not an error.
func LoadUser() (map[string][]string, error) {
	path := File()
	if path == "" {
		return map[string][]string{}, nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string][]string{}, nil
		}
		return nil, err
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return map[string][]string{}, nil
	}
	doc := map[string]any{}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("keybindings: %w", err)
	}
	out := map[string][]string{}
	for k, v := range doc {
		if keys, ok := asKeys(v); ok {
			out[k] = keys
		}
	}
	return out, nil
}

// Set writes one action. keys == nil deletes the override (Pi default).
// Empty keys writes [] (action off). Unknown keys in the file stay.
func Set(action string, keys []string) error {
	if !known(action) {
		return fmt.Errorf("unknown action")
	}
	if keys != nil {
		for _, k := range keys {
			if !ValidKey(k) {
				return fmt.Errorf("bad key %q", k)
			}
		}
	}
	return patchDoc(func(doc map[string]any) bool {
		if keys == nil {
			delete(doc, action)
		} else {
			doc[action] = keys
		}
		return true
	})
}

// ResetKnown removes every override whose action this catalog knows, in one
// write, and returns how many it removed. A key the file carries but the
// catalog does not stays — the same rule Set keeps for one action, so the
// pane's "Reset all" cannot throw away a binding written by a pi version
// PiCode has not read yet. A no-op writes nothing (no file is created just to
// hold an empty object).
func ResetKnown() (int, error) {
	removed := 0
	err := patchDoc(func(doc map[string]any) bool {
		for _, a := range Catalog {
			if _, ok := doc[a.ID]; ok {
				delete(doc, a.ID)
				removed++
			}
		}
		return removed > 0
	})
	if err != nil {
		return 0, err
	}
	return removed, nil
}

// patchDoc is the one place a keybindings write happens (Set, ResetKnown):
// read the file, let edit change the decoded object, write it back whole.
func patchDoc(edit func(map[string]any) bool) error {
	path := File()
	if path == "" {
		return fmt.Errorf("no home directory")
	}
	doc := map[string]any{}
	raw, err := os.ReadFile(path)
	if err == nil && len(strings.TrimSpace(string(raw))) > 0 {
		if err := json.Unmarshal(raw, &doc); err != nil {
			return fmt.Errorf("keybindings: %w", err)
		}
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	if !edit(doc) {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(out, '\n'), 0o644)
}

func asKeys(v any) ([]string, bool) {
	switch t := v.(type) {
	case string:
		return []string{t}, true
	case []any:
		out := make([]string, 0, len(t))
		for _, x := range t {
			s, ok := x.(string)
			if !ok {
				return nil, false
			}
			out = append(out, s)
		}
		return out, true
	default:
		return nil, false
	}
}

var mods = map[string]bool{"ctrl": true, "shift": true, "alt": true, "super": true}

// ValidKey is modifier+key in Pi's format (ctrl+backspace, pageUp, ...).
func ValidKey(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" || strings.ContainsAny(s, " \t\n") {
		return false
	}
	parts := strings.Split(s, "+")
	if len(parts) == 0 {
		return false
	}
	for i, p := range parts {
		if p == "" {
			return false
		}
		if i < len(parts)-1 {
			if !mods[strings.ToLower(p)] {
				return false
			}
			continue
		}
		for _, r := range p {
			if r > unicode.MaxASCII {
				return false
			}
		}
	}
	return true
}
