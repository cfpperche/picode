package server

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestBrowseIgnored(t *testing.T) {
	root := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		if out, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git: %s: %v", out, err)
		}
	}
	git("init")
	write := func(path, body string) {
		t.Helper()
		p := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write(".gitignore", "*.log\n!keep.log\nartifacts/\n")
	for _, p := range []string{"debug.log", "keep.log", "tracked.log", "normal.txt", "space name.log", "line\nbreak.log", "artifacts/a.txt", "nested/a.log"} {
		write(p, "test")
	}
	git("add", "-f", "tracked.log")
	for _, tc := range []struct {
		dir, path string
		ignored   bool
	}{
		{"", "debug.log", true}, {"", "keep.log", false}, {"", "tracked.log", false}, {"", "normal.txt", false},
		{"", "space name.log", true}, {"", "line\nbreak.log", true}, {"", "artifacts", true},
		{"artifacts", "artifacts/a.txt", true}, {"nested", "nested/a.log", true},
	} {
		t.Run(tc.path, func(t *testing.T) {
			out, err := browseAgentDir(root, tc.dir)
			if err != nil {
				t.Fatal(err)
			}
			for _, key := range []string{"dirs", "files"} {
				for _, hit := range out[key].([]browseHit) {
					if hit.Path == tc.path {
						if hit.Ignored != tc.ignored {
							t.Fatalf("ignored=%v, want %v", hit.Ignored, tc.ignored)
						}
						return
					}
				}
			}
			t.Fatal("missing entry")
		})
	}
	out, err := browseAgentDir(filepath.Join(root, "nested"), "")
	if err != nil {
		t.Fatal(err)
	}
	if !out["files"].([]browseHit)[0].Ignored {
		t.Fatal("nested browse root lost parent ignore rules")
	}
	plain := t.TempDir()
	if err := os.WriteFile(filepath.Join(plain, "normal.txt"), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}
	out, err = browseAgentDir(plain, "")
	if err != nil || out["files"].([]browseHit)[0].Ignored {
		t.Fatal("non-Git browsing failed")
	}
	t.Setenv("PATH", t.TempDir())
	out, err = browseAgentDir(root, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, hit := range out["files"].([]browseHit) {
		if hit.Ignored {
			t.Fatal("unavailable Git should keep ordinary browsing")
		}
	}
}
