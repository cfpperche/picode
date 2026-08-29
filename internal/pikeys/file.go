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
	if keys == nil {
		delete(doc, action)
	} else {
		doc[action] = keys
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
