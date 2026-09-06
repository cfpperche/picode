package clisession

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// GrokSource lists Grok CLI sessions from the prompt history under
// ~/.grok/sessions/<url-encoded-cwd>/prompt_history.jsonl (format verified
// against a real installation, 2026-09). The history is prompts only — one
// file per folder, many session ids inside — so a Grok row summarizes the
// prompts of one session id and claims no transcript it does not have.
//
// Verified resume flag (grok --help): `--resume <SESSION_ID>`; UUID-shaped
// values always mean ids, which is exactly what this source reports.
type GrokSource struct{}

func (GrokSource) CLI() string { return "grok" }

func (GrokSource) List(cwd string) ([]Summary, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, nil
	}
	root := filepath.Join(home, ".grok", "sessions")
	ents, err := os.ReadDir(root)
	if err != nil {
		return nil, nil
	}
	var out []Summary
	for _, e := range ents {
		if !e.IsDir() {
			continue
		}
		dir := decodePathDir(e.Name())
		if dir == "" {
			continue
		}
		if cwd != "" && dir != cwd {
			continue // the directory name is the cwd: skip without parsing
		}
		p := filepath.Join(root, e.Name(), "prompt_history.jsonl")
		out = append(out, grokSessions(p, dir)...)
	}
	sortNewest(out)
	return out, nil
}

// grokSessions folds one prompt-history file into per-session summaries.
func grokSessions(path, dir string) []Summary {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	order := []string{} // first-seen order keeps the fold deterministic
	byID := map[string]*Summary{}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	for sc.Scan() {
		var raw struct {
			Timestamp string `json:"timestamp"`
			SessionID string `json:"session_id"`
			Prompt    string `json:"prompt"`
		}
		if json.Unmarshal(sc.Bytes(), &raw) != nil || raw.SessionID == "" {
			continue
		}
		s, ok := byID[raw.SessionID]
		if !ok {
			s = &Summary{
				CLI:        "grok",
				ID:         raw.SessionID,
				Path:       path,
				ResumeArgs: []string{"--resume", raw.SessionID},
				Cwd:        dir,
				CreatedAt:  raw.Timestamp,
			}
			byID[raw.SessionID] = s
			order = append(order, raw.SessionID)
		}
		s.Messages++
		s.UpdatedAt = raw.Timestamp
		if s.Preview == "" && strings.TrimSpace(raw.Prompt) != "" {
			s.Preview = clip(raw.Prompt, 120)
		}
	}
	out := make([]Summary, 0, len(order))
	for _, id := range order {
		s := byID[id]
		if s.UpdatedAt == "" {
			continue
		}
		s.Size = fileSize(path) // shared file; informational only
		out = append(out, *s)
	}
	sortNewest(out)
	return out
}
