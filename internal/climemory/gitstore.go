package climemory

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// A memory store the CLI keeps under version control.
//
// Codex is the one that does it today: `~/.codex/memories` is a git repository
// whose first commit is the baseline Codex wrote, and every later revision is
// Codex reorganising the same three files. It is the only thing that store can
// answer that the others cannot — it has no citation graph, no declared index
// budget and no frontmatter kinds, so its rows carried a size and a date and
// nothing else.
//
// It is also the more truthful date. The filesystem's mtime moves when the CLI
// rewrites a file with the same bytes, and on the day this was measured all
// three of Codex's files said "today" while their content had not changed since
// the 12:10 baseline. A revision date says when the content changed.
//
// Only a repository whose `.git` sits **directly** in the store folder counts.
// Muse keeps its memory in `<workspace>/.agents/memory`, inside the user's own
// repository: walking up would report the project's history as the memory's,
// which is a different thing entirely.

// History is what a store's own repository knows about itself.
type History struct {
	Revisions int    `json:"revisions"`
	LastAt    string `json:"lastAt,omitempty"`
	Subject   string `json:"subject,omitempty"`
	// Changed names the store's own files that differ from the last revision.
	Changed []string `json:"changed,omitempty"`
	// at maps a tracked file to the revision that last changed it. Folded into
	// each Item's Modified rather than serialized on its own.
	at map[string]string
}

// gitTimeout bounds every call. A store PiCode cannot read in three seconds
// reports no history at all — a half-answer about what changed is worse than
// the size and the mtime the row already carries.
const gitTimeout = 3 * time.Second

// logCap bounds the walk. These stores hold a handful of files; a history
// longer than this cannot change which revision last touched one of them
// enough to be worth the read.
const logCap = 500

// recordSep and fieldSep are bytes a commit subject cannot contain, so a
// subject with a newline or a tab in it cannot fake a record boundary.
const recordSep = "\x01"
const fieldSep = "\x02"

func history(dir string) *History {
	if dir == "" {
		return nil
	}
	// A `.git` directory, or the file a worktree leaves in its place.
	if info, err := os.Lstat(filepath.Join(dir, ".git")); err != nil || !(info.IsDir() || info.Mode().IsRegular()) {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), gitTimeout)
	defer cancel()

	count, ok := gitOut(ctx, dir, "rev-list", "--count", "HEAD")
	if !ok {
		return nil
	}
	n, err := strconv.Atoi(strings.TrimSpace(count))
	if err != nil || n <= 0 {
		// A repository with no commit yet has nothing to report.
		return nil
	}

	h := &History{Revisions: n, at: map[string]string{}}
	log, ok := gitOut(ctx, dir, "log", "--max-count="+strconv.Itoa(logCap),
		"--format="+recordSep+"%cI"+fieldSep+"%s", "--name-only")
	if !ok {
		return nil
	}
	when := ""
	for _, line := range strings.Split(log, "\n") {
		if strings.HasPrefix(line, recordSep) {
			date, subject, _ := strings.Cut(strings.TrimPrefix(line, recordSep), fieldSep)
			when = date
			if h.LastAt == "" {
				h.LastAt, h.Subject = date, subject
			}
			continue
		}
		name := strings.TrimSpace(line)
		if name == "" || when == "" || !storeFile(name) {
			continue
		}
		// The log walks newest first, so the first date a path wears is the
		// revision that last changed it.
		if _, seen := h.at[name]; !seen {
			h.at[name] = when
		}
	}

	// -z, because git quotes and backslash-escapes a path with a space in the
	// plain format: a name we failed to parse would be a file wearing a
	// revision date it no longer matches, which is the one wrong answer this
	// can give.
	if status, ok := gitOut(ctx, dir, "status", "--porcelain", "-z", "-uall", "--no-renames"); ok {
		for _, rec := range strings.Split(status, "\x00") {
			// "XY path": the code is two columns wide and a space follows it.
			if len(rec) < 4 {
				continue
			}
			name := strings.TrimSpace(rec[2:])
			if storeFile(name) {
				h.Changed = append(h.Changed, name)
			}
		}
		sort.Strings(h.Changed)
	}
	return h
}

// storeFile keeps the history to the files the pane actually lists: ordinary
// markdown at the top of the store. Codex's folder also holds `.agents/` and
// `extensions/`, which are not memories.
func storeFile(name string) bool {
	return !strings.ContainsAny(name, "/\\") && strings.EqualFold(filepath.Ext(name), ".md")
}

func gitOut(ctx context.Context, dir string, args ...string) (string, bool) {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", dir}, args...)...)
	// A read must never take the index lock: the CLI may be writing this very
	// folder while the pane loads.
	cmd.Env = append(os.Environ(), "GIT_OPTIONAL_LOCKS=0")
	out, err := cmd.Output()
	if err != nil {
		return "", false
	}
	return string(out), true
}

// stamp folds the history into the rows. A date the CLI wrote into the file
// itself still wins — that is the CLI's own statement about its memory. Git
// only replaces the filesystem's mtime, and only for a file that matches the
// revision: one with uncommitted changes is newer than any revision, and the
// mtime is then the honest answer.
func (h *History) stamp(items []Item) []Item {
	if h == nil || len(h.at) == 0 {
		return items
	}
	changed := map[string]bool{}
	for _, name := range h.Changed {
		changed[name] = true
	}
	for i := range items {
		if items[i].ModifiedFrom != "file" || changed[items[i].ID] {
			continue
		}
		if at := h.at[items[i].ID]; at != "" {
			items[i].Modified, items[i].ModifiedFrom = at, "git"
		}
	}
	return items
}
