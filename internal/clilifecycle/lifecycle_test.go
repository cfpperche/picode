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
	museWrapper := filepath.Join(dir, ".local", "bin", "muse")
	write(museWrapper, "#!/usr/bin/env bash\n# muse-code/launcher\nchannel_url=https://api.meta.ai/muse-code/channels/muse-stable\nexec muse-bin\n")
	if got := DetectMethod(museWrapper); got != MethodVendor {
		t.Errorf("wrapper muse: got %q, want vendor", got)
	}
	agyBinary := filepath.Join(dir, ".local", "bin", "agy")
	write(agyBinary, "binary")
	if got := DetectMethod(agyBinary); got != MethodVendor {
		t.Errorf("agy: got %q, want vendor", got)
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
		{"/home/u/.bun/install/global/node_modules/@oh-my-pi/pi-coding-agent/dist/cli.js", MethodUnknown},
		{"/home/u/.local/bin/omp", MethodVendor},
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
		updateEnv         []string
		uninstall         UninstallKind
		uninstallArgs     []string
	}{
		{"pi", "npm", true, "npm", []string{"install", "-g", "@earendil-works/pi-coding-agent@latest"}, []string{"install", "-g", "@earendil-works/pi-coding-agent@latest"}, nil, "npm", []string{"remove", "-g", "@earendil-works/pi-coding-agent"}},
		{"pi", "unknown", false, "", nil, nil, nil, "", nil},
		{"codex", "npm", true, "npm", []string{"update"}, []string{"install", "-g", "@openai/codex@latest"}, nil, "npm", []string{"remove", "-g", "@openai/codex"}},
		{"claude-code", "native", true, "npm", []string{"update"}, []string{"install"}, nil, "guided", nil},
		{"claude-code", "npm", true, "npm", []string{"install", "-g", "@anthropic-ai/claude-code@latest"}, []string{"install", "-g", "@anthropic-ai/claude-code@latest"}, nil, "npm", []string{"remove", "-g", "@anthropic-ai/claude-code"}},
		{"claude-code", "unknown", false, "", nil, nil, nil, "", nil},
		{"grok", "vendor", true, "vendor", []string{"update"}, []string{"update", "--force-reinstall"}, nil, "guided", nil},
		{"grok", "npm", false, "", nil, nil, nil, "", nil},
		{"hermes", "git", true, "vendor", []string{"update", "--yes"}, []string{"update", "--force", "--yes"}, nil, "vendor", []string{"uninstall", "--yes"}},
		{"hermes", "unknown", false, "", nil, nil, nil, "", nil},
		{"opencode", "npm", true, "npm", []string{"upgrade"}, []string{"upgrade"}, nil, "vendor", []string{"uninstall", "--keep-config", "--keep-data", "--force"}},
		{"opencode", "unknown", false, "", nil, nil, nil, "", nil},
		{"muse", "vendor", true, "channel", nil, nil, []string{"MUSE_LAUNCHER_INSTALL=1"}, "guided", nil},
		{"muse", "unknown", false, "", nil, nil, nil, "", nil},
		{"agy", "vendor", true, "channel", []string{"update"}, []string{"update"}, nil, "guided", nil},
		{"agy", "unknown", false, "", nil, nil, nil, "", nil},
		// omp: npm installs mutate through npm (deterministic target — the
		// vendor updater resolves by PATH and hit an npm copy while run
		// from a bun install, measured 2026-09-17); native installs run
		// the vendor updater with its own check command.
		{"omp", "npm", true, "npm", []string{"install", "-g", "@oh-my-pi/pi-coding-agent@latest"}, []string{"install", "-g", "@oh-my-pi/pi-coding-agent@latest"}, nil, "npm", []string{"remove", "-g", "@oh-my-pi/pi-coding-agent"}},
		{"omp", "vendor", true, "vendor", []string{"update"}, []string{"update", "--force"}, nil, "guided", nil},
		{"omp", "unknown", false, "", nil, nil, nil, "", nil},
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
		if strings.Join(p.UpdateEnv, " ") != strings.Join(c.updateEnv, " ") {
			t.Errorf("For(%s,%s) UpdateEnv = %v, want %v", c.cli, c.method, p.UpdateEnv, c.updateEnv)
		}
		if p.CanUpdate() != (len(c.update) > 0 || len(c.updateEnv) > 0) {
			t.Errorf("For(%s,%s) CanUpdate = %v", c.cli, c.method, p.CanUpdate())
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
	for _, id := range []string{"grok", "hermes", "muse", "agy"} {
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

func TestParseChannelVersion(t *testing.T) {
	// Muse's channel JSON and Antigravity's manifest share one shape: a
	// `version` field. Verified 2026-09-14 against both live endpoints.
	c, err := ParseChannelVersion([]byte(`{"channel":"muse-stable","version":"1.2.1-R2847.1"}`))
	if err != nil || c.Version != "1.2.1-R2847.1" {
		t.Fatalf("muse channel = %+v, err %v", c, err)
	}
	m, err := ParseChannelVersion([]byte(`{"version":"1.2.2","url":"https://storage.googleapis.com/x","sha512":"abc"}`))
	if err != nil || m.Version != "1.2.2" {
		t.Fatalf("antigravity manifest = %+v, err %v", m, err)
	}
	if _, err := ParseChannelVersion([]byte(`<html>`)); err == nil {
		t.Error("unreadable output must error")
	}
	if _, err := ParseChannelVersion([]byte(`{"channel":"muse-stable","version":""}`)); err == nil {
		t.Error("missing version must error")
	}
}

func TestChannelURLFor(t *testing.T) {
	if got, ok := ChannelURLFor("muse", "linux", "amd64", false); !ok || got != MuseChannelURL {
		t.Fatalf("muse url = %q ok=%v", got, ok)
	}
	want := AntigravityManifestURL + "linux_amd64.json"
	if got, ok := ChannelURLFor("agy", "linux", "amd64", false); !ok || got != want {
		t.Fatalf("agy url = %q, want %q", got, want)
	}
	if got, ok := ChannelURLFor("agy", "linux", "amd64", true); !ok || got != AntigravityManifestURL+"linux_amd64_musl.json" {
		t.Fatalf("agy musl url = %q", got)
	}
	if got, ok := ChannelURLFor("agy", "darwin", "arm64", false); !ok || got != AntigravityManifestURL+"darwin_arm64.json" {
		t.Fatalf("agy darwin url = %q", got)
	}
	if _, ok := ChannelURLFor("pi", "linux", "amd64", false); ok {
		t.Error("pi has no version channel")
	}
}

func TestExtractMuseVersion(t *testing.T) {
	if got := ExtractMuseVersion("Muse Code 1.2.1 (1.2.1-R2847.1)"); got != "1.2.1-R2847.1" {
		t.Fatalf("got %q", got)
	}
	if got := ExtractMuseVersion("1.2.2"); got != "1.2.2" {
		t.Fatalf("semver fallback got %q", got)
	}
}

func TestParseOmpCheck(t *testing.T) {
	latest, err := ParseOmpCheck("Current version: 18.2.4\n✔ Already up to date\n")
	if err != nil || latest != "" {
		t.Errorf("up to date: latest = %q err = %v, want empty/nil", latest, err)
	}
	latest, err = ParseOmpCheck("Current version: 18.2.3\nNew version available: 18.2.4\n")
	if err != nil || latest != "18.2.4" {
		t.Errorf("outdated: latest = %q err = %v, want 18.2.4/nil", latest, err)
	}
	if _, err := ParseOmpCheck("some future layout"); err == nil {
		t.Error("unreadable output accepted")
	}
}
