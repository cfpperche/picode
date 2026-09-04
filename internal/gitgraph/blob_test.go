package gitgraph

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// runGit runs one git command in dir, failing the test on error.
func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// blobRepo is a repository with one committed asset and one working change on
// top: img.png is "v1\n" at HEAD and "v2\n" in the tree — the shape a binary
// row in the graph's Uncommitted Changes meets.
func blobRepo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git missing")
	}
	dir := t.TempDir()
	runGit(t, dir, "init", "-b", "main")
	runGit(t, dir, "config", "user.email", "t@t")
	runGit(t, dir, "config", "user.name", "t")
	if err := os.WriteFile(filepath.Join(dir, "img.png"), []byte("v1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "asset v1")
	if err := os.WriteFile(filepath.Join(dir, "img.png"), []byte("v2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// A committed asset comes back byte-exact through both its hash and HEAD.
func TestBlobReadsCommittedAsset(t *testing.T) {
	dir := blobRepo(t)
	want := []byte("v1\n")
	for _, hash := range []string{headHash(t, dir), "HEAD"} {
		got, err := Blob(dir, hash, "img.png")
		if err != nil {
			t.Fatalf("Blob(%q): %v", hash, err)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("Blob(%q) = %q, want %q", hash, got, want)
		}
	}
}

// An asset living only in a subdirectory resolves like any repo path.
func TestBlobReadsNestedPath(t *testing.T) {
	dir := blobRepo(t)
	runGit(t, dir, "checkout", "--", "img.png")
	if err := os.MkdirAll(filepath.Join(dir, "docs-videos", "stills"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "docs-videos", "stills", "a.png"), []byte("deep"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "nested")
	got, err := Blob(dir, "HEAD", "docs-videos/stills/a.png")
	if err != nil || !bytes.Equal(got, []byte("deep")) {
		t.Fatalf("nested blob = %q, %v", got, err)
	}
}

// A blob whose last byte is whitespace must survive the round trip — the
// text-oriented git() helper trims newlines and would corrupt media here.
func TestBlobKeepsTrailingWhitespaceBytes(t *testing.T) {
	dir := blobRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "img2.png"), []byte("v2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "asset ends in newline")
	got, err := Blob(dir, "HEAD", "img2.png")
	if err != nil || !bytes.Equal(got, []byte("v2\n")) {
		t.Fatalf("trailing newline lost: %q, %v", got, err)
	}
}

func TestBlobRefusesBadInput(t *testing.T) {
	dir := blobRepo(t)
	hash := headHash(t, dir)
	cases := []struct {
		name string
		hash string
		path string
		want error
	}{
		{"short hash", "abc123", "img.png", ErrBadHash},
		{"ref name instead of hash", "main", "img.png", ErrBadHash},
		{"flag-shaped hash", "-x", "img.png", ErrBadHash},
		{"empty path", hash, "", ErrBadPath},
		{"climbs out", hash, "../secret.png", ErrBadPath},
		{"absolute", hash, "/etc/passwd", ErrBadPath},
		{"flag-shaped path", hash, "-o", ErrBadPath},
	}
	for _, c := range cases {
		if _, err := Blob(dir, c.hash, c.path); !errors.Is(err, c.want) {
			t.Fatalf("%s: err = %v, want %v", c.name, err, c.want)
		}
	}
}

func TestBlobMissingAssetAndRepo(t *testing.T) {
	dir := blobRepo(t)
	if _, err := Blob(dir, headHash(t, dir), "gone.png"); !errors.Is(err, ErrNoBlob) {
		t.Fatalf("missing file: err = %v, want ErrNoBlob", err)
	}
	if _, err := Blob(t.TempDir(), "HEAD", "img.png"); !errors.Is(err, ErrNoRepo) {
		t.Fatalf("no repo: err = %v, want ErrNoRepo", err)
	}
}

func TestBlobCapsSize(t *testing.T) {
	dir := blobRepo(t)
	old := MaxBlobBytes
	MaxBlobBytes = 1
	t.Cleanup(func() { MaxBlobBytes = old })
	if _, err := Blob(dir, "HEAD", "img.png"); !errors.Is(err, ErrTooBig) {
		t.Fatalf("err = %v, want ErrTooBig", err)
	}
}

func TestValidBlobPath(t *testing.T) {
	cases := map[string]bool{
		"a.png": true, "d/s/a.png": true, "a b.png": true, "a\\b.png": true,
		"": false, "-a.png": false, "/a.png": false, "a/../b.png": false, "..": false,
	}
	for p, want := range cases {
		if got := validBlobPath(p); got != want {
			t.Fatalf("validBlobPath(%q) = %v, want %v", p, got, want)
		}
	}
}

func TestSplitPatchReadsStatus(t *testing.T) {
	patch := strings.Join([]string{
		"diff --git a/docs-videos/assets/stills/v2-3-inbox.png b/docs-videos/assets/stills/v2-3-inbox.png",
		"index 0f4e6c1..30d7d33 100644",
		"GIT binary patch",
		"",
		"diff --git a/new.png b/new.png",
		"new file mode 100644",
		"index 0000000..9d2f1c0",
		"GIT binary patch",
		"",
		"diff --git a/old.png b/old.png",
		"deleted file mode 100644",
		"index 9d2f1c0..0000000",
		"GIT binary patch",
		"",
		"diff --git a/before.png b/after.png",
		"similarity index 95%",
		"rename from before.png",
		"rename to after.png",
		"",
	}, "\n")
	byPath := map[string]FileDiff{}
	for _, f := range splitPatch(patch) {
		byPath[f.Path] = f
	}
	want := map[string]string{
		"docs-videos/assets/stills/v2-3-inbox.png": "",
		"new.png":   "added",
		"old.png":   "deleted",
		"after.png": "renamed",
	}
	for p, s := range want {
		if byPath[p].Status != s {
			t.Fatalf("%s status = %q, want %q", p, byPath[p].Status, s)
		}
	}
	if byPath["after.png"].OldPath != "before.png" {
		t.Fatalf("rename oldPath = %q", byPath["after.png"].OldPath)
	}
	if !byPath["docs-videos/assets/stills/v2-3-inbox.png"].Binary {
		t.Fatal("GIT binary patch must set Binary")
	}
}
