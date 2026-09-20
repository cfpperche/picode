package clisettings

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The fixtures in testdata are shaped like the real files of the nine CLIs
// installed on the owner's machine on 2026-09-20 — same keys, same nesting,
// same comments, same quirks (Omp's file ends without a newline; Antigravity
// carries the title reporter PiCode installs and a workspace list the CLI
// keeps) — with personal paths replaced. They are the golden files ADR-0150
// asks a config driver to carry.
var golden = map[string]string{
	"claude-code": "claude-code.settings.json",
	"codex":       "codex.config.toml",
	"grok":        "grok.config.toml",
	"hermes":      "hermes.config.yaml",
	"opencode":    "opencode.opencode.json",
	"muse":        "muse.settings.json",
	"agy":         "agy.settings.json",
	"omp":         "omp.config.yml",
}

func seedGolden(t *testing.T, cli string) (Paths, string) {
	t.Helper()
	home := t.TempDir()
	s := For(cli)
	path := s.layers[0].file(Paths{Home: home})
	body, err := os.ReadFile(filepath.Join("testdata", golden[cli]))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatal(err)
	}
	return Paths{Home: home}, path
}

// TestGoldenReadFindsEveryDeclaredKeyThatIsSet proves the declarations name
// keys the vendors actually use: every key the fixture sets is read back, and
// no key the fixture leaves out is reported as set.
func TestGoldenReadFindsEveryDeclaredKeyThatIsSet(t *testing.T) {
	want := map[string][]string{
		"claude-code": {"model", "permissions.defaultMode"},
		"codex":       {"model", "approval_policy", "sandbox_mode", "model_reasoning_effort", "features.memories"},
		"grok":        {"models.default", "models.default_reasoning_effort", "ui.permission_mode", "ui.yolo", "ui.compact_mode", "cli.auto_update"},
		"hermes":      {"model.default", "model.provider", "agent.max_turns", "display.compact", "display.show_reasoning", "memory.memory_enabled", "memory.user_profile_enabled", "memory.memory_char_limit", "memory.user_char_limit"},
		"opencode":    {"model", "autoupdate"},
		"muse":        {"model", "provider", "reasoning_effort", "permissions.default_profile"},
		"agy":         {"model"},
		"omp":         {"modelRoles.default", "symbolPreset", "theme.dark"},
	}
	for cli, keys := range want {
		t.Run(cli, func(t *testing.T) {
			p, _ := seedGolden(t, cli)
			rep, err := Read(cli, p)
			if err != nil {
				t.Fatal(err)
			}
			values := rep.Layers[0].Values
			for _, key := range keys {
				if _, ok := values[key]; !ok {
					t.Errorf("%s is set in the fixture but was not read: %#v", key, values)
				}
			}
			if len(values) != len(keys) {
				t.Errorf("read %d keys, the fixture sets %d: %#v", len(values), len(keys), values)
			}
		})
	}
}

// TestGoldenAddThenResetRestoresTheFile is the round-trip that caught two real
// bugs: a `[memories]` table and a `memory:` block left behind after their last
// key was handed back. Adding every key a CLI declares and then resetting them
// all must return the document to the bytes it had.
//
// The one accepted difference is a final newline: appending to a file that had
// none adds one, and a later removal cannot take it back. Ending a text file
// with a newline is the POSIX shape, so PiCode keeps it rather than restoring
// a truncated last line.
func TestGoldenAddThenResetRestoresTheFile(t *testing.T) {
	for cli := range golden {
		t.Run(cli, func(t *testing.T) {
			p, path := seedGolden(t, cli)
			before := read(t, path)
			rep, err := Read(cli, p)
			if err != nil {
				t.Fatal(err)
			}
			set := map[string]any{}
			var reset []string
			for _, f := range For(cli).fields {
				if _, ok := rep.Layers[0].Values[f.Key]; ok {
					continue
				}
				switch f.Kind {
				case KindBool:
					set[f.Key] = true
				case KindNumber:
					set[f.Key] = 5
				case KindSelect:
					set[f.Key] = f.Options[len(f.Options)-1].Value
				default:
					set[f.Key] = "picode-probe"
				}
				reset = append(reset, f.Key)
			}
			if len(set) == 0 {
				t.Skip("the fixture already sets every declared key")
			}
			if err := Apply(cli, p, Patch{Scope: "user", Set: set}); err != nil {
				t.Fatalf("add: %v", err)
			}
			added, err := Read(cli, p)
			if err != nil {
				t.Fatalf("the added keys do not parse: %v", err)
			}
			for key := range set {
				if _, ok := added.Layers[0].Values[key]; !ok {
					t.Fatalf("%s was written but is not read back", key)
				}
			}
			if err := Apply(cli, p, Patch{Scope: "user", Reset: reset}); err != nil {
				t.Fatalf("reset: %v", err)
			}
			after := read(t, path)
			if strings.TrimRight(after, "\n") != strings.TrimRight(before, "\n") {
				t.Fatalf("add+reset of %d keys did not restore the file\n--- before ---\n%s\n--- after ---\n%s", len(set), before, after)
			}
			if strings.HasSuffix(before, "\n") && after != before {
				t.Fatalf("a file that already ended with a newline must come back byte-identical")
			}
		})
	}
}

// TestGoldenSingleKeyWriteIsSurgical: one save changes one value and nothing
// else, on documents that carry comments, nested tables and arrays.
func TestGoldenSingleKeyWriteIsSurgical(t *testing.T) {
	for _, tc := range []struct {
		cli, key string
		value    any
		was, now string
	}{
		{"codex", "sandbox_mode", "workspace-write", `sandbox_mode = "danger-full-access"`, `sandbox_mode = "workspace-write"`},
		{"grok", "ui.permission_mode", "plan", `permission_mode = "always-approve"`, `permission_mode = "plan"`},
		{"grok", "ui.yolo", true, "yolo = false", "yolo = true"},
		{"hermes", "memory.memory_enabled", false, "  memory_enabled: true", "  memory_enabled: false"},
		{"hermes", "memory.memory_char_limit", 4000, "  memory_char_limit: 2200", "  memory_char_limit: 4000"},
		{"opencode", "model", "openai/gpt-5.3", `  "model": "anthropic/claude-sonnet-4.6",`, `  "model": "openai/gpt-5.3",`},
		{"claude-code", "permissions.defaultMode", "plan", `    "defaultMode": "auto",`, `    "defaultMode": "plan",`},
		{"muse", "permissions.default_profile", ":safe", `    "default_profile": ":unrestricted",`, `    "default_profile": ":safe",`},
		{"agy", "model", "Gemini 3.8 Pro", `  "model": "Gemini 3.8 Flash (Medium)",`, `  "model": "Gemini 3.8 Pro",`},
		{"omp", "theme.dark", "carbon", "  dark: titanium", "  dark: carbon"},
	} {
		t.Run(tc.cli+"/"+tc.key, func(t *testing.T) {
			p, path := seedGolden(t, tc.cli)
			before := read(t, path)
			if err := Apply(tc.cli, p, Patch{Scope: "user", Set: map[string]any{tc.key: tc.value}}); err != nil {
				t.Fatal(err)
			}
			got := read(t, path)
			want := strings.Replace(before, tc.was, tc.now, 1)
			if got != want {
				t.Fatalf("the save moved more than the value:\n--- got ---\n%s\n--- want ---\n%s", got, want)
			}
		})
	}
}
