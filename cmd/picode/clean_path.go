package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/cfpperche/picode/internal/hostfs"
	"github.com/cfpperche/picode/internal/install"
)

// toolPath finds a cleanup tool the way the person's own terminal would.
//
// `picode clean` is usually started by the Windows shell through wsl.exe,
// which runs it with the distro's bare PATH: no nvm, no ~/go-sdk, no
// ~/.local/bin. A Go build cache then "cannot be cleaned" because go "is not
// installed" while the person's terminal runs it fine (2026-09-23). So the
// lookup widens in order, stopping at the first hit:
//
//  1. this process's PATH;
//  2. the PATH the PiCode service runs with — the unit's snapshot of the
//     person's PATH at install, and the operator's drop-in over it — the
//     same PATH every agent CLI already resolves through;
//  3. the person's login shell, asked `command -v` (a tool installed after
//     the snapshot).
//
// It returns the absolute path, or "" when none of them has it.
func toolPath(name, home string) string {
	toolMu.Lock()
	defer toolMu.Unlock()
	if p, ok := toolCache[name]; ok {
		return p
	}
	p := findTool(name, home)
	toolCache[name] = p
	return p
}

// toolCache: one lookup per tool per process — a missing tool costs a
// login-shell probe, and a scan asks for each tool more than once.
var (
	toolMu    sync.Mutex
	toolCache = map[string]string{}
)

func findTool(name, home string) string {
	if p, err := exec.LookPath(name); err == nil && !onWindowsDrive(p) {
		return p
	}
	for _, path := range servicePATHs(home) {
		if p := lookIn(name, path); p != "" {
			return p
		}
	}
	return loginShellLookup(name)
}

// onWindowsDrive: a tool under /mnt/<drive> is the Windows install (WSL
// appends the Windows PATH by default). `npm` there is node.exe cleaning
// the Windows cache — the measured ~/.npm would not shrink at all.
func onWindowsDrive(p string) bool { return strings.HasPrefix(p, "/mnt/") }

// lookIn searches one PATH string for an executable name, skipping the
// Windows drives for the same reason as above.
func lookIn(name, path string) string {
	for _, dir := range filepath.SplitList(path) {
		if dir == "" || onWindowsDrive(dir) {
			continue
		}
		p := filepath.Join(dir, name)
		if st, err := os.Stat(p); err == nil && !st.IsDir() && st.Mode()&0o111 != 0 {
			return p
		}
	}
	return ""
}

// servicePATHs are the PATHs the PiCode service may run with, the one
// systemd applies first: the drop-in's (applied last, so it wins), then
// the unit's own snapshot — tried too, since a narrower drop-in PATH must
// not hide a tool the snapshot had. Both are read with systemd's own
// parsing (quotes, several assignments per line), never line-prefix
// guessing.
func servicePATHs(home string) []string {
	var out []string
	if env, err := install.ReadEnvDropIn(home); err == nil && env["PATH"] != "" {
		out = append(out, env["PATH"])
	}
	unit := filepath.Join(home, ".config", "systemd", "user", install.UnitName)
	if b, err := os.ReadFile(unit); err == nil {
		if p := install.ParseEnvironment(string(b))["PATH"]; p != "" {
			out = append(out, p)
		}
	}
	return out
}

// toolName is what may reach the shell below: a bare command name, never
// anything the shell would read as syntax.
var toolName = regexp.MustCompile(`^[A-Za-z0-9._+-]+$`)

// loginShellLookup asks the person's shell, as a login and interactive
// shell (nvm is set up in ~/.bashrc, which a non-interactive shell skips),
// where name lives. Bounded: a slow rc file must not hang a cleanup. Only
// an absolute path on the last line counts — rc files may print banners.
func loginShellLookup(name string) string {
	if !toolName.MatchString(name) {
		return ""
	}
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/bash"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, shell, "-lic", "command -v "+name)
	cmd.Stdin = nil
	// The context kills the shell only; a job an rc file left in the
	// background can hold stdout open and keep Output waiting for it.
	// WaitDelay closes the pipes a second after the kill, so the bound is
	// the bound.
	cmd.WaitDelay = time.Second
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	last := strings.TrimSpace(lines[len(lines)-1])
	if filepath.IsAbs(last) && !onWindowsDrive(last) && lookIn(filepath.Base(last), filepath.Dir(last)) != "" {
		return last
	}
	return ""
}

// withToolDir puts the tool's own directory first on the child's PATH:
// npm is a `#!/usr/bin/env node` script, so finding npm is not enough —
// its node has to be found the same way.
func withToolDir(env []string, tool string) []string {
	dir := filepath.Dir(tool)
	out := make([]string, 0, len(env)+1)
	set := false
	for _, kv := range env {
		if v, ok := strings.CutPrefix(kv, "PATH="); ok {
			out = append(out, "PATH="+dir+string(os.PathListSeparator)+v)
			set = true
			continue
		}
		out = append(out, kv)
	}
	if !set {
		out = append(out, "PATH="+dir)
	}
	return out
}

// locatedConsumers is the cache table with each tool-owned cache measured
// where its tool says it is (hostfs.LocateConsumers): the tool is found the
// way toolPath finds it, asked with a bound, and only its last output line
// counts — some tools print a notice first.
func locatedConsumers(home string) []hostfs.Consumer {
	return hostfs.LocateConsumers(hostfs.Consumers(home), func(argv []string) (string, error) {
		tool := toolPath(argv[0], home)
		if tool == "" {
			return "", fmt.Errorf("%s is not installed", argv[0])
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, tool, argv[1:]...)
		// From home, not from whatever project the caller stands in: a
		// go.mod asking for another toolchain would make `go env` download
		// one, a project .npmrc or pnpm workspace would answer for that
		// project. And never a download or a prompt from a shim.
		cmd.Dir = home
		cmd.Env = append(withToolDir(os.Environ(), tool), "GOTOOLCHAIN=local", "COREPACK_ENABLE_DOWNLOAD_PROMPT=0", "COREPACK_ENABLE_STRICT=0")
		cmd.WaitDelay = time.Second
		out, err := cmd.Output()
		if err != nil {
			return "", err
		}
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		return strings.TrimSpace(lines[len(lines)-1]), nil
	}, home)
}
