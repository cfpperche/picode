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
