package clisession

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/cfpperche/picode/internal/session"
)

// Locator finds one recorded session on disk again (ADR-0205): the agent
// history lists a removed agent only while its transcript exists, and a
// restore resumes it with the arguments the listing verifies. A Locator
// memoizes each CLI's listing per folder, so a page of exits reads every
// store once.
type Locator struct {
	lists map[string][]Summary
}

// A history refresh checks the file again, but unchanged transcripts need
// not be parsed again. The bounded cache holds summaries, never file bodies.
var locatedFiles = struct {
	sync.Mutex
	items map[string]locatedFile
	order []string
}{items: map[string]locatedFile{}}

type locatedFile struct {
	size    int64
	mtime   int64
	summary Summary
}

// NewLocator returns an empty Locator; use one per request.
func NewLocator() *Locator { return &Locator{lists: map[string][]Summary{}} }

// Locate returns the session a removed agent pointed at, or nil when it is
// no longer on disk. Pi is found by its file (id is ignored); every other
// CLI by its session id in the folder it ran in (cwd), falling back to the
// whole machine when that folder has no match (exits written before the
// folder was recorded, a CLI that files by something else). A DB-backed
// CLI's path names the database every row shares, so it never identifies
// a session on its own.
func (l *Locator) Locate(cli, id, path, cwd string) (*Summary, error) {
	id, path, cwd = strings.TrimSpace(id), strings.TrimSpace(path), strings.TrimSpace(cwd)
	if cli == "pi" {
		return locatePi(path, cwd)
	}
	if id == "" {
		return nil, nil
	}
	if sum := locateFile(cli, id, path); sum != nil {
		return sum, nil
	}
	src, ok := Get(cli)
	if !ok {
		return nil, nil
	}
	scopes := []string{cwd}
	if cwd != "" {
		scopes = append(scopes, "")
	}
	for _, scope := range scopes {
		list, err := l.list(src, scope)
		if err != nil {
			return nil, err
		}
		for i := range list {
			if list[i].ID == id {
				s := list[i]
				return &s, nil
			}
		}
	}
	return nil, nil
}

// A file-backed CLI records the exact transcript path on removal. Verify the
// id and root before using it; old exits and moved files keep the listing
// fallback below. Database paths identify a store, not one conversation.
func locateFile(cli, id, path string) *Summary {
	var root string
	var summarize func(string) (Summary, bool)
	switch cli {
	case "codex":
		root, summarize = CodexSessionsRoot(), summarizeCodex
	case "claude-code":
		root, summarize = ClaudeProjectsRoot(), summarizeClaude
	case "omp":
		root, summarize = OmpSessionsRoot(), scanOmpFile
	default:
		return nil
	}
	if root == "" || path == "" || filepath.Ext(path) != ".jsonl" {
		return nil
	}
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return nil
	}
	st, err := os.Stat(path)
	if err != nil || !st.Mode().IsRegular() {
		return nil
	}
	key := cli + "\x00" + filepath.Clean(path)
	locatedFiles.Lock()
	entry, ok := locatedFiles.items[key]
	locatedFiles.Unlock()
	if ok && entry.size == st.Size() && entry.mtime == st.ModTime().UnixNano() {
		if entry.summary.ID == id {
			s := entry.summary
			return &s
		}
		return nil
	}
	s, ok := summarize(path)
	if !ok || s.ID != id {
		return nil
	}
	locatedFiles.Lock()
	if _, exists := locatedFiles.items[key]; !exists {
		locatedFiles.order = append(locatedFiles.order, key)
	}
	locatedFiles.items[key] = locatedFile{size: st.Size(), mtime: st.ModTime().UnixNano(), summary: s}
	if len(locatedFiles.order) > 256 {
		delete(locatedFiles.items, locatedFiles.order[0])
		locatedFiles.order = locatedFiles.order[1:]
	}
	locatedFiles.Unlock()
	return &s
}

func (l *Locator) list(src Source, cwd string) ([]Summary, error) {
	key := src.CLI() + "\x00" + cwd
	if got, ok := l.lists[key]; ok {
		return got, nil
	}
	got, err := src.List(cwd)
	if err != nil {
		return nil, err
	}
	l.lists[key] = got
	return got, nil
}

func locatePi(path, cwd string) (*Summary, error) {
	if path == "" {
		return nil, nil
	}
	st, err := os.Stat(path)
	if err != nil || !st.Mode().IsRegular() {
		return nil, nil
	}
	sum, err := session.Summarize(path)
	if err != nil {
		// The file is there; a header it cannot parse still resumes in Pi.
		sum = session.Summary{Path: path, Size: st.Size()}
	}
	out := piSummaries([]session.Summary{sum})[0]
	if out.UpdatedAt == "" {
		out.UpdatedAt = mtime(path)
	}
	if out.Cwd == "" {
		out.Cwd = cwd
	}
	return &out, nil
}
