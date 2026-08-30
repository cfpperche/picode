package gitgraph

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestKeyNotRepo(t *testing.T) {
	if got := Key(t.TempDir()); got != "" {
		t.Fatalf("expected empty key, got %q", got)
	}
	if Load(t.TempDir(), 10) != nil {
		t.Fatal("expected nil graph")
	}
}

// The whole of ADR-0022 rests on this: a worktree is a different checkout of
// the same repository, so it must answer with the same key.
func TestWorktreeSharesTheKey(t *testing.T) {
	dir := repo(t)
	wt := filepath.Join(t.TempDir(), "side")
	run(t, dir, "git", "worktree", "add", "-b", "side", wt)

	main, side := Key(dir), Key(wt)
	if main == "" || side == "" {
		t.Fatalf("empty key: main=%q side=%q", main, side)
	}
	if main != side {
		t.Fatalf("worktrees must share the key:\n main = %s\n side = %s", main, side)
	}

	// ...and the refs are shared too, which is why one graph serves both.
	g := Load(wt, 50)
	if g == nil {
		t.Fatal("nil graph from the worktree")
	}
	if !hasRef(g.Refs, "main", "head") || !hasRef(g.Refs, "side", "head") {
		t.Fatalf("both branches must be visible from the worktree: %+v", g.Refs)
	}
	if len(g.Worktrees) != 2 {
		t.Fatalf("expected 2 worktrees, got %d: %+v", len(g.Worktrees), g.Worktrees)
	}
}

func TestLoadCommitsAndParents(t *testing.T) {
	dir := repo(t)
	run(t, dir, "git", "checkout", "-b", "feature")
	write(t, dir, "b", "b")
	run(t, dir, "git", "add", ".")
	run(t, dir, "git", "commit", "-m", "second")
	run(t, dir, "git", "checkout", "main")
	run(t, dir, "git", "merge", "--no-ff", "feature", "-m", "merge feature")

	g := Load(dir, 50)
	if g == nil {
		t.Fatal("nil graph")
	}
	if len(g.Commits) != 3 {
		t.Fatalf("expected 3 commits, got %d", len(g.Commits))
	}
	head := g.Commits[0]
	if head.Subject != "merge feature" {
		t.Fatalf("newest first expected, got %q", head.Subject)
	}
	if len(head.Parents) != 2 {
		t.Fatalf("a merge has two parents, got %v", head.Parents)
	}
	if head.At == 0 || head.Author == "" {
		t.Fatalf("author/date not parsed: %+v", head)
	}
	root := g.Commits[len(g.Commits)-1]
	if len(root.Parents) != 0 {
		t.Fatalf("root commit has no parents, got %v", root.Parents)
	}
}

// git preserves a literal 0x1f typed into a commit message (verified: it
// comes back byte for byte in %s). A naive Split would then see six fields
// and drop the commit, losing it from the graph. The subject is the last
// field, so it must absorb the extra separator instead.
func TestSubjectKeepsALiteralSeparator(t *testing.T) {
	dir := repo(t)
	write(t, dir, "c", "c")
	run(t, dir, "git", "add", ".")
	run(t, dir, "git", "commit", "-m", "weird\x1fsubject")

	g := Load(dir, 50)
	if g == nil || len(g.Commits) != 2 {
		t.Fatalf("the commit must not vanish: %+v", g)
	}
	head := g.Commits[0]
	if head.Subject != "weird\x1fsubject" {
		t.Fatalf("subject mangled: %q", head.Subject)
	}
	if head.Author == "" || head.At == 0 {
		t.Fatalf("fields shifted: %+v", head)
	}
}

// A 0x1e in a message splits the record. The tail must be dropped, never
// parsed into a commit whose hash is really the rest of someone's subject.
func TestRecordSeparatorInMessageProducesNoPhantomCommit(t *testing.T) {
	dir := repo(t)
	write(t, dir, "c", "c")
	run(t, dir, "git", "add", ".")
	run(t, dir, "git", "commit", "-m", "split\x1ehere")

	g := Load(dir, 50)
	if g == nil {
		t.Fatal("nil graph")
	}
	for _, c := range g.Commits {
		if !isHash(c.Hash) {
			t.Fatalf("phantom commit with a non-hash id: %+v", c)
		}
	}
}

func TestIsHash(t *testing.T) {
	sha1 := "cf23665181e55e52adba61b7347b924a00c9aa43"
	for _, c := range []struct {
		in   string
		want bool
	}{
		{sha1, true},
		{strings.Repeat("a", 64), true},
		{"", false},
		{sha1[:39], false},
		{strings.ToUpper(sha1), false},
		{"zf23665181e55e52adba61b7347b924a00c9aa43", false},
		{"here\ncf23665181e55e52adba61b7347b924a00c9aa4", false},
	} {
		if got := isHash(c.in); got != c.want {
			t.Fatalf("isHash(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestEmptyRepoIsAStateNotAnError(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git missing")
	}
	dir := t.TempDir()
	run(t, dir, "git", "init", "-b", "main")

	g := Load(dir, 50)
	if g == nil {
		t.Fatal("an initialised repo with no commits is still a repo")
	}
	if len(g.Commits) != 0 {
		t.Fatalf("expected no commits, got %d", len(g.Commits))
	}
	if g.More {
		t.Fatal("no commits cannot mean there are more")
	}
}

func TestMoreWhenLimitIsReached(t *testing.T) {
	dir := repo(t)
	for i := 0; i < 3; i++ {
		write(t, dir, "f", string(rune('a'+i)))
		run(t, dir, "git", "add", ".")
		run(t, dir, "git", "commit", "-m", "c")
	}
	if g := Load(dir, 2); g == nil || !g.More {
		t.Fatalf("limit reached must set More: %+v", g)
	}
	if g := Load(dir, 50); g == nil || g.More {
		t.Fatalf("whole history must not set More: %+v", g)
	}
}

func TestNameFromKey(t *testing.T) {
	for _, c := range []struct{ key, want string }{
		{"/home/u/picode/.git", "picode"},
		{"/srv/mirrors/picode.git", "picode"},
		{"/home/u/picode/.git/", "picode"},
	} {
		if got := repoName(filepath.Clean(c.key)); got != c.want {
			t.Fatalf("repoName(%q) = %q, want %q", c.key, got, c.want)
		}
	}
}

func TestClassifyRef(t *testing.T) {
	for _, c := range []struct {
		in, name, kind string
		ok             bool
	}{
		{"refs/heads/main", "main", "head", true},
		{"refs/heads/feat/x", "feat/x", "head", true},
		{"refs/remotes/origin/main", "origin/main", "remote", true},
		{"refs/tags/v1.0", "v1.0", "tag", true},
		{"refs/remotes/origin/HEAD", "", "", false},
		{"refs/stash", "", "", false},
	} {
		name, kind, ok := classifyRef(c.in)
		if ok != c.ok || name != c.name || kind != c.kind {
			t.Fatalf("classifyRef(%q) = %q,%q,%v want %q,%q,%v",
				c.in, name, kind, ok, c.name, c.kind, c.ok)
		}
	}
}

func TestParseWorktreePorcelain(t *testing.T) {
	dir := repo(t)
	list := loadWorktrees(dir)
	if len(list) != 1 {
		t.Fatalf("expected the primary worktree, got %+v", list)
	}
	if list[0].Branch != "main" {
		t.Fatalf("branch must be short: %+v", list[0])
	}
	if list[0].Head == "" {
		t.Fatalf("head missing: %+v", list[0])
	}

	// Detached HEAD reports no branch, and that is not a failure.
	run(t, dir, "git", "checkout", "--detach")
	list = loadWorktrees(dir)
	if len(list) != 1 || list[0].Branch != "" {
		t.Fatalf("detached worktree must have no branch: %+v", list)
	}
}

func hasRef(refs []Ref, name, kind string) bool {
	for _, r := range refs {
		if r.Name == name && r.Kind == kind {
			return true
		}
	}
	return false
}

func repo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git missing")
	}
	dir := t.TempDir()
	run(t, dir, "git", "init", "-b", "main")
	run(t, dir, "git", "config", "user.email", "t@t")
	run(t, dir, "git", "config", "user.name", "t")
	write(t, dir, "a", "a")
	run(t, dir, "git", "add", ".")
	run(t, dir, "git", "commit", "-m", "first")
	return dir
}

func write(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func run(t *testing.T, dir, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s %v: %v\n%s", name, args, err, out)
	}
}
