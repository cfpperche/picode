//go:build !embedui

package web

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDirDefaultsToTheRepoPathAndHonoursTheOverride(t *testing.T) {
	t.Setenv(DirEnv, "")
	if got := Dir(); got != filepath.FromSlash("internal/web/public") {
		t.Fatalf("Dir() = %q", got)
	}
	t.Setenv(DirEnv, "  /somewhere/else  ")
	if got := Dir(); got != "/somewhere/else" {
		t.Fatalf("override not honoured or not trimmed: %q", got)
	}
}

// A disk build is the one that can legitimately have no UI yet — that is the
// state a fresh clone is in, and the server has to say so rather than 404.
func TestBuiltFollowsWhatIsOnDisk(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(DirEnv, dir)
	if Built() {
		t.Fatal("an empty directory is not a built UI")
	}
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !Built() {
		t.Fatal("index.html is there; Built() must say so")
	}
}

func TestDiskBuildDoesNotClaimToBeEmbedded(t *testing.T) {
	if Embedded() {
		t.Fatal("this build embeds nothing")
	}
}
