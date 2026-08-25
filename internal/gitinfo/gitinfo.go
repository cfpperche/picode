// Package gitinfo reads branch/worktree for a directory. Missing git is not an error.
package gitinfo

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Info is facts from git. Nil Inspect means not a repo.
type Info struct {
	Branch   string `json:"branch,omitempty"`
	Worktree string `json:"worktree,omitempty"`
}

// Inspect returns git identity for dir, or nil.
func Inspect(dir string) *Info {
	if strings.TrimSpace(dir) == "" {
		return nil
	}
	if _, err := exec.LookPath("git"); err != nil {
		return nil
	}
	inside := git(dir, "rev-parse", "--is-inside-work-tree")
	if inside != "true" {
		return nil
	}
	info := &Info{}
	branch := git(dir, "branch", "--show-current")
	if branch == "" {
		branch = git(dir, "rev-parse", "--short", "HEAD")
	}
	info.Branch = branch
	gitDir := git(dir, "rev-parse", "--git-dir")
	if strings.Contains(filepath.ToSlash(gitDir), "/worktrees/") {
		info.Worktree = filepath.Base(gitDir)
	}
	if info.Branch == "" && info.Worktree == "" {
		return nil
	}
	return info
}

func git(dir string, args ...string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
