//go:build !windows

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// write creates a file (and its directory) or fails the test.
func write(t *testing.T, path, body string, mode os.FileMode) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), mode); err != nil {
		t.Fatal(err)
	}
	return path
}

// fakeTool writes an executable named name into dir.
func fakeTool(t *testing.T, dir, name string) string {
	t.Helper()
	return write(t, filepath.Join(dir, name), "#!/bin/sh\n", 0o755)
}

// hermetic points PATH and SHELL at fakes, so no case reaches the real
// rc files or the real login shell.
func hermetic(t *testing.T, home, shellBody string) {
	t.Helper()
	t.Setenv("PATH", filepath.Join(home, "proc"))
	t.Setenv("SHELL", write(t, filepath.Join(home, "sh", "fakesh"), shellBody, 0o755))
}

// TestToolPathWidensLikeTheTerminal is the decision table for where a
// cleanup tool is found when picode runs with wsl.exe's bare PATH.
//
//	process PATH | drop-in PATH | unit PATH           | login shell | → found in
//	has it       | has it       | has it              | —           | process PATH
//	no           | has it       | has it              | —           | drop-in (systemd applies it last)
//	no           | lacks it     | has it              | —           | unit (a narrow drop-in hides nothing)
//	no           | none         | has it              | —           | unit
//	no           | none         | quoted, with spaces | —           | unit (systemd's own parsing)
//	/mnt/c only  | none         | none                | —           | "" (the Windows tool is never used)
//	no           | none         | none                | banner+path | login shell's last line
//	no           | none         | none                | silent      | ""
func TestToolPathWidensLikeTheTerminal(t *testing.T) {
	const silent = "#!/bin/sh\n"
	cases := []struct {
		name      string
		setup     func(t *testing.T, home string) (want string)
		shellBody string
	}{
		{"process PATH wins", func(t *testing.T, home string) string {
			p := fakeTool(t, filepath.Join(home, "proc"), "gofake")
			dropIn(t, home, `Environment=PATH=`+fakeDir(t, home, "d"))
			unit(t, home, `Environment=PATH=`+fakeDir(t, home, "u"))
			return p
		}, silent},
		{"drop-in over unit", func(t *testing.T, home string) string {
			d := fakeDir(t, home, "d")
			dropIn(t, home, `Environment=PATH=`+d)
			unit(t, home, `Environment=PATH=`+fakeDir(t, home, "u"))
			return filepath.Join(d, "gofake")
		}, silent},
		{"narrow drop-in falls through to the unit", func(t *testing.T, home string) string {
			dropIn(t, home, `Environment=PATH=/usr/bin`)
			u := fakeDir(t, home, "u")
			unit(t, home, `Environment=PATH=`+u)
			return filepath.Join(u, "gofake")
		}, silent},
		{"unit snapshot", func(t *testing.T, home string) string {
			u := fakeDir(t, home, "u")
			unit(t, home, "Environment=HOME="+home+"\nEnvironment=PATH=/nonexistent:"+u)
			return filepath.Join(u, "gofake")
		}, silent},
		{"quoted unit line with a space", func(t *testing.T, home string) string {
			u := fakeDir(t, home, "with space")
			unit(t, home, `Environment="PATH=/nonexistent:`+u+`" "OTHER=x"`)
			return filepath.Join(u, "gofake")
		}, silent},
		{"a Windows-drive tool is never used", func(t *testing.T, home string) string {
			// The process PATH is faked to a /mnt path shape via the unit:
			// lookIn must skip it even when the file exists elsewhere.
			unit(t, home, `Environment=PATH=/mnt/c/Program Files/nodejs`)
			return ""
		}, silent},
		{"login shell answers after a banner", func(t *testing.T, home string) string {
			return fakeTool(t, filepath.Join(home, "late"), "gofake")
		}, "#!/bin/sh\necho 'Welcome to Ubuntu'\necho \"$HOME_LATE/gofake\"\n"},
		{"nowhere", func(t *testing.T, home string) string { return "" }, silent},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME_LATE", filepath.Join(home, "late"))
			hermetic(t, home, tc.shellBody)
			want := tc.setup(t, home)
			if got := findTool("gofake", home); got != want {
				t.Errorf("toolPath = %q, want %q", got, want)
			}
		})
	}
}

func fakeDir(t *testing.T, home, name string) string {
	t.Helper()
	dir := filepath.Join(home, name)
	fakeTool(t, dir, "gofake")
	return dir
}

func unit(t *testing.T, home, env string) {
	t.Helper()
	write(t, filepath.Join(home, ".config", "systemd", "user", "picode.service"), "[Service]\n"+env+"\n", 0o644)
}

func dropIn(t *testing.T, home, env string) {
	t.Helper()
	write(t, filepath.Join(home, ".config", "systemd", "user", "picode.service.d", "env.conf"), "[Service]\n"+env+"\n", 0o644)
}

// TestLoginShellLookupRefusesShellSyntax: only a bare command name reaches
// `sh -lic`, so a cache table entry can never become a shell command. The
// shell is a fake that would print a path — a refusal must not ask it.
func TestLoginShellLookupRefusesShellSyntax(t *testing.T) {
	home := t.TempDir()
	tool := fakeTool(t, filepath.Join(home, "bin"), "x")
	hermetic(t, home, "#!/bin/sh\necho "+tool+"\n")
	for _, name := range []string{"go; rm -rf ~", "$(id)", "a b", "`x`", "", "go\n"} {
		if got := loginShellLookup(name); got != "" {
			t.Errorf("loginShellLookup(%q) = %q, want refusal", name, got)
		}
	}
}

// TestLoginShellLookupIsBounded: an rc file that leaves a job holding
// stdout must not stretch the 5 s bound (the context kills the shell only;
// WaitDelay closes the pipe).
func TestLoginShellLookupIsBounded(t *testing.T) {
	if testing.Short() {
		t.Skip("waits out the bound")
	}
	sleep, err := exec.LookPath("sleep") // before hermetic hides PATH
	if err != nil {
		t.Skip("no sleep binary")
	}
	home := t.TempDir()
	hermetic(t, home, "#!/bin/sh\n"+sleep+" 30 &\n"+sleep+" 30\n")
	start := time.Now()
	if got := loginShellLookup("gofake"); got != "" {
		t.Errorf("got %q from a shell that never answered", got)
	}
	if d := time.Since(start); d > 10*time.Second {
		t.Errorf("lookup took %s; the bound is 5 s plus the 1 s wait delay", d)
	}
}

// TestWithToolDirPutsTheToolFirst: npm is `#!/usr/bin/env node`, so the
// directory npm was found in has to lead the child's PATH for node.
func TestWithToolDirPutsTheToolFirst(t *testing.T) {
	env := withToolDir([]string{"HOME=/h", "PATH=/usr/bin"}, "/n/v24/bin/npm")
	if !strings.Contains(strings.Join(env, "\n"), "PATH=/n/v24/bin:/usr/bin") {
		t.Errorf("env = %v", env)
	}
	env = withToolDir([]string{"HOME=/h"}, "/n/v24/bin/npm")
	if env[len(env)-1] != "PATH=/n/v24/bin" {
		t.Errorf("no PATH: env = %v", env)
	}
}
