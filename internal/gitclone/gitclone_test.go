package gitclone

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestParseRemote(t *testing.T) {
	ok := []struct {
		in           string
		url, name, b string
	}{
		{"https://github.com/octo/hello", "https://github.com/octo/hello", "hello", ""},
		{"https://github.com/octo/hello.git", "https://github.com/octo/hello.git", "hello", ""},
		{"https://github.com/octo/hello/", "https://github.com/octo/hello", "hello", ""},
		{"https://github.com/octo/hello/tree/main", "https://github.com/octo/hello", "hello", "main"},
		{"https://github.com/octo/hello/tree/feat/x/", "https://github.com/octo/hello", "hello", "feat/x"},
		{"git@github.com:octo/hello.git", "git@github.com:octo/hello.git", "hello", ""},
		{"ssh://git@host.tld/org/repo.git", "ssh://git@host.tld/org/repo.git", "repo", ""},
		{"git://host.tld/org/repo", "git://host.tld/org/repo", "repo", ""},
		{"  https://gitlab.com/a/b/c.git  ", "https://gitlab.com/a/b/c.git", "c", ""},
	}
	for _, c := range ok {
		rem, err := ParseRemote(c.in)
		if err != nil {
			t.Fatalf("ParseRemote(%q): %v", c.in, err)
		}
		if rem.URL != c.url || rem.Name != c.name || rem.Branch != c.b {
			t.Fatalf("ParseRemote(%q) = %+v, want url=%q name=%q branch=%q", c.in, rem, c.url, c.name, c.b)
		}
	}

	bad := []string{
		"",
		"   ",
		"--upload-pack=/bin/sh",
		"-https://github.com/o/r",
		"-ogit@github.com:o/r.git",
		"https://github.com/o/r; rm -rf /",
		"https://github.com/o/r $(x)",
		"https://github.com/o/r`x`",
		"url with space",
		"/tmp/some/local/path",
		"~/code/repo",
		"file:///tmp/repo",
		"plainword",
		"https://github.com/",
	}
	for _, in := range bad {
		if _, err := ParseRemote(in); err == nil {
			t.Fatalf("ParseRemote(%q): want error, got none", in)
		}
	}
}

func TestNormalizeForCompare(t *testing.T) {
	cases := [][2]string{
		{"git@github.com:Octo/Hello.git", "https://github.com/octo/hello"},
		{"ssh://git@github.com/octo/hello.git", "https://github.com/octo/hello/"},
		{"https://user@github.com:443/octo/hello.git", "https://github.com/octo/hello"},
	}
	for _, c := range cases {
		if a, b := NormalizeForCompare(c[0]), NormalizeForCompare(c[1]); a != b {
			t.Fatalf("NormalizeForCompare: %q → %q != %q → %q", c[0], a, c[1], b)
		}
	}
	if a, b := NormalizeForCompare("https://github.com/octo/hello"), NormalizeForCompare("https://github.com/octo/other"); a == b {
		t.Fatalf("different repos compare equal: %q", a)
	}
}

func TestClassifyStderr(t *testing.T) {
	cases := map[string]string{
		"fatal: could not read Username for 'https://github.com': terminal prompts disabled": "auth",
		"git@github.com: Permission denied (publickey).":                                     "auth",
		"remote: Repository not found.":                                                      "notfound",
		"fatal: unable to access 'https://x/y.git/': Could not resolve host: x":              "network",
		"something else entirely":                                                            "",
	}
	for in, want := range cases {
		if got := ClassifyStderr(in); got != want {
			t.Fatalf("ClassifyStderr(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCloneLocalRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	src := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", src}, args...)...)
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init", "-q", "-b", "main")
	if err := os.WriteFile(filepath.Join(src, "a.txt"), []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "a.txt")
	run("commit", "-q", "-m", "init")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	dest := filepath.Join(t.TempDir(), "cloned")
	// file:// is rejected by ParseRemote on purpose; the test feeds Clone directly.
	rem := Remote{URL: "file://" + src, Name: "cloned"}
	if err := Clone(ctx, rem, dest); err != nil {
		t.Fatalf("Clone: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dest, "a.txt")); err != nil {
		t.Fatalf("cloned file missing: %v", err)
	}
	if !SameOrigin(dest, "file://"+src) {
		t.Fatalf("SameOrigin: clone does not match its own source")
	}
	if SameOrigin(dest, "https://github.com/other/repo") {
		t.Fatalf("SameOrigin: matched an unrelated URL")
	}

	if err := Clone(ctx, Remote{URL: "file://" + filepath.Join(src, "missing"), Name: "x"},
		filepath.Join(t.TempDir(), "y")); err == nil {
		t.Fatalf("Clone of a missing repo: want error")
	}
}

func TestDirUsable(t *testing.T) {
	d := t.TempDir()
	if ex, _ := DirUsable(filepath.Join(d, "nope")); ex {
		t.Fatal("missing dir reported as existing")
	}
	if ex, empty := DirUsable(d); !ex || !empty {
		t.Fatalf("empty dir: exists=%v empty=%v", ex, empty)
	}
	if err := os.WriteFile(filepath.Join(d, "f"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if ex, empty := DirUsable(d); !ex || empty {
		t.Fatalf("non-empty dir: exists=%v empty=%v", ex, empty)
	}
	if ex, empty := DirUsable(filepath.Join(d, "f")); !ex || empty {
		t.Fatalf("file: exists=%v empty=%v", ex, empty)
	}
}
