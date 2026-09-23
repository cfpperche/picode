package clisettings

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The fixtures below are trimmed copies of the real files on the owner's
// machine (2026-09-20), comments and key order included: the whole point of
// the surgical writer is that a hand-maintained config survives a save.

const codexTOML = `model = "gpt-6-astra"
approval_policy = "never"   # set during the tachyon run
sandbox_mode = "danger-full-access"
model_reasoning_effort = "medium"

[features]
terminal_resize_reflow = true
memories = true

[projects."/home/goat/spock"]
trust_level = "trusted"
`

const hermesYAML = `model:
  default: glm-5.3-flash
  provider: zai
  base_url: https://api.z.ai/api/coding/paas/v4
agent:
  max_turns: 60
  verify_on_stop: false
display:
  compact: false
  show_reasoning: false
memory:
  # raised after the 2200 default truncated a note
  memory_enabled: true
  user_profile_enabled: true
  memory_char_limit: 2200
_config_version: 45
`

const museJSON = `{
  "model": "muse-spark-1.3-contributor",
  "permissions": {
    "default_profile": ":unrestricted",
    "schema_version": 1
  },
  "provider": "meta",
  "reasoning_effort": "max",
  "schema_version": 1
}
`

const opencodeJSONC = `{
  // the team's shared default
  "model": "anthropic/claude-sonnet-4.6",
  "autoupdate": true
}
`

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// TestWriteTouchesOnlyTheValue is the load-bearing test of ADR-0163: after a
// save, every byte the patch did not name is still where it was. A parser
// round-trip would pass a "keys survived" check and still reflow the file;
// this compares the text.
func TestWriteTouchesOnlyTheValue(t *testing.T) {
	for _, tc := range []struct {
		name   string
		cli    string
		file   []string
		body   string
		key    string
		value  any
		before string
		after  string
	}{
		{
			name: "toml root scalar keeps its trailing comment",
			cli:  "codex", file: []string{".codex", "config.toml"}, body: codexTOML,
			key: "approval_policy", value: "on-request",
			before: `approval_policy = "never"   # set during the tachyon run`,
			after:  `approval_policy = "on-request"   # set during the tachyon run`,
		},
		{
			name: "toml nested bool",
			cli:  "codex", file: []string{".codex", "config.toml"}, body: codexTOML,
			key: "features.memories", value: false,
			before: "memories = true", after: "memories = false",
		},
		{
			name: "yaml nested scalar under a commented block",
			cli:  "hermes", file: []string{".hermes", "config.yaml"}, body: hermesYAML,
			key: "memory.memory_char_limit", value: 4000,
			before: "  memory_char_limit: 2200", after: "  memory_char_limit: 4000",
		},
		{
			name: "yaml nested string",
			cli:  "hermes", file: []string{".hermes", "config.yaml"}, body: hermesYAML,
			key: "model.default", value: "glm-5.3",
			before: "  default: glm-5.3-flash", after: "  default: glm-5.3",
		},
		{
			name: "json nested string",
			cli:  "muse", file: []string{".config", "muse", "settings.json"}, body: museJSON,
			key: "permissions.default_profile", value: ":standard",
			before: `    "default_profile": ":unrestricted",`, after: `    "default_profile": ":standard",`,
		},
		{
			name: "jsonc keeps its comment",
			cli:  "opencode", file: []string{".config", "opencode", "opencode.json"}, body: opencodeJSONC,
			key: "model", value: "openai/gpt-5.3",
			before: `  "model": "anthropic/claude-sonnet-4.6",`, after: `  "model": "openai/gpt-5.3",`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			path := filepath.Join(append([]string{home}, tc.file...)...)
			write(t, path, tc.body)
			if err := Apply(tc.cli, Paths{Home: home}, Patch{Scope: "user", Set: map[string]any{tc.key: tc.value}}); err != nil {
				t.Fatalf("apply: %v", err)
			}
			got := read(t, path)
			if !strings.Contains(got, tc.after) {
				t.Fatalf("want line %q in:\n%s", tc.after, got)
			}
			// Everything else is byte-identical.
			want := strings.Replace(tc.body, tc.before, tc.after, 1)
			if got != want {
				t.Fatalf("file changed beyond the value:\n--- got ---\n%s\n--- want ---\n%s", got, want)
			}
		})
	}
}

// TestReadReportsOnlyWhatALayerSets keeps the pane honest: a value a file does
// not carry is not reported as set there, which is what makes "Set here"
// versus "From Global" true.
func TestReadReportsOnlyWhatALayerSets(t *testing.T) {
	home := t.TempDir()
	cwd := t.TempDir()
	write(t, filepath.Join(home, ".grok", "config.toml"), "[models]\ndefault = \"grok-4.6\"\n\n[ui]\nyolo = false\n")
	write(t, filepath.Join(cwd, ".grok", "config.toml"), "[ui]\nyolo = true\n")
	rep, err := Read("grok", Paths{Home: home, Cwd: cwd})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Layers) != 2 {
		t.Fatalf("want user and project layers, got %d", len(rep.Layers))
	}
	user, project := rep.Layers[0], rep.Layers[1]
	if user.Values["models.default"] != "grok-4.6" || user.Values["ui.yolo"] != false {
		t.Fatalf("user layer: %#v", user.Values)
	}
	if _, ok := project.Values["models.default"]; ok {
		t.Fatalf("project layer must not claim a key it does not set: %#v", project.Values)
	}
	if project.Values["ui.yolo"] != true {
		t.Fatalf("project layer: %#v", project.Values)
	}
}

// TestMissingFileIsNotAnError: a CLI running on its own defaults has no file,
// and that is a layer with no values, not a broken pane.
func TestMissingFileIsNotAnError(t *testing.T) {
	rep, err := Read("codex", Paths{Home: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Layers) != 1 || rep.Layers[0].Exists || len(rep.Layers[0].Values) != 0 {
		t.Fatalf("want one empty layer, got %#v", rep.Layers)
	}
	if !rep.Layers[0].Writable {
		t.Fatal("a missing file is still writable")
	}
}

// TestDeclaredLayerWithoutAWorkspaceIsReportedNotHidden: a CLI that has a
// workspace file, opened without a workspace, must still show the layer and
// say what it needs. Hiding it let a link to that layer edit the machine file
// (live QA, 2026-09-20).
func TestDeclaredLayerWithoutAWorkspaceIsReportedNotHidden(t *testing.T) {
	rep, err := Read("grok", Paths{Home: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Layers) != 2 {
		t.Fatalf("want both declared layers, got %d", len(rep.Layers))
	}
	project := rep.Layers[1]
	if project.Scope != "project" || project.Writable || project.Path != "" || project.Note == "" {
		t.Fatalf("the workspace layer must be reported, unwritable and explained: %#v", project)
	}
	// And it is still refused as a write target.
	if err := Apply("grok", Paths{Home: t.TempDir()}, Patch{Scope: "project", Set: map[string]any{"ui.yolo": true}}); err == nil {
		t.Fatal("a write with no workspace must be refused")
	}
}

// TestSingleFileCLIHasNoProjectLayer: no invented scope (ADR-0099 §5).
func TestSingleFileCLIHasNoProjectLayer(t *testing.T) {
	for _, cli := range []string{"codex", "hermes", "muse", "agy"} {
		rep, err := Read(cli, Paths{Home: t.TempDir(), Cwd: t.TempDir()})
		if err != nil {
			t.Fatal(err)
		}
		if len(rep.Layers) != 1 {
			t.Fatalf("%s: want one layer, got %d", cli, len(rep.Layers))
		}
		if err := Apply(cli, Paths{Home: t.TempDir(), Cwd: t.TempDir()}, Patch{Scope: "project", Set: map[string]any{"model": "x"}}); err == nil {
			t.Fatalf("%s: a project write must be refused", cli)
		}
	}
}

// TestUndeclaredKeyIsRefused: PiCode writes the vendor's names, and only the
// ones a CLI declared.
func TestUndeclaredKeyIsRefused(t *testing.T) {
	home := t.TempDir()
	write(t, filepath.Join(home, ".codex", "config.toml"), codexTOML)
	err := Apply("codex", Paths{Home: home}, Patch{Scope: "user", Set: map[string]any{"telemetry": false}})
	if err == nil || !strings.Contains(err.Error(), "not a setting PiCode manages") {
		t.Fatalf("want a refusal naming the key, got %v", err)
	}
	if read(t, filepath.Join(home, ".codex", "config.toml")) != codexTOML {
		t.Fatal("a refused patch must not touch the file")
	}
}

// TestStaleFileIsRefused: the CLI writes this file too. A save built on a
// report older than the file on disk is refused, not merged blindly.
func TestStaleFileIsRefused(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, ".codex", "config.toml")
	write(t, path, codexTOML)
	rep, err := Read("codex", Paths{Home: home})
	if err != nil {
		t.Fatal(err)
	}
	stale := rep.Layers[0].Revision
	write(t, path, codexTOML+"\nnotify = []\n")
	err = Apply("codex", Paths{Home: home}, Patch{Scope: "user", Revision: stale, Set: map[string]any{"model": "gpt-6"}})
	if err != ErrStale {
		t.Fatalf("want ErrStale, got %v", err)
	}
	if !strings.Contains(read(t, path), "notify = []") {
		t.Fatal("the other writer's line must survive a refused save")
	}
	// Force is the explicit override, and only then.
	if err := Apply("codex", Paths{Home: home}, Patch{Scope: "user", Revision: stale, Force: true, Set: map[string]any{"model": "gpt-6"}}); err != nil {
		t.Fatalf("forced apply: %v", err)
	}
	if !strings.Contains(read(t, path), `model = "gpt-6"`) {
		t.Fatal("forced apply did not write")
	}
}

// TestUnparseableFileIsNeverOverwritten (ADR-0099 §4).
func TestUnparseableFileIsNeverOverwritten(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, ".config", "muse", "settings.json")
	broken := "{ \"model\": \"muse\", oops }\n"
	write(t, path, broken)
	rep, err := Read("muse", Paths{Home: home})
	if err != nil {
		t.Fatal(err)
	}
	layer := rep.Layers[0]
	if layer.Error == "" || layer.Writable {
		t.Fatalf("a rejected file must be reported and not writable: %#v", layer)
	}
	if err := Apply("muse", Paths{Home: home}, Patch{Scope: "user", Set: map[string]any{"model": "x"}}); err == nil {
		t.Fatal("want a refusal")
	}
	if read(t, path) != broken {
		t.Fatal("the file changed under a refused save")
	}
}

// TestResetRemovesTheKey: an override handed back to the CLI's own default.
func TestResetRemovesTheKey(t *testing.T) {
	for _, tc := range []struct {
		cli, key string
		file     []string
		body     string
		gone     string
		kept     string
	}{
		{"codex", "model_reasoning_effort", []string{".codex", "config.toml"}, codexTOML, "model_reasoning_effort", "sandbox_mode"},
		{"hermes", "display.compact", []string{".hermes", "config.yaml"}, hermesYAML, "compact:", "show_reasoning"},
		{"muse", "provider", []string{".config", "muse", "settings.json"}, museJSON, `"provider"`, `"reasoning_effort"`},
	} {
		t.Run(tc.cli, func(t *testing.T) {
			home := t.TempDir()
			path := filepath.Join(append([]string{home}, tc.file...)...)
			write(t, path, tc.body)
			if err := Apply(tc.cli, Paths{Home: home}, Patch{Scope: "user", Reset: []string{tc.key}}); err != nil {
				t.Fatalf("apply: %v", err)
			}
			got := read(t, path)
			if strings.Contains(got, tc.gone) {
				t.Fatalf("%s should be gone:\n%s", tc.gone, got)
			}
			if !strings.Contains(got, tc.kept) {
				t.Fatalf("%s must survive:\n%s", tc.kept, got)
			}
			if _, err := Read(tc.cli, Paths{Home: home}); err != nil {
				t.Fatalf("the result must still parse: %v", err)
			}
		})
	}
}

// TestInsertCreatesTheKeyInPlace: a setting the file does not carry yet is
// added where the format expects it, and the document still parses.
func TestInsertCreatesTheKeyInPlace(t *testing.T) {
	for _, tc := range []struct {
		cli, key string
		value    any
		file     []string
		body     string
		want     string
	}{
		{"codex", "memories.use_memories", true, []string{".codex", "config.toml"}, codexTOML, "[memories]\nuse_memories = true"},
		{"codex", "features.memories", false, []string{".codex", "config.toml"}, "model = \"gpt-6\"\n", "[features]\nmemories = false"},
		{"hermes", "memory.user_char_limit", 900, []string{".hermes", "config.yaml"}, hermesYAML, "  user_char_limit: 900"},
		{"hermes", "agent.max_turns", 12, []string{".hermes", "config.yaml"}, "model:\n  default: x\n", "agent:\n  max_turns: 12"},
		{"muse", "reasoning_effort", "low", []string{".config", "muse", "settings.json"}, "{\n  \"model\": \"muse\"\n}\n", `"reasoning_effort": "low"`},
		{"muse", "permissions.default_profile", ":safe", []string{".config", "muse", "settings.json"}, "{\n  \"model\": \"muse\"\n}\n", `"default_profile": ":safe"`},
		{"agy", "model", "Gemini 3.8 Pro", []string{".gemini", "antigravity-cli", "settings.json"}, "{}\n", `"model": "Gemini 3.8 Pro"`},
	} {
		t.Run(tc.cli+"/"+tc.key, func(t *testing.T) {
			home := t.TempDir()
			path := filepath.Join(append([]string{home}, tc.file...)...)
			write(t, path, tc.body)
			if err := Apply(tc.cli, Paths{Home: home}, Patch{Scope: "user", Set: map[string]any{tc.key: tc.value}}); err != nil {
				t.Fatalf("apply: %v", err)
			}
			got := read(t, path)
			if !strings.Contains(got, tc.want) {
				t.Fatalf("want %q in:\n%s", tc.want, got)
			}
			rep, err := Read(tc.cli, Paths{Home: home})
			if err != nil {
				t.Fatalf("result does not parse: %v", err)
			}
			if _, ok := rep.Layers[0].Values[tc.key]; !ok {
				t.Fatalf("the key is not read back: %#v", rep.Layers[0].Values)
			}
		})
	}
}

// TestAgyStructuresAreNeverTouched: Antigravity's settings file carries the
// title reporter PiCode itself installs and a workspace list the CLI keeps.
// Declaring one scalar is the point — the rest must come out byte-identical.
func TestAgyStructuresAreNeverTouched(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, ".gemini", "antigravity-cli", "settings.json")
	body := `{
  "model": "Gemini 3.8 Flash (Medium)",
  "title": {
    "type": "command",
    "command": "/home/goat/.picode/intercept/agy-title.sh",
    "enabled": true
  },
  "trustedWorkspaces": [
    "/home/goat",
    "/home/goat/picode"
  ]
}
`
	write(t, path, body)
	if err := Apply("agy", Paths{Home: home}, Patch{Scope: "user", Set: map[string]any{"model": "Gemini 3.8 Pro"}}); err != nil {
		t.Fatal(err)
	}
	got := read(t, path)
	want := strings.Replace(body, `"Gemini 3.8 Flash (Medium)"`, `"Gemini 3.8 Pro"`, 1)
	if got != want {
		t.Fatalf("the reporter or the workspace list moved:\n%s", got)
	}
}

// TestEveryDeclaredFieldRoundTrips walks the whole catalog: for each CLI and
// each declared key, a write followed by a read returns the value. This is the
// guard that catches a schema row whose dotted path the writer cannot place.
func TestEveryDeclaredFieldRoundTrips(t *testing.T) {
	for _, cli := range Supported() {
		s := For(cli)
		for _, f := range s.fields {
			t.Run(cli+"/"+f.Key, func(t *testing.T) {
				home, cwd := t.TempDir(), t.TempDir()
				p := Paths{Home: home, Cwd: cwd}
				var value any
				switch f.Kind {
				case KindBool:
					value = true
				case KindNumber:
					value = 7
				case KindSelect:
					value = f.Options[0].Value
				case KindList:
					// A list field is written and read back as a list (ADR-0181).
					value = []any{"picode-probe"}
				default:
					value = "picode-probe"
				}
				for _, layer := range s.layers {
					if !f.allows(layer.scope) {
						continue
					}
					if err := Apply(cli, p, Patch{Scope: layer.scope, Set: map[string]any{f.Key: value}}); err != nil {
						t.Fatalf("%s layer: %v", layer.scope, err)
					}
				}
				rep, err := Read(cli, p)
				if err != nil {
					t.Fatal(err)
				}
				for _, layer := range rep.Layers {
					if !f.allows(layer.Scope) {
						continue // a field kept to one layer is written and read there only
					}
					got, ok := layer.Values[f.Key]
					if !ok {
						t.Fatalf("%s layer lost %s", layer.Scope, f.Key)
					}
					if f.Kind == KindNumber {
						// TOML and YAML decode integers as int64, JSON as float64.
						switch n := got.(type) {
						case int64:
							got = int(n)
						case float64:
							got = int(n)
						}
					}
					if f.Kind == KindList {
						gotList, err := asStrings(got)
						if err != nil || strings.Join(gotList, ",") != "picode-probe" {
							t.Fatalf("%s layer: got %#v want [picode-probe]", layer.Scope, got)
						}
						continue
					}
					if got != value {
						t.Fatalf("%s layer: got %#v want %#v", layer.Scope, got, value)
					}
				}
			})
		}
	}
}

// TestSecretValuesAreRedacted keeps credential-shaped settings out of the API.
func TestSecretValuesAreRedacted(t *testing.T) {
	f := Field{Key: "token", Secret: true}
	if got := redact(f, "sk-live-1234"); got != "••••••" {
		t.Fatalf("got %v", got)
	}
	if got := redact(Field{Key: "model"}, "opus"); got != "opus" {
		t.Fatalf("a plain field must not be redacted, got %v", got)
	}
}

// TestLayerNoteReadsAsASentence: the note is chrome copy, so it has to be a
// sentence a terminal-averse reader can act on, not a label glued into one.
func TestLayerNoteReadsAsASentence(t *testing.T) {
	rep, err := Read("grok", Paths{Home: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	note := rep.Layers[1].Note
	if note != "Open this CLI from a workspace to edit its workspace settings." {
		t.Fatalf("note reads %q", note)
	}
}

// TestInsertedJSONMemberReadsLikeTheFile: a key added to a JSON document must
// land as a normal member. Anchoring the comma on the closing brace left it
// alone on its own line (live QA, 2026-09-20).
func TestInsertedJSONMemberReadsLikeTheFile(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, ".config", "opencode", "opencode.json")
	write(t, path, "{\n  // the team's shared default\n  \"model\": \"anthropic/claude-sonnet-4.6\",\n  \"mcp\": {}\n}\n")
	if err := Apply("opencode", Paths{Home: home}, Patch{Scope: "user", Set: map[string]any{"theme": "tokyonight"}}); err != nil {
		t.Fatal(err)
	}
	want := "{\n  // the team's shared default\n  \"model\": \"anthropic/claude-sonnet-4.6\",\n  \"mcp\": {},\n  \"theme\": \"tokyonight\"\n}\n"
	if got := read(t, path); got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
	// And an empty document gets its first member indented, not glued.
	empty := filepath.Join(home, ".config", "muse", "settings.json")
	write(t, empty, "{}\n")
	if err := Apply("muse", Paths{Home: home}, Patch{Scope: "user", Set: map[string]any{"model": "muse"}}); err != nil {
		t.Fatal(err)
	}
	if got := read(t, empty); got != "{\n  \"model\": \"muse\"\n}\n" {
		t.Fatalf("empty document: %q", got)
	}
}

// TestYAMLNeverWritesASameNamedKeyAtAnotherDepth is the regression for the
// worst defect the adversarial review found: `memory.memory_enabled` landed
// on `foo.memory.memory_enabled` and left the key the pane showed untouched.
// The reader resolves the dotted path correctly, so a mis-targeted write was
// silent — the pane said saved and the CLI kept its old value.
func TestYAMLNeverWritesASameNamedKeyAtAnotherDepth(t *testing.T) {
	for _, tc := range []struct{ name, body, want string }{
		{
			name: "decoy block before the real one",
			body: "foo:\n  memory:\n    memory_enabled: true\nmemory:\n  memory_enabled: true\n",
			want: "foo:\n  memory:\n    memory_enabled: true\nmemory:\n  memory_enabled: false\n",
		},
		{
			name: "decoy block after the real one",
			body: "memory:\n  memory_enabled: true\nfoo:\n  memory:\n    memory_enabled: true\n",
			want: "memory:\n  memory_enabled: false\nfoo:\n  memory:\n    memory_enabled: true\n",
		},
		{
			name: "only a nested namesake: the root key is created, not hijacked",
			body: "agent:\n  memory:\n    memory_enabled: true\n",
			want: "agent:\n  memory:\n    memory_enabled: true\nmemory:\n  memory_enabled: false\n",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			path := filepath.Join(home, ".hermes", "config.yaml")
			write(t, path, tc.body)
			if err := Apply("hermes", Paths{Home: home}, Patch{Scope: "user", Set: map[string]any{"memory.memory_enabled": false}}); err != nil {
				t.Fatal(err)
			}
			if got := read(t, path); got != tc.want {
				t.Fatalf("got:\n%s\nwant:\n%s", got, tc.want)
			}
			rep, err := Read("hermes", Paths{Home: home})
			if err != nil {
				t.Fatal(err)
			}
			if rep.Layers[0].Values["memory.memory_enabled"] != false {
				t.Fatalf("the key the pane reads is not the key that was written: %#v", rep.Layers[0].Values)
			}
		})
	}
}

// TestYAMLRefusesBlockValues: a key whose value is a block, a block scalar, an
// anchor or an alias is not a scalar, so the writer inserts beside it rather
// than splicing bytes into a structure it does not understand.
func TestYAMLRefusesBlockValues(t *testing.T) {
	for _, body := range []string{
		"memory:\n  memory_enabled: |\n    a long\n    block\n",
		"memory:\n  memory_enabled: &anchor true\n",
		"memory:\n  memory_enabled:\n    nested: true\n",
	} {
		home := t.TempDir()
		path := filepath.Join(home, ".hermes", "config.yaml")
		write(t, path, body)
		// The write must either refuse or leave the block intact; it must never
		// splice into the middle of it.
		_ = Apply("hermes", Paths{Home: home}, Patch{Scope: "user", Set: map[string]any{"memory.memory_enabled": false}})
		got := read(t, path)
		if _, err := decode([]byte(got), FormatYAML, path); err != nil {
			t.Fatalf("the result does not parse:\n%s", got)
		}
	}
}

// TestYAMLInsertLandsInTheRightBlock: the same depth confusion in the insert
// path would add a key under a namesake block.
func TestYAMLInsertLandsInTheRightBlock(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, ".hermes", "config.yaml")
	write(t, path, "foo:\n  memory:\n    other: 1\nmemory:\n  memory_enabled: true\n")
	if err := Apply("hermes", Paths{Home: home}, Patch{Scope: "user", Set: map[string]any{"memory.user_char_limit": 900}}); err != nil {
		t.Fatal(err)
	}
	want := "foo:\n  memory:\n    other: 1\nmemory:\n  memory_enabled: true\n  user_char_limit: 900\n"
	if got := read(t, path); got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}
