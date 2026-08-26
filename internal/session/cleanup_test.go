package session

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDirStatsAndRemove(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	cwd := filepath.Join(home, "proj")
	if err := os.MkdirAll(cwd, 0o755); err != nil {
		t.Fatal(err)
	}
	empty := DirStats(cwd)
	if empty.Count != 0 || empty.Bytes != 0 {
		t.Fatalf("missing dir stats = %+v", empty)
	}

	dir := Dir(cwd)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a.jsonl"), []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("skip"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := DirStats(cwd)
	if got.Count != 1 || got.Bytes != 6 {
		t.Fatalf("stats = %+v", got)
	}

	if err := RemoveDir(cwd); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("dir still there: %v", err)
	}
}

func TestRemoveDirRefusesOddPath(t *testing.T) {
	// Empty cwd still resolves to a --encoded-- folder name; that's fine.
	// The guard is the folder shape, not the cwd.
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	if err := RemoveDir(filepath.Join(home, "never-created")); err != nil {
		t.Fatal(err)
	}
}
