package gitgraph

import (
	"os"
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
