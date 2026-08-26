package mcp

import (
	_ "embed"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

//go:embed live.js
var liveExt []byte

// Live values written onto Server.Live (file On only).
const (
	LiveIdle   = "idle"
	LiveOn     = "live"
	LiveFailed = "failed"
	LiveAuth   = "signin"
)

// EnsureLiveExt writes the silent -e bridge into dataDir.
func EnsureLiveExt(dataDir string) (string, error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dataDir, "mcp-live.js")
	if err := os.WriteFile(path, liveExt, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// LivePath is the snapshot file for one managed agent.
func LivePath(dataDir, agentID string) string {
	id := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return -1
	}, agentID)
	if id == "" {
		return ""
	}
	return filepath.Join(dataDir, "mcp-live", id+".json")
}

// AttachLive returns pi argv + env so a managed start can receive snapshots.
func AttachLive(dataDir, agentID string) (args, env []string) {
	if dataDir == "" || agentID == "" {
		return nil, nil
	}
	ext, err := EnsureLiveExt(dataDir)
	if err != nil {
		return nil, nil
	}
	path := LivePath(dataDir, agentID)
	if path == "" {
		return nil, nil
	}
	return []string{"-e", ext}, []string{"PICODE_MCP_LIVE=" + path}
}

// ClearLive drops a stale snapshot when the agent stops.
func ClearLive(dataDir, agentID string) {
	if path := LivePath(dataDir, agentID); path != "" {
		_ = os.Remove(path)
	}
}

type liveSnap struct {
	Servers []struct {
		Name   string `json:"name"`
		Status string `json:"status"`
	} `json:"servers"`
}

// ReadLive maps server name → adapter status. Missing or stale → nil.
func ReadLive(path string, maxAge time.Duration) map[string]string {
	if path == "" {
		return nil
	}
	st, err := os.Stat(path)
	if err != nil || st.IsDir() {
		return nil
	}
	if maxAge > 0 && time.Since(st.ModTime()) > maxAge {
		return nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var snap liveSnap
	if json.Unmarshal(b, &snap) != nil {
		return nil
	}
	out := map[string]string{}
	for _, s := range snap.Servers {
		if s.Name != "" {
			out[s.Name] = s.Status
		}
	}
	return out
}

// ApplyLive sets Server.Live from a running-agent snapshot.
func ApplyLive(rep *Report, live map[string]string, running bool) {
	if rep == nil {
		return
	}
	for i := range rep.Servers {
		if rep.Servers[i].Disabled {
			rep.Servers[i].Live = ""
			continue
		}
		if !running {
			rep.Servers[i].Live = LiveIdle
			continue
		}
		switch live[rep.Servers[i].Name] {
		case "connected":
			rep.Servers[i].Live = LiveOn
		case "failed":
			rep.Servers[i].Live = LiveFailed
		case "needs-auth":
			rep.Servers[i].Live = LiveAuth
		default:
			rep.Servers[i].Live = LiveIdle
		}
	}
}
