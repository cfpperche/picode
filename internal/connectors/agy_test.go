package connectors

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/mcp"
)

// The golden fixture: Antigravity's own per-entry fields PiCode does not
// model. Every write must keep them.
const agyGolden = `{
  "mcpServers": {
    "keep": {
      "command": "npx",
      "args": ["-y", "keep-mcp"],
      "authProviderType": "google",
      "oauth": {"scopes": ["read"]},
      "disabledTools": ["wipe"]
    }
  }
}`

func seedAgy(t *testing.T, scope, text string) (Paths, string) {
	t.Helper()
	if text == "skip" { // dirs exist, file does not
		return Paths{Home: t.TempDir(), Cwd: t.TempDir()}, ""
	}
	home := t.TempDir()
	if scope == "user" {
		var path string
		if text != "" {
			if err := os.MkdirAll(filepath.Join(home, ".gemini", "config"), 0o755); err != nil {
				t.Fatal(err)
			}
			path = filepath.Join(home, ".gemini", "config", "mcp_config.json")
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
		if err := os.MkdirAll(filepath.Join(cwd, ".agents"), 0o755); err != nil {
			t.Fatal(err)
		}
		path = filepath.Join(cwd, ".agents", "mcp_config.json")
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return Paths{Home: home, Cwd: cwd}, path
}

func TestAGYGoldenRoundTrip(t *testing.T) {
	writeFakeBin(t, "agy")
	p, path := seedAgy(t, "project", agyGolden)

	// Add: remote servers ride serverUrl, never the legacy url key; unknown
	// per-entry keys on the surviving entry survive the rewrite.
	if err := (AGY{}).Add(p, "project", "docs", mcp.Entry{URL: "https://mcp.deepwiki.com/mcp", Auth: "oauth"}); err != nil {
		t.Fatal(err)
	}
	raw, err := readJSONFile(path)
	if err != nil {
		t.Fatal(err)
	}
	keep, _ := raw["mcpServers"].(map[string]any)["keep"].(map[string]any)
	if keep == nil || keep["authProviderType"] != "google" || keep["disabledTools"] == nil || keep["oauth"] == nil {
		t.Fatalf("unknown entry keys not preserved: %v", raw["mcpServers"])
	}
	docs, _ := raw["mcpServers"].(map[string]any)["docs"].(map[string]any)
	if docs == nil || docs["serverUrl"] != "https://mcp.deepwiki.com/mcp" {
		t.Fatalf("docs = %v", docs)
	}
	if _, has := docs["url"]; has {
		t.Fatalf("managed writes never use the legacy url key: %v", docs)
	}
	if _, has := docs["auth"]; has {
		t.Fatalf("oauth must not be written into the vendor file: %v", docs)
	}

	rep, err := AGY{}.List(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Servers) != 2 || rep.Adapter.Source != "native:agy" || !rep.Adapter.Installed {
		t.Fatalf("report = %+v", rep)
	}
	for _, s := range rep.Servers {
		if s.Live != "" || !s.Owned {
			t.Fatalf("row = %+v", s)
		}
	}

	// Update merges by key: switching transport drops the other side, keeps
	// Antigravity's fields on the surviving entry.
	if err := (AGY{}).Add(p, "project", "keep", mcp.Entry{URL: "https://kept.example/mcp"}); err != nil {
		t.Fatal(err)
	}
	raw, _ = readJSONFile(path)
	keep, _ = raw["mcpServers"].(map[string]any)["keep"].(map[string]any)
	if keep["serverUrl"] != "https://kept.example/mcp" || keep["command"] != nil {
		t.Fatalf("transport switch did not clean the other side: %v", keep)
	}
	if keep["authProviderType"] != "google" || keep["disabledTools"] == nil {
		t.Fatalf("Antigravity fields lost by update: %v", keep)
	}

	// Remove deletes the entry and nothing else.
	if err := (AGY{}).Remove(p, "project", "docs"); err != nil {
		t.Fatal(err)
	}
	raw, _ = readJSONFile(path)
	if _, still := raw["mcpServers"].(map[string]any)["docs"]; still {
		t.Fatal("docs survived Remove")
	}
	if _, has := raw["mcpServers"].(map[string]any)["keep"]; !has {
		t.Fatal("keep lost by Remove")
	}
}

func TestAGYToggleFlipsDisabledInPlace(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	p, path := seedAgy(t, "user", agyGolden)

	if err := (AGY{}).Toggle(p, "user", "keep", true); err != nil {
		t.Fatal(err)
	}
	raw, _ := readJSONFile(path)
	keep, _ := raw["mcpServers"].(map[string]any)["keep"].(map[string]any)
	if keep["disabled"] != true || keep["authProviderType"] != "google" || keep["oauth"] == nil || keep["disabledTools"] == nil {
		t.Fatalf("toggle off: %v", keep)
	}
	if err := (AGY{}).Toggle(p, "user", "keep", false); err != nil {
		t.Fatal(err)
	}
	raw, _ = readJSONFile(path)
	keep, _ = raw["mcpServers"].(map[string]any)["keep"].(map[string]any)
	if keep["disabled"] != false {
		t.Fatalf("toggle on: %v", keep)
	}

	rep, err := AGY{}.List(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Servers) != 1 || rep.Servers[0].Disabled || rep.Servers[0].Live != "" {
		t.Fatalf("servers = %+v", rep.Servers)
	}
}

// A stdio entry's cwd is Antigravity's field: written when the add form
// carries one, and preserved through a toggle that rewrites the entry.
func TestAGYCwdPreservedThroughToggle(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	home := t.TempDir()
	p := Paths{Home: home}
	if err := (AGY{}).Add(p, "user", "local", mcp.Entry{Command: "node", Args: []string{"server.js"}, Cwd: "/opt/srv"}); err != nil {
		t.Fatal(err)
	}
	if err := (AGY{}).Toggle(p, "user", "local", true); err != nil {
		t.Fatal(err)
	}
	raw, err := readJSONFile(filepath.Join(home, ".gemini", "config", "mcp_config.json"))
	if err != nil {
		t.Fatal(err)
	}
	local, _ := raw["mcpServers"].(map[string]any)["local"].(map[string]any)
	if local["cwd"] != "/opt/srv" || local["disabled"] != true || local["command"] != "node" {
		t.Fatalf("cwd not preserved: %v", local)
	}
}

// The legacy url-only shape Antigravity wrote before serverUrl: displayed,
// reported not owned, refused on Toggle/Remove — and converted by an Add
// over the same name.
func TestAGYLegacyURLEntry(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	seed := `{"mcpServers":{"old":{"url":"https://old.example/mcp"}}}`
	p, path := seedAgy(t, "project", seed)

	rep, err := AGY{}.List(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Servers) != 1 || rep.Servers[0].Owned || rep.Servers[0].URL != "https://old.example/mcp" || rep.Servers[0].Transport != "url" {
		t.Fatalf("legacy row = %+v", rep.Servers)
	}

	if err := (AGY{}).Toggle(p, "project", "old", true); err == nil || !strings.Contains(err.Error(), "legacy entry managed in Antigravity") {
		t.Fatalf("legacy toggle = %v", err)
	}
	if err := (AGY{}).Remove(p, "project", "old"); err == nil || !strings.Contains(err.Error(), "legacy entry managed in Antigravity") {
		t.Fatalf("legacy remove = %v", err)
	}
	if got, _ := os.ReadFile(path); string(got) != seed {
		t.Fatalf("refused op still modified the file:\n%q", got)
	}

	// An Add over the legacy name converts it to the managed shape.
	if err := (AGY{}).Add(p, "project", "old", mcp.Entry{URL: "https://old.example/mcp"}); err != nil {
		t.Fatal(err)
	}
	raw, _ := readJSONFile(path)
	old, _ := raw["mcpServers"].(map[string]any)["old"].(map[string]any)
	if old["serverUrl"] != "https://old.example/mcp" || old["url"] != nil {
		t.Fatalf("legacy conversion: %v", old)
	}
	rep, err = AGY{}.List(p)
	if err != nil {
		t.Fatal(err)
	}
	if !rep.Servers[0].Owned {
		t.Fatalf("converted row still unowned: %+v", rep.Servers)
	}
}

func TestAGYDisabledReadsDisabled(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	seed := `{"mcpServers":{"docs":{"serverUrl":"https://d.example/mcp","disabled":true}}}`
	p, _ := seedAgy(t, "project", seed)
	rep, err := AGY{}.List(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Servers) != 1 || !rep.Servers[0].Disabled {
		t.Fatalf("servers = %+v", rep.Servers)
	}
}

func TestAGYMalformedFileRefuses(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	p, path := seedAgy(t, "project", "{not json")
	if err := (AGY{}).Add(p, "project", "docs", mcp.Entry{URL: "https://x.example/mcp"}); err == nil {
		t.Fatal("add on malformed file must fail")
	}
	if err := (AGY{}).Toggle(p, "project", "docs", true); err == nil {
		t.Fatal("toggle on malformed file must fail")
	}
	if err := (AGY{}).Remove(p, "project", "docs"); err == nil {
		t.Fatal("remove on malformed file must fail")
	}
	if raw, _ := os.ReadFile(path); string(raw) != "{not json" {
		t.Fatalf("malformed file was modified: %q", raw)
	}
}

// The decision table (ADR-0150): add/toggle/remove × scope × missing file ×
// malformed file × legacy entry — every row ends in an observable result,
// never a panic and never a silently rewritten file.
func TestAGYDecisionTable(t *testing.T) {
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
		{name: "toggle legacy url entry", scope: "user", op: "toggle", seed: `{"mcpServers":{"docs":{"url":"https://d.example/mcp"}}}`, wantErr: "legacy entry managed in Antigravity"},
		{name: "remove legacy url entry", scope: "user", op: "remove", seed: `{"mcpServers":{"docs":{"url":"https://d.example/mcp"}}}`, wantErr: "legacy entry managed in Antigravity"},
		{name: "add agent scope", scope: "agent", op: "add", wantErr: "not available yet"},
		{name: "toggle agent scope", scope: "agent", op: "toggle", wantErr: "not available yet"},
		{name: "remove agent scope", scope: "agent", op: "remove", wantErr: "not available yet"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("PATH", t.TempDir())
			p, path := seedAgy(t, tc.scope, tc.seed)
			var err error
			switch tc.op {
			case "add":
				err = AGY{}.Add(p, tc.scope, "docs", mcp.Entry{URL: "https://d.example/mcp"})
			case "toggle":
				err = AGY{}.Toggle(p, tc.scope, "docs", true)
			case "remove":
				err = AGY{}.Remove(p, tc.scope, "docs")
			}
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("want success, got %v", err)
				}
				if tc.seed == "skip" {
					if _, serr := os.Stat(filepath.Join(p.Cwd, ".agents", "mcp_config.json")); serr != nil {
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

func TestAGYToggleWritePathKeepsOtherScope(t *testing.T) {
	// A docs server in the user file is untouched by a project toggle of a
	// same-named server.
	t.Setenv("PATH", t.TempDir())
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".gemini", "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	userSeed := `{"mcpServers":{"docs":{"serverUrl":"https://user.example/mcp"}}}`
	if err := os.WriteFile(filepath.Join(home, ".gemini", "config", "mcp_config.json"), []byte(userSeed), 0o644); err != nil {
		t.Fatal(err)
	}
	p, projectPath := seedAgy(t, "project", `{"mcpServers":{"docs":{"serverUrl":"https://project.example/mcp"}}}`)
	p.Home = home
	if err := (AGY{}).Toggle(p, "project", "docs", true); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(filepath.Join(home, ".gemini", "config", "mcp_config.json")); string(got) != userSeed {
		t.Fatalf("user file changed:\n%q", got)
	}
	raw, _ := readJSONFile(projectPath)
	docs, _ := raw["mcpServers"].(map[string]any)["docs"].(map[string]any)
	if docs["disabled"] != true {
		t.Fatalf("project file not toggled: %v", docs)
	}
}
