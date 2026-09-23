package clisession

import (
	"os"
	"strings"

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
