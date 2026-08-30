// Package gitgraph reads the commit DAG, refs and worktrees of a repository.
// It knows nothing about agents: the caller pairs worktrees with occupants.
//
// ADR-0022: one graph per repository. The identity is the common dir, so
// every worktree of a repo answers with the same Key.
package gitgraph

import (
	"context"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Field and record separators, emitted literally by git through %x1f / %x1e.
// They are not sentinels: a commit message may contain them, and git hands
// them back byte for byte. The parser is what makes them safe — the subject
// is last, so a capped split lets it absorb any extra separator.
const (
	fieldSep  = "\x1f"
	recordSep = "\x1e"
)

// gitTimeout bounds every call. A repository on a cold or network filesystem
// must not hang a request; an empty graph is a better answer than a stall.
const gitTimeout = 10 * time.Second

// Commit is one node of the DAG. Parents are hashes, in git's own order, so
// the first parent stays first.
type Commit struct {
	Hash    string   `json:"hash"`
	Parents []string `json:"parents"`
	Author  string   `json:"author"`
	At      int64    `json:"at"`
	Subject string   `json:"subject"`
}

// Ref is a branch, remote branch or tag pointing at a commit.
type Ref struct {
	Name string `json:"name"`
	Kind string `json:"kind"` // head | remote | tag
	Hash string `json:"hash"`
}

// Worktree is one checkout of the repository. Branch is empty when detached.
type Worktree struct {
	Path   string `json:"path"`
	Head   string `json:"head"`
	Branch string `json:"branch,omitempty"`
}

// Graph is everything the browser needs to draw in one payload.
type Graph struct {
	Key       string     `json:"key"`
	Name      string     `json:"name"`
	Head      string     `json:"head"`
	Commits   []Commit   `json:"commits"`
	Refs      []Ref      `json:"refs"`
	Worktrees []Worktree `json:"worktrees"`
	More      bool       `json:"more"`
}

// Key is the repository identity: the common dir, absolute. Every worktree of
// a repo returns the same value, which is what collapses them into one graph.
// Empty means dir is not a repository.
func Key(dir string) string {
	if strings.TrimSpace(dir) == "" {
		return ""
	}
	if _, err := exec.LookPath("git"); err != nil {
		return ""
	}
	common := git(dir, "rev-parse", "--git-common-dir")
	if common == "" {
		return ""
	}
	if !filepath.IsAbs(common) {
		top := git(dir, "rev-parse", "--show-toplevel")
		if top == "" {
			return ""
		}
		common = filepath.Join(top, common)
	}
	abs, err := filepath.Abs(common)
	if err != nil {
		return common
	}
	return filepath.Clean(abs)
}

// Load reads up to limit commits reachable from all refs plus HEAD. A
// repository with no commits yet is not an error: it returns a Graph whose
// Commits is empty, because the view for "no history" is a real state.
func Load(dir string, limit int) *Graph {
	key := Key(dir)
	if key == "" {
		return nil
	}
	if limit <= 0 {
		limit = 250
	}
	g := &Graph{
		Key:  key,
		Name: repoName(key),
		Head: git(dir, "rev-parse", "HEAD"),
	}
	g.Commits = loadCommits(dir, limit)
	g.More = len(g.Commits) == limit
	g.Refs = loadRefs(dir)
	g.Worktrees = loadWorktrees(dir)
	return g
}

// repoName is the working name of the repository: the directory holding the
// common dir. For a bare repo the common dir is the repo, so use it directly.
func repoName(key string) string {
	base := filepath.Base(key)
	if base == ".git" {
		return filepath.Base(filepath.Dir(key))
	}
	return strings.TrimSuffix(base, ".git")
}

func loadCommits(dir string, limit int) []Commit {
	format := "--format=" + strings.Join([]string{"%H", "%P", "%an", "%at", "%s"}, fieldSep) + recordSep
	out := git(dir,
		"-c", "log.showSignature=false", "log",
		"--max-count="+strconv.Itoa(limit),
		format, "--date-order",
		"--branches", "--tags", "--remotes", "HEAD", "--")
	if out == "" {
		return []Commit{}
	}
	records := strings.Split(out, recordSep)
	commits := make([]Commit, 0, len(records))
	for _, rec := range records {
		rec = strings.TrimLeft(rec, "\r\n")
		if rec == "" {
			continue
		}
		// SplitN, not Split: git preserves a literal 0x1f typed into a commit
		// message, and the subject is the last field. Capping the split lets
		// the subject absorb it instead of shifting every field and dropping
		// the commit from the graph.
		f := strings.SplitN(rec, fieldSep, 5)
		// A 0x1e in a message splits one record in two, and the tail would
		// otherwise parse into a commit with a fabricated hash. Every record
		// begins with %H, so anything else is debris: drop it.
		if len(f) != 5 || !isHash(f[0]) {
			continue
		}
		at, _ := strconv.ParseInt(f[3], 10, 64)
		c := Commit{Hash: f[0], Author: f[2], At: at, Subject: f[4], Parents: []string{}}
		if p := strings.TrimSpace(f[1]); p != "" {
			c.Parents = strings.Fields(p)
		}
		commits = append(commits, c)
	}
	return commits
}

// isHash reports whether s is an object name as %H writes it: lowercase hex,
// SHA-1 or SHA-256 length. It is the guard that keeps debris out of the DAG.
func isHash(s string) bool {
	if len(s) != 40 && len(s) != 64 {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}

func loadRefs(dir string) []Ref {
	out := git(dir, "for-each-ref", "--format=%(objectname)"+fieldSep+"%(refname)",
		"refs/heads", "refs/remotes", "refs/tags")
	refs := []Ref{}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		f := strings.SplitN(line, fieldSep, 2)
		if len(f) != 2 {
			continue
		}
		name, kind, ok := classifyRef(f[1])
		if !ok {
			continue
		}
		refs = append(refs, Ref{Name: name, Kind: kind, Hash: f[0]})
	}
	return refs
}

// classifyRef turns a full refname into a short name and a kind. A remote's
// HEAD pointer (origin/HEAD) is dropped: it is an alias, not a branch, and
// drawing it doubles a label the user already sees.
func classifyRef(refname string) (name, kind string, ok bool) {
	switch {
	case strings.HasPrefix(refname, "refs/heads/"):
		return strings.TrimPrefix(refname, "refs/heads/"), "head", true
	case strings.HasPrefix(refname, "refs/remotes/"):
		short := strings.TrimPrefix(refname, "refs/remotes/")
		if strings.HasSuffix(short, "/HEAD") {
			return "", "", false
		}
		return short, "remote", true
	case strings.HasPrefix(refname, "refs/tags/"):
		return strings.TrimPrefix(refname, "refs/tags/"), "tag", true
	}
	return "", "", false
}

// loadWorktrees parses `git worktree list --porcelain`. Records are separated
// by a blank line; a record is "worktree <path>" then HEAD, then either
// "branch <ref>" or "detached".
func loadWorktrees(dir string) []Worktree {
	out := git(dir, "worktree", "list", "--porcelain")
	list := []Worktree{}
	var cur *Worktree
	flush := func() {
		if cur != nil && cur.Path != "" {
			list = append(list, *cur)
		}
		cur = nil
	}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimRight(line, "\r")
		switch {
		case line == "":
			flush()
		case strings.HasPrefix(line, "worktree "):
			flush()
			cur = &Worktree{Path: filepath.Clean(strings.TrimPrefix(line, "worktree "))}
		case cur == nil:
			// A stray line before any worktree header; nothing to attach it to.
		case strings.HasPrefix(line, "HEAD "):
			cur.Head = strings.TrimPrefix(line, "HEAD ")
		case strings.HasPrefix(line, "branch "):
			cur.Branch = strings.TrimPrefix(strings.TrimPrefix(line, "branch "), "refs/heads/")
		}
	}
	flush()
	return list
}

func git(dir string, args ...string) string {
	ctx, cancel := context.WithTimeout(context.Background(), gitTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimRight(string(out), "\n")
}
