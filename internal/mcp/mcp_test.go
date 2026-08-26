package mcp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
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
	if err := validEntry(Entry{Command: "npx", Auth: "oauth"}); err == nil {
		t.Fatal("auth on command")
	}
	if err := validEntry(Entry{URL: "https://x", Env: map[string]string{"A": "1"}}); err == nil {
		t.Fatal("env on url")
	}
	if err := validEntry(Entry{Command: "npx", Env: map[string]string{"1BAD": "x"}}); err == nil {
		t.Fatal("bad env key")
	}
	if err := validEntry(Entry{URL: "https://x", Auth: "basic"}); err == nil {
		t.Fatal("bad auth")
	}
	if err := validEntry(Entry{URL: "https://x", Auth: "bearer"}); err == nil {
		t.Fatal("bearer without token")
	}
}

func TestAddEnvHeadersAuth(t *testing.T) {
	home := t.TempDir()
	p := Paths{Home: home}

	if err := Add(p, "user", "local", Entry{Command: "npx", Args: []string{"-y", "x"}, Env: map[string]string{"API_KEY": "sekrit"}}); err != nil {
		t.Fatal(err)
	}
	if err := Add(p, "user", "remote", Entry{URL: "https://mcp.example/mcp", Auth: "bearer", BearerToken: "tok", Headers: map[string]string{"X-Trace": "1"}}); err != nil {
		t.Fatal(err)
	}
	if err := Add(p, "user", "oauth", Entry{URL: "https://mcp.example/oauth", Auth: "oauth"}); err != nil {
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
	if _, ok := by["local"].Env["API_KEY"]; !ok || by["local"].Env["API_KEY"] != "" || by["local"].Auth != "" {
		t.Fatalf("local = %+v", by["local"])
	}
	if by["remote"].Auth != "bearer" || by["remote"].Headers["X-Trace"] != "" {
		t.Fatalf("remote = %+v", by["remote"])
	}
	if by["oauth"].Auth != "oauth" {
		t.Fatalf("oauth = %+v", by["oauth"])
	}

	raw, err := os.ReadFile(p.PiGlobal())
	if err != nil {
		t.Fatal(err)
	}
	var file map[string]any
	if err := json.Unmarshal(raw, &file); err != nil {
		t.Fatal(err)
	}
	servers := file["mcpServers"].(map[string]any)
	remote := servers["remote"].(map[string]any)
	if remote["bearerToken"] != "tok" {
		t.Fatalf("file remote = %v", remote)
	}
	if _, ok := servers["local"].(map[string]any)["headers"]; ok {
		t.Fatal("command kept headers")
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

func TestImportHosts(t *testing.T) {
	home := t.TempDir()
	p := Paths{Home: home}

	res, err := ImportHosts(p, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Added) != 0 || len(res.Found) != 0 {
		t.Fatalf("empty home: %+v", res)
	}

	if err := os.MkdirAll(filepath.Join(home, ".cursor"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".cursor", "mcp.json"), []byte(`{"mcpServers":{"from-cursor":{"url":"https://c.example/mcp"}}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := ImportHosts(p, []ImportPick{{Kind: "codex", Servers: []string{"x"}}}); err == nil {
		t.Fatal("codex not on this machine")
	}
	res, err = ImportHosts(p, []ImportPick{{Kind: "cursor", Servers: []string{"from-cursor"}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Added) != 1 || res.Added[0] != "cursor" {
		t.Fatalf("added = %+v", res)
	}
	rep, err := List(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Imports) != 1 || rep.Imports[0] != "cursor" {
		t.Fatalf("imports = %v", rep.Imports)
	}
	if len(rep.Found) != 1 || !rep.Found[0].On || len(rep.Found[0].Servers) != 1 {
		t.Fatalf("found = %+v", rep.Found)
	}
	if len(rep.Servers) != 1 || rep.Servers[0].Name != "from-cursor" || rep.Servers[0].Owned {
		t.Fatalf("servers = %+v", rep.Servers)
	}

	again, err := ImportHosts(p, []ImportPick{{Kind: "cursor", Servers: []string{"from-cursor"}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(again.Added) != 0 || len(again.Removed) != 0 {
		t.Fatalf("second = %+v", again)
	}
	off, err := ImportHosts(p, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(off.Removed) != 1 || off.Removed[0] != "cursor" {
		t.Fatalf("remove = %+v", off)
	}
}

func TestImportPickDisablesOther(t *testing.T) {
	home := t.TempDir()
	p := Paths{Home: home}
	if err := os.MkdirAll(filepath.Join(home, ".cursor"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".cursor", "mcp.json"), []byte(`{"mcpServers":{"keep":{"url":"https://a.example/mcp"},"drop":{"url":"https://b.example/mcp"}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ImportHosts(p, []ImportPick{{Kind: "cursor", Servers: []string{"keep"}}}); err != nil {
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
	if by["keep"].Disabled || !by["drop"].Disabled {
		t.Fatalf("%+v", rep.Servers)
	}
	if by["drop"].Owned || by["keep"].Owned {
		t.Fatalf("import overlay must not be owned: %+v", rep.Servers)
	}
	if err := Remove(p, "user", "drop"); err == nil {
		t.Fatal("remove overlay stub")
	}
	rep, _ = List(p)
	for _, s := range rep.Servers {
		if s.Name == "drop" && !s.Disabled {
			t.Fatal("remove unmasked import")
		}
	}
}

func TestServersFromToml(t *testing.T) {
	s := "# x\n[mcp_servers.picode-dogfood]\nurl = \"https://mcp.context7.com/mcp\"\n"
	got := serversFromToml(s)
	if got["picode-dogfood"]["url"] != "https://mcp.context7.com/mcp" {
		t.Fatalf("%+v", got)
	}
}

func TestFoundHostsSkipsEmptyFile(t *testing.T) {
	home := t.TempDir()
	p := Paths{Home: home}
	if err := os.MkdirAll(filepath.Join(home, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".claude.json"), []byte(`{"theme":"dark"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := FoundHosts(p); len(got) != 0 {
		t.Fatalf("empty claude offered: %+v", got)
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

func TestListSortsByName(t *testing.T) {
	home := t.TempDir()
	p := Paths{Home: home}
	if err := Add(p, "user", "zeta", Entry{URL: "https://z.example/mcp"}); err != nil {
		t.Fatal(err)
	}
	if err := Add(p, "user", "Alpha", Entry{URL: "https://a.example/mcp"}); err != nil {
		t.Fatal(err)
	}
	if err := Add(p, "user", "mid", Entry{URL: "https://m.example/mcp"}); err != nil {
		t.Fatal(err)
	}
	rep, err := List(p)
	if err != nil {
		t.Fatal(err)
	}
	got := []string{}
	for _, s := range rep.Servers {
		got = append(got, s.Name)
	}
	want := []string{"Alpha", "mid", "zeta"}
	if len(got) != 3 || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
		t.Fatalf("order = %v want %v", got, want)
	}
}

func TestApplyLive(t *testing.T) {
	rep := Report{Servers: []Server{
		{Name: "on", Disabled: false},
		{Name: "off", Disabled: true},
		{Name: "auth", Disabled: false},
		{Name: "bad", Disabled: false},
		{Name: "ok", Disabled: false},
	}}
	ApplyLive(&rep, nil, false)
	if rep.Servers[0].Live != LiveIdle || rep.Servers[1].Live != "" {
		t.Fatalf("stopped = %+v", rep.Servers)
	}
	ApplyLive(&rep, map[string]string{
		"auth": "needs-auth",
		"bad":  "failed",
		"ok":   "connected",
	}, true)
	want := []string{LiveIdle, "", LiveAuth, LiveFailed, LiveOn}
	for i, s := range rep.Servers {
		if s.Live != want[i] {
			t.Fatalf("%s live=%q want %q", s.Name, s.Live, want[i])
		}
	}
}

func TestReadLive(t *testing.T) {
	dir := t.TempDir()
	path := LivePath(dir, "agent-1")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if ReadLive(path, 0) != nil {
		t.Fatal("missing file")
	}
	if err := os.WriteFile(path, []byte(`{"servers":[{"name":"docs","status":"connected"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	got := ReadLive(path, time.Hour)
	if got["docs"] != "connected" {
		t.Fatalf("got %#v", got)
	}
	args, env := AttachLive(dir, "agent-1")
	if len(args) != 2 || args[0] != "-e" || len(env) != 1 {
		t.Fatalf("attach %v %v", args, env)
	}
	ClearLive(dir, "agent-1")
	if ReadLive(path, 0) != nil {
		t.Fatal("cleared file remains")
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
