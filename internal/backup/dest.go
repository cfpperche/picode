package backup

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ValidateDest refuses empty paths and destinations inside the live data trees.
func ValidateDest(dest, dataDir, piHome string) error {
	dest = strings.TrimSpace(dest)
	if dest == "" {
		return fmt.Errorf("backup: choose a folder")
	}
	abs, err := filepath.Abs(dest)
	if err != nil {
		return fmt.Errorf("backup: dest: %w", err)
	}
	if inside(abs, dataDir) {
		return fmt.Errorf("backup: destination cannot be inside the PiCode data folder")
	}
	if piHome != "" && inside(abs, piHome) {
		return fmt.Errorf("backup: destination cannot be inside the pi folder")
	}
	return nil
}

func inside(path, root string) bool {
	path, root = canon(path), canon(root)
	if path == "" || root == "" {
		return false
	}
	if path == root {
		return true
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func canon(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return filepath.Clean(p)
	}

	// The destination normally does not exist yet, so EvalSymlinks(abs)
	// cannot resolve aliases in an existing ancestor. Resolve the longest
	// existing prefix instead, then put the missing tail back. This matters
	// on macOS, where t.TempDir commonly arrives below /var while the same
	// directory's canonical spelling starts with /private/var.
	cur := abs
	var tail []string
	for {
		if resolved, err := filepath.EvalSymlinks(cur); err == nil {
			for i := len(tail) - 1; i >= 0; i-- {
				resolved = filepath.Join(resolved, tail[i])
			}
			return filepath.Clean(resolved)
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return filepath.Clean(abs)
		}
		tail = append(tail, filepath.Base(cur))
		cur = parent
	}
}

func ensureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}
