package gitgraph

import "strings"

// WorktreeOfRef maps a branch name, or a full commit hash for a detached
// checkout, to the absolute path of the worktree of this repository where it
// is checked out. It is how a URL can name a sibling worktree without ever
// carrying a filesystem path: the caller's own cwd identifies the repository,
// and git — not the request — decides where that checkout lives. This keeps
// ADR-0022's confinement intact: no repository or directory is ever resolved
// from the URL.
//
// "" means no such worktree: an unknown branch, an unknown hash, or a checkout
// that git does not list for this repository.
func WorktreeOfRef(dir, ref string) string {
	if Key(dir) == "" {
		return ""
	}
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return ""
	}
	// A 40/64-hex ref is answered as a commit: detached worktrees have no
	// branch to be named by. When several worktrees sit on that commit, a
	// detached one wins — an attached worktree is better addressed by its
	// branch name, which the caller has.
	hash := isHash(ref)
	fallback := ""
	for _, wt := range loadWorktrees(dir) {
		if wt.Bare || wt.Prunable {
			continue
		}
		if hash {
			if wt.Head != ref {
				continue
			}
			if wt.Detached {
				return wt.Path
			}
			if fallback == "" {
				fallback = wt.Path
			}
			continue
		}
		if wt.Branch == ref && branchExists(dir, ref) {
			return wt.Path
		}
	}
	return fallback
}

// branchExists requires an exact match against refs/heads. The worktree list
// alone would accept a branch git has since deleted, and the full-refname
// comparison keeps a pattern-shaped name (`feat/*`) from matching more than
// the one branch it names.
func branchExists(dir, branch string) bool {
	full := "refs/heads/" + branch
	for _, name := range strings.Split(git(dir, "for-each-ref", "--format=%(refname)", full), "\n") {
		if strings.TrimSpace(name) == full {
			return true
		}
	}
	return false
}
