package backup

import (
	"fmt"
	"os"
	"path/filepath"
)

// Restore copies a snapshot back over the live trees. Files absent from the
// snapshot (sessions/secrets omitted) are left untouched.
func (e *Engine) Restore(dest, id string, currentSchema int) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.Store == nil {
		return fmt.Errorf("backup: no store")
	}
	list, err := List(dest)
	if err != nil {
		return err
	}
	var snap Snapshot
	for _, s := range list {
		if s.ID == id {
			snap = s
			break
		}
	}
	if snap.ID == "" {
		return fmt.Errorf("backup: snapshot not found")
	}
	m, err := readManifest(snap.Path)
	if err != nil {
		return fmt.Errorf("backup: manifest: %w", err)
	}
	if m.Schema > currentSchema {
		return fmt.Errorf("backup: snapshot schema %d is newer than this PiCode (%d) — upgrade first", m.Schema, currentSchema)
	}

	dbSrc := filepath.Join(snap.Path, "picode", "picode.db")
	if _, err := os.Stat(dbSrc); err != nil {
		return fmt.Errorf("backup: snapshot has no database")
	}
	if err := e.Store.ReplaceFrom(dbSrc); err != nil {
		return err
	}

	dataDir := e.DataDir
	srcPins := filepath.Join(snap.Path, "picode", "pins")
	if st, err := os.Stat(srcPins); err == nil && st.IsDir() {
		dst := filepath.Join(dataDir, "pins")
		_ = os.RemoveAll(dst)
		if err := copyTree(srcPins, dst); err != nil {
			return fmt.Errorf("backup: restore pins: %w", err)
		}
	}
	srcAcc := filepath.Join(snap.Path, "picode", "accounts.json")
	if _, err := os.Stat(srcAcc); err == nil {
		if err := copyRegular(srcAcc, filepath.Join(dataDir, "accounts.json"), 0o600); err != nil {
			return err
		}
	}

	pi := e.piDir()
	if pi != "" {
		for _, name := range []string{"settings.json", "trust.json", "auth.json"} {
			src := filepath.Join(snap.Path, "pi", name)
			if _, err := os.Stat(src); err != nil {
				continue
			}
			mode := os.FileMode(0o644)
			if name == "auth.json" {
				mode = 0o600
			}
			if err := copyRegular(src, filepath.Join(pi, name), mode); err != nil {
				return err
			}
		}
		srcSess := filepath.Join(snap.Path, "pi", "sessions")
		if st, err := os.Stat(srcSess); err == nil && st.IsDir() {
			dst := filepath.Join(pi, "sessions")
			_ = os.RemoveAll(dst)
			if err := copyTree(srcSess, dst); err != nil {
				return fmt.Errorf("backup: restore sessions: %w", err)
			}
		}
	}
	return nil
}

func copyTree(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		return copyRegular(path, target, info.Mode().Perm())
	})
}
