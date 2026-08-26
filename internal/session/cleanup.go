package session

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Stats is the JSONL footprint of one cwd's session folder.
type Stats struct {
	Dir   string `json:"dir"`
	Count int    `json:"count"`
	Bytes int64  `json:"bytes"`
}

// DirStats counts .jsonl files under Dir(cwd). Missing dir is empty, not an error.
func DirStats(cwd string) Stats {
	dir := Dir(cwd)
	st := Stats{Dir: dir}
	if dir == "" {
		return st
	}
	ents, err := os.ReadDir(dir)
	if err != nil {
		return st
	}
	for _, e := range ents {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		st.Count++
		if info, err := e.Info(); err == nil {
			st.Bytes += info.Size()
		}
	}
	return st
}

// RemoveDir deletes the session folder for cwd. Refuses anything that is
// not a pi session dir (name must look like --encoded-cwd--).
func RemoveDir(cwd string) error {
	dir := Dir(cwd)
	if dir == "" {
		return nil
	}
	base := filepath.Base(dir)
	if !strings.HasPrefix(base, "--") || !strings.HasSuffix(base, "--") || len(base) < 5 {
		return fmt.Errorf("session: refuse to remove %s", dir)
	}
	parent := filepath.Base(filepath.Dir(dir))
	if parent != "sessions" {
		return fmt.Errorf("session: refuse to remove %s", dir)
	}
	if err := os.RemoveAll(dir); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
