package server

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/cfpperche/picode/internal/gitclone"
)

// makeLocalRepo builds a committed git repo the clone tests use as origin.
func makeLocalRepo(t *testing.T) string {
	t.Helper()
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
	if err := os.WriteFile(filepath.Join(src, "readme.md"), []byte("hi\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "readme.md")
	run("commit", "-q", "-m", "init")
	return src
}

// stubCloneToLocal redirects cloneFn to clone from src regardless of the
// requested URL — the handler's ParseRemote refuses file:// on purpose, so
// tests exercise the wiring with a local origin standing in for the remote.
func stubCloneToLocal(t *testing.T, src string) {
	t.Helper()
	orig := cloneFn
	cloneFn = func(ctx context.Context, rem gitclone.Remote, dest string) error {
		return gitclone.Clone(ctx, gitclone.Remote{URL: "file://" + src, Name: rem.Name, Branch: rem.Branch}, dest)
	}
	t.Cleanup(func() { cloneFn = orig })
}

func TestCloneWorkspace(t *testing.T) {
	src := makeLocalRepo(t)
	stubCloneToLocal(t, src)
	ts := newTestServer(t, "cat")
	dest := filepath.Join(t.TempDir(), "proj")

	res := postJSON(t, ts, "/api/workspaces/clone",
		map[string]string{"url": "https://example.com/octo/proj.git", "name": "", "path": dest})
	if res.StatusCode != 201 {
		t.Fatalf("clone = %d", res.StatusCode)
	}
	var wk workspaceView
	if err := json.NewDecoder(res.Body).Decode(&wk); err != nil {
		t.Fatalf("decode: %v", err)
	}
	res.Body.Close()
	if wk.Name != "proj" {
		t.Fatalf("name not derived from repo: %q", wk.Name)
	}
	if wk.Path != dest {
		t.Fatalf("path = %q, want %q", wk.Path, dest)
	}
	if _, err := os.Stat(filepath.Join(dest, "readme.md")); err != nil {
		t.Fatalf("cloned file missing: %v", err)
	}
	if len(wk.Agents) != 0 {
		t.Fatalf("workspace should start empty (ADR-0027), has %d agents", len(wk.Agents))
	}
}

func TestCloneWorkspaceAdoptsSameOrigin(t *testing.T) {
	src := makeLocalRepo(t)
	stubCloneToLocal(t, src)
	ts := newTestServer(t, "cat")
	dest := filepath.Join(t.TempDir(), "proj")

	// First clone for real, then POST again at the same destination: the
	// origin matches (file://src), so the handler adopts instead of failing.
	ctx := context.Background()
	if err := gitclone.Clone(ctx, gitclone.Remote{URL: "file://" + src, Name: "proj"}, dest); err != nil {
		t.Fatalf("seed clone: %v", err)
	}
	res := postJSON(t, ts, "/api/workspaces/clone",
		map[string]string{"url": "file://" + src, "name": "proj", "path": dest})
	if res.StatusCode != 400 {
		t.Fatalf("file:// url should be rejected by ParseRemote, got %d", res.StatusCode)
	}
	res.Body.Close()

	// A same-origin adopt needs the origin to normalize equal to the URL the
	// user pasted. Rewrite origin to the https spelling and adopt via that.
	cmd := exec.Command("git", "-C", dest, "remote", "set-url", "origin", "git@example.com:octo/proj.git")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("set-url: %v\n%s", err, out)
	}
	res = postJSON(t, ts, "/api/workspaces/clone",
		map[string]string{"url": "https://example.com/octo/proj", "name": "proj", "path": dest})
	if res.StatusCode != 200 {
		t.Fatalf("adopt = %d", res.StatusCode)
	}
	var got struct {
		workspaceView
		Adopted bool `json:"adopted"`
	}
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	res.Body.Close()
	if !got.Adopted {
		t.Fatal("expected adopted:true")
	}
}

func TestCloneWorkspaceErrors(t *testing.T) {
	src := makeLocalRepo(t)
	stubCloneToLocal(t, src)
	ts := newTestServer(t, "cat")

	// Invalid URL → 400.
	res := postJSON(t, ts, "/api/workspaces/clone",
		map[string]string{"url": "not a url", "path": filepath.Join(t.TempDir(), "x")})
	if res.StatusCode != 400 {
		t.Fatalf("bad url = %d, want 400", res.StatusCode)
	}
	res.Body.Close()

	// Missing destination → 400.
	res = postJSON(t, ts, "/api/workspaces/clone",
		map[string]string{"url": "https://example.com/o/r.git", "path": ""})
	if res.StatusCode != 400 {
		t.Fatalf("empty path = %d, want 400", res.StatusCode)
	}
	res.Body.Close()

	// Occupied destination with unrelated content → 409.
	occupied := t.TempDir()
	if err := os.WriteFile(filepath.Join(occupied, "other.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	res = postJSON(t, ts, "/api/workspaces/clone",
		map[string]string{"url": "https://example.com/o/r.git", "path": occupied})
	if res.StatusCode != 409 {
		t.Fatalf("occupied dest = %d, want 409", res.StatusCode)
	}
	res.Body.Close()
}
