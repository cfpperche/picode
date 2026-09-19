package connectors

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/mcp"
)

// The golden fixture: Hermes's own document keys and comments, one entry
// with the vendor fields PiCode does not model (auth, tools) and ${VAR}
// placeholders. Every write must keep them — yaml.Node edits preserve
// order and comments by construction; these tests hold the codec to it.
const hermesGolden = `# Hermes main config — PiCode edits only mcp_servers entries.
model: sonnet

# MCP servers Hermes loads at startup.
mcp_servers:
  # docs serves the library search
  docs:
    command: npx
    args: ["-y", "${DOCS_PKG}"] # inline note
    enabled: true
    auth: oauth
    tools:
      include: [search, fetch]
  relay:
    url: https://relay.example/mcp
    enabled: false
`

func seedHermes(t *testing.T, text string) (Paths, string) {
	t.Helper()
	home := t.TempDir()
	// The lesson of the phase-3 incident: a fixture must never be able to
	// reach the running machine's real config, so HOME and XDG resolve
	// inside this test's own tree even if a driver branch stops using
	// Paths.Home.
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	path := ""
	if text != "" {
		dir := filepath.Join(home, ".hermes")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		path = filepath.Join(dir, "config.yaml")
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return Paths{Home: home}, path
}

func TestHermesGoldenRoundTrip(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	p, path := seedHermes(t, hermesGolden)

	// Add: the new entry joins the block; the document's comments, key
	// order, vendor fields and ${VAR} placeholders survive untouched.
	if err := (Hermes{}).Add(p, "user", "deepwiki", mcp.Entry{URL: "https://mcp.deepwiki.com/mcp", Auth: "oauth"}); err != nil {
		t.Fatal(err)
	}
	out := mustRead(t, path)
	for _, want := range []string{
		"# Hermes main config — PiCode edits only mcp_servers entries.",
		"# MCP servers Hermes loads at startup.",
		"# docs serves the library search",
		"# inline note",
		"model: sonnet",
		"auth: oauth",
		"include: [search, fetch]",
		"${DOCS_PKG}",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("write lost %q:\n%s", want, out)
		}
	}
	if i, j := strings.Index(out, "model: sonnet"), strings.Index(out, "mcp_servers:"); i < 0 || j < 0 || i > j {
		t.Fatalf("document order not preserved:\n%s", out)
	}
	if i, j := strings.Index(out, "docs:"), strings.Index(out, "deepwiki:"); i < 0 || j < 0 || i > j {
		t.Fatalf("entry order not preserved (new entry must append):\n%s", out)
	}
	if strings.Count(out, "deepwiki:") != 1 {
		t.Fatalf("unexpected duplicate deepwiki entries:\n%s", out)
	}

	// The round-trip reports both servers; deepwiki rides its URL.
	rep, err := Hermes{}.List(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Servers) != 3 || rep.Adapter.Source != "native:hermes" || rep.Adapter.Installed {
		t.Fatalf("report = %+v", rep)
	}
	byName := map[string]mcp.Server{}
	for _, s := range rep.Servers {
		byName[s.Name] = s
	}
	if byName["docs"].Command != "npx" || len(byName["docs"].Args) != 2 || byName["docs"].Args[1] != "${DOCS_PKG}" {
		t.Fatalf("docs row = %+v", byName["docs"])
	}
	if byName["docs"].Disabled {
		t.Fatalf("docs must read enabled: %+v", byName["docs"])
	}
	if !byName["relay"].Disabled || byName["relay"].Transport != "url" {
		t.Fatalf("relay row = %+v", byName["relay"])
	}
	if byName["deepwiki"].URL != "https://mcp.deepwiki.com/mcp" || byName["deepwiki"].Live != "" {
		t.Fatalf("deepwiki row = %+v", byName["deepwiki"])
	}

	// Update merges by key: switching transport drops the other side and
	// keeps auth, tools and the comments; env values stay ${VAR} verbatim.
	if err := (Hermes{}).Add(p, "user", "docs", mcp.Entry{Command: "node", Args: []string{"docs.js"}, Env: map[string]string{"TOKEN": "${HERMES_TOKEN}"}}); err != nil {
		t.Fatal(err)
	}
	out = mustRead(t, path)
	for _, want := range []string{"# docs serves the library search", "auth: oauth", "include: [search, fetch]", "TOKEN: ${HERMES_TOKEN}", "command: node"} {
		if !strings.Contains(out, want) {
			t.Fatalf("update lost %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "${DOCS_PKG}") || strings.Contains(out, "npx") {
		t.Fatalf("transport switch did not clean the other side:\n%s", out)
	}

	// Remove deletes the entry and nothing else.
	if err := (Hermes{}).Remove(p, "user", "deepwiki"); err != nil {
		t.Fatal(err)
	}
	out = mustRead(t, path)
	if strings.Contains(out, "deepwiki") {
		t.Fatalf("deepwiki survived Remove:\n%s", out)
	}
	for _, want := range []string{"# Hermes main config — PiCode edits only mcp_servers entries.", "mcp_servers:", "model: sonnet", "command: node"} {
		if !strings.Contains(out, want) {
			t.Fatalf("Remove lost %q:\n%s", want, out)
		}
	}
}

func TestHermesToggleFlipsEnabledInPlace(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	p, path := seedHermes(t, hermesGolden)

	if err := (Hermes{}).Toggle(p, "user", "docs", true); err != nil {
		t.Fatal(err)
	}
	out := mustRead(t, path)
	if !strings.Contains(out, "# docs serves the library search") || !strings.Contains(out, "${DOCS_PKG}") || !strings.Contains(out, "# inline note") {
		t.Fatalf("toggle lost comments or placeholders:\n%s", out)
	}
	docs := out[strings.Index(out, "docs:"):]
	if i := strings.Index(docs, "enabled: false"); i < 0 || i > strings.Index(docs, "relay:") {
		t.Fatalf("docs not disabled in place:\n%s", out)
	}
	if strings.Count(out, "enabled: false") != 2 { // docs flipped + relay already off
		t.Fatalf("another entry's flag moved:\n%s", out)
	}

	// Toggle back on and read the flag through List.
	if err := (Hermes{}).Toggle(p, "user", "docs", false); err != nil {
		t.Fatal(err)
	}
	rep, err := Hermes{}.List(p)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range rep.Servers {
		if s.Name == "docs" && s.Disabled {
			t.Fatalf("docs still disabled: %+v", rep.Servers)
		}
	}
}

// Add on a missing config creates the one file with the mcp_servers block.
func TestHermesAddCreatesConfig(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	p, _ := seedHermes(t, "")
	if err := (Hermes{}).Add(p, "user", "docs", mcp.Entry{Command: "npx", Args: []string{"-y", "docs"}}); err != nil {
		t.Fatal(err)
	}
	out := mustRead(t, filepath.Join(p.Home, ".hermes", "config.yaml"))
	if !strings.Contains(out, "mcp_servers:") || !strings.Contains(out, "command: npx") {
		t.Fatalf("created config:\n%s", out)
	}
	// The created document reads back as the entry it holds.
	rep, err := Hermes{}.List(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Servers) != 1 || rep.Servers[0].Name != "docs" || rep.Servers[0].Command != "npx" || len(rep.Servers[0].Args) != 2 {
		t.Fatalf("created config reads back = %+v", rep.Servers)
	}
}

// Without the binary the pane stays honest: Installed is false, but the
// plain file is still fully manageable.
func TestHermesWorksWithoutBinary(t *testing.T) {
	t.Setenv("PATH", t.TempDir()) // no hermes anywhere
	p, path := seedHermes(t, hermesGolden)
	rep, err := Hermes{}.List(p)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Adapter.Installed || len(rep.Servers) != 2 {
		t.Fatalf("report = %+v", rep)
	}
	if err := (Hermes{}).Toggle(p, "user", "docs", true); err != nil {
		t.Fatal(err)
	}
	if out := mustRead(t, path); !strings.Contains(out, "enabled: false") {
		t.Fatalf("toggle without binary:\n%s", out)
	}
}

func TestHermesMalformedFileRefuses(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	p, path := seedHermes(t, "mcp_servers: [unclosed")
	if err := (Hermes{}).Add(p, "user", "docs", mcp.Entry{URL: "https://x.example/mcp"}); err == nil || !strings.Contains(err.Error(), "not valid YAML") {
		t.Fatalf("add on malformed file = %v", err)
	}
	if err := (Hermes{}).Toggle(p, "user", "docs", true); err == nil || !strings.Contains(err.Error(), "not valid YAML") {
		t.Fatalf("toggle on malformed file = %v", err)
	}
	if err := (Hermes{}).Remove(p, "user", "docs"); err == nil || !strings.Contains(err.Error(), "not valid YAML") {
		t.Fatalf("remove on malformed file = %v", err)
	}
	if raw, _ := os.ReadFile(path); string(raw) != "mcp_servers: [unclosed" {
		t.Fatalf("malformed file was modified: %q", raw)
	}
}

// A scalar document is valid YAML but not a config: refuse loudly, touch
// nothing.
func TestHermesNonMappingRootRefuses(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	p, path := seedHermes(t, "- just\n- a list\n")
	if err := (Hermes{}).Add(p, "user", "docs", mcp.Entry{URL: "https://x.example/mcp"}); err == nil || !strings.Contains(err.Error(), "config mapping") {
		t.Fatalf("add on scalar-root file = %v", err)
	}
	if raw, _ := os.ReadFile(path); !strings.Contains(string(raw), "just") {
		t.Fatalf("file was modified: %q", raw)
	}
}

// The decision table (ADR-0150): add/toggle/remove × scope (Hermes has one
// file, so project refuses) × missing file × malformed file — every row
// ends in an observable result, never a panic and never a silently
// rewritten file.
func TestHermesDecisionTable(t *testing.T) {
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
		{name: "toggle user missing server", scope: "user", op: "toggle", seed: "other: 1\n", wantErr: "is not in"},
		{name: "remove missing server", scope: "user", op: "remove", seed: "mcp_servers: {}\n", wantErr: "is not in"},
		{name: "toggle malformed", scope: "user", op: "toggle", seed: "mcp_servers: [unclosed", wantErr: "not valid YAML"},
		{name: "remove malformed", scope: "user", op: "remove", seed: "mcp_servers: [unclosed", wantErr: "not valid YAML"},
		{name: "add on malformed", scope: "user", op: "add", seed: "mcp_servers: [unclosed", wantErr: "not valid YAML"},
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
			p, _ := seedHermes(t, tc.seed)
			var err error
			switch tc.op {
			case "add":
				err = Hermes{}.Add(p, tc.scope, "docs", mcp.Entry{URL: "https://d.example/mcp"})
			case "toggle":
				err = Hermes{}.Toggle(p, tc.scope, "docs", true)
			case "remove":
				err = Hermes{}.Remove(p, tc.scope, "docs")
			}
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("want success, got %v", err)
				}
				if tc.seed == "" {
					if _, serr := os.Stat(filepath.Join(p.Home, ".hermes", "config.yaml")); serr != nil {
						t.Fatalf("add did not create the file: %v", serr)
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

func mustRead(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// `hermes mcp list` is the headless status signal (ADR-0150 d4). Measured
// v0.21.3: a table whose last column carries the status ("✗ disabled");
// healthy rows show ✓ (token per vendor docs — the disabled row is the
// measured one). List merges the vendor verdict into the file rows.
func TestHermesLiveStatusFromVendorList(t *testing.T) {
	dir := t.TempDir()
	log := filepath.Join(dir, "calls")
	script := "#!/bin/sh\nPATH=\"$PATH:/usr/bin:/bin\"\necho \"$@\" >> \"" + log + "\"\n" +
		"if [ \"$1\" = mcp ] && [ \"$2\" = list ]; then printf '  MCP Servers:\\n\\n  Name             Transport                      Tools        Status    \\n  \\342\\224\\200\\342\\224\\200\\342\\224\\200\\342\\224\\200\\n  probe            /bin/echo hi                   all          \\342\\234\\227 disabled\\n  remote           https://x.example/mcp          -            \\342\\234\\223 connected\\n'; exit 0; fi\nexit 0\n"
	if err := os.WriteFile(filepath.Join(dir, "hermes"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	home := t.TempDir()
	p := Paths{Home: home}
	if err := os.MkdirAll(filepath.Join(home, ".hermes"), 0o755); err != nil {
		t.Fatal(err)
	}
	seed := "mcp_servers:\n  probe:\n    command: /bin/echo\n  remote:\n    url: https://x.example/mcp\n"
	if err := os.WriteFile(filepath.Join(home, ".hermes", "config.yaml"), []byte(seed), 0o644); err != nil {
		t.Fatal(err)
	}
	rep, err := (Hermes{}).List(p)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"probe": "", "remote": "live"}
	for _, s := range rep.Servers {
		if s.Live != want[s.Name] {
			t.Fatalf("%s live = %q, want %q", s.Name, s.Live, want[s.Name])
		}
	}
	before := readCalls(t, log)
	if _, err := (Hermes{}).List(p); err != nil {
		t.Fatal(err)
	}
	if after := readCalls(t, log); len(after) != len(before) {
		t.Fatalf("probe re-ran inside the TTL: %v -> %v", before, after)
	}
}
