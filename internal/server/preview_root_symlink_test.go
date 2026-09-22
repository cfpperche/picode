package server

import (
	"os"
	"path/filepath"
	"testing"
)

// A root reached through a symlink is still that root.
//
// previewWithin canonicalises the requested file, so it has to canonicalise
// the root as well or the two are never comparable: with a linked ancestor
// every file in the project resolved "outside" and the pane answered 404 for
// its own files. That is not an exotic case — on macOS TMPDIR lives under
// /var, which is a symlink to /private/var, so it is every path there.
//
// The second half is the part that must not regress: canonicalising both
// sides is meant to fix a false refusal, never to allow an escape.
func TestPreviewRootReachedThroughASymlink(t *testing.T) {
	real := t.TempDir()
	if err := os.WriteFile(filepath.Join(real, "index.html"), []byte("<h1>hi</h1>"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "root-link")
	if err := os.Symlink(real, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	got, err := previewWithin(link, filepath.Join(link, "index.html"))
	if err != nil {
		t.Fatalf("a file inside the root was refused through the root's own symlink: %v", err)
	}
	want, _ := filepath.EvalSymlinks(filepath.Join(real, "index.html"))
	if got != want {
		t.Fatalf("resolved to %q, want %q", got, want)
	}

	// An escape is still an escape, whichever spelling the root arrived in.
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret"), []byte("no"), 0o600); err != nil {
		t.Fatal(err)
	}
	bridge := filepath.Join(real, "bridge")
	if err := os.Symlink(filepath.Join(outside, "secret"), bridge); err != nil {
		t.Fatal(err)
	}
	if _, err := previewWithin(link, bridge); err == nil {
		t.Fatal("a symlink out of the root was served — the containment guard is gone")
	}
}
