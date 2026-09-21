package backup

import (
	"os"
	"path/filepath"
	"testing"
)

// Restoring a directory or a file is a swap, not a delete followed by a
// copy. The rows below are the conditions that change the outcome:
//
//	live exists | copy succeeds | outcome
//	------------|---------------|---------------------------------------
//	yes         | yes           | target is the snapshot; old one dropped
//	yes         | no            | target untouched, nothing half-written
//	no          | yes           | target created
//
// The remaining branch — the final rename failing after the live tree was
// moved aside — is the rollback that puts it back. Forcing it needs the
// parent directory to become unwritable between two renames in the same
// function, which would block the staging copy too, so it stays covered
// by inspection rather than by a test that cannot be written honestly.

func writeTree(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for rel, body := range files {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func readTree(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		out[rel] = string(b)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestSwapTreeReplacesLiveContent(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "snap")
	dst := filepath.Join(dir, "live")
	writeTree(t, src, map[string]string{"a.md": "from snapshot", "sub/b.md": "nested"})
	writeTree(t, dst, map[string]string{"a.md": "live", "gone.md": "was here"})

	if err := swapTree(src, dst); err != nil {
		t.Fatal(err)
	}
	got := readTree(t, dst)
	if got["a.md"] != "from snapshot" || got["sub/b.md"] != "nested" {
		t.Fatalf("snapshot content not in place: %v", got)
	}
	if _, ok := got["gone.md"]; ok {
		t.Fatal("a file only the live tree had survived the swap")
	}
	for _, leftover := range []string{dst + ".restoring", dst + ".replaced"} {
		if _, err := os.Stat(leftover); err == nil {
			t.Fatalf("%s left behind", filepath.Base(leftover))
		}
	}
}

// The regression this whole change is about: before, the live tree was
// RemoveAll'd first, so a copy that died halfway destroyed it.
func TestSwapTreeLeavesLiveTreeIntactWhenTheCopyFails(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "snap")
	dst := filepath.Join(dir, "live")
	writeTree(t, src, map[string]string{"ok.md": "fine"})
	// A dangling symlink: Walk reports it as a file, copyRegular opens it
	// and gets ENOENT. No permission games, same result on every runner.
	if err := os.Symlink(filepath.Join(dir, "nothing-here"), filepath.Join(src, "broken.md")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	live := map[string]string{"keep.md": "precious", "sub/also.md": "precious too"}
	writeTree(t, dst, live)

	if err := swapTree(src, dst); err == nil {
		t.Fatal("swapTree reported success on an unreadable source")
	}
	if got := readTree(t, dst); len(got) != len(live) || got["keep.md"] != "precious" || got["sub/also.md"] != "precious too" {
		t.Fatalf("live tree was damaged by a failed restore: %v", got)
	}
	if _, err := os.Stat(dst + ".restoring"); err == nil {
		t.Fatal("staging directory left behind")
	}
}

func TestSwapTreeCreatesAMissingTarget(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "snap")
	dst := filepath.Join(dir, "live")
	writeTree(t, src, map[string]string{"a.md": "new"})

	if err := swapTree(src, dst); err != nil {
		t.Fatal(err)
	}
	if got := readTree(t, dst); got["a.md"] != "new" {
		t.Fatalf("got %v", got)
	}
}

func TestSwapRegularKeepsOrDiscardsWhatItReplaced(t *testing.T) {
	for _, tc := range []struct {
		name     string
		keep     bool
		wantKept bool
	}{
		{"vault keeps the file it replaced", keepReplaced, true},
		{"pi's own files leave no debris", discardReplaced, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			src := filepath.Join(dir, "snap.json")
			dst := filepath.Join(dir, "live.json")
			if err := os.WriteFile(src, []byte(`{"from":"snapshot"}`), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(dst, []byte(`{"from":"live"}`), 0o600); err != nil {
				t.Fatal(err)
			}

			if err := swapRegular(src, dst, 0o600, tc.keep); err != nil {
				t.Fatal(err)
			}
			if b, _ := os.ReadFile(dst); string(b) != `{"from":"snapshot"}` {
				t.Fatalf("target = %s", b)
			}
			b, err := os.ReadFile(dst + ".replaced")
			if tc.wantKept {
				if err != nil {
					t.Fatal("the replaced file was not kept — a vault that will not decrypt has nothing to go back to")
				}
				if string(b) != `{"from":"live"}` {
					t.Fatalf(".replaced = %s", b)
				}
			} else if err == nil {
				t.Fatal(".replaced left in a directory PiCode does not own")
			}
			if _, err := os.Stat(dst + ".restoring"); err == nil {
				t.Fatal("staging file left behind")
			}
		})
	}
}

func TestSwapRegularLeavesTheLiveFileWhenTheCopyFails(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "live.json")
	if err := os.WriteFile(dst, []byte(`{"from":"live"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := swapRegular(filepath.Join(dir, "absent.json"), dst, 0o600, keepReplaced); err == nil {
		t.Fatal("swapRegular reported success on a missing source")
	}
	if b, _ := os.ReadFile(dst); string(b) != `{"from":"live"}` {
		t.Fatalf("live file was damaged: %s", b)
	}
	if _, err := os.Stat(dst + ".restoring"); err == nil {
		t.Fatal("staging file left behind")
	}
}
