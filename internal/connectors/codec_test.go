package connectors

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/mcp"
)

func TestStripJSONCTable(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string // parsed JSON compared as content; "" must stay unparseable
	}{
		{name: "trailing comma object", in: `{"a":1,}`, want: `{"a":1}`},
		{name: "trailing comma array", in: `{"a":[1,2,],}`, want: `{"a":[1,2]}`},
		{name: "line comment", in: "{\n// note\n\"a\":1\n}", want: `{"a":1}`},
		{name: "block comment", in: `{"a": /* x */ 1}`, want: `{"a":1}`},
		{name: "url keeps slashes", in: `{"u":"https://x.example//mcp"}`, want: `{"u":"https://x.example//mcp"}`},
		{name: "comma inside string stays", in: `{"u":"a,}"}`, want: `{"u":"a,}"}`},
		{name: "comment inside string stays", in: `{"u":"http://a // b"}`, want: `{"u":"http://a // b"}`},
		{name: "escaped quote keeps string", in: `{"u":"say \"hi//"}`, want: `{"u":"say \"hi//"}`},
		{name: "missing value stays broken", in: `{"a":1 "b":2}`, want: ""},
		{name: "stray brace stays broken", in: `{not json`, want: ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := lenientJSON([]byte(tc.in))
			if tc.want == "" {
				if err == nil {
					t.Fatalf("lenientJSON(%q) = %v, want error", tc.in, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("lenientJSON(%q) = %v", tc.in, err)
			}
			wantRaw := map[string]any{}
			if err := json.Unmarshal([]byte(tc.want), &wantRaw); err != nil {
				t.Fatal(err)
			}
			gotB, _ := json.Marshal(got)
			wantB, _ := json.Marshal(wantRaw)
			if string(gotB) != string(wantB) {
				t.Fatalf("strip(%q) parsed as %s, want %s", tc.in, gotB, wantB)
			}
		})
	}
}

// The owner's repro (connectors-parity debt): OpenCode writes JSONC-ish
// config — trailing commas after a schema-only block, hand comments. Every
// JSON-codec driver must read it; a rewrite through Add lands strict JSON.
func TestGuestListToleratesJSONC(t *testing.T) {
	const server = `"docs": {"type": "remote", "url": "https://d.example/mcp",} // remote`
	cases := []struct {
		name     string
		driver   Driver
		addScope string
		seed     func(t *testing.T) (Paths, string)
		scroll   string // document key holding the servers block
	}{
		{
			name:     "opencode user",
			driver:   OpenCode{},
			addScope: "user",
			seed: func(t *testing.T) (Paths, string) {
				return seedJSONCUser(t, filepath.Join(".config", "opencode", "opencode.json"), `{
					"$schema": "https://opencode.ai/config.json", // schema pin
					"mcp": {`+server+`
					},
				}`)
			},
			scroll: "mcp",
		},
		{
			name:     "omp user",
			driver:   Omp{},
			addScope: "user",
			seed: func(t *testing.T) (Paths, string) {
				return seedJSONCUser(t, filepath.Join(".omp", "agent", "mcp.json"), `{
					/* omp config */
					"mcpServers": {`+server+`
					},
				}`)
			},
			scroll: "mcpServers",
		},
		{
			name:     "muse user",
			driver:   Muse{},
			addScope: "user",
			seed: func(t *testing.T) (Paths, string) {
				return seedJSONCUser(t, filepath.Join(".config", "muse", "settings.json"), `{
					"mcp_servers": {`+server+`
					},
				}`)
			},
			scroll: "mcp_servers",
		},
		{
			name:     "agy user",
			driver:   AGY{},
			addScope: "user",
			seed: func(t *testing.T) (Paths, string) {
				return seedJSONCUser(t, filepath.Join(".gemini", "config", "mcp_config.json"), `{
					"mcpServers": {`+server+`
					},
				}`)
			},
			scroll: "mcpServers",
		},
		{
			name:     "claude project",
			driver:   Claude{},
			addScope: "project",
			seed: func(t *testing.T) (Paths, string) {
				home := t.TempDir()
				cwd := t.TempDir()
				path := filepath.Join(cwd, ".mcp.json")
				text := `{
					"mcpServers": { /* project servers */ ` + server + `
					},
				}`
				if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
					t.Fatal(err)
				}
				return Paths{Home: home, Cwd: cwd}, path
			},
			scroll: "mcpServers",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("PATH", t.TempDir())
			p, path := tc.seed(t)
			rep, err := tc.driver.List(p)
			if err != nil {
				t.Fatalf("List on JSONC file: %v", err)
			}
			if len(rep.Servers) != 1 || rep.Servers[0].Name != "docs" || rep.Servers[0].URL != "https://d.example/mcp" {
				t.Fatalf("servers = %+v", rep.Servers)
			}
			for _, l := range rep.Layers {
				if l.Error != "" {
					t.Fatalf("layer %s blocked: %q", l.ID, l.Error)
				}
			}
			// Add works on top and the rewrite lands strict JSON: the same
			// bytes parse without the lenient second chance, and the
			// vendor's own top-level key survives.
			if err := tc.driver.Add(p, tc.addScope, "keep", mcp.Entry{URL: "https://k.example/mcp"}); err != nil {
				t.Fatal(err)
			}
			b, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var strict map[string]any
			if err := json.Unmarshal(b, &strict); err != nil {
				t.Fatalf("rewrite is not strict JSON: %v\n%s", err, b)
			}
			if strict[tc.scroll] == nil {
				t.Fatalf("servers block lost: %s", b)
			}
			if strings.Contains(string(b), ",}") || strings.Contains(string(b), "/*") {
				t.Fatalf("rewrite kept JSONC escapes: %s", b)
			}
		})
	}
}

// A malformed file degrades: List answers nil error, the layer reports
// exists + the reason, and it contributes no servers. Every codec gets the
// same treatment (JSON, TOML, YAML — ADR-0150).
func TestGuestListDegradesMalformedLayer(t *testing.T) {
	cases := []struct {
		name    string
		seed    func(t *testing.T) (Paths, string)
		layer   string
		driver  Driver
		wantErr string
	}{
		{name: "opencode", driver: OpenCode{}, layer: "opencode-user", wantErr: "is not valid JSON", seed: func(t *testing.T) (Paths, string) {
			return seedJSONCUser(t, filepath.Join(".config", "opencode", "opencode.json"), "{not json")
		}},
		{name: "omp", driver: Omp{}, layer: "omp-user", wantErr: "is not valid JSON", seed: func(t *testing.T) (Paths, string) {
			return seedJSONCUser(t, filepath.Join(".omp", "agent", "mcp.json"), "{not json")
		}},
		{name: "muse", driver: Muse{}, layer: "muse-user", wantErr: "is not valid JSON", seed: func(t *testing.T) (Paths, string) {
			return seedJSONCUser(t, filepath.Join(".config", "muse", "settings.json"), "{not json")
		}},
		{name: "agy", driver: AGY{}, layer: "agy-user", wantErr: "is not valid JSON", seed: func(t *testing.T) (Paths, string) {
			return seedJSONCUser(t, filepath.Join(".gemini", "config", "mcp_config.json"), "{not json")
		}},
		{name: "claude project", driver: Claude{}, layer: "claude-project", wantErr: "is not valid JSON", seed: func(t *testing.T) (Paths, string) {
			home := t.TempDir()
			cwd := t.TempDir()
			if err := os.WriteFile(filepath.Join(cwd, ".mcp.json"), []byte("{not json"), 0o644); err != nil {
				t.Fatal(err)
			}
			return Paths{Home: home, Cwd: cwd}, ""
		}},
		{name: "grok", driver: Grok{}, layer: "grok-user", wantErr: "is not valid TOML", seed: func(t *testing.T) (Paths, string) {
			return seedJSONCUser(t, filepath.Join(".grok", "config.toml"), "[mcp_servers.docs\nbroken")
		}},
		{name: "hermes", driver: Hermes{}, layer: "hermes-user", wantErr: "is not valid YAML", seed: func(t *testing.T) (Paths, string) {
			return seedJSONCUser(t, filepath.Join(".hermes", "config.yaml"), "mcp_servers: [unclosed")
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("PATH", t.TempDir())
			p, _ := tc.seed(t)
			rep, err := tc.driver.List(p)
			if err != nil {
				t.Fatalf("List must degrade, got error: %v", err)
			}
			if len(rep.Servers) != 0 {
				t.Fatalf("servers from a broken layer: %+v", rep.Servers)
			}
			blocked := 0
			for _, l := range rep.Layers {
				if l.ID != tc.layer {
					if l.Error != "" {
						t.Fatalf("healthy layer %s blocked: %q", l.ID, l.Error)
					}
					continue
				}
				blocked++
				if !l.Exists || l.Error != tc.wantErr {
					t.Fatalf("blocked layer = %+v", l)
				}
			}
			if blocked != 1 {
				t.Fatalf("layer %s not in report: %+v", tc.layer, rep.Layers)
			}
		})
	}
}

// A blocked user file must not hide a healthy workspace file: the project
// layer still lists, the user layer names its file.
func TestOpencodeBlockedUserKeepsProject(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	if err := os.MkdirAll(filepath.Join(home, ".config", "opencode"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".config", "opencode", "opencode.json"), []byte("{,broken"), 0o644); err != nil {
		t.Fatal(err)
	}
	cwd := t.TempDir()
	if err := os.WriteFile(filepath.Join(cwd, "opencode.json"), []byte(`{"mcp":{"docs":{"type":"remote","url":"https://d.example/mcp"}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	rep, err := (OpenCode{}).List(Paths{Home: home, Cwd: cwd})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Servers) != 1 || rep.Servers[0].Name != "docs" || rep.Servers[0].Scope != "project" {
		t.Fatalf("servers = %+v", rep.Servers)
	}
	for _, l := range rep.Layers {
		switch l.Scope {
		case "user":
			if !l.Exists || l.Error != "is not valid JSON" {
				t.Fatalf("user layer = %+v", l)
			}
		case "project":
			if l.Error != "" {
				t.Fatalf("project layer = %+v", l)
			}
		}
	}
}

// seedJSONCUser plants text at ~/.rooted (under a fresh HOME, XDG pinned
// empty so the default resolution lands there) and returns Paths pointing
// there plus the file path.
func seedJSONCUser(t *testing.T, rel, text string) (Paths, string) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", "")
	home := t.TempDir()
	path := filepath.Join(home, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	return Paths{Home: home}, path
}
