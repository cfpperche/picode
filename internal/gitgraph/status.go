package gitgraph

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Change is one working-tree difference, relative to the repository root.
// Kind collapses git's two-column status into the one word a file tree can
// decorate with: staged vs unstaged is a review distinction, and review
// belongs to a diff surface, not to a dot on a row.
type Change struct {
	Path string `json:"path"`
	Kind string `json:"kind"` // "untracked" | "added" | "deleted" | "renamed" | "conflicted" | "modified"
}

// ChangeStat is a Change with the line counts a review list shows beside the
// name (Inspector rail): lines added and removed against HEAD, exactly what
// the per-file working diff renders, so the list and the detail agree. A
// binary file has no line counts; Binary says why both are zero. Truncated
// marks an untracked file whose count stopped at the read cap.
type ChangeStat struct {
	Change
	Add       int  `json:"add"`
	Del       int  `json:"del"`
	Binary    bool `json:"binary,omitempty"`
	Truncated bool `json:"truncated,omitempty"`
}

// StatusInfo is the working tree with its counts and the branch facts a
// header can show. Top is "" when dir is not inside a repository.
type StatusInfo struct {
	Top      string
	Branch   string
	Worktree string
	Changes  []ChangeStat
}

// newFileReadCap bounds how much of an untracked file is read to count its
// lines. A generated asset can run to hundreds of megabytes; the count is a
// hint beside a name, not a reason to stream the file.
const newFileReadCap = 4 << 20

// binaryProbeBytes is how far into an untracked file a NUL is looked for
// before calling it binary — the same heuristic git applies to new files.
const binaryProbeBytes = 8 << 10

// Status reads the working tree state of the repository containing dir.
// top is the repository's toplevel ("" when dir is not inside one); change
// paths are relative to that toplevel, exactly as git reports them — the
// caller re-anchors them to whatever directory its own reader is confined to.
func Status(dir string) (top string, changes []Change) {
	top = git(dir, "rev-parse", "--show-toplevel")
	if top == "" {
		return "", nil
	}
	// -uall: without it git collapses an untracked directory into one
	// "dir/" record, and a tree cannot decorate files it was never told about.
	raw := git(dir, "status", "--porcelain", "-z", "--untracked-files=all")
	recs := strings.Split(raw, "\x00")
	for i := 0; i < len(recs); i++ {
		rec := recs[i]
		if len(rec) < 4 || rec[2] != ' ' {
			continue
		}
		x, y, path := rec[0], rec[1], rec[3:]
		if x == 'R' || x == 'C' || y == 'R' || y == 'C' {
			// A rename/copy record is two NUL-terminated fields: the new
			// path, then the old one — consume the old path so it is not
			// misread as the next record's status header.
			i++
		}
		changes = append(changes, Change{Path: path, Kind: changeKind(x, y)})
	}
	return top, changes
}

// StatusWithStats is Status plus, for every change, the lines added and
// removed against HEAD, and the branch/worktree the directory is on. Tracked
// changes come from one `git diff HEAD --numstat` (staged or not — the same
// comparison WorkingDiff draws); untracked files are counted here, since git
// has nothing to compare them against. Status callers that only need kinds
// keep the cheaper read.
func StatusWithStats(dir string) StatusInfo {
	top, plain := Status(dir)
	if top == "" {
		return StatusInfo{}
	}
	info := StatusInfo{Top: top, Changes: make([]ChangeStat, 0, len(plain))}
	info.Branch = git(dir, "branch", "--show-current")
	if info.Branch == "" {
		info.Branch = git(dir, "rev-parse", "--short", "HEAD")
	}
	if gitDir := git(dir, "rev-parse", "--git-dir"); strings.Contains(filepath.ToSlash(gitDir), "/worktrees/") {
		info.Worktree = filepath.Base(gitDir)
	}
	idx := make(map[string]int, len(plain))
	for _, c := range plain {
		idx[c.Path] = len(info.Changes)
		info.Changes = append(info.Changes, ChangeStat{Change: c})
	}
	// A repository with no commit yet has no HEAD to diff against; every
	// change there is a new file and is counted below.
	if git(dir, "rev-parse", "--verify", "--quiet", "HEAD") != "" {
		walkNumstat(git(dir, "diff", "HEAD", "--numstat", "-z", "-M", "--"), func(path string, add, del int, binary bool) {
			j, ok := idx[path]
			if !ok {
				return
			}
			info.Changes[j].Add, info.Changes[j].Del, info.Changes[j].Binary = add, del, binary
		})
	}
	for i := range info.Changes {
		if info.Changes[i].Kind != "untracked" {
			continue
		}
		lines, binary, truncated := countNewFile(filepath.Join(top, filepath.FromSlash(info.Changes[i].Path)))
		info.Changes[i].Add, info.Changes[i].Binary, info.Changes[i].Truncated = lines, binary, truncated
	}
	return info
}

// walkNumstat parses `--numstat -z` output, calling fn once per record. -z
// is what makes a rename parseable: instead of the brace shorthand
// (`dir/{old => new}/f`) the record ends with an empty path and the two names
// follow as their own NUL-separated tokens, old then new — fn receives the
// new name. A binary file reports "-" for both counts; binary says so.
func walkNumstat(out string, fn func(path string, add, del int, binary bool)) {
	tok := strings.Split(out, "\x00")
	for i := 0; i < len(tok); i++ {
		f := strings.SplitN(tok[i], "\t", 3)
		if len(f) != 3 {
			continue
		}
		path := f[2]
		if path == "" {
			if i+2 >= len(tok) {
				break
			}
			path = tok[i+2]
			i += 2
		}
		add, errA := strconv.Atoi(f[0])
		del, errD := strconv.Atoi(f[1])
		binary := errA != nil || errD != nil
		if binary {
			add, del = 0, 0
		}
		fn(path, add, del, binary)
	}
}

// countNewFile counts the lines git would report as added for an untracked
// file: one per newline, plus one for a final line without one. Reading
// stops at newFileReadCap (truncated says so); a NUL within the first
// binaryProbeBytes makes the file binary, with no count at all.
func countNewFile(path string) (lines int, binary bool, truncated bool) {
	f, err := os.Open(path)
	if err != nil {
		return 0, false, false
	}
	defer f.Close()
	buf := make([]byte, 64<<10)
	var read int64
	first := true
	last := byte('\n')
	for read < newFileReadCap {
		n, err := f.Read(buf)
		if n > 0 {
			chunk := buf[:n]
			if first {
				first = false
				if bytes.IndexByte(chunk[:min(n, binaryProbeBytes)], 0) >= 0 {
					return 0, true, false
				}
			}
			lines += bytes.Count(chunk, []byte{'\n'})
			last = chunk[n-1]
			read += int64(n)
		}
		if err != nil {
			if err != io.EOF {
				return lines, false, false
			}
			break
		}
	}
	if read >= newFileReadCap {
		// Whatever follows the cap is uncounted; do not guess a final line.
		var probe [1]byte
		if n, _ := f.Read(probe[:]); n > 0 {
			return lines, false, true
		}
	}
	if read > 0 && last != '\n' {
		lines++
	}
	return lines, false, false
}

func changeKind(x, y byte) string {
	switch {
	case x == '?' || y == '?':
		return "untracked"
	case x == 'U' || y == 'U' || (x == 'A' && y == 'A') || (x == 'D' && y == 'D'):
		return "conflicted"
	case x == 'R' || y == 'R' || x == 'C' || y == 'C':
		return "renamed"
	case x == 'D' || y == 'D':
		return "deleted"
	case x == 'A':
		return "added"
	default:
		return "modified"
	}
}
