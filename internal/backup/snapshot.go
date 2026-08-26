package backup

import (
	"fmt"
	"os"
	"path/filepath"
)

// Snapshot writes a new complete snapshot into dest. Incomplete dirs are removed.
func (e *Engine) Snapshot(sessions, secrets bool, dest string) (Snapshot, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.snapshotLocked(sessions, secrets, dest)
}

func (e *Engine) snapshotLocked(sessions, secrets bool, dest string) (Snapshot, error) {
	if e.Store == nil {
		return Snapshot{}, fmt.Errorf("backup: no store")
	}
	dataDir := e.DataDir
	if err := ValidateDest(dest, dataDir, defaultPiHome()); err != nil {
		e.setLast(false, 0, err)
		return Snapshot{}, err
	}
	root := Root(dest)
	if err := ensureDir(root); err != nil {
		e.setLast(false, 0, err)
		return Snapshot{}, err
	}
	id := e.now().Format("2006-01-02T150405Z")
	dir := filepath.Join(root, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		e.setLast(false, 0, err)
		return Snapshot{}, err
	}
	ok := false
	defer func() {
		if !ok {
			_ = os.RemoveAll(dir)
		}
	}()

	prev := latestComplete(dest)
	var files []FileEnt

	picodeDst := filepath.Join(dir, "picode")
	if err := os.MkdirAll(picodeDst, 0o755); err != nil {
		e.setLast(false, 0, err)
		return Snapshot{}, err
	}
	dbDst := filepath.Join(picodeDst, "picode.db")
	if err := e.Store.VacuumInto(dbDst); err != nil {
		e.setLast(false, 0, err)
		return Snapshot{}, err
	}
	if err := noteFile(dbDst, &files); err != nil {
		e.setLast(false, 0, err)
		return Snapshot{}, err
	}

	prevPicode := ""
	if prev != "" {
		prevPicode = filepath.Join(prev, "picode")
	}
	if secrets {
		acc := filepath.Join(dataDir, "accounts.json")
		if _, err := os.Stat(acc); err == nil {
			if err := putFile(acc, filepath.Join(picodeDst, "accounts.json"), join(prevPicode, "accounts.json"), 0o600, &files); err != nil {
				e.setLast(false, 0, err)
				return Snapshot{}, err
			}
		}
	}
	if err := walkCopy(filepath.Join(dataDir, "pins"), filepath.Join(picodeDst, "pins"), join(prevPicode, "pins"), &files); err != nil {
		e.setLast(false, 0, err)
		return Snapshot{}, err
	}

	piSrc := e.piDir()
	prevPi := ""
	if prev != "" {
		prevPi = filepath.Join(prev, "pi")
	}
	if piSrc != "" {
		piDst := filepath.Join(dir, "pi")
		if err := os.MkdirAll(piDst, 0o755); err != nil {
			e.setLast(false, 0, err)
			return Snapshot{}, err
		}
		for _, name := range []string{"settings.json", "trust.json"} {
			src := filepath.Join(piSrc, name)
			if _, err := os.Stat(src); err != nil {
				continue
			}
			if err := putFile(src, filepath.Join(piDst, name), join(prevPi, name), 0o644, &files); err != nil {
				e.setLast(false, 0, err)
				return Snapshot{}, err
			}
		}
		if secrets {
			src := filepath.Join(piSrc, "auth.json")
			if _, err := os.Stat(src); err == nil {
				if err := putFile(src, filepath.Join(piDst, "auth.json"), join(prevPi, "auth.json"), 0o600, &files); err != nil {
					e.setLast(false, 0, err)
					return Snapshot{}, err
				}
			}
		}
		if sessions {
			if err := walkCopy(filepath.Join(piSrc, "sessions"), filepath.Join(piDst, "sessions"), join(prevPi, "sessions"), &files); err != nil {
				e.setLast(false, 0, err)
				return Snapshot{}, err
			}
		}
	}

	var bytes int64
	for i := range files {
		rel, err := filepath.Rel(dir, files[i].Path)
		if err == nil {
			files[i].Path = filepath.ToSlash(rel)
		}
		bytes += files[i].Size
	}

	schema, err := e.Store.SchemaVersion()
	if err != nil {
		e.setLast(false, 0, err)
		return Snapshot{}, err
	}
	host, _ := os.Hostname()
	m := Manifest{
		Format: FormatVersion, ID: id, Created: e.now().Format("2006-01-02T15:04:05Z"),
		Hostname: host, AppVersion: e.Version, Schema: schema,
		Sessions: sessions, Secrets: secrets, SameFS: SameFS(dataDir, dest),
		Bytes: bytes, Files: files,
	}
	if err := writeManifest(dir, m); err != nil {
		e.setLast(false, 0, err)
		return Snapshot{}, err
	}
	ok = true
	e.setLast(true, bytes, nil)
	return Snapshot{
		ID: id, Path: dir, Created: m.Created, Bytes: bytes,
		Sessions: sessions, Secrets: secrets, Schema: schema,
		Hostname: host, AppVersion: e.Version,
	}, nil
}

func noteFile(path string, files *[]FileEnt) error {
	sum, size, err := hashFile(path)
	if err != nil {
		return err
	}
	*files = append(*files, FileEnt{Path: path, Size: size, SHA: sum})
	return nil
}

func join(dir, name string) string {
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, name)
}
