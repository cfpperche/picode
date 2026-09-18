package connectors

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/mcp"
)

func TestClaudeProjectGoldenRoundTrip(t *testing.T) {
	// No real `claude` may answer here: the user-scope listing shells out to
	// the CLI, and this test owns only the project file.
	t.Setenv("PATH", t.TempDir())
	home := t.TempDir()
	cwd := t.TempDir()
	p := Paths{Home: home, Cwd: cwd}

	seed := `{
  "mcpServers": {
    "keep": {
      "command": "npx",
      "args": ["-y", "keep-mcp"],
      "vendorHint": "${HOME}/bin"
    }
  },
  "otherTopLevel": {"untouched": true}
}`
	path := filepath.Join(cwd, ".mcp.json")
	if err := os.WriteFile(path, []byte(seed), 0o644); err != nil {
		t.Fatal(err)
	}

	// Add: unknown keys — top-level and per-entry — survive the rewrite.
	if err := (Claude{}).Add(p, "project", "docs", mcp.Entry{URL: "https://mcp.deepwiki.com/mcp", Auth: "oauth"}); err != nil {
		t.Fatal(err)
	}
	raw, err := readClaudeFile(path)
	if err != nil {
		t.Fatal(err)
	}
	keep, _ := raw["mcpServers"].(map[string]any)["keep"].(map[string]any)
	if keep == nil || keep["vendorHint"] != "${HOME}/bin" {
		t.Fatalf("unknown entry keys not preserved: %v", raw["mcpServers"])
	}
	if raw["otherTopLevel"] == nil {
		t.Fatalf("unknown top-level keys not preserved: %v", raw)
	}
	docs, _ := raw["mcpServers"].(map[string]any)["docs"].(map[string]any)
	if docs == nil || docs["type"] != "http" || docs["url"] != "https://mcp.deepwiki.com/mcp" {
		t.Fatalf("docs = %v", docs)
	}
	if _, has := docs["auth"]; has {
		t.Fatalf("oauth must not be written into the vendor file: %v", docs)
	}

	// The round-trip re-marshals; key order may move, so the golden check is
	// on parsed content (documented contract), not bytes.
	rep, err := Claude{}.List(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Servers) != 2 || rep.Adapter.Source != "native:claude" {
		t.Fatalf("report = %+v", rep)
	}

	// Update merges by key: switching transport drops the other side, keeps
	// the unknown key on the surviving entry.
	if err := (Claude{}).Add(p, "project", "docs", mcp.Entry{Command: "npx", Args: []string{"-y", "docs"}}); err != nil {
		t.Fatal(err)
	}
	raw, _ = readClaudeFile(path)
	docs, _ = raw["mcpServers"].(map[string]any)["docs"].(map[string]any)
	if docs["command"] != "npx" || docs["url"] != nil || docs["type"] != nil {
		t.Fatalf("transport switch did not clean the other side: %v", docs)
	}

	// Remove.
	if err := (Claude{}).Remove(p, "project", "docs"); err != nil {
		t.Fatal(err)
	}
	raw, _ = readClaudeFile(path)
	if _, still := raw["mcpServers"].(map[string]any)["docs"]; still {
		t.Fatal("docs survived Remove")
	}
	if _, has := raw["mcpServers"].(map[string]any)["keep"]; !has {
		t.Fatal("keep lost by Remove")
	}
}

func TestClaudeToggleRefusedEverywhere(t *testing.T) {
	p := Paths{Home: t.TempDir(), Cwd: t.TempDir()}
	for _, scope := range []string{"user", "project"} {
		err := Claude{}.Toggle(p, scope, "docs", true)
		if err == nil || !strings.Contains(err.Error(), "turn on and off in Claude Code") {
			t.Fatalf("toggle %s = %v", scope, err)
		}
	}
	if err := (Claude{}).Toggle(p, "agent", "docs", true); err == nil {
		t.Fatal("agent scope must be refused")
	}
}

func TestClaudeUserScopeNeedsBinary(t *testing.T) {
	t.Setenv("PATH", t.TempDir()) // no claude anywhere
	p := Paths{Home: t.TempDir()}
	add := Claude{}.Add(p, "user", "docs", mcp.Entry{URL: "https://mcp.example/mcp"})
	if add == nil || !strings.Contains(add.Error(), "not installed or not on PATH") {
		t.Fatalf("add without binary = %v", add)
	}
	rm := Claude{}.Remove(p, "user", "docs")
	if rm == nil || !strings.Contains(rm.Error(), "not installed or not on PATH") {
		t.Fatalf("remove without binary = %v", rm)
	}
	rep, err := Claude{}.List(p)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Adapter.Installed {
		t.Fatal("installed must be false without the binary")
	}
	if len(rep.Servers) != 0 {
		t.Fatalf("servers = %+v", rep.Servers)
	}
}

// writeFakeClaude plants a `claude` shell script that logs argv and answers
// `mcp list` from a fixed file — the repo's fake-binary pattern
// (term_wiring_test.go). No network, no real CLI.
func writeFakeClaude(t *testing.T, listOut string) string {
	t.Helper()
	dir := t.TempDir()
	log := filepath.Join(dir, "args.log")
	fixture := filepath.Join(dir, "list.out")
	if err := os.WriteFile(fixture, []byte(listOut), 0o644); err != nil {
		t.Fatal(err)
	}
	// The test isolates PATH to the fake dir; the script itself still needs
	// the standard dirs for `cat`.
	script := "#!/bin/sh\nPATH=\"$PATH:/usr/bin:/bin\"\nprintf '%s\\n' \"$*\" >> \"" + log + "\"\n" +
		"if [ \"$1\" = mcp ] && [ \"$2\" = list ]; then cat \"" + fixture + "\"; exit 0; fi\n" +
		"exit 0\n"
	if err := os.WriteFile(filepath.Join(dir, "claude"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	return log
}

func readLog(t *testing.T, log string) []string {
	t.Helper()
	raw, err := os.ReadFile(log)
	if err != nil {
		t.Fatalf("argv log: %v", err)
	}
	var out []string
	for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

func TestClaudeUserScopeViaVendorCLI(t *testing.T) {
	log := writeFakeClaude(t, `{"mcpServers":{"relay":{"type":"http","url":"https://relay.example/mcp","headers":{"Authorization":"Bearersekrit"}}}}`)
	p := Paths{Home: t.TempDir()}

	if err := (Claude{}).Add(p, "user", "docs", mcp.Entry{
		URL: "https://docs.example/mcp", Auth: "bearer", BearerToken: "tok",
	}); err != nil {
		t.Fatal(err)
	}
	lines := readLog(t, log)
	if len(lines) == 0 {
		t.Fatal("claude was never called")
	}
	argv := lines[len(lines)-1]
	for _, want := range []string{"mcp add", "--transport http", "--header Authorization: Bearer tok", "docs", "https://docs.example/mcp", "--scope user"} {
		if !strings.Contains(argv, want) {
			t.Fatalf("argv %q missing %q", argv, want)
		}
	}

	// A command entry carries env through --env instead.
	if err := (Claude{}).Add(p, "user", "local", mcp.Entry{
		Command: "npx", Args: []string{"-y", "docs"}, Env: map[string]string{"K": "v"},
	}); err != nil {
		t.Fatal(err)
	}
	lines = readLog(t, log)
	argv = lines[len(lines)-1]
	for _, want := range []string{"mcp add", "--env K=v", "local", "npx -y docs", "--scope user"} {
		if !strings.Contains(argv, want) {
			t.Fatalf("command argv %q missing %q", argv, want)
		}
	}

	rep, err := Claude{}.List(p)
	if err != nil {
		t.Fatal(err)
	}
	var relay *mcp.Server
	for i := range rep.Servers {
		if rep.Servers[i].Name == "relay" {
			relay = &rep.Servers[i]
		}
	}
	if relay == nil || relay.Scope != "user" || relay.Transport != "url" || relay.Live != "" {
		t.Fatalf("relay row = %+v", relay)
	}
	// GET must not echo secrets: header keys survive, values do not.
	if relay.Headers["Authorization"] != "" {
		t.Fatalf("headers leaked: %v", relay.Headers)
	}

	if err := (Claude{}).Remove(p, "user", "docs"); err != nil {
		t.Fatal(err)
	}
	lines = readLog(t, log)
	if !strings.Contains(lines[len(lines)-1], "mcp remove docs --scope user") {
		t.Fatalf("remove argv = %q", lines[len(lines)-1])
	}
}

func TestClaudeListTolerantLines(t *testing.T) {
	writeFakeClaude(t, "Checking MCP server health…\n\ndocs: npx -y docs-mcp - ✓ Connected\nrelay: https://relay.example/mcp - ✗ Failed to connect\n!!! not a name\n")
	p := Paths{Home: t.TempDir()}
	rep, err := Claude{}.List(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Servers) != 2 {
		t.Fatalf("servers = %+v", rep.Servers)
	}
	if rep.Servers[0].Name != "docs" || !strings.HasPrefix(rep.Servers[0].Command, "npx -y docs-mcp") {
		t.Fatalf("docs row = %+v", rep.Servers[0])
	}
	if rep.Servers[1].Name != "relay" || strings.Contains(rep.Servers[1].Command, "Failed") {
		t.Fatalf("relay row = %+v", rep.Servers[1])
	}
}

func TestClaudeMalformedProjectFileRefuses(t *testing.T) {
	cwd := t.TempDir()
	p := Paths{Home: t.TempDir(), Cwd: cwd}
	path := filepath.Join(cwd, ".mcp.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := (Claude{}).Add(p, "project", "docs", mcp.Entry{URL: "https://x.example/mcp"}); err == nil {
		t.Fatal("add on malformed file must fail")
	}
	if err := (Claude{}).Remove(p, "project", "docs"); err == nil {
		t.Fatal("remove on malformed file must fail")
	}
	if raw, _ := os.ReadFile(path); string(raw) != "{not json" {
		t.Fatalf("malformed file was modified: %q", raw)
	}
}

func TestClaudeAgentScopeRefused(t *testing.T) {
	p := Paths{Home: t.TempDir(), Cwd: t.TempDir()}
	err := Claude{}.Add(p, "agent", "docs", mcp.Entry{URL: "https://x.example/mcp"})
	if err == nil || !strings.Contains(err.Error(), "not available yet") {
		t.Fatalf("agent scope = %v", err)
	}
}
