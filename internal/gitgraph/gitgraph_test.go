package gitgraph

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
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

func TestShowRefusesAnythingButAFullHash(t *testing.T) {
	dir := repo(t)
	// The hash is the only user-controlled part of the command line. A leading
	// dash would be read by git as a flag, so nothing but hex may reach it.
	for _, bad := range []string{"", "HEAD", "main", "--help", "-n1", "../etc", "abc123", strings.Repeat("z", 40)} {
		if got := Show(dir, bad); got != nil {
			t.Fatalf("Show(%q) must be refused, got %+v", bad, got)
		}
	}
}

func TestShowOrdinaryCommit(t *testing.T) {
	dir := repo(t)
	write(t, dir, "b.txt", "one\ntwo\n")
	run(t, dir, "git", "add", ".")
	run(t, dir, "git", "commit", "-m", "add b\n\nA body line.\nAnd another.")

	head := headHash(t, dir)
	d := Show(dir, head)
	if d == nil {
		t.Fatal("nil detail")
	}
	if d.Subject != "add b" {
		t.Fatalf("subject = %q", d.Subject)
	}
	if d.Body != "A body line.\nAnd another." {
		t.Fatalf("body = %q", d.Body)
	}
	if d.Email == "" || d.At == 0 {
		t.Fatalf("metadata missing: %+v", d)
	}
	if len(d.Files) != 1 || d.Files[0].Path != "b.txt" {
		t.Fatalf("files = %+v", d.Files)
	}
	if !strings.Contains(d.Files[0].Patch, "+one") {
		t.Fatalf("patch missing content: %q", d.Files[0].Patch)
	}
}

// A merge must arrive as a plain patch against the first parent. Left alone,
// git emits a combined diff (`diff --cc`, `@@@`) that a unified-diff reader
// misreads line by line without ever failing.
func TestShowMergeIsPlainNotCombined(t *testing.T) {
	dir := repo(t)
	run(t, dir, "git", "checkout", "-q", "-b", "side")
	write(t, dir, "side.txt", "side\n")
	run(t, dir, "git", "add", ".")
	run(t, dir, "git", "commit", "-m", "side work")
	run(t, dir, "git", "checkout", "-q", "main")
	write(t, dir, "main.txt", "main\n")
	run(t, dir, "git", "add", ".")
	run(t, dir, "git", "commit", "-m", "main work")
	run(t, dir, "git", "merge", "--no-ff", "side", "-m", "merge side")

	d := Show(dir, headHash(t, dir))
	if d == nil || len(d.Parents) != 2 {
		t.Fatalf("expected a merge: %+v", d)
	}
	if len(d.Files) == 0 {
		t.Fatal("a merge against its first parent still has changes")
	}
	for _, f := range d.Files {
		if strings.Contains(f.Patch, "diff --cc") || strings.Contains(f.Patch, "@@@") {
			t.Fatalf("combined diff leaked into %s:\n%s", f.Path, f.Patch)
		}
	}
	if d.Files[0].Path != "side.txt" {
		t.Fatalf("first-parent diff should bring the side file, got %+v", d.Files)
	}
}

func TestShowRootCommit(t *testing.T) {
	dir := repo(t)
	root := strings.TrimSpace(output(t, dir, "git", "rev-list", "--max-parents=0", "HEAD"))
	d := Show(dir, root)
	if d == nil || len(d.Parents) != 0 {
		t.Fatalf("root commit: %+v", d)
	}
	if len(d.Files) != 1 || d.Files[0].Path != "a" {
		t.Fatalf("root commit must still show its files: %+v", d.Files)
	}
}

func TestShowRenameAndDelete(t *testing.T) {
	dir := repo(t)
	write(t, dir, "old.txt", strings.Repeat("stable line\n", 20))
	run(t, dir, "git", "add", ".")
	run(t, dir, "git", "commit", "-m", "add old")
	run(t, dir, "git", "mv", "old.txt", "new.txt")
	run(t, dir, "git", "rm", "-q", "a")
	run(t, dir, "git", "commit", "-m", "rename and delete")

	d := Show(dir, headHash(t, dir))
	if d == nil {
		t.Fatal("nil")
	}
	var renamed, deleted *FileDiff
	for i := range d.Files {
		switch d.Files[i].Path {
		case "new.txt":
			renamed = &d.Files[i]
		case "a":
			deleted = &d.Files[i]
		}
	}
	if renamed == nil || renamed.OldPath != "old.txt" {
		t.Fatalf("rename not carried: %+v", d.Files)
	}
	if deleted == nil {
		t.Fatalf("delete missing: %+v", d.Files)
	}
}

func TestSplitPatchShapes(t *testing.T) {
	patch := strings.Join([]string{
		"diff --git a/keep.txt b/keep.txt",
		"index 1111111..2222222 100644",
		"--- a/keep.txt",
		"+++ b/keep.txt",
		"@@ -1 +1 @@",
		"-old",
		"+new",
		"diff --git a/gone.txt b/gone.txt",
		"deleted file mode 100644",
		"--- a/gone.txt",
		"+++ /dev/null",
		"diff --git a/pic.png b/pic.png",
		"index 3333333..4444444 100644",
		"Binary files a/pic.png and b/pic.png differ",
	}, "\n")

	files := splitPatch(patch)
	if len(files) != 3 {
		t.Fatalf("expected 3 files, got %d: %+v", len(files), files)
	}
	if files[0].Path != "keep.txt" || files[0].OldPath != "" {
		t.Fatalf("unchanged path should not repeat as oldPath: %+v", files[0])
	}
	// A delete keeps its name from the --- line; +++ is /dev/null.
	if files[1].Path != "gone.txt" && files[1].OldPath != "gone.txt" {
		t.Fatalf("deleted file lost its name: %+v", files[1])
	}
	if !files[2].Binary {
		t.Fatalf("binary not flagged: %+v", files[2])
	}
	if !strings.Contains(files[0].Patch, "+new") || strings.Contains(files[0].Patch, "gone.txt") {
		t.Fatalf("file patches bled into each other: %q", files[0].Patch)
	}
	if len(splitPatch("")) != 0 {
		t.Fatal("an empty patch has no files")
	}
}

func TestHeaderPathsCutsAtTheLastMarker(t *testing.T) {
	for _, c := range []struct{ in, old, new string }{
		{"a/x.go b/x.go", "x.go", "x.go"},
		{"a/old.go b/new.go", "old.go", "new.go"},
		// A directory literally called "b" must not fool the split.
		{"a/src/b/f.go b/src/b/f.go", "src/b/f.go", "src/b/f.go"},
	} {
		o, n := headerPaths(c.in)
		if o != c.old || n != c.new {
			t.Fatalf("headerPaths(%q) = %q,%q want %q,%q", c.in, o, n, c.old, c.new)
		}
	}
}

func headHash(t *testing.T, dir string) string {
	t.Helper()
	return strings.TrimSpace(output(t, dir, "git", "rev-parse", "HEAD"))
}

func output(t *testing.T, dir, name string, args ...string) string {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("%s %v: %v", name, args, err)
	}
	return string(out)
}

// git reports the common dir relative to the directory it ran in — `.git` from
// the top, `../.git` one level down. Resolving that against the work tree root
// instead of against the directory pointed one level above the repository, and
// every caller comparing keys then saw a mismatch that was not there.
func TestKeyIsTheSameFromAnyDepth(t *testing.T) {
	dir := repo(t)
	deep := filepath.Join(dir, "one", "two", "three")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	want := Key(dir)
	if want == "" {
		t.Fatal("no key at the root")
	}
	for _, d := range []string{
		filepath.Join(dir, "one"),
		filepath.Join(dir, "one", "two"),
		deep,
	} {
		if got := Key(d); got != want {
			t.Fatalf("Key(%s) = %q, want %q", d, got, want)
		}
	}
}

// Numstat is what keeps the per-file counts honest: it is computed by git over
// the whole diff, not by us over whatever survived the patch cap.
func TestShowNumstatOrdinary(t *testing.T) {
	dir := repo(t)
	write(t, dir, "n.txt", "one\ntwo\nthree\n")
	run(t, dir, "git", "add", ".")
	run(t, dir, "git", "commit", "-m", "add n")
	write(t, dir, "n.txt", "one\nTWO\n")
	run(t, dir, "git", "commit", "-am", "edit n")

	d := Show(dir, headHash(t, dir))
	if d == nil || len(d.Files) != 1 {
		t.Fatalf("files = %+v", d)
	}
	if d.Files[0].Add != 1 || d.Files[0].Del != 2 {
		t.Fatalf("numstat = +%d -%d, want +1 -2", d.Files[0].Add, d.Files[0].Del)
	}
}

func TestShowNumstatMergeFollowsFirstParent(t *testing.T) {
	dir := repo(t)
	run(t, dir, "git", "checkout", "-q", "-b", "side")
	write(t, dir, "side.txt", "s1\ns2\n")
	run(t, dir, "git", "add", ".")
	run(t, dir, "git", "commit", "-m", "side work")
	run(t, dir, "git", "checkout", "-q", "main")
	write(t, dir, "main.txt", "m\n")
	run(t, dir, "git", "add", ".")
	run(t, dir, "git", "commit", "-m", "main work")
	run(t, dir, "git", "merge", "--no-ff", "side", "-m", "merge side")

	d := Show(dir, headHash(t, dir))
	if d == nil || len(d.Parents) != 2 {
		t.Fatalf("expected a merge: %+v", d)
	}
	var side *FileDiff
	for i := range d.Files {
		if d.Files[i].Path == "side.txt" {
			side = &d.Files[i]
		}
	}
	if side == nil || side.Add != 2 || side.Del != 0 {
		t.Fatalf("first-parent numstat should count the side file: %+v", d.Files)
	}
}

func TestShowNumstatRename(t *testing.T) {
	dir := repo(t)
	write(t, dir, "old.txt", strings.Repeat("stable line\n", 20))
	run(t, dir, "git", "add", ".")
	run(t, dir, "git", "commit", "-m", "add old")
	run(t, dir, "git", "mv", "old.txt", "new.txt")
	write(t, dir, "new.txt", strings.Repeat("stable line\n", 20)+"tail\n")
	run(t, dir, "git", "add", ".")
	run(t, dir, "git", "commit", "-m", "rename with edit")

	d := Show(dir, headHash(t, dir))
	var renamed *FileDiff
	for i := range d.Files {
		if d.Files[i].Path == "new.txt" {
			renamed = &d.Files[i]
		}
	}
	if renamed == nil || renamed.OldPath != "old.txt" {
		t.Fatalf("rename lost: %+v", d.Files)
	}
	if renamed.Add != 1 || renamed.Del != 0 {
		t.Fatalf("rename numstat = +%d -%d, want +1 -0", renamed.Add, renamed.Del)
	}
}

func TestShowNumstatBinaryStaysZero(t *testing.T) {
	dir := repo(t)
	if err := os.WriteFile(filepath.Join(dir, "blob.bin"), []byte{0, 1, 2, 3, 0, 255}, 0o644); err != nil {
		t.Fatal(err)
	}
	run(t, dir, "git", "add", ".")
	run(t, dir, "git", "commit", "-m", "binary")

	d := Show(dir, headHash(t, dir))
	var bin *FileDiff
	for i := range d.Files {
		if d.Files[i].Path == "blob.bin" {
			bin = &d.Files[i]
		}
	}
	if bin == nil || !bin.Binary {
		t.Fatalf("binary file not flagged: %+v", d.Files)
	}
	if bin.Add != 0 || bin.Del != 0 {
		t.Fatalf("binary numstat must stay 0/0, got +%d -%d", bin.Add, bin.Del)
	}
}

// A truncated patch used to poison the counts, because the browser derived
// them from patch lines. Numstat sees the whole diff regardless of the cap.
func TestShowNumstatSurvivesTruncation(t *testing.T) {
	dir := repo(t)
	var b strings.Builder
	for i := 0; b.Len() < maxPatchBytes+4096; i++ {
		b.WriteString("line ")
		b.WriteString(strconv.Itoa(i))
		b.WriteString(" of a very repetitive generated asset\n")
	}
	lines := strings.Count(b.String(), "\n")
	write(t, dir, "big.txt", b.String())
	run(t, dir, "git", "add", ".")
	run(t, dir, "git", "commit", "-m", "huge")

	d := Show(dir, headHash(t, dir))
	if d == nil || !d.Truncated {
		t.Fatalf("expected a truncated detail, got %+v", d)
	}
	var big *FileDiff
	for i := range d.Files {
		if d.Files[i].Path == "big.txt" {
			big = &d.Files[i]
		}
	}
	if big == nil || big.Add != lines || big.Del != 0 {
		t.Fatalf("numstat should count the whole file: got %+v, want +%d", big, lines)
	}
}
