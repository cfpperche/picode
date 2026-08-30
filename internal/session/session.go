// Package session is a read-only index of pi JSONL session files (ADR-0005).
package session

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Summary is one session file, enough for a picker.
type Summary struct {
	ID        string  `json:"id"`
	Path      string  `json:"path"`
	Name      string  `json:"name,omitempty"`
	Cwd       string  `json:"cwd,omitempty"`
	CreatedAt string  `json:"createdAt"`
	UpdatedAt string  `json:"updatedAt"`
	Preview   string  `json:"preview,omitempty"`
	Messages  int     `json:"messages"`
	Cost      float64 `json:"cost"`
	Provider  string  `json:"provider,omitempty"`
	Model     string  `json:"model,omitempty"`
	Thinking  string  `json:"thinking,omitempty"`
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

// TestRoot, when set, is Root() (tests only).
var TestRoot string

// Root is ~/.pi/agent/sessions.
func Root() string {
	if TestRoot != "" {
		return TestRoot
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".pi", "agent", "sessions")
}

// ListAll summaries every JSONL under Root, newest first.
func ListAll() ([]Summary, error) {
	return ListRoot(Root())
}

// ListRoot is ListAll against an explicit sessions root (tests).
func ListRoot(root string) ([]Summary, error) {
	if root == "" {
		return []Summary{}, nil
	}
	ents, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return []Summary{}, nil
		}
		return nil, err
	}
	var out []Summary
	for _, e := range ents {
		if !e.IsDir() {
			continue
		}
		list, err := listDir(filepath.Join(root, e.Name()))
		if err != nil {
			continue
		}
		out = append(out, list...)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UpdatedAt > out[j].UpdatedAt })
	if out == nil {
		out = []Summary{}
	}
	return out, nil
}

func listDir(dir string) ([]Summary, error) {
	ents, err := os.ReadDir(dir)
	if err != nil {
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
	return out, nil
}

// UnderRoot reports whether path is a JSONL inside the sessions root.
func UnderRoot(root, path string) bool {
	if root == "" || path == "" {
		return false
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(absRoot, abs)
	if err != nil {
		return false
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return false
	}
	return strings.HasSuffix(abs, ".jsonl")
}

// CopyFile writes a new JSONL next to src. The original file is not touched.
func CopyFile(src string) (string, error) {
	src, err := filepath.Abs(src)
	if err != nil {
		return "", err
	}
	in, err := os.Open(src)
	if err != nil {
		return "", err
	}
	defer in.Close()
	name := time.Now().UTC().Format("2006-01-02T15-04-05") + "_adopt_" + strconv.FormatInt(time.Now().UnixNano()%1e9, 36) + ".jsonl"
	dst := filepath.Join(filepath.Dir(src), name)
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return "", err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		_ = os.Remove(dst)
		return "", err
	}
	return dst, nil
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
			if cwd, _ := raw["cwd"].(string); cwd != "" {
				s.Cwd = cwd
			}
		case "session_info":
			if n, _ := raw["name"].(string); n != "" {
				s.Name = n
			}
		case "model_change":
			if p, _ := raw["provider"].(string); p != "" {
				s.Provider = p
			}
			if m, _ := raw["modelId"].(string); m != "" {
				s.Model = m
			}
		case "thinking_level_change":
			if t, _ := raw["thinkingLevel"].(string); t != "" {
				s.Thinking = t
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
