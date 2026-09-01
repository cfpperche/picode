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
	return DirStatsAt(Dir(cwd))
}

// DirStatsAt counts .jsonl files directly under an already-resolved
// directory (Dir(cwd) or AgentDir(agentID), ADR-0040). Missing dir is
// empty, not an error.
func DirStatsAt(dir string) Stats {
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

// RemoveAgentDir deletes the private session folder for agentID (ADR-0040).
// Same safety spirit as RemoveDir, adapted: an agent-id directory name
// never looks like --encoded-cwd--, so the shape check that protects
// RemoveDir doesn't apply here — the parent-is-"sessions" check is what's
// left, and is sufficient since AgentDir is always Root()/agentID.
func RemoveAgentDir(agentID string) error {
	agentID = strings.TrimSpace(agentID)
	if agentID == "" {
		return fmt.Errorf("session: refuse to remove empty agent id")
	}
	dir := AgentDir(agentID)
	if filepath.Base(filepath.Dir(dir)) != "sessions" {
		return fmt.Errorf("session: refuse to remove %s", dir)
	}
	if err := os.RemoveAll(dir); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
