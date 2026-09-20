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

// seedAgy plants the one config file Antigravity loads (the user file,
// measured against agy 1.2.6); scope "project" only opens an empty
// workspace directory — there is no workspace file to read or write.
func seedAgy(t *testing.T, scope, text string) (Paths, string) {
	t.Helper()
	home := t.TempDir()
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
	p := Paths{Home: home}
	if scope == "project" {
		p.Cwd = t.TempDir()
	}
	return p, path
}

func TestAGYGoldenRoundTrip(t *testing.T) {
	writeFakeBin(t, "agy")
	p, path := seedAgy(t, "user", agyGolden)

	// Add: remote servers ride serverUrl, never the legacy url key; unknown
	// per-entry keys on the surviving entry survive the rewrite.
	if err := (AGY{}).Add(p, "user", "docs", mcp.Entry{URL: "https://mcp.deepwiki.com/mcp", Auth: "oauth"}); err != nil {
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
	if err := (AGY{}).Add(p, "user", "keep", mcp.Entry{URL: "https://kept.example/mcp"}); err != nil {
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
	if err := (AGY{}).Remove(p, "user", "docs"); err != nil {
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
	p, path := seedAgy(t, "user", seed)

	rep, err := AGY{}.List(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Servers) != 1 || rep.Servers[0].Owned || rep.Servers[0].URL != "https://old.example/mcp" || rep.Servers[0].Transport != "url" {
		t.Fatalf("legacy row = %+v", rep.Servers)
	}

	if err := (AGY{}).Toggle(p, "user", "old", true); err == nil || !strings.Contains(err.Error(), "legacy entry managed in Antigravity") {
		t.Fatalf("legacy toggle = %v", err)
	}
	if err := (AGY{}).Remove(p, "user", "old"); err == nil || !strings.Contains(err.Error(), "legacy entry managed in Antigravity") {
		t.Fatalf("legacy remove = %v", err)
	}
	if got, _ := os.ReadFile(path); string(got) != seed {
		t.Fatalf("refused op still modified the file:\n%q", got)
	}

	// An Add over the legacy name converts it to the managed shape.
	if err := (AGY{}).Add(p, "user", "old", mcp.Entry{URL: "https://old.example/mcp"}); err != nil {
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
	p, _ := seedAgy(t, "user", seed)
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
	p, path := seedAgy(t, "user", "{not json")
	if err := (AGY{}).Add(p, "user", "docs", mcp.Entry{URL: "https://x.example/mcp"}); err == nil {
		t.Fatal("add on malformed file must fail")
	}
	if err := (AGY{}).Toggle(p, "user", "docs", true); err == nil {
		t.Fatal("toggle on malformed file must fail")
	}
	if err := (AGY{}).Remove(p, "user", "docs"); err == nil {
		t.Fatal("remove on malformed file must fail")
	}
	if raw, _ := os.ReadFile(path); string(raw) != "{not json" {
		t.Fatalf("malformed file was modified: %q", raw)
	}
}

// Live verification (agy 1.2.6): only ~/.gemini/config/mcp_config.json is
// ever loaded, so project scope refuses all three writes and touches
// nothing — no workspace file appears and the user file keeps its bytes.
func TestAGYProjectScopeRefuses(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	p, path := seedAgy(t, "project", agyGolden)
	before := fileText(t, path)

	// One layer even with a workspace open: the user file is all there is.
	rep, err := AGY{}.List(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Layers) != 1 || rep.Layers[0].Scope != "user" {
		t.Fatalf("layers = %+v", rep.Layers)
	}

	addErr := (AGY{}).Add(p, "project", "docs", mcp.Entry{URL: "https://d.example/mcp"})
	toggleErr := (AGY{}).Toggle(p, "project", "docs", true)
	removeErr := (AGY{}).Remove(p, "project", "docs")
	for _, err := range []error{addErr, toggleErr, removeErr} {
		if err == nil || !strings.Contains(err.Error(), "keeps one config file") {
			t.Fatalf("project-scope op = %v, want the one-config-file refusal", err)
		}
	}
	entries, err := os.ReadDir(p.Cwd)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("workspace not empty: %v", entries)
	}
	if got := fileText(t, path); got != before {
		t.Fatalf("user file changed:\n%q", got)
	}
}

// The decision table (ADR-0150): add/toggle/remove × scope × missing file ×
// malformed file × legacy entry — every row ends in an observable result,
// never a panic and never a silently rewritten file. Project scope refuses:
// Antigravity keeps one config file and there is no workspace file to write.
func TestAGYDecisionTable(t *testing.T) {
	cases := []struct {
		name    string
		scope   string
		op      string // add | toggle | remove
		seed    string // "" → missing file
		wantErr string
	}{
		{name: "add user no file", scope: "user", op: "add"},
		{name: "toggle user no file", scope: "user", op: "toggle", wantErr: "is not in"},
		{name: "remove user no file", scope: "user", op: "remove", wantErr: "is not in"},
		{name: "toggle user missing server", scope: "user", op: "toggle", seed: `{"other":1}`, wantErr: "is not in"},
		{name: "remove missing server", scope: "user", op: "remove", seed: `{"other":1}`, wantErr: "is not in"},
		{name: "toggle malformed", scope: "user", op: "toggle", seed: "{not json", wantErr: "not valid JSON"},
		{name: "remove malformed", scope: "user", op: "remove", seed: "{not json", wantErr: "not valid JSON"},
		{name: "add on malformed", scope: "user", op: "add", seed: "{not json", wantErr: "not valid JSON"},
		{name: "toggle legacy url entry", scope: "user", op: "toggle", seed: `{"mcpServers":{"docs":{"url":"https://d.example/mcp"}}}`, wantErr: "legacy entry managed in Antigravity"},
		{name: "remove legacy url entry", scope: "user", op: "remove", seed: `{"mcpServers":{"docs":{"url":"https://d.example/mcp"}}}`, wantErr: "legacy entry managed in Antigravity"},
		{name: "add project scope", scope: "project", op: "add", wantErr: "no per-workspace file"},
		{name: "toggle project scope", scope: "project", op: "toggle", wantErr: "no per-workspace file"},
		{name: "remove project scope", scope: "project", op: "remove", wantErr: "no per-workspace file"},
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
				if tc.seed == "" && tc.op == "add" {
					if _, serr := os.Stat(filepath.Join(p.Home, ".gemini", "config", "mcp_config.json")); serr != nil {
						t.Fatalf("add did not create the user file: %v", serr)
					}
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("err = %v, want it to contain %q", err, tc.wantErr)
			}
			if tc.seed != "" {
				if got, _ := os.ReadFile(path); string(got) != tc.seed {
					t.Fatalf("refused op still modified the file:\n%q", got)
				}
			}
		})
	}
}
