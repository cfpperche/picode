package connectors

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/mcp"
)

// The golden fixture: OpenCode's own top-level keys ($schema, theme,
// provider) and one entry carrying the command array plus vendor fields
// PiCode does not model. Every write must keep them.
const opencodeGolden = `{
  "$schema": "https://opencode.ai/config.json",
  "theme": "opencode",
  "provider": {"copied": {"options": {"baseURL": "${OPENCODE_BASE}"}}},
  "mcp": {
    "keep": {
      "type": "local",
      "command": ["npx", "-y", "keep-mcp"],
      "environment": {"TOKEN": "${KEEP_TOKEN}"},
      "timeout": 30,
      "oauth": {"scopes": ["read"]}
    }
  }
}`

func seedOpencode(t *testing.T, scope, text string) (Paths, string) {
	t.Helper()
	if text == "skip" { // dirs exist, file does not
		home := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
		return Paths{Home: home, Cwd: t.TempDir()}, ""
	}
	home := t.TempDir()
	// The user config must resolve inside this test's own tree, never the
	// running machine's — route it through the XDG branch the driver honors.
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	if scope == "user" {
		var path string
		if text != "" {
			if err := os.MkdirAll(filepath.Join(home, ".config", "opencode"), 0o755); err != nil {
				t.Fatal(err)
			}
			path = filepath.Join(home, ".config", "opencode", "opencode.json")
			if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		return Paths{Home: home}, path
	}
	cwd := ""
	var path string
	if text != "" {
		cwd = t.TempDir()
		path = filepath.Join(cwd, "opencode.json")
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return Paths{Home: home, Cwd: cwd}, path
}

func writeFakeOpencodeBin(t *testing.T, name string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, name), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
}

func TestOpencodeGoldenRoundTrip(t *testing.T) {
	writeFakeOpencodeBin(t, "opencode")
	p, path := seedOpencode(t, "project", opencodeGolden)

	// Add: unknown keys — top-level and per-entry — survive the rewrite, and
	// the command array round-trips as an array.
	if err := (OpenCode{}).Add(p, "project", "docs", mcp.Entry{URL: "https://mcp.deepwiki.com/mcp", Auth: "oauth"}); err != nil {
		t.Fatal(err)
	}
	raw, err := readJSONFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if raw["$schema"] != "https://opencode.ai/config.json" || raw["theme"] != "opencode" {
		t.Fatalf("top-level keys not preserved: %v", raw)
	}
	if _, ok := raw["provider"].(map[string]any); !ok {
		t.Fatalf("provider block not preserved: %v", raw)
	}
	keep, _ := raw["mcp"].(map[string]any)["keep"].(map[string]any)
	if keep == nil || keep["timeout"] != float64(30) {
		t.Fatalf("unknown entry keys not preserved: %v", raw["mcp"])
	}
	if _, ok := keep["oauth"].(map[string]any); !ok {
		t.Fatalf("oauth field not preserved: %v", keep)
	}
	env, _ := keep["environment"].(map[string]any)
	if env["TOKEN"] != "${KEEP_TOKEN}" {
		t.Fatalf("placeholder not verbatim: %v", keep)
	}
	cmd, _ := keep["command"].([]any)
	if len(cmd) != 3 || cmd[0] != "npx" || cmd[1] != "-y" || cmd[2] != "keep-mcp" {
		t.Fatalf("command array not preserved: %v", keep["command"])
	}
	docs, _ := raw["mcp"].(map[string]any)["docs"].(map[string]any)
	if docs == nil || docs["type"] != "remote" || docs["url"] != "https://mcp.deepwiki.com/mcp" {
		t.Fatalf("docs = %v", docs)
	}
	if _, has := docs["auth"]; has {
		t.Fatalf("oauth must not be written into the vendor file: %v", docs)
	}

	// The round-trip re-marshals; key order may move, so the golden check is
	// on parsed content (documented contract), not bytes.
	rep, err := OpenCode{}.List(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Servers) != 2 || rep.Adapter.Source != "native:opencode" || !rep.Adapter.Installed {
		t.Fatalf("report = %+v", rep)
	}
	byName := map[string]mcp.Server{}
	for _, s := range rep.Servers {
		byName[s.Name] = s
	}
	if byName["keep"].Command != "npx" || len(byName["keep"].Args) != 2 || byName["keep"].Args[0] != "-y" || byName["keep"].Args[1] != "keep-mcp" {
		t.Fatalf("keep row = %+v", byName["keep"])
	}
	if byName["docs"].Transport != "url" || byName["docs"].Live != "" {
		t.Fatalf("docs row = %+v", byName["docs"])
	}

	// Update merges by key: switching transport drops the other side, keeps
	// the unknown key and the previous enabled flag on the surviving entry.
	if err := (OpenCode{}).Add(p, "project", "docs", mcp.Entry{Command: "npx", Args: []string{"-y", "docs"}}); err != nil {
		t.Fatal(err)
	}
	raw, _ = readJSONFile(path)
	docs, _ = raw["mcp"].(map[string]any)["docs"].(map[string]any)
	if docs["type"] != "local" || docs["url"] != nil {
		t.Fatalf("transport switch did not clean the other side: %v", docs)
	}
	cmd, _ = docs["command"].([]any)
	if len(cmd) != 3 || cmd[0] != "npx" {
		t.Fatalf("added command not an array: %v", docs["command"])
	}

	// Remove deletes the entry and nothing else.
	if err := (OpenCode{}).Remove(p, "project", "docs"); err != nil {
		t.Fatal(err)
	}
	raw, _ = readJSONFile(path)
	if _, still := raw["mcp"].(map[string]any)["docs"]; still {
		t.Fatal("docs survived Remove")
	}
	if _, has := raw["mcp"].(map[string]any)["keep"]; !has {
		t.Fatal("keep lost by Remove")
	}
	if raw["$schema"] == nil || raw["provider"] == nil {
		t.Fatalf("top-level keys lost by Remove: %v", raw)
	}
}

func TestOpencodeToggleFlipsEnabledInPlace(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	p, path := seedOpencode(t, "user", opencodeGolden)

	if err := (OpenCode{}).Toggle(p, "user", "keep", true); err != nil {
		t.Fatal(err)
	}
	raw, _ := readJSONFile(path)
	keep, _ := raw["mcp"].(map[string]any)["keep"].(map[string]any)
	if keep["enabled"] != false || keep["timeout"] != float64(30) {
		t.Fatalf("toggle off: %v", keep)
	}
	env, _ := keep["environment"].(map[string]any)
	if env["TOKEN"] != "${KEEP_TOKEN}" {
		t.Fatalf("placeholder not verbatim: %v", keep)
	}
	if raw["$schema"] == nil || raw["provider"] == nil {
		t.Fatalf("top-level keys lost by Toggle: %v", raw)
	}

	// ${VAR} stays verbatim through another full read/write cycle.
	if err := (OpenCode{}).Toggle(p, "user", "keep", false); err != nil {
		t.Fatal(err)
	}
	raw, _ = readJSONFile(path)
	keep, _ = raw["mcp"].(map[string]any)["keep"].(map[string]any)
	if keep["enabled"] != true {
		t.Fatalf("toggle on: %v", keep)
	}

	// The disabled flag reads back through List.
	rep, err := OpenCode{}.List(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Servers) != 1 || rep.Servers[0].Disabled || rep.Servers[0].Live != "" {
		t.Fatalf("servers = %+v", rep.Servers)
	}
}

func TestOpencodeEnabledFalseReadsDisabled(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	seed := `{"mcp":{"docs":{"type":"remote","url":"https://d.example/mcp","enabled":false}}}`
	p, _ := seedOpencode(t, "project", seed)
	rep, err := OpenCode{}.List(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Servers) != 1 || !rep.Servers[0].Disabled || rep.Servers[0].Transport != "url" {
		t.Fatalf("servers = %+v", rep.Servers)
	}
}

// The user config follows XDG_CONFIG_HOME when it is set (the XDG
// base-directory rule OpenCode itself uses).
func TestOpencodeXDGConfigHome(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	t.Setenv("PATH", t.TempDir())
	p := Paths{Home: t.TempDir()}
	if err := (OpenCode{}).Add(p, "user", "docs", mcp.Entry{Command: "npx", Args: []string{"-y", "docs"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(xdg, "opencode", "opencode.json")); err != nil {
		t.Fatalf("add ignored XDG_CONFIG_HOME: %v", err)
	}
}

// Unlike Claude Code's user store, OpenCode's user file is a plain file:
// every scope edits it directly, no vendor binary needed — the binary only
// feeds the honest Installed flag.
func TestOpencodeWorksWithoutBinary(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("PATH", t.TempDir()) // no opencode anywhere
	home := t.TempDir()
	p := Paths{Home: home}
	if err := (OpenCode{}).Add(p, "user", "docs", mcp.Entry{
		Command: "npx", Args: []string{"-y", "docs"}, Env: map[string]string{"TOKEN": "${OC_TOKEN}"},
	}); err != nil {
		t.Fatal(err)
	}
	raw, err := readJSONFile(filepath.Join(home, ".config", "opencode", "opencode.json"))
	if err != nil {
		t.Fatal(err)
	}
	docs, _ := raw["mcp"].(map[string]any)["docs"].(map[string]any)
	cmd, _ := docs["command"].([]any)
	if len(cmd) != 3 || cmd[0] != "npx" || cmd[1] != "-y" || cmd[2] != "docs" {
		t.Fatalf("command array = %v", docs["command"])
	}
	env, _ := docs["environment"].(map[string]any)
	if env["TOKEN"] != "${OC_TOKEN}" {
		t.Fatalf("placeholder not verbatim: %v", docs)
	}
	rep, err := OpenCode{}.List(p)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Adapter.Installed {
		t.Fatal("installed must be false without the binary")
	}
	if len(rep.Servers) != 1 {
		t.Fatalf("servers = %+v", rep.Servers)
	}
}

func TestOpencodeMalformedFileRefuses(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	p, path := seedOpencode(t, "project", "{not json")
	if err := (OpenCode{}).Add(p, "project", "docs", mcp.Entry{URL: "https://x.example/mcp"}); err == nil {
		t.Fatal("add on malformed file must fail")
	}
	if err := (OpenCode{}).Toggle(p, "project", "docs", true); err == nil {
		t.Fatal("toggle on malformed file must fail")
	}
	if err := (OpenCode{}).Remove(p, "project", "docs"); err == nil {
		t.Fatal("remove on malformed file must fail")
	}
	if raw, _ := os.ReadFile(path); string(raw) != "{not json" {
		t.Fatalf("malformed file was modified: %q", raw)
	}
}

// The decision table (ADR-0150): add/toggle/remove × scope × missing file ×
// malformed file — every row ends in an observable result, never a panic
// and never a silently rewritten file.
func TestOpencodeDecisionTable(t *testing.T) {
	cases := []struct {
		name    string
		scope   string
		op      string // add | toggle | remove
		seed    string // "" → missing file
		wantErr string
	}{
		{name: "add user no file", scope: "user", op: "add"},
		{name: "add project no cwd", scope: "project", op: "add", wantErr: "select a workspace"},
		{name: "add creates project file", scope: "project", op: "add", seed: "skip"},
		{name: "toggle user no file", scope: "user", op: "toggle", wantErr: "is not in"},
		{name: "toggle project no cwd", scope: "project", op: "toggle", wantErr: "select a workspace"},
		{name: "remove user no file", scope: "user", op: "remove", wantErr: "is not in"},
		{name: "toggle user missing server", scope: "user", op: "toggle", seed: `{"other":1}`, wantErr: "is not in"},
		{name: "remove missing server", scope: "user", op: "remove", seed: `{"other":1}`, wantErr: "is not in"},
		{name: "toggle malformed", scope: "user", op: "toggle", seed: "{not json", wantErr: "not valid JSON"},
		{name: "remove malformed", scope: "user", op: "remove", seed: "{not json", wantErr: "not valid JSON"},
		{name: "add on malformed", scope: "user", op: "add", seed: "{not json", wantErr: "not valid JSON"},
		{name: "add agent scope", scope: "agent", op: "add", wantErr: "not available yet"},
		{name: "toggle agent scope", scope: "agent", op: "toggle", wantErr: "not available yet"},
		{name: "remove agent scope", scope: "agent", op: "remove", wantErr: "not available yet"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("PATH", t.TempDir())
			p, _ := seedOpencode(t, tc.scope, tc.seed)
			var err error
			switch tc.op {
			case "add":
				err = OpenCode{}.Add(p, tc.scope, "docs", mcp.Entry{URL: "https://d.example/mcp"})
			case "toggle":
				err = OpenCode{}.Toggle(p, tc.scope, "docs", true)
			case "remove":
				err = OpenCode{}.Remove(p, tc.scope, "docs")
			}
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("want success, got %v", err)
				}
				if tc.seed == "skip" {
					if _, serr := os.Stat(filepath.Join(p.Cwd, "opencode.json")); serr != nil {
						t.Fatalf("add did not create the project file: %v", serr)
					}
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("err = %v, want it to contain %q", err, tc.wantErr)
			}
		})
	}
}

func TestOpencodeToggleWritePathKeepsOtherScope(t *testing.T) {
	// A docs server in the user file is untouched by a project toggle of a
	// same-named server.
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("PATH", t.TempDir())
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".config", "opencode"), 0o755); err != nil {
		t.Fatal(err)
	}
	userSeed := `{"mcp":{"docs":{"type":"remote","url":"https://user.example/mcp"}}}`
	if err := os.WriteFile(filepath.Join(home, ".config", "opencode", "opencode.json"), []byte(userSeed), 0o644); err != nil {
		t.Fatal(err)
	}
	p, projectPath := seedOpencode(t, "project", `{"mcp":{"docs":{"type":"remote","url":"https://project.example/mcp"}}}`)
	p.Home = home
	if err := (OpenCode{}).Toggle(p, "project", "docs", true); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(filepath.Join(home, ".config", "opencode", "opencode.json")); string(got) != userSeed {
		t.Fatalf("user file changed:\n%q", got)
	}
	raw, _ := readJSONFile(projectPath)
	docs, _ := raw["mcp"].(map[string]any)["docs"].(map[string]any)
	if docs["enabled"] != false {
		t.Fatalf("project file not toggled: %v", docs)
	}
}
