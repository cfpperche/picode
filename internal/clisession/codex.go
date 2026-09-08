package clisession

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// CodexSource lists Codex CLI rollouts under
// ~/.codex/sessions/YYYY/MM/DD/rollout-<ts>-<uuid>.jsonl (format verified
// against a real installation, 2026-09). The first session_meta line
// carries id and cwd; turn_context lines carry the model; response_item
// message lines carry the turns (role user/developer/assistant).
//
// Verified resume form (codex --help): the `resume` subcommand takes the
// session id — args are positional, so they replace any configured
// defaults when the terminal launch is composed.
type CodexSource struct{}

func (CodexSource) CLI() string { return "codex" }

// CodexTestRoot, when set, is CodexSessionsRoot() (tests only).
var CodexTestRoot string

// CodexSessionsRoot is ~/.codex/sessions, where Codex keeps rollouts under
// a YYYY/MM/DD tree. Exported so climetrics reads the same path this
// package lists from.
func CodexSessionsRoot() string {
	if CodexTestRoot != "" {
		return CodexTestRoot
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".codex", "sessions")
}

func (CodexSource) List(cwd string) ([]Summary, error) {
	root := CodexSessionsRoot()
	if root == "" {
		return nil, nil
	}
	var out []Summary
	for _, p := range jsonlFiles(root) {
		s, ok := summarizeCodex(p)
		if !ok {
			continue
		}
		if cwd != "" && s.Cwd != cwd {
			continue
		}
		out = append(out, s)
	}
	sortNewest(out)
	return out, nil
}

func summarizeCodex(path string) (Summary, bool) {
	f, err := os.Open(path)
	if err != nil {
		return Summary{}, false
	}
	defer f.Close()

	s := Summary{
		CLI:        "codex",
		Path:       path,
		ResumeArgs: []string{"resume", codexIDFromName(path)},
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
			Payload   struct {
				Type      string `json:"type"`
				ID        string `json:"id"`
				Cwd       string `json:"cwd"`
				Timestamp string `json:"timestamp"`
				Model     string `json:"model"`
				Role      string `json:"role"`
				Content   []struct {
					Type string `json:"type"`
					Text string `json:"text"`
				} `json:"content"`
			} `json:"payload"`
		}
		if json.Unmarshal(line, &raw) != nil {
			continue
		}
		switch raw.Type {
		case "session_meta":
			if s.ID == "" {
				s.ID = raw.Payload.ID
			}
			s.Cwd = raw.Payload.Cwd
			if s.CreatedAt == "" && raw.Payload.Timestamp != "" {
				s.CreatedAt = raw.Payload.Timestamp
			} else if s.CreatedAt == "" && raw.Timestamp != "" {
				s.CreatedAt = raw.Timestamp
			}
			if raw.Timestamp != "" {
				s.UpdatedAt = raw.Timestamp
			}
		case "turn_context":
			if raw.Payload.Model != "" {
				s.Model = raw.Payload.Model
			}
			if raw.Payload.Cwd != "" {
				s.Cwd = raw.Payload.Cwd
			}
			if raw.Timestamp != "" {
				s.UpdatedAt = raw.Timestamp
			}
		case "response_item":
			if raw.Payload.Type != "message" {
				continue
			}
			var text string
			for _, c := range raw.Payload.Content {
				if c.Text != "" {
					text = c.Text
					break
				}
			}
			if text == "" {
				continue
			}
			switch raw.Payload.Role {
			case "user", "assistant":
				s.Messages++
				if s.Preview == "" && raw.Payload.Role == "user" && !codexInjected(text) {
					s.Preview = clip(text, 120)
				}
			}
			if raw.Timestamp != "" {
				s.UpdatedAt = raw.Timestamp
			}
		}
	}
	if s.ID == "" {
		s.ID = codexIDFromName(path)
	}
	s.ResumeArgs = []string{"resume", s.ID}
	if s.UpdatedAt == "" {
		s.UpdatedAt = mtime(path)
	}
	if s.Cwd == "" {
		return Summary{}, false // no folder declared: unusable for opening
	}
	return s, true
}

// codexIDFromName pulls the trailing uuid out of a rollout file name:
// rollout-2025-12-10T08-28-53-019b0805-….jsonl → 019b0805-…
func codexIDFromName(path string) string {
	base := strings.TrimSuffix(filepath.Base(path), ".jsonl")
	if i := strings.LastIndex(base, "-"); i >= 0 && i < len(base)-1 {
		return base[i+1:]
	}
	return base
}

// codexInjected reports whether a user-role message is an injected
// instruction block (AGENTS.md preamble, environment context) rather than a
// human turn — those dominate the opening lines of many rollouts and would
// otherwise become the session's preview. Heuristic, deliberately narrow:
// injected blocks on real machines start with a heading or an XML-ish tag.
func codexInjected(text string) bool {
	return strings.HasPrefix(text, "# ") || strings.HasPrefix(text, "<")
}
