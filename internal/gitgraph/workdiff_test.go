package gitgraph

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWorkingDiffModifiedFile(t *testing.T) {
	dir := repo(t)
	write(t, dir, "a", "changed body\n")
	f, truncated := WorkingDiff(dir, "a")
	if f == nil || truncated {
		t.Fatalf("diff = %+v truncated=%v", f, truncated)
	}
	if f.Path != "a" || f.Binary {
		t.Fatalf("file = %+v", f)
	}
	if !strings.Contains(f.Patch, "+changed body") || !strings.Contains(f.Patch, "-a") {
		t.Fatalf("patch misses the change:\n%s", f.Patch)
	}
}

// An untracked file has no HEAD side: `git diff HEAD` says nothing and
// `git diff --no-index` answers with EXIT CODE 1 — the diff must still
// arrive, whole-file additions, with no phantom /dev/null rename.
func TestWorkingDiffUntrackedFile(t *testing.T) {
	dir := repo(t)
	write(t, dir, "fresh", "line one\nline two\n")
	f, _ := WorkingDiff(dir, "fresh")
	if f == nil {
		t.Fatal("untracked file produced no diff — exit code 1 was swallowed")
	}
	if f.Path != "fresh" || f.OldPath != "" {
		t.Fatalf("file = %+v, want Path fresh with no OldPath", f)
	}
	if !strings.Contains(f.Patch, "+line one") || !strings.Contains(f.Patch, "+line two") {
		t.Fatalf("patch is not all-additions:\n%s", f.Patch)
	}
	if strings.Contains(f.Patch, "-line") {
		t.Fatalf("an addition-only diff carries deletions:\n%s", f.Patch)
	}
}

func TestWorkingDiffDeletedFile(t *testing.T) {
	dir := repo(t)
	if err := os.Remove(filepath.Join(dir, "a")); err != nil {
		t.Fatal(err)
	}
	f, _ := WorkingDiff(dir, "a")
	if f == nil {
		t.Fatal("deleted file produced no diff")
	}
	if !strings.Contains(f.Patch, "-a") {
		t.Fatalf("patch misses the deletion:\n%s", f.Patch)
	}
}

func TestWorkingDiffBinaryFile(t *testing.T) {
	dir := repo(t)
	if err := os.WriteFile(filepath.Join(dir, "blob.bin"), []byte{0, 1, 2, 250, 251}, 0o644); err != nil {
		t.Fatal(err)
	}
	f, _ := WorkingDiff(dir, "blob.bin")
	if f == nil || !f.Binary {
		t.Fatalf("binary file = %+v, want Binary true", f)
	}
}

func TestWorkingDiffCleanAndMissingAreNoDiff(t *testing.T) {
	dir := repo(t)
	if f, _ := WorkingDiff(dir, "a"); f != nil {
		t.Fatalf("clean file answered %+v", f)
	}
	if f, _ := WorkingDiff(dir, "nope.txt"); f != nil {
		t.Fatalf("missing file answered %+v", f)
	}
	if f, _ := WorkingDiff(t.TempDir(), "a"); f != nil {
		t.Fatalf("non-repo answered %+v", f)
	}
}

// A repository with no commits yet has no HEAD to diff against; every file
// falls back to the /dev/null side.
func TestWorkingDiffRepoWithoutCommits(t *testing.T) {
	dir := t.TempDir()
	run(t, dir, "git", "init", "-b", "main")
	write(t, dir, "first.txt", "hello\n")
	f, _ := WorkingDiff(dir, "first.txt")
	if f == nil || !strings.Contains(f.Patch, "+hello") {
		t.Fatalf("no-commit repo diff = %+v", f)
	}
}

func TestWorkingDiffTruncatesHugePatches(t *testing.T) {
	dir := repo(t)
	write(t, dir, "huge", strings.Repeat("x", maxPatchBytes+4096)+"\n")
	f, truncated := WorkingDiff(dir, "huge")
	if f == nil || !truncated {
		t.Fatalf("huge diff = %+v truncated=%v, want truncated", f, truncated)
	}
	if len(f.Patch) > maxPatchBytes {
		t.Fatalf("patch len %d exceeds the cap", len(f.Patch))
	}
}
