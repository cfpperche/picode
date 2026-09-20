package connectors

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/mcp"
)

// The golden fixture: Omp's own top-level keys and one entry carrying
// vendor fields PiCode does not model. Every write must keep them.
const ompGolden = `{
  "$schema": "https://omp.sh/schemas/mcp-config.json",
  "mcpServers": {
    "keep": {
      "command": "npx",
      "args": ["-y", "keep-mcp"],
      "timeout": 30,
      "vendorHint": "${HOME}/bin"
    }
  },
  "disabledServers": ["retired"],
  "enabledServers": ["keep"]
}`

func seedOmp(t *testing.T, scope, text string) (Paths, string) {
	t.Helper()
	if text == "skip" { // dirs exist, file does not
		return Paths{Home: t.TempDir(), Cwd: t.TempDir()}, ""
	}
	home := t.TempDir()
	if scope == "user" {
		var path string
		if text != "" {
			if err := os.MkdirAll(filepath.Join(home, ".omp", "agent"), 0o755); err != nil {
				t.Fatal(err)
			}
			path = filepath.Join(home, ".omp", "agent", "mcp.json")
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
		if err := os.MkdirAll(filepath.Join(cwd, ".omp"), 0o755); err != nil {
			t.Fatal(err)
		}
		path = filepath.Join(cwd, ".omp", "mcp.json")
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return Paths{Home: home, Cwd: cwd}, path
}

// writeFakeBin plants an empty vendor binary on PATH so Installed reports
// true — the repo's fake-binary pattern, no real omp/agy involved.
func writeFakeBin(t *testing.T, name string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, name), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
}

func TestOmpGoldenRoundTrip(t *testing.T) {
	writeFakeBin(t, "omp")
	p, path := seedOmp(t, "project", ompGolden)

	// Add: unknown keys — top-level and per-entry — survive the rewrite.
	if err := (Omp{}).Add(p, "project", "docs", mcp.Entry{URL: "https://mcp.deepwiki.com/mcp", Auth: "oauth"}); err != nil {
		t.Fatal(err)
	}
	raw, err := readJSONFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if raw["$schema"] != "https://omp.sh/schemas/mcp-config.json" {
		t.Fatalf("$schema not preserved: %v", raw)
	}
	if ds, ok := raw["disabledServers"].([]any); !ok || len(ds) != 1 || ds[0] != "retired" {
		t.Fatalf("disabledServers not preserved: %v", raw)
	}
	if es, ok := raw["enabledServers"].([]any); !ok || len(es) != 1 {
		t.Fatalf("enabledServers not preserved: %v", raw)
	}
	keep, _ := raw["mcpServers"].(map[string]any)["keep"].(map[string]any)
	if keep == nil || keep["vendorHint"] != "${HOME}/bin" || keep["timeout"] != float64(30) {
		t.Fatalf("unknown entry keys not preserved: %v", raw["mcpServers"])
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
	rep, err := Omp{}.List(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Servers) != 2 || rep.Adapter.Source != "native:omp" || !rep.Adapter.Installed {
		t.Fatalf("report = %+v", rep)
	}

	// Update merges by key: switching transport drops the other side, keeps
	// the unknown key on the surviving entry.
	if err := (Omp{}).Add(p, "project", "docs", mcp.Entry{Command: "npx", Args: []string{"-y", "docs"}}); err != nil {
		t.Fatal(err)
	}
	raw, _ = readJSONFile(path)
	docs, _ = raw["mcpServers"].(map[string]any)["docs"].(map[string]any)
	if docs["command"] != "npx" || docs["url"] != nil || docs["type"] != nil {
		t.Fatalf("transport switch did not clean the other side: %v", docs)
	}

	// Remove deletes the entry and nothing else.
	if err := (Omp{}).Remove(p, "project", "docs"); err != nil {
		t.Fatal(err)
	}
	raw, _ = readJSONFile(path)
	if _, still := raw["mcpServers"].(map[string]any)["docs"]; still {
		t.Fatal("docs survived Remove")
	}
	if _, has := raw["mcpServers"].(map[string]any)["keep"]; !has {
		t.Fatal("keep lost by Remove")
	}
	if raw["$schema"] == nil || raw["disabledServers"] == nil {
		t.Fatalf("top-level keys lost by Remove: %v", raw)
	}
}

func TestOmpToggleFlipsEnabledInPlace(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	p, path := seedOmp(t, "user", ompGolden)

	if err := (Omp{}).Toggle(p, "user", "keep", true); err != nil {
		t.Fatal(err)
	}
	raw, _ := readJSONFile(path)
	keep, _ := raw["mcpServers"].(map[string]any)["keep"].(map[string]any)
	if keep["enabled"] != false || keep["vendorHint"] != "${HOME}/bin" || keep["timeout"] != float64(30) {
		t.Fatalf("toggle off: %v", keep)
	}
	// ${VAR} stays verbatim through a full read/write cycle.
	if err := (Omp{}).Toggle(p, "user", "keep", false); err != nil {
		t.Fatal(err)
	}
	raw, _ = readJSONFile(path)
	keep, _ = raw["mcpServers"].(map[string]any)["keep"].(map[string]any)
	if keep["enabled"] != true || keep["vendorHint"] != "${HOME}/bin" {
		t.Fatalf("toggle on: %v", keep)
	}
	if raw["$schema"] == nil || raw["disabledServers"] == nil {
		t.Fatalf("top-level keys lost by Toggle: %v", raw)
	}

	// The disabled flag reads back through List.
	rep, err := Omp{}.List(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Servers) != 1 || rep.Servers[0].Disabled || rep.Servers[0].Live != "" {
		t.Fatalf("servers = %+v", rep.Servers)
	}
}

func TestOmpEnabledFalseReadsDisabled(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	seed := `{"mcpServers":{"docs":{"url":"https://d.example/mcp","enabled":false}}}`
	p, _ := seedOmp(t, "project", seed)
	rep, err := Omp{}.List(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Servers) != 1 || !rep.Servers[0].Disabled || rep.Servers[0].Transport != "url" {
		t.Fatalf("servers = %+v", rep.Servers)
	}
}

// Unlike Claude Code's user store, Omp's user file is a plain file: every
// scope edits it directly, no vendor binary needed — the binary only feeds
// the honest Installed flag.
func TestOmpWorksWithoutBinary(t *testing.T) {
	t.Setenv("PATH", t.TempDir()) // no omp anywhere
	home := t.TempDir()
	p := Paths{Home: home}
	if err := (Omp{}).Add(p, "user", "docs", mcp.Entry{
		Command: "npx", Args: []string{"-y", "docs"}, Env: map[string]string{"TOKEN": "${OMP_TOKEN}"},
	}); err != nil {
		t.Fatal(err)
	}
	raw, err := readJSONFile(filepath.Join(home, ".omp", "agent", "mcp.json"))
	if err != nil {
		t.Fatal(err)
	}
	docs, _ := raw["mcpServers"].(map[string]any)["docs"].(map[string]any)
	env, _ := docs["env"].(map[string]any)
	if env["TOKEN"] != "${OMP_TOKEN}" {
		t.Fatalf("placeholder not verbatim: %v", docs)
	}
	rep, err := Omp{}.List(p)
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

func TestOmpMalformedFileRefuses(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	p, path := seedOmp(t, "project", "{not json")
	if err := (Omp{}).Add(p, "project", "docs", mcp.Entry{URL: "https://x.example/mcp"}); err == nil {
		t.Fatal("add on malformed file must fail")
	}
	if err := (Omp{}).Toggle(p, "project", "docs", true); err == nil {
		t.Fatal("toggle on malformed file must fail")
	}
	if err := (Omp{}).Remove(p, "project", "docs"); err == nil {
		t.Fatal("remove on malformed file must fail")
	}
	if raw, _ := os.ReadFile(path); string(raw) != "{not json" {
		t.Fatalf("malformed file was modified: %q", raw)
	}
}

// The decision table (ADR-0150): add/toggle/remove × scope × missing file ×
// malformed file — every row ends in an observable result, never a panic
// and never a silently rewritten file.
func TestOmpDecisionTable(t *testing.T) {
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
			p, path := seedOmp(t, tc.scope, tc.seed)
			var err error
			switch tc.op {
			case "add":
				err = Omp{}.Add(p, tc.scope, "docs", mcp.Entry{URL: "https://d.example/mcp"})
			case "toggle":
				err = Omp{}.Toggle(p, tc.scope, "docs", true)
			case "remove":
				err = Omp{}.Remove(p, tc.scope, "docs")
			}
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("want success, got %v", err)
				}
				if tc.seed == "skip" {
					if _, serr := os.Stat(filepath.Join(p.Cwd, ".omp", "mcp.json")); serr != nil {
						t.Fatalf("add did not create the project file: %v", serr)
					}
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("err = %v, want it to contain %q", err, tc.wantErr)
			}
			if tc.seed != "" && tc.seed != "skip" {
				if got, _ := os.ReadFile(path); string(got) != tc.seed {
					t.Fatalf("refused op still modified the file:\n%q", got)
				}
			}
		})
	}
}

func TestOmpToggleWritePathKeepsOtherScope(t *testing.T) {
	// A docs server in the user file is untouched by a project toggle of a
	// same-named server.
	t.Setenv("PATH", t.TempDir())
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".omp", "agent"), 0o755); err != nil {
		t.Fatal(err)
	}
	userSeed := `{"mcpServers":{"docs":{"url":"https://user.example/mcp"}}}`
	if err := os.WriteFile(filepath.Join(home, ".omp", "agent", "mcp.json"), []byte(userSeed), 0o644); err != nil {
		t.Fatal(err)
	}
	p, projectPath := seedOmp(t, "project", `{"mcpServers":{"docs":{"url":"https://project.example/mcp"}}}`)
	p.Home = home
	if err := (Omp{}).Toggle(p, "project", "docs", true); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(filepath.Join(home, ".omp", "agent", "mcp.json")); string(got) != userSeed {
		t.Fatalf("user file changed:\n%q", got)
	}
	raw, _ := readJSONFile(projectPath)
	docs, _ := raw["mcpServers"].(map[string]any)["docs"].(map[string]any)
	if docs["enabled"] != false {
		t.Fatalf("project file not toggled: %v", docs)
	}
}
