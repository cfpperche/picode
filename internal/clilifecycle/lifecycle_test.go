package clilifecycle

import (
	"strings"
	"testing"
)

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
