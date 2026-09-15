// Package clisession is a read-only index of coding-CLI session files
// (ADR-0079 phase 2): pi JSONL under ~/.pi/agent/sessions, Claude Code
// transcripts under ~/.claude/projects, Codex rollouts under
// ~/.codex/sessions, Grok prompt history under ~/.grok/sessions,
// Hermes Agent rows in ~/.hermes/state.db (or $HERMES_HOME/state.db),
// OpenCode rows in ~/.local/share/opencode/opencode.db (or
// $XDG_DATA_HOME/opencode/opencode.db), Muse Code rows in
// ~/.local/share/muse/session-index.db (or $XDG_DATA_HOME/muse/...) and
// Antigravity rows in ~/.gemini/antigravity-cli/conversation_summaries.db.
// Nothing here writes, deletes or resumes; it only lists what is on disk.
package clisession

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Summary is one session of one CLI, enough for the sessions picker.
// Cost is intentionally pi-only on this surface: Claude Code, Codex, Grok,
// Hermes and OpenCode listings do not populate it. Hermes and OpenCode store
// spend in SQLite, but the guest session row does not render cost.
type Summary struct {
	CLI        string   `json:"cli"`
	ID         string   `json:"id"`
	Path       string   `json:"path,omitempty"`
	ResumeArgs []string `json:"resumeArgs,omitempty"`
	Name       string   `json:"name,omitempty"`
	Cwd        string   `json:"cwd,omitempty"`
	CreatedAt  string   `json:"createdAt"`
	UpdatedAt  string   `json:"updatedAt"`
	Preview    string   `json:"preview,omitempty"`
	Messages   int      `json:"messages"`
	Size       int64    `json:"size"`
	Model      string   `json:"model,omitempty"`
	Cost       float64  `json:"cost,omitempty"`
}

// Source lists one CLI's sessions. cwd filters to one folder; "" lists
// every folder on the machine. Missing roots are simply empty, never an
// error — a CLI that never ran here still renders a sane empty state.
type Source interface {
	CLI() string
	List(cwd string) ([]Summary, error)
}

// Sources returns every registered source keyed by catalog CLI id.
func Sources() map[string]Source {
	out := map[string]Source{}
	for _, s := range []Source{PISource{}, ClaudeCodeSource{}, CodexSource{}, GrokSource{}, HermesSource{}, OpenCodeSource{}, MuseSource{}, AgySource{}} {
		out[s.CLI()] = s
	}
	return out
}

// Get returns the source for a catalog CLI id.
func Get(cli string) (Source, bool) {
	s, ok := Sources()[cli]
	return s, ok
}

// Latest returns the most recently updated session of one CLI in one
// folder, or nil when the CLI has none there. Sessions last written before
// notBefore never win — pass a run's start time so a terminal can't pin
// another terminal's conversation in a shared folder (ADR-0084); the zero
// time accepts any.
func Latest(cliID, cwd string, notBefore time.Time) (*Summary, error) {
	src, ok := Get(cliID)
	if !ok {
		return nil, fmt.Errorf("Unknown CLI.")
	}
	list, err := src.List(cwd)
	if err != nil {
		return nil, err
	}
	sortNewest(list)
	for i := range list {
		s := &list[i]
		if !notBefore.IsZero() {
			at, err := time.Parse(time.RFC3339, s.UpdatedAt)
			if err != nil || at.Before(notBefore) {
				continue
			}
		}
		return s, nil
	}
	return nil, nil
}

// sortNewest orders summaries by UpdatedAt, newest first, and is the
// single ordering every source applies.
func sortNewest(out []Summary) {
	sort.Slice(out, func(i, j int) bool { return out[i].UpdatedAt > out[j].UpdatedAt })
}

// jsonlFiles walks root for *.jsonl files, newest-mtime first is NOT
// guaranteed — callers sort summaries themselves. A missing root yields
// no files and no error.
func jsonlFiles(root string) []string {
	if root == "" {
		return nil
	}
	var out []string
	_ = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // unreadable subtree: skip, never fail the listing
		}
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".jsonl") {
			out = append(out, p)
		}
		return nil
	})
	return out
}

// mtime is a file's modification time in RFC3339 UTC, or the zero string.
func mtime(path string) string {
	if st, err := os.Stat(path); err == nil {
		return st.ModTime().UTC().Format(time.RFC3339)
	}
	return ""
}

// clip mirrors pi's preview clipping so every CLI's rows stay one line.
func clip(s string, n int) string {
	s = strings.TrimSpace(strings.ReplaceAll(s, "\n", " "))
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// decodePathDir undoes an URL-escaped directory name (Grok encodes the
// cwd as one path segment: %2Fhome%2Fgoat). A malformed name decodes to "".
// DecodePathDir turns a url-encoded directory name back into a path.
// Exported so climetrics resolves Grok's folders the same way this package
// does, instead of re-deriving the encoding.
func DecodePathDir(name string) string { return decodePathDir(name) }

func decodePathDir(name string) string {
	d, err := url.PathUnescape(name)
	if err != nil {
		return ""
	}
	return d
}
