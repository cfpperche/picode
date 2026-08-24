// Package session is a read-only index of pi JSONL session files (ADR-0005).
package session

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Summary is one session file, enough for a picker.
type Summary struct {
	ID        string  `json:"id"`
	Path      string  `json:"path"`
	Name      string  `json:"name,omitempty"`
	CreatedAt string  `json:"createdAt"`
	UpdatedAt string  `json:"updatedAt"`
	Preview   string  `json:"preview,omitempty"`
	Messages  int     `json:"messages"`
	Cost      float64 `json:"cost"`
}

// DirName is pi's folder name for a workspace cwd.
func DirName(cwd string) string {
	clean := filepath.ToSlash(filepath.Clean(cwd))
	enc := strings.ReplaceAll(clean, "/", "-")
	enc = strings.Trim(enc, "-")
	return "--" + enc + "--"
}

// Dir is ~/.pi/agent/sessions/<DirName>.
func Dir(cwd string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".pi", "agent", "sessions", DirName(cwd))
}

// List summaries for cwd, newest first.
func List(cwd string) ([]Summary, error) {
	dir := Dir(cwd)
	ents, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []Summary{}, nil
		}
		return nil, err
	}
	var out []Summary
	for _, e := range ents {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		p := filepath.Join(dir, e.Name())
		s, err := Summarize(p)
		if err != nil {
			continue
		}
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UpdatedAt > out[j].UpdatedAt })
	return out, nil
}

// Summarize reads a JSONL session file.
func Summarize(path string) (Summary, error) {
	st, err := os.Stat(path)
	if err != nil {
		return Summary{}, err
	}
	f, err := os.Open(path)
	if err != nil {
		return Summary{}, err
	}
	defer f.Close()

	s := Summary{
		Path:      path,
		UpdatedAt: st.ModTime().UTC().Format(time.RFC3339),
	}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		var raw map[string]any
		if json.Unmarshal(line, &raw) != nil {
			continue
		}
		switch raw["type"] {
		case "session":
			if id, _ := raw["id"].(string); id != "" {
				s.ID = id
			}
			if ts, _ := raw["timestamp"].(string); ts != "" {
				s.CreatedAt = ts
			}
		case "session_info":
			if n, _ := raw["name"].(string); n != "" {
				s.Name = n
			}
		case "message":
			s.Messages++
			if s.Preview == "" {
				s.Preview = previewFrom(raw["message"])
			}
			s.Cost += costFrom(raw["message"])
		}
	}
	if s.ID == "" {
		s.ID = strings.TrimSuffix(filepath.Base(path), ".jsonl")
	}
	if s.CreatedAt == "" {
		s.CreatedAt = s.UpdatedAt
	}
	return s, sc.Err()
}

func previewFrom(msg any) string {
	m, _ := msg.(map[string]any)
	if m == nil {
		return ""
	}
	if m["role"] != "user" {
		return ""
	}
	switch c := m["content"].(type) {
	case string:
		return clip(c, 120)
	case []any:
		for _, part := range c {
			p, _ := part.(map[string]any)
			if p != nil && p["type"] == "text" {
				if t, _ := p["text"].(string); t != "" {
					return clip(t, 120)
				}
			}
		}
	}
	return ""
}

func costFrom(msg any) float64 {
	m, _ := msg.(map[string]any)
	if m == nil {
		return 0
	}
	u, _ := m["usage"].(map[string]any)
	if u == nil {
		return 0
	}
	c, _ := u["cost"].(map[string]any)
	if c == nil {
		return 0
	}
	switch v := c["total"].(type) {
	case float64:
		return v
	}
	return 0
}

func clip(s string, n int) string {
	s = strings.TrimSpace(strings.ReplaceAll(s, "\n", " "))
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
