package clisession

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// ClaudeCodeSource lists Claude Code transcripts under
// ~/.claude/projects/<encoded-cwd>/<sessionId>.jsonl (format verified
// against a real installation, 2026-09). Each line carries the cwd and the
// session id explicitly, so the encoded directory name is never decoded —
// dash-for-slash is ambiguous, the content is not.
//
// Verified resume flag (claude --help): `claude --resume <session-id>`.
type ClaudeCodeSource struct{}

func (ClaudeCodeSource) CLI() string { return "claude-code" }

func (ClaudeCodeSource) List(cwd string) ([]Summary, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, nil
	}
	root := filepath.Join(home, ".claude", "projects")
	var out []Summary
	for _, dir := range projectDirs(root) {
		for _, p := range jsonlFiles(dir) {
			s, ok := summarizeClaude(p)
			if !ok {
				continue
			}
			if cwd != "" && s.Cwd != cwd {
				continue
			}
			out = append(out, s)
		}
	}
	sortNewest(out)
	return out, nil
}

// projectDirs returns the immediate subdirectories of root. Claude Code
// encodes a cwd by replacing "/" with "-", so an exact decode is impossible
// and the cwd each transcript declares on its lines is used instead.
func projectDirs(root string) []string {
	ents, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	var dirs []string
	for _, e := range ents {
		if e.IsDir() {
			dirs = append(dirs, filepath.Join(root, e.Name()))
		}
	}
	return dirs
}

// summarizeClaude scans one transcript. It reads every line because the
// newest activity and the model ride on later entries; malformed lines are
// skipped. A file with no session id anywhere falls back to its file name.
func summarizeClaude(path string) (Summary, bool) {
	f, err := os.Open(path)
	if err != nil {
		return Summary{}, false
	}
	defer f.Close()

	s := Summary{
		CLI:        "claude-code",
		Path:       path,
		ResumeArgs: []string{"--resume", strings.TrimSuffix(filepath.Base(path), ".jsonl")},
		Size:       fileSize(path),
	}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		var raw struct {
			Type      string `json:"type"`
			Timestamp string `json:"timestamp"`
			Cwd       string `json:"cwd"`
			SessionID string `json:"sessionId"`
			SessionLo string `json:"session_id"`
			Sidechain bool   `json:"isSidechain"`
			Summary   string `json:"summary"`
			Message   struct {
				Role    string          `json:"role"`
				Model   string          `json:"model"`
				Content json.RawMessage `json:"content"`
			} `json:"message"`
		}
		if json.Unmarshal(line, &raw) != nil {
			continue
		}
		if s.ID == "" {
			s.ID = raw.SessionID
			if s.ID == "" {
				s.ID = raw.SessionLo
			}
		}
		if s.Cwd == "" {
			s.Cwd = raw.Cwd
		}
		if raw.Timestamp != "" {
			if s.CreatedAt == "" {
				s.CreatedAt = raw.Timestamp
			}
			s.UpdatedAt = raw.Timestamp
		}
		switch raw.Type {
		case "summary":
			if s.Name == "" {
				s.Name = clip(raw.Summary, 120)
			}
		case "user", "assistant":
			if raw.Sidechain {
				continue
			}
			if raw.Message.Model != "" {
				s.Model = raw.Message.Model
			}
			text := claudeText(raw.Message.Content, raw.Message.Role)
			if text == "" {
				continue
			}
			s.Messages++
			if s.Preview == "" && raw.Message.Role == "user" {
				s.Preview = clip(text, 120)
			}
		}
	}
	if s.ID == "" {
		s.ID = strings.TrimSuffix(filepath.Base(path), ".jsonl")
	}
	if s.UpdatedAt == "" {
		s.UpdatedAt = mtime(path)
	}
	if s.Cwd == "" {
		return Summary{}, false // no folder declared: unusable for opening
	}
	return s, true
}

// claudeText extracts human text from a message content that may be a
// plain string or an array of typed blocks ({type:"text",text:"..."}).
// Tool results and tool_use blocks are not human turns and are skipped.
func claudeText(content json.RawMessage, role string) string {
	if len(content) == 0 {
		return ""
	}
	var str string
	if json.Unmarshal(content, &str) == nil {
		return strings.TrimSpace(str)
	}
	var blocks []struct {
		Type    string          `json:"type"`
		Text    string          `json:"text"`
		Content json.RawMessage `json:"content"`
	}
	if json.Unmarshal(content, &blocks) != nil {
		return ""
	}
	for _, b := range blocks {
		if b.Type == "text" && strings.TrimSpace(b.Text) != "" {
			return strings.TrimSpace(b.Text)
		}
	}
	return ""
}

// fileSize stats a file, 0 when unreadable.
func fileSize(path string) int64 {
	if st, err := os.Stat(path); err == nil {
		return st.Size()
	}
	return 0
}
