package pisettings

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// TrustFile is ~/.pi/agent/trust.json (path → bool).
func TrustFile() string {
	return filepath.Join(filepath.Dir(UserFile()), "trust.json")
}

// Trusted reports whether cwd (or a parent) is true in trust.json.
func Trusted(cwd string) bool {
	return trustedAt(cwd, TrustFile())
}

// Set writes cwd → allow in trust.json (read-modify-write). Other keys stay.
func Set(cwd string, allow bool) error {
	return setAt(cwd, allow, TrustFile())
}

func setAt(cwd string, allow bool, trustPath string) error {
	cwd = filepath.Clean(cwd)
	if cwd == "" || cwd == "." {
		return os.ErrInvalid
	}
	m := map[string]bool{}
	if raw, err := os.ReadFile(trustPath); err == nil {
		_ = json.Unmarshal(raw, &m)
	} else if !os.IsNotExist(err) {
		return err
	}
	m[cwd] = allow
	raw, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(trustPath), 0o700); err != nil {
		return err
	}
	return os.WriteFile(trustPath, append(raw, '\n'), 0o600)
}

func trustedAt(cwd, trustPath string) bool {
	cwd = filepath.Clean(cwd)
	if cwd == "" || cwd == "." {
		return false
	}
	raw, err := os.ReadFile(trustPath)
	if err != nil {
		return false
	}
	var m map[string]bool
	if err := json.Unmarshal(raw, &m); err != nil {
		return false
	}
	clean := map[string]bool{}
	for k, v := range m {
		clean[filepath.Clean(k)] = v
	}
	for dir := cwd; ; {
		if v, ok := clean[dir]; ok {
			return v
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return false
		}
		dir = parent
	}
}
