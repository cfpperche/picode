package gitinfo

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestInspectNotRepo(t *testing.T) {
	if Inspect(t.TempDir()) != nil {
		t.Fatal("expected nil")
	}
}

func TestInspectBranch(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git missing")
	}
	dir := t.TempDir()
	run(t, dir, "git", "init", "-b", "main")
	run(t, dir, "git", "config", "user.email", "t@t")
	run(t, dir, "git", "config", "user.name", "t")
	if err := os.WriteFile(filepath.Join(dir, "a"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	run(t, dir, "git", "add", ".")
	run(t, dir, "git", "commit", "-m", "i")
	info := Inspect(dir)
	if info == nil || info.Branch != "main" {
		t.Fatalf("%+v", info)
	}
	if info.Worktree != "" {
		t.Fatalf("primary worktree should be empty: %+v", info)
	}
}

func run(t *testing.T, dir, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s %v: %v\n%s", name, args, err, out)
	}
}
