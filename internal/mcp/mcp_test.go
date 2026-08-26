package mcp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestAddToggleRemoveRoundTrip(t *testing.T) {
	home := t.TempDir()
	p := Paths{Home: home}

	if err := Add(p, "user", "deepwiki", Entry{URL: "https://mcp.deepwiki.com/mcp"}); err != nil {
		t.Fatal(err)
	}
	rep, err := List(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Servers) != 1 || rep.Servers[0].Name != "deepwiki" || rep.Servers[0].Transport != "url" {
		t.Fatalf("list = %+v", rep.Servers)
	}
	if !rep.Servers[0].Owned || rep.Servers[0].Layer != "pi-global" {
		t.Fatalf("layer = %+v", rep.Servers[0])
	}

	if err := Toggle(p, "user", "deepwiki", true); err != nil {
		t.Fatal(err)
	}
	rep, _ = List(p)
	if !rep.Servers[0].Disabled {
		t.Fatal("expected disabled")
	}
	if err := Toggle(p, "user", "deepwiki", false); err != nil {
		t.Fatal(err)
	}
	rep, _ = List(p)
	if rep.Servers[0].Disabled {
		t.Fatal("expected enabled")
	}

	if err := Remove(p, "user", "deepwiki"); err != nil {
		t.Fatal(err)
	}
	rep, _ = List(p)
	if len(rep.Servers) != 0 {
		t.Fatalf("after remove: %+v", rep.Servers)
	}
}

func TestAddKeepsUnknownKeys(t *testing.T) {
	home := t.TempDir()
	p := Paths{Home: home}
	path := p.PiGlobal()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	seed := []byte(`{
  "imports": ["cursor"],
  "settings": { "toolPrefix": "server" },
  "mcpServers": { "keep": { "url": "https://a.example/mcp", "custom": 1 } }
}
`)
	if err := os.WriteFile(path, seed, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Add(p, "user", "two", Entry{Command: "npx", Args: []string{"-y", "x"}}); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatal(err)
	}
	if _, ok := raw["imports"]; !ok {
		t.Fatal("lost imports")
	}
	if _, ok := raw["settings"]; !ok {
		t.Fatal("lost settings")
	}
	servers := raw["mcpServers"].(map[string]any)
	keep := servers["keep"].(map[string]any)
	if keep["custom"] != float64(1) {
		t.Fatalf("lost custom key: %v", keep)
	}
	if _, ok := servers["two"]; !ok {
		t.Fatal("missing two")
	}
}

func TestProjectAndAgentLayers(t *testing.T) {
	home := t.TempDir()
	cwd := t.TempDir()
	agent := t.TempDir()
	p := Paths{Home: home, Cwd: cwd, AgentCwd: agent}

	if err := Add(p, "user", "machine", Entry{URL: "https://m.example/mcp"}); err != nil {
		t.Fatal(err)
	}
	if err := Add(p, "project", "folder", Entry{URL: "https://f.example/mcp"}); err != nil {
		t.Fatal(err)
	}
	if err := Add(p, "agent", "only-me", Entry{Command: "npx", Args: []string{"-y", "x"}}); err != nil {
		t.Fatal(err)
	}
	// Same name in agent overrides machine.
	if err := Add(p, "agent", "machine", Entry{URL: "https://override.example/mcp"}); err != nil {
		t.Fatal(err)
	}
	rep, err := List(p)
	if err != nil {
		t.Fatal(err)
	}
	by := map[string]Server{}
	for _, s := range rep.Servers {
		by[s.Name] = s
	}
	if by["folder"].Layer != "shared-project" {
		t.Fatalf("folder layer %s", by["folder"].Layer)
	}
	if by["only-me"].Layer != "agent" || by["only-me"].Transport != "stdio" {
		t.Fatalf("agent = %+v", by["only-me"])
	}
	if by["machine"].URL != "https://override.example/mcp" || by["machine"].Layer != "agent" {
		t.Fatalf("override = %+v", by["machine"])
	}
}

func TestValidNameAndEntry(t *testing.T) {
	if err := ValidName(""); err == nil {
		t.Fatal("empty name")
	}
	if err := ValidName("../x"); err == nil {
		t.Fatal("path name")
	}
	if err := ValidName("ok-1"); err != nil {
		t.Fatal(err)
	}
	if err := validEntry(Entry{}); err == nil {
		t.Fatal("empty entry")
	}
	if err := validEntry(Entry{Command: "npx", URL: "https://x"}); err == nil {
		t.Fatal("both")
	}
	if err := validEntry(Entry{URL: "ftp://x"}); err == nil {
		t.Fatal("ftp")
	}
	if err := validEntry(Entry{Command: "npx; rm -rf /"}); err == nil {
		t.Fatal("metachar")
	}
}

func TestStripComments(t *testing.T) {
	home := t.TempDir()
	p := Paths{Home: home}
	path := p.PiGlobal()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	src := []byte(`{
  // cursor style
  "mcpServers": {
    "a": { "url": "https://a.example/mcp" /* keep */ }
  }
}
`)
	if err := os.WriteFile(path, src, 0o644); err != nil {
		t.Fatal(err)
	}
	rep, err := List(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Servers) != 1 || rep.Servers[0].Name != "a" {
		t.Fatalf("%+v", rep.Servers)
	}
}

func TestToggleStubDoesNotCopyURL(t *testing.T) {
	home := t.TempDir()
	p := Paths{Home: home}
	shared := p.SharedGlobal()
	if err := os.MkdirAll(filepath.Dir(shared), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(shared, []byte(`{"mcpServers":{"ext":{"url":"https://secret.example/mcp","headers":{"Authorization":"Bearer x"}}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Toggle(p, "user", "ext", true); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(p.PiGlobal())
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "" && containsAny(string(b), "secret.example", "Bearer") {
		t.Fatalf("stub leaked credentials: %s", b)
	}
	rep, _ := List(p)
	if len(rep.Servers) != 1 || !rep.Servers[0].Disabled || rep.Servers[0].URL != "https://secret.example/mcp" {
		t.Fatalf("merged = %+v", rep.Servers)
	}
}

func TestAdapterConfigured(t *testing.T) {
	if AdapterConfigured(nil) {
		t.Fatal("empty")
	}
	if !AdapterConfigured([]string{"npm:pi-web-search", "npm:pi-mcp-adapter"}) {
		t.Fatal("missed")
	}
}

func containsAny(s string, parts ...string) bool {
	for _, p := range parts {
		if len(p) > 0 && len(s) > 0 && (len(s) >= len(p)) {
			for i := 0; i+len(p) <= len(s); i++ {
				if s[i:i+len(p)] == p {
					return true
				}
			}
		}
	}
	return false
}
