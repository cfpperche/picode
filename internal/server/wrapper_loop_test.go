package server

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Two PiCode instances' bin dirs on one PATH (a scratch terminal opened
// inside PiCode): each wrapper used to skip only its own dir, found the other
// instance's wrapper, and the two exec'd each other forever — one pid at full
// CPU for half an hour (2026-09-25). Every wrapper flavour must reach the real
// binary past any other PiCode wrapper.
func TestWrappersSkipOtherInstancesWrappers(t *testing.T) {
	root := t.TempDir()
	realDir := filepath.Join(root, "real")
	for _, n := range []string{"tmux", "muse", "xdg-open"} {
		if err := writeExecutable(filepath.Join(realDir, n), "#!/bin/sh\nprintf 'REAL %s\\n' \"$*\"\n"); err != nil {
			t.Fatal(err)
		}
	}
	a, b := filepath.Join(root, "a"), filepath.Join(root, "b")
	for _, d := range []string{a, b} {
		if err := installTmuxGuard(d); err != nil {
			t.Fatal(err)
		}
		cli := "#!/bin/sh\n# PiCode test integration.\nname=muse\n" + wrapperFindReal + "exec \"$real\" \"$@\"\n"
		if err := writeExecutable(wrapperPath(d, "muse"), cli); err != nil {
			t.Fatal(err)
		}
		opener := strings.Replace(openURLWrapper, "__NAME__", "xdg-open", 1)
		if err := writeExecutable(wrapperPath(d, "xdg-open"), opener); err != nil {
			t.Fatal(err)
		}
	}
	path := interceptBinDir(a) + string(os.PathListSeparator) + interceptBinDir(b) + string(os.PathListSeparator) + realDir + string(os.PathListSeparator) + "/usr/bin:/bin"
	for _, c := range []struct {
		bin  string
		args []string
	}{
		{"tmux", []string{"list-sessions"}},
		{"muse", []string{"--version"}},
		{"xdg-open", []string{"file.txt"}},
	} {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		cmd := exec.CommandContext(ctx, wrapperPath(a, c.bin), c.args...)
		cmd.Env = []string{"PATH=" + path, "HOME=" + root}
		out, err := cmd.CombinedOutput()
		looped := ctx.Err() == context.DeadlineExceeded
		cancel()
		if looped {
			t.Fatalf("%s: the wrapper looped (no answer in 10s)", c.bin)
		}
		if err != nil || !strings.Contains(string(out), "REAL") {
			t.Fatalf("%s: %v %q, want the real binary", c.bin, err, out)
		}
	}
}
