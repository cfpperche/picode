package clisession

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// OmpSource lists Omp (oh-my-pi) sessions from ~/.omp/agent/sessions, the
// same bucket-per-cwd JSONL layout pi uses with omp's own encoding (each
// "/" becomes "-", no trimming: /tmp/omp-probe → -tmp-omp-probe). Read-only.
// Resume flag verified against a real install (omp 18.2.4, 2026-09-17):
// `omp --resume <id>` resumes by id prefix; `-r` without a value is the
// interactive picker, so the row always carries the id.
//
// omp resolves its agent dir from $PI_CODING_AGENT_DIR before ~/.omp/agent
// (the launcher reserves that key, so a PiCode launch can never point this
// listing somewhere else); XDG relocation via `omp config init-xdg` is a
// user migration this source does not follow yet.
//
// The file is a pi-family transcript at schema version 3: a `session`
// header (id, timestamp, cwd), `title` records, a `model_change` per switch,
// and `message` records whose content is a block array. Everything parses
// defensively — a malformed file is skipped, never fails the listing.
type OmpSource struct{}

func (OmpSource) CLI() string { return "omp" }

// OmpTestRoot, when set, replaces the sessions root (tests only).
var OmpTestRoot string

func (OmpSource) List(cwd string) ([]Summary, error) {
	root := ompSessionsRoot()
	if root == "" {
		return nil, nil
	}
	var files []string
	if cwd == "" {
		files = jsonlFiles(root)
	} else {
		// The bucket name is omp's encoding, but the filter reads the cwd
		// each session header recorded — robust to bucket collisions.
		for _, f := range jsonlFiles(ompBucket(root, cwd)) {
			files = append(files, f)
		}
	}
	out := make([]Summary, 0, len(files))
	for _, f := range files {
		if s, ok := scanOmpFile(f); ok {
			if cwd != "" && s.Cwd != cwd {
				continue
			}
			out = append(out, s)
		}
	}
	sortNewest(out)
	return out, nil
}

// ompSessionsRoot is omp's sessions directory: $PI_CODING_AGENT_DIR/sessions
// when the override is set, else ~/.omp/agent/sessions.
func ompSessionsRoot() string {
	if OmpTestRoot != "" {
		return OmpTestRoot
	}
	if dir := strings.TrimSpace(os.Getenv("PI_CODING_AGENT_DIR")); dir != "" {
		return filepath.Join(dir, "sessions")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".omp", "agent", "sessions")
}

// ompBucket is the observed encoding: every "/" becomes "-", nothing trimmed
// (/tmp/omp-probe → -tmp-omp-probe).
func ompBucket(root, cwd string) string {
	return filepath.Join(root, strings.ReplaceAll(filepath.ToSlash(filepath.Clean(cwd)), "/", "-"))
}

// ompFile carries what one pass over a session file extracted.
type ompFile struct {
	id        string
	cwd       string
	started   string
	updated   string
	title     string
	preview   string
	modelFrom string // newest model_change: provider-qualified (google/gemini-3.6-flash)
	msgModel  string // newest assistant message model, the fallback
	messages  int
	hasUser   bool
	hasHeader bool
}

// scanOmpFile reads one session file once, keeping the fields the picker
// shows: the header's id/cwd/timestamp, the newest non-empty title, the
// newest model_change, the first user text as preview and the message count.
func scanOmpFile(path string) (Summary, bool) {
	raw, err := os.ReadFile(path)
	if err != nil || len(raw) == 0 {
		return Summary{}, false
	}
	var f ompFile
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || line[0] != '{' {
			continue
		}
		var rec struct {
			Type      string `json:"type"`
			ID        string `json:"id"`
			Timestamp string `json:"timestamp"`
			Cwd       string `json:"cwd"`
			Title     string `json:"title"`
			Model     string `json:"model"`
			Message   *struct {
				Role       string          `json:"role"`
				Timestamp  json.Number     `json:"timestamp"` // epoch millis on message records
				Model      string          `json:"model"`
				StopReason string          `json:"stopReason"`
				Content    json.RawMessage `json:"content"`
			} `json:"message"`
		}
		if json.Unmarshal([]byte(line), &rec) != nil {
			continue
		}
		if rec.Timestamp != "" {
			f.updated = rec.Timestamp
		}
		switch rec.Type {
		case "session":
			f.hasHeader = true
			f.id, f.cwd, f.started = rec.ID, rec.Cwd, rec.Timestamp
		case "title":
			if t := strings.TrimSpace(rec.Title); t != "" {
				f.title = t
			}
		case "model_change":
			if rec.Model != "" {
				f.modelFrom = rec.Model
			}
		case "message":
			if rec.Message == nil {
				continue
			}
			f.messages++
			if rec.Message.Model != "" {
				f.msgModel = rec.Message.Model
			}
			if !f.hasUser && rec.Message.Role == "user" {
				if t := ompText(rec.Message.Content); t != "" {
					f.preview, f.hasUser = t, true
				}
			}
		}
	}
	if !f.hasHeader || f.id == "" || (f.messages == 0 && f.preview == "") {
		// Not a session (or a session nobody prompted): skip both.
		return Summary{}, false
	}
	if f.updated == "" {
		f.updated = mtime(path)
	}
	if f.started == "" {
		f.started = f.updated
	}
	if f.updated == "" {
		return Summary{}, false
	}
	preview := f.preview
	name := f.title
	if name == "" {
		name = preview
	}
	model := f.modelFrom
	if model == "" {
		model = f.msgModel
	}
	return Summary{
		CLI:        "omp",
		ID:         f.id,
		Path:       path,
		ResumeArgs: []string{"--resume", f.id},
		Name:       clip(name, 120),
		Cwd:        f.cwd,
		CreatedAt:  f.started,
		UpdatedAt:  f.updated,
		Preview:    clip(preview, 120),
		Messages:   f.messages,
		Size:       fileSize(path),
		Model:      model,
	}, true
}

// ompText extracts the first text block of a message content array
// ([{"type":"text","text":"…"}]); non-text blocks and plain strings that
// are not text are skipped.
func ompText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if json.Unmarshal(raw, &blocks) != nil {
		return ""
	}
	for _, b := range blocks {
		if b.Type == "text" && strings.TrimSpace(b.Text) != "" {
			return strings.TrimSpace(b.Text)
		}
	}
	return ""
}
