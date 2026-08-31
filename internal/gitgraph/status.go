package gitgraph

import "strings"

// Change is one working-tree difference, relative to the repository root.
// Kind collapses git's two-column status into the one word a file tree can
// decorate with: staged vs unstaged is a review distinction, and review
// belongs to a diff surface, not to a dot on a row.
type Change struct {
	Path string `json:"path"`
	Kind string `json:"kind"` // "untracked" | "added" | "deleted" | "renamed" | "conflicted" | "modified"
}

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
