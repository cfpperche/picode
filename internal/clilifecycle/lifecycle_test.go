package clilifecycle

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Real installs are symlinks into install roots or wrapper scripts that
// exec the real binary (hermes). Regression for the 2026-09-07 production
// report: every CLI classified unknown because detection never resolved
// links or read the wrapper.
func TestDetectMethodResolvesSymlinksAndWrappers(t *testing.T) {
	dir := t.TempDir()
	write := func(p, body string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	npmTarget := filepath.Join(dir, "nvm", "lib", "node_modules", "@x", "pi", "bin", "pi")
	write(npmTarget, "#!/usr/bin/env node\n")
	nativeTarget := filepath.Join(dir, ".local", "share", "claude", "versions", "2.1.263")
	write(nativeTarget, "binary")
	vendorTarget := filepath.Join(dir, ".grok", "downloads", "grok-1.0.13-linux-x86_64")
	write(vendorTarget, "binary")
	cases := []struct {
		path string
		want Method
	}{
		{npmTarget, MethodNpm},
		{nativeTarget, MethodNative},
		{vendorTarget, MethodVendor},
	}
	for _, c := range cases {
		link := filepath.Join(dir, "bin", filepath.Base(c.path)+"-link")
		if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(c.path, link); err != nil {
			t.Fatal(err)
		}
		if got := DetectMethod(link); got != c.want {
			t.Errorf("symlink %s → %s: got %q, want %q", link, c.path, got, c.want)
		}
	}
	// A plain wrapper script that execs the real venv binary.
	wrapper := filepath.Join(dir, ".local", "bin", "hermes")
	write(wrapper, "#!/usr/bin/env bash\nexec \""+filepath.Join(dir, ".hermes", "hermes-agent", "venv", "bin", "hermes")+"\" \"$@\"\n")
	if got := DetectMethod(wrapper); got != MethodGit {
		t.Errorf("wrapper hermes: got %q, want %q", got, MethodGit)
	}
	// Unrelated binaries stay unknown — no invented lifecycle.
	plain := filepath.Join(dir, "plain")
	write(plain, "#!/bin/sh\necho hi\n")
	if got := DetectMethod(plain); got != MethodUnknown {
		t.Errorf("plain script: got %q, want unknown", got)
	}
	if got := DetectMethod("/nonexistent/binary"); got != MethodUnknown {
		t.Errorf("missing file: got %q, want unknown", got)
	}
	if got := DetectMethod(""); got != MethodUnknown {
		t.Errorf("empty path: got %q, want unknown", got)
	}
}

func TestDetectMethodDecisionTable(t *testing.T) {
	cases := []struct {
		path string
		want Method
	}{
		{"/home/u/.nvm/versions/node/v24.11.1/lib/node_modules/@earendil-works/pi-coding-agent/bin/pi.js", MethodNpm},
		{"/usr/lib/node_modules/@openai/codex/bin/codex.js", MethodNpm},
		{"/home/u/.local/share/claude/versions/2.1.263", MethodNative},
		{"/home/u/.claude/local/claude", MethodUnknown},
		{"/home/u/.grok/downloads/grok-1.0.13-linux-x86_64", MethodVendor},
		{"/home/u/.hermes/hermes-agent/venv/bin/hermes", MethodGit},
		{"/home/u/.bun/install/global/node_modules/opencode-ai/bin/opencode.exe", MethodNpm},
		{"/opt/homebrew/bin/codex", MethodUnknown},
		{"", MethodUnknown},
	}
	for _, c := range cases {
		if got := DetectMethod(c.path); got != c.want {
			t.Errorf("DetectMethod(%q) = %q, want %q", c.path, got, c.want)
		}
	}
}

// Recorded against each vendor's --help on 2026-09-06 (see ADR-0087).
func TestForDecisionTable(t *testing.T) {
	cases := []struct {
		cli, method       string
		ok                bool
		latestFrom        string
		update, reinstall []string
		uninstall         UninstallKind
		uninstallArgs     []string
	}{
		{"pi", "npm", true, "npm", []string{"update"}, []string{"install", "-g", "@earendil-works/pi-coding-agent@latest"}, "npm", []string{"remove", "-g", "@earendil-works/pi-coding-agent"}},
		{"pi", "unknown", false, "", nil, nil, "", nil},
		{"codex", "npm", true, "npm", []string{"update"}, []string{"install", "-g", "@openai/codex@latest"}, "npm", []string{"remove", "-g", "@openai/codex"}},
		{"claude-code", "native", true, "npm", []string{"update"}, []string{"install"}, "guided", nil},
		{"claude-code", "npm", true, "npm", []string{"install", "-g", "@anthropic-ai/claude-code@latest"}, []string{"install", "-g", "@anthropic-ai/claude-code@latest"}, "npm", []string{"remove", "-g", "@anthropic-ai/claude-code"}},
		{"claude-code", "unknown", false, "", nil, nil, "", nil},
		{"grok", "vendor", true, "vendor", []string{"update"}, []string{"update", "--force-reinstall"}, "guided", nil},
		{"grok", "npm", false, "", nil, nil, "", nil},
		{"hermes", "git", true, "vendor", []string{"update", "--yes"}, []string{"update", "--force", "--yes"}, "vendor", []string{"uninstall", "--yes"}},
		{"hermes", "unknown", false, "", nil, nil, "", nil},
		{"opencode", "npm", true, "npm", []string{"upgrade"}, []string{"upgrade"}, "vendor", []string{"uninstall", "--keep-config", "--keep-data", "--force"}},
		{"opencode", "unknown", false, "", nil, nil, "", nil},
	}
	for _, c := range cases {
		p, ok := For(c.cli, Method(c.method))
		if ok != c.ok {
			t.Errorf("For(%s,%s) ok = %v, want %v", c.cli, c.method, ok, c.ok)
			continue
		}
		if !ok {
			continue
		}
		if p.LatestFrom != c.latestFrom {
			t.Errorf("For(%s,%s) LatestFrom = %q, want %q", c.cli, c.method, p.LatestFrom, c.latestFrom)
		}
		if strings.Join(p.UpdateArgs, " ") != strings.Join(c.update, " ") {
			t.Errorf("For(%s,%s) UpdateArgs = %v, want %v", c.cli, c.method, p.UpdateArgs, c.update)
		}
		if strings.Join(p.ReinstallArgs, " ") != strings.Join(c.reinstall, " ") {
			t.Errorf("For(%s,%s) ReinstallArgs = %v, want %v", c.cli, c.method, p.ReinstallArgs, c.reinstall)
		}
		if p.Uninstall != c.uninstall {
			t.Errorf("For(%s,%s) Uninstall = %q, want %q", c.cli, c.method, p.Uninstall, c.uninstall)
		}
		if strings.Join(p.UninstallArgs, " ") != strings.Join(c.uninstallArgs, " ") {
			t.Errorf("For(%s,%s) UninstallArgs = %v, want %v", c.cli, c.method, p.UninstallArgs, c.uninstallArgs)
		}
	}
}

func TestForMissingDecisionTable(t *testing.T) {
	for _, id := range []string{"pi", "codex", "claude-code"} {
		p, ok := ForMissing(id)
		if !ok || p.NpmPackage == "" || p.Method != MethodNpm {
			t.Errorf("ForMissing(%s) = %+v ok=%v, want npm-backed plan", id, p, ok)
			continue
		}
		argv, err := p.Args(ActionInstall)
		if err != nil || strings.Join(argv, " ") != "install -g "+p.NpmPackage+"@latest" {
			t.Errorf("ForMissing(%s) install argv = %v, err %v", id, argv, err)
		}
	}
	for _, id := range []string{"grok", "hermes"} {
		p, ok := ForMissing(id)
		if ok {
			t.Errorf("ForMissing(%s) = %+v, want guided (ok=false)", id, p)
		}
		if InstallDocs(id) == "" {
			t.Errorf("InstallDocs(%s) empty", id)
		}
	}
	if _, ok := ForMissing("nope"); ok {
		t.Error("unknown CLI must refuse install")
	}
	// The install action refuses on plans without an npm package (vendor
	// methods) — only ForMissing plans carry install argv.
	grokPlan, _ := For("grok", MethodVendor)
	if _, err := grokPlan.Args(ActionInstall); err == nil {
		t.Error("vendor plan must refuse install argv")
	}
}

func TestPlanArgsRefusals(t *testing.T) {
	guided, _ := For("grok", MethodVendor)
	if _, err := guided.Args(ActionUninstall); err == nil {
		t.Error("guided uninstall must refuse argv")
	}
	unknown := Plan{CLI: "codex", Method: MethodUnknown}
	for _, a := range []Action{ActionUpdate, ActionReinstall, ActionUninstall} {
		if _, err := unknown.Args(a); err == nil {
			t.Errorf("unknown method must refuse %s", a)
		}
	}
	if _, err := guided.Args("drop"); err == nil {
		t.Error("unknown action must refuse")
	}
}

func TestParseGrokCheck(t *testing.T) {
	good := []byte(`{"currentVersion":"1.0.13","latestVersion":"1.0.14","updateAvailable":true,"installer":"internal","channel":"stable","autoUpdate":true,"error":null}`)
	c, err := ParseGrokCheck(good)
	if err != nil || c.LatestVersion != "1.0.14" || !c.UpdateAvailable {
		t.Fatalf("ParseGrokCheck good = %+v, err %v", c, err)
	}
	if _, err := ParseGrokCheck([]byte("<html>")); err == nil {
		t.Error("unreadable output must error")
	}
	msg := "network down"
	if _, err := ParseGrokCheck([]byte(`{"currentVersion":"1","latestVersion":"","updateAvailable":false,"error":"network down"}`)); err == nil || !strings.Contains(err.Error(), msg) {
		t.Errorf("vendor error must surface, got %v", err)
	}
	if _, err := ParseGrokCheck([]byte(`{"currentVersion":"1","latestVersion":"","updateAvailable":false,"error":null}`)); err == nil {
		t.Error("missing latest must error")
	}
}

func TestParseHermesCheck(t *testing.T) {
	available := "→ Fetching from upstream...\n⚕ Update available (behind origin/main).\n  Run 'hermes update' to install."
	current := "Hermes Agent is up to date with origin/main."
	if ok, err := ParseHermesCheck(available); err != nil || !ok {
		t.Errorf("available case = %v, %v", ok, err)
	}
	if ok, err := ParseHermesCheck(current); err != nil || ok {
		t.Errorf("current case = %v, %v", ok, err)
	}
	if _, err := ParseHermesCheck("something unexpected"); err == nil {
		t.Error("unrecognized output must error, never guess")
	}
}
