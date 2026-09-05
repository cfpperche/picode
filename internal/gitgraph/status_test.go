package gitgraph

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func changeByPath(changes []Change) map[string]string {
	out := map[string]string{}
	for _, c := range changes {
		out[c.Path] = c.Kind
	}
	return out
}

func TestStatusNotARepo(t *testing.T) {
	top, changes := Status(t.TempDir())
	if top != "" || changes != nil {
		t.Fatalf("Status on a plain dir = %q, %v; want empty", top, changes)
	}
}

func TestStatusCleanRepo(t *testing.T) {
	dir := repo(t)
	top, changes := Status(dir)
	if top == "" {
		t.Fatal("clean repo reported no toplevel")
	}
	if len(changes) != 0 {
		t.Fatalf("clean repo reported changes: %v", changes)
	}
}

// One dirty repo, every kind at once — and the rename record, whose second
// NUL field (the old path) must be consumed, not misread as a new record.
func TestStatusKindsAndRenameRecord(t *testing.T) {
	dir := repo(t)
	write(t, dir, "b", "b")
	// Distinct bodies: git pairs renames by content similarity, and two
	// identical one-byte files would let it match the wrong pair.
	write(t, dir, "gone", "the deleted file's own body\n")
	write(t, dir, "xy old", "the spaced rename's own body\n")
	run(t, dir, "git", "add", ".")
	run(t, dir, "git", "commit", "-m", "base")

	write(t, dir, "a", "changed")    // modified
	write(t, dir, "fresh", "new")    // untracked
	write(t, dir, "staged-new", "s") // added (staged)
	run(t, dir, "git", "add", "staged-new")
	run(t, dir, "git", "rm", "-q", "gone") // deleted
	// The rename's OLD path is chosen to LOOK like a status header
	// ("xy old": two letters, a space) — if the parser fails to consume
	// the second NUL field, this is the shape that turns into a phantom
	// "old" change instead of being silently skipped.
	run(t, dir, "git", "mv", "b", "b-renamed") // renamed
	run(t, dir, "git", "mv", "xy old", "spaced-renamed")

	if err := os.MkdirAll(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, dir, "sub/deep", "d") // untracked inside a subdir

	top, changes := Status(dir)
	if top == "" {
		t.Fatal("no toplevel on a repo")
	}
	got := changeByPath(changes)
	want := map[string]string{
		"a":              "modified",
		"fresh":          "untracked",
		"staged-new":     "added",
		"gone":           "deleted",
		"b-renamed":      "renamed",
		"spaced-renamed": "renamed",
		"sub/deep":       "untracked",
	}
	for path, kind := range want {
		if got[path] != kind {
			t.Errorf("%s = %q, want %q (all: %v)", path, got[path], kind, got)
		}
	}
	for _, leak := range []string{"b", "xy old", "old"} {
		if _, ok := got[leak]; ok {
			t.Errorf("the rename's OLD path leaked in as %q", leak)
		}
	}
	if len(got) != len(want) {
		t.Errorf("got %d changes, want %d: %v", len(got), len(want), got)
	}
}

// Paths come back relative to the toplevel even when Status is asked from a
// subdirectory — the caller re-anchors, so the contract must not drift.
func TestStatusFromASubdirStaysTopRelative(t *testing.T) {
	dir := repo(t)
	if err := os.MkdirAll(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, dir, "sub/f", "x")
	top, changes := Status(filepath.Join(dir, "sub"))
	if top == "" {
		t.Fatal("no toplevel from subdir")
	}
	got := changeByPath(changes)
	if got["sub/f"] != "untracked" {
		t.Fatalf("changes = %v, want sub/f untracked (top-relative)", got)
	}
}

func statByPath(changes []ChangeStat) map[string]ChangeStat {
	out := map[string]ChangeStat{}
	for _, c := range changes {
		out[c.Path] = c
	}
	return out
}

// Counts follow the comparison WorkingDiff draws — working tree against
// HEAD, staged or not — and untracked files are counted in Go, since git has
// nothing to diff them against. A final line without a newline still counts.
func TestStatusWithStatsCountsTrackedAndUntracked(t *testing.T) {
	dir := repo(t)
	write(t, dir, "notes", "one\ntwo\nthree\n")
	run(t, dir, "git", "add", ".")
	run(t, dir, "git", "commit", "-m", "base")

	write(t, dir, "a", "changed\nagain")            // 1 line replaced by 2: +2 −1
	write(t, dir, "notes", "one\nthree\n")          // −1
	write(t, dir, "fresh", "new\nfile\nno newline") // untracked, 3 lines
	write(t, dir, "staged-new", "s1\ns2\n")         // added (staged): +2
	run(t, dir, "git", "add", "staged-new")

	info := StatusWithStats(dir)
	if info.Top == "" {
		t.Fatal("no toplevel on a repo")
	}
	if info.Branch != "main" {
		t.Fatalf("branch = %q, want main", info.Branch)
	}
	if info.Worktree != "" {
		t.Fatalf("worktree = %q on the main checkout, want empty", info.Worktree)
	}
	got := statByPath(info.Changes)
	want := map[string][2]int{"a": {2, 1}, "notes": {0, 1}, "fresh": {3, 0}, "staged-new": {2, 0}}
	for path, counts := range want {
		c, ok := got[path]
		if !ok {
			t.Fatalf("%s missing from %v", path, got)
		}
		if c.Add != counts[0] || c.Del != counts[1] || c.Binary || c.Truncated {
			t.Errorf("%s = +%d −%d binary=%v truncated=%v, want +%d −%d", path, c.Add, c.Del, c.Binary, c.Truncated, counts[0], counts[1])
		}
	}
	if len(got) != len(want) {
		t.Errorf("got %d changes, want %d: %v", len(got), len(want), got)
	}
}

// A rename carries its edit's counts under the NEW name — the -z record
// walk must pair the two path tokens the same way Status does.
func TestStatusWithStatsRenameWithEdit(t *testing.T) {
	dir := repo(t)
	write(t, dir, "before", "line one\nline two\nline three\nline four\n")
	run(t, dir, "git", "add", ".")
	run(t, dir, "git", "commit", "-m", "base")
	run(t, dir, "git", "mv", "before", "after")
	write(t, dir, "after", "line one\nline two\nline three\nline four\nline five\n")
	run(t, dir, "git", "add", "after")

	got := statByPath(StatusWithStats(dir).Changes)
	c, ok := got["after"]
	if !ok {
		t.Fatalf("renamed file missing: %v", got)
	}
	if c.Kind != "renamed" || c.Add != 1 || c.Del != 0 {
		t.Fatalf("after = %+v, want renamed +1 −0", c)
	}
	if _, leaked := got["before"]; leaked {
		t.Fatalf("the rename's old path leaked: %v", got)
	}
}

// Binary files have no line counts: git says "-" for tracked ones, and an
// untracked file with a NUL in its head is called binary here.
func TestStatusWithStatsBinary(t *testing.T) {
	dir := repo(t)
	write(t, dir, "tracked.bin", "\x00\x01\x02")
	run(t, dir, "git", "add", ".")
	run(t, dir, "git", "commit", "-m", "base")
	write(t, dir, "tracked.bin", "\x00\x01\x02\x03\x04")
	write(t, dir, "fresh.bin", "\x00\x07\x08\x09")

	got := statByPath(StatusWithStats(dir).Changes)
	for _, path := range []string{"tracked.bin", "fresh.bin"} {
		c, ok := got[path]
		if !ok {
			t.Fatalf("%s missing: %v", path, got)
		}
		if !c.Binary || c.Add != 0 || c.Del != 0 {
			t.Errorf("%s = %+v, want binary with zero counts", path, c)
		}
	}
}

// Before the first commit there is no HEAD to diff against; every change is a
// new file and still gets its count instead of a failed git call.
func TestStatusWithStatsNoCommitsYet(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git missing")
	}
	dir := t.TempDir()
	run(t, dir, "git", "init", "-b", "main")
	write(t, dir, "first", "a\nb\n")
	write(t, dir, "staged", "x\n")
	run(t, dir, "git", "add", "staged")

	info := StatusWithStats(dir)
	if info.Top == "" {
		t.Fatal("no toplevel on a fresh repo")
	}
	got := statByPath(info.Changes)
	if got["first"].Add != 2 || got["first"].Kind != "untracked" {
		t.Errorf("first = %+v, want untracked +2", got["first"])
	}
	// Staged in an unborn repository: git reports it as added but has no
	// HEAD to count against — zero is honest, not wrong.
	if got["staged"].Kind != "added" {
		t.Errorf("staged = %+v, want kind added", got["staged"])
	}
}

// A linked worktree names itself; the main checkout stays unnamed.
func TestStatusWithStatsWorktreeName(t *testing.T) {
	dir := repo(t)
	wt := filepath.Join(t.TempDir(), "side")
	run(t, dir, "git", "worktree", "add", "-b", "side", wt)
	write(t, wt, "a", "edited in the worktree\n")

	info := StatusWithStats(wt)
	if info.Branch != "side" || info.Worktree != "side" {
		t.Fatalf("worktree read = branch %q worktree %q, want side/side", info.Branch, info.Worktree)
	}
	if got := statByPath(info.Changes); got["a"].Add != 1 || got["a"].Del != 1 {
		t.Fatalf("a = %+v, want +1 −1", got["a"])
	}
}

func TestCountNewFileCapsAndFinalLine(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "nl", "a\nb\n")
	write(t, dir, "nonl", "a\nb")
	write(t, dir, "empty", "")
	big := bytes.Repeat([]byte("0123456789\n"), (newFileReadCap/11)+2)
	if err := os.WriteFile(filepath.Join(dir, "big"), big, 0o644); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name      string
		lines     int
		truncated bool
	}{{"nl", 2, false}, {"nonl", 2, false}, {"empty", 0, false}} {
		lines, binary, truncated := countNewFile(filepath.Join(dir, tc.name))
		if lines != tc.lines || binary || truncated != tc.truncated {
			t.Errorf("%s = %d binary=%v truncated=%v, want %d/false/%v", tc.name, lines, binary, truncated, tc.lines, tc.truncated)
		}
	}
	lines, _, truncated := countNewFile(filepath.Join(dir, "big"))
	if !truncated || lines < newFileReadCap/11-1 {
		t.Fatalf("big = %d truncated=%v, want a capped count flagged truncated", lines, truncated)
	}
	if lines, _, _ := countNewFile(filepath.Join(dir, "missing")); lines != 0 {
		t.Fatalf("missing file counted %d lines", lines)
	}
}
