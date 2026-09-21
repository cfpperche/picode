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
		if err := swapTree(srcPins, filepath.Join(dataDir, "pins")); err != nil {
			return fmt.Errorf("backup: restore pins: %w", err)
		}
	}
	for _, name := range []string{"credentials.json", "accounts.json"} {
		src := filepath.Join(snap.Path, "picode", name)
		if _, err := os.Stat(src); err != nil {
			continue
		}
		if err := swapRegular(src, filepath.Join(dataDir, name), 0o600, keepReplaced); err != nil {
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
			if err := swapRegular(src, filepath.Join(pi, name), mode, discardReplaced); err != nil {
				return err
			}
		}
		srcSess := filepath.Join(snap.Path, "pi", "sessions")
		if st, err := os.Stat(srcSess); err == nil && st.IsDir() {
			if err := swapTree(srcSess, filepath.Join(pi, "sessions")); err != nil {
				return fmt.Errorf("backup: restore sessions: %w", err)
			}
		}
	}
	return nil
}

// swapTree restores one directory without ever destroying the live one
// before the replacement is complete: build it beside the target, then
// rename into place (one filesystem, so the rename is atomic) and only
// then drop what it replaced.
//
// It used to be RemoveAll followed by a walking copy. A copy that failed
// halfway — a full disk, one unreadable file — left the live tree deleted
// and half rebuilt, with no way back, and it ran *after* ReplaceFrom had
// already swapped the database in: a restore that failed on ~/.pi/sessions
// took the conversations with it. Store.ReplaceFrom stashes and rolls back
// for exactly this reason; these two halves now do the same.
func swapTree(src, dst string) error {
	staging := dst + ".restoring"
	if err := os.RemoveAll(staging); err != nil {
		return err
	}
	if err := copyTree(src, staging); err != nil {
		_ = os.RemoveAll(staging)
		return err
	}
	replaced := dst + ".replaced"
	_ = os.RemoveAll(replaced)
	hadLive := true
	if err := os.Rename(dst, replaced); err != nil {
		if !os.IsNotExist(err) {
			_ = os.RemoveAll(staging)
			return err
		}
		hadLive = false // nothing there to replace: a first restore
	}
	if err := os.Rename(staging, dst); err != nil {
		if hadLive {
			_ = os.Rename(replaced, dst) // put the live tree back
		}
		_ = os.RemoveAll(staging)
		return err
	}
	if hadLive {
		_ = os.RemoveAll(replaced)
	}
	return nil
}

// Whether swapRegular leaves the file it replaced behind, named at the
// call site so the two behaviours read as the decision they are.
const (
	keepReplaced    = true
	discardReplaced = false
)

// swapRegular restores one file the same way: write beside the target,
// then rename into place, so a failed copy never leaves a half-written
// file where a whole one used to be.
//
// With keepReplaced the file it replaced stays as "<name>.replaced".
// That matters for credentials.json: the snapshot carries the vault but
// never the key that opens it (see Snapshot), so a vault restored where
// the local key has since changed cannot be decrypted — and the live one
// is then still on disk to put back. It is the promise ADR-0166 already
// makes when activating a CLI credential. Pi's own files are restored
// with discardReplaced: leaving debris in a directory another tool owns
// is not ours to do.
func swapRegular(src, dst string, mode os.FileMode, keep bool) error {
	staging := dst + ".restoring"
	if err := copyRegular(src, staging, mode); err != nil {
		_ = os.Remove(staging)
		return err
	}
	replaced := dst + ".replaced"
	restore := func() {}
	if _, err := os.Stat(dst); keep && err == nil {
		_ = os.Remove(replaced)
		if err := os.Rename(dst, replaced); err != nil {
			_ = os.Remove(staging)
			return err
		}
		restore = func() { _ = os.Rename(replaced, dst) }
	}
	if err := os.Rename(staging, dst); err != nil {
		restore()
		_ = os.Remove(staging)
		return err
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
