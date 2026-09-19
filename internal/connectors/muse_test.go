package connectors

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/mcp"
)

// The golden fixture: Muse's own document keys (schema_version, unknown
// vendor keys) and one entry carrying mode plus fields PiCode does not
// model. Every write must keep them.
const museGolden = `{
  "schema_version": 1,
  "theme": "muse-dark",
  "mcp_servers": {
    "keep": {
      "transport": "stdio",
      "command": "npx",
      "args": ["-y", "keep-mcp"],
      "env": {"TOKEN": "${KEEP_TOKEN}"},
      "mode": "optional",
      "timeout": 30
    }
  }
}`

func seedMuse(t *testing.T, text string) (Paths, string) {
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
		dir := filepath.Join(home, ".config", "muse")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		path = filepath.Join(dir, "settings.json")
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return Paths{Home: home}, path
}

func TestMuseGoldenRoundTrip(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	p, path := seedMuse(t, museGolden)

	// Add: unknown keys — top-level and per-entry — survive the rewrite.
	if err := (Muse{}).Add(p, "user", "docs", mcp.Entry{URL: "https://mcp.deepwiki.com/mcp", Auth: "oauth"}); err != nil {
		t.Fatal(err)
	}
	raw, err := readJSONFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if raw["schema_version"] != float64(1) || raw["theme"] != "muse-dark" {
		t.Fatalf("document keys not preserved: %v", raw)
	}
	keep, _ := raw["mcp_servers"].(map[string]any)["keep"].(map[string]any)
	if keep == nil {
		t.Fatalf("keep entry lost: %v", raw["mcp_servers"])
	}
	if keep["mode"] != "optional" || keep["timeout"] != float64(30) {
		t.Fatalf("unknown entry keys not preserved: %v", keep)
	}
	if keep["transport"] != "stdio" {
		t.Fatalf("existing transport not preserved: %v", keep)
	}
	env, _ := keep["env"].(map[string]any)
	if env["TOKEN"] != "${KEEP_TOKEN}" {
		t.Fatalf("placeholder not verbatim: %v", keep)
	}
	docs, _ := raw["mcp_servers"].(map[string]any)["docs"].(map[string]any)
	if docs == nil || docs["transport"] != "streamable_http" || docs["url"] != "https://mcp.deepwiki.com/mcp" {
		t.Fatalf("docs = %v", docs)
	}
	if _, has := docs["auth"]; has {
		t.Fatalf("oauth must not be written into the vendor file: %v", docs)
	}

	// The round-trip re-marshals; the golden check is on parsed content
	// (documented contract), not bytes.
	rep, err := Muse{}.List(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Servers) != 2 || rep.Adapter.Source != "native:muse" || rep.Adapter.Installed {
		t.Fatalf("report = %+v", rep)
	}
	byName := map[string]mcp.Server{}
	for _, s := range rep.Servers {
		byName[s.Name] = s
	}
	if byName["keep"].Command != "npx" || len(byName["keep"].Args) != 2 || byName["keep"].Args[1] != "keep-mcp" {
		t.Fatalf("keep row = %+v", byName["keep"])
	}
	if byName["docs"].Transport != "url" || byName["docs"].Live != "" {
		t.Fatalf("docs row = %+v", byName["docs"])
	}

	// Update merges by key: switching transport drops the other side and
	// keeps the unknown keys plus the previous mode.
	if err := (Muse{}).Add(p, "user", "docs", mcp.Entry{Command: "npx", Args: []string{"-y", "docs"}}); err != nil {
		t.Fatal(err)
	}
	raw, _ = readJSONFile(path)
	docs, _ = raw["mcp_servers"].(map[string]any)["docs"].(map[string]any)
	if docs["transport"] != "stdio" || docs["url"] != nil {
		t.Fatalf("transport switch did not clean the other side: %v", docs)
	}

	// Remove deletes the entry and nothing else.
	if err := (Muse{}).Remove(p, "user", "docs"); err != nil {
		t.Fatal(err)
	}
	raw, _ = readJSONFile(path)
	if _, still := raw["mcp_servers"].(map[string]any)["docs"]; still {
		t.Fatal("docs survived Remove")
	}
	if _, has := raw["mcp_servers"].(map[string]any)["keep"]; !has {
		t.Fatal("keep lost by Remove")
	}
	if raw["schema_version"] == nil || raw["theme"] == nil {
		t.Fatalf("document keys lost by Remove: %v", raw)
	}
}

func TestMuseToggleFlipsEnabledInPlace(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	p, path := seedMuse(t, museGolden)

	if err := (Muse{}).Toggle(p, "user", "keep", true); err != nil {
		t.Fatal(err)
	}
	raw, _ := readJSONFile(path)
	keep, _ := raw["mcp_servers"].(map[string]any)["keep"].(map[string]any)
	if keep["enabled"] != false || keep["mode"] != "optional" || keep["timeout"] != float64(30) {
		t.Fatalf("toggle off: %v", keep)
	}
	env, _ := keep["env"].(map[string]any)
	if env["TOKEN"] != "${KEEP_TOKEN}" {
		t.Fatalf("placeholder not verbatim: %v", keep)
	}
	if raw["schema_version"] != float64(1) || raw["theme"] == nil {
		t.Fatalf("document keys lost by Toggle: %v", raw)
	}

	// The disabled flag reads back through List.
	rep, err := Muse{}.List(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Servers) != 1 || !rep.Servers[0].Disabled || rep.Servers[0].Live != "" {
		t.Fatalf("servers = %+v", rep.Servers)
	}
}

// Without the binary the pane stays honest: Installed is false, but the
// plain file is still fully manageable.
func TestMuseWorksWithoutBinary(t *testing.T) {
	t.Setenv("PATH", t.TempDir()) // no muse anywhere
	home := t.TempDir()
	p := Paths{Home: home}
	if err := (Muse{}).Add(p, "user", "docs", mcp.Entry{
		Command: "npx", Args: []string{"-y", "docs"}, Env: map[string]string{"TOKEN": "${MUSE_TOKEN}"},
	}); err != nil {
		t.Fatal(err)
	}
	raw, err := readJSONFile(filepath.Join(home, ".config", "muse", "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	docs, _ := raw["mcp_servers"].(map[string]any)["docs"].(map[string]any)
	if docs["transport"] != "stdio" || docs["command"] != "npx" {
		t.Fatalf("docs = %v", docs)
	}
	env, _ := docs["env"].(map[string]any)
	if env["TOKEN"] != "${MUSE_TOKEN}" {
		t.Fatalf("placeholder not verbatim: %v", docs)
	}
	rep, err := Muse{}.List(p)
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

func TestMuseMalformedFileRefuses(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	p, path := seedMuse(t, "{not json")
	if err := (Muse{}).Add(p, "user", "docs", mcp.Entry{URL: "https://x.example/mcp"}); err == nil {
		t.Fatal("add on malformed file must fail")
	}
	if err := (Muse{}).Toggle(p, "user", "docs", true); err == nil {
		t.Fatal("toggle on malformed file must fail")
	}
	if err := (Muse{}).Remove(p, "user", "docs"); err == nil {
		t.Fatal("remove on malformed file must fail")
	}
	if raw, _ := os.ReadFile(path); string(raw) != "{not json" {
		t.Fatalf("malformed file was modified: %q", raw)
	}
}

// The decision table (ADR-0150): add/toggle/remove × scope (Muse has one
// file, so project refuses) × missing file × malformed file — every row
// ends in an observable result, never a panic and never a silently
// rewritten file.
func TestMuseDecisionTable(t *testing.T) {
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
			p, _ := seedMuse(t, tc.seed)
			var err error
			switch tc.op {
			case "add":
				err = Muse{}.Add(p, tc.scope, "docs", mcp.Entry{URL: "https://d.example/mcp"})
			case "toggle":
				err = Muse{}.Toggle(p, tc.scope, "docs", true)
			case "remove":
				err = Muse{}.Remove(p, tc.scope, "docs")
			}
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("want success, got %v", err)
				}
				if tc.seed == "" {
					if _, serr := os.Stat(filepath.Join(p.Home, ".config", "muse", "settings.json")); serr != nil {
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

// Measured against Muse Code 1.3.0 (2026-09-18): muse refuses a
// settings.json without schema_version as malformed, so every write path
// must keep the key present — a file PiCode creates from scratch, and an
// older file a Toggle or Remove rewrites.
func TestMuseWriteKeepsSchemaVersion(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	addPath := filepath.Join(t.TempDir(), ".config", "muse", "settings.json")
	pAdd := Paths{Home: filepath.Dir(filepath.Dir(filepath.Dir(addPath)))}
	if err := (Muse{}).Add(pAdd, "user", "docs", mcp.Entry{URL: "https://docs.example/mcp"}); err != nil {
		t.Fatal(err)
	}
	raw, err := readJSONFile(addPath)
	if err != nil {
		t.Fatal(err)
	}
	if raw["schema_version"] == nil {
		t.Fatalf("created file has no schema_version: %v", raw)
	}

	// An existing file that predates the key gains it on the next write,
	// and an existing value is never overwritten.
	p, path := seedMuse(t, `{"mcp_servers":{"docs":{"transport":"stdio","command":"npx"}}}`)
	if err := (Muse{}).Toggle(p, "user", "docs", true); err != nil {
		t.Fatal(err)
	}
	raw, _ = readJSONFile(path)
	if raw["schema_version"] == nil {
		t.Fatalf("toggle did not add schema_version: %v", raw)
	}
	if err := (Muse{}).Remove(p, "user", "docs"); err != nil {
		t.Fatal(err)
	}
	raw, _ = readJSONFile(path)
	if raw["schema_version"] == nil {
		t.Fatalf("remove did not add schema_version: %v", raw)
	}

	p, path = seedMuse(t, `{"schema_version": 7, "mcp_servers":{"docs":{"transport":"stdio","command":"npx"}}}`)
	if err := (Muse{}).Toggle(p, "user", "docs", true); err != nil {
		t.Fatal(err)
	}
	raw, _ = readJSONFile(path)
	if raw["schema_version"] != float64(7) {
		t.Fatalf("existing schema_version overwritten: %v", raw)
	}
}
