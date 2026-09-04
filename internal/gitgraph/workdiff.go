package gitgraph

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// gitLoose is git() for commands where a non-zero exit is an answer, not a
// failure: `git diff --no-index` exits 1 exactly when the two sides differ,
// which is the very output the caller wants. Exit codes past 1 still fail.
func gitLoose(dir string, args ...string) string {
	ctx, cancel := context.WithTimeout(context.Background(), gitTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); !ok || ee.ExitCode() != 1 {
			return ""
		}
	}
	return strings.TrimRight(string(out), "\n")
}

// WorkingDiff is one file's difference between the working tree and HEAD —
// what a change dot expands into. relPath must already be confined by the
// caller (relUnderCwd); it is passed after `--` so a leading dash cannot
// become a flag. An untracked file has no side in HEAD, so it is diffed
// against /dev/null — the whole file as additions — and the same path serves
// a repository with no commits yet. nil means no difference to show.
func WorkingDiff(dir, relPath string) (*FileDiff, bool) {
	if relPath == "" || Key(dir) == "" {
		return nil, false
	}
	var raw string
	untracked := false
	if git(dir, "rev-parse", "HEAD") == "" {
		raw = untrackedDiff(dir, relPath)
		untracked = true
	} else {
		raw = git(dir, "diff", "HEAD", "--no-color", "-M", "--", relPath)
		// An empty answer is ambiguous: clean-and-tracked, or untracked.
		// Only the untracked file takes the /dev/null fallback — a clean
		// tracked file fed to --no-index would answer with its whole body.
		if strings.TrimSpace(raw) == "" && git(dir, "ls-files", "--", relPath) == "" {
			raw = untrackedDiff(dir, relPath)
			untracked = true
		}
	}
	if strings.TrimSpace(raw) == "" {
		return nil, false
	}
	truncated := false
	if len(raw) > maxPatchBytes {
		raw = raw[:maxPatchBytes]
		truncated = true
	}
	files := splitPatch(raw)
	if len(files) == 0 {
		return nil, false
	}
	f := files[0]
	// The /dev/null side of an untracked diff is bookkeeping, not a rename.
	if f.OldPath == "/dev/null" || f.OldPath == "dev/null" {
		f.OldPath = ""
	}
	// An untracked file is new by definition, and --no-index is the one diff
	// whose mode lines cannot be trusted to say so.
	if untracked {
		f.Status = "added"
	}
	return &f, truncated
}

func untrackedDiff(dir, relPath string) string {
	if st, err := os.Stat(filepath.Join(dir, relPath)); err != nil || st.IsDir() {
		return ""
	}
	return gitLoose(dir, "diff", "--no-index", "--no-color", "--", os.DevNull, relPath)
}
