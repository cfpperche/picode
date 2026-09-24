package skills

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/clisettings"
)

func readFile(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	return string(b)
}

// rowOf reads the report and returns the row for one skill at one scope.
func rowOf(t *testing.T, cli, homeDir, wsDir, name string, scope Scope) Row {
	t.Helper()
	rep, err := Read(Query{CLI: cli, Home: homeDir, Workspace: wsDir, Trusted: func(string, string) *bool { v := true; return &v }})
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rep.Rows {
		if r.Name == name && r.Scope == scope {
			return r
		}
	}
	t.Fatalf("%s: no %s row for %s in %+v", cli, scope, name, rep.Rows)
	return Row{}
}

func flip(t *testing.T, cli, homeDir, wsDir string, row Row, on bool) ToggleResult {
	t.Helper()
	res, err := SetEnabled(context.Background(), ToggleReq{CLI: cli, Home: homeDir, Workspace: wsDir, Row: row, Enabled: on})
	if err != nil {
		t.Fatalf("%s %s → %v: %v", cli, row.Name, on, err)
	}
	return res
}

// The decision table of slice 3 (docs/plans/skills.md, "toggle" rows): for each
// CLI, off writes the vendor's key at the row's scope, the report reads it
// back as disabled, and on restores the user's bytes exactly.
func TestSwitchRoundTripPerCLI(t *testing.T) {
	cases := []struct {
		cli        string
		skillDir   func(h, ws string) string // where the skill lives
		scope      Scope
		file       func(h, ws string) string // the file the switch writes
		seed       string                    // the user's own bytes, kept
		wantOff    string                    // a fragment the off write adds
		machineNot bool                      // a machine-only switch: the row says so
	}{
		{"claude-code", func(h, _ string) string { return filepath.Join(h, ".claude", "skills") }, Machine,
			func(h, _ string) string { return filepath.Join(h, ".claude", "settings.json") },
			"{\n  \"model\": \"opus\"\n}\n", `"skillOverrides": {"pdf": "off"}`, false},
		{"claude-code", func(_, ws string) string { return filepath.Join(ws, ".claude", "skills") }, Workspace,
			func(_, ws string) string { return filepath.Join(ws, ".claude", "settings.local.json") },
			"", `"skillOverrides": {"pdf": "off"}`, false},
		{"codex", func(h, _ string) string { return filepath.Join(h, ".agents", "skills") }, Machine,
			func(h, _ string) string { return filepath.Join(h, ".codex", "config.toml") },
			"# mine\nmodel = \"gpt-5\"\n", "[[skills.config]]\npath = ", false},
		{"codex", func(_, ws string) string { return filepath.Join(ws, ".agents", "skills") }, Workspace,
			func(h, _ string) string { return filepath.Join(h, ".codex", "config.toml") },
			"", "enabled = false", false},
		{"opencode", func(h, _ string) string { return filepath.Join(h, ".config", "opencode", "skills") }, Machine,
			func(h, _ string) string { return filepath.Join(h, ".config", "opencode", "opencode.json") },
			"{\n  // theme\n  \"theme\": \"dark\"\n}\n", `"permission": {"skill": {"pdf": "deny"}}`, false},
		{"opencode", func(_, ws string) string { return filepath.Join(ws, ".opencode", "skills") }, Workspace,
			func(_, ws string) string { return filepath.Join(ws, "opencode.json") },
			"", `"deny"`, false},
		{"omp", func(h, _ string) string { return filepath.Join(h, ".omp", "agent", "skills") }, Machine,
			func(h, _ string) string { return filepath.Join(h, ".omp", "agent", "config.yml") },
			"theme: dark\n", "ignoredSkills", false},
		{"grok", func(h, _ string) string { return filepath.Join(h, ".grok", "skills") }, Machine,
			func(h, _ string) string { return filepath.Join(h, ".grok", "config.toml") },
			"model = \"grok-5\"\n", `disabled = ["pdf"]`, false},
		{"grok", func(_, ws string) string { return filepath.Join(ws, ".grok", "skills") }, Workspace,
			func(h, _ string) string { return filepath.Join(h, ".grok", "config.toml") },
			"", `disabled = ["pdf"]`, true},
		{"hermes", func(h, _ string) string { return filepath.Join(h, ".hermes", "skills") }, Machine,
			func(h, _ string) string { return filepath.Join(h, ".hermes", "config.yaml") },
			"model: x\n", "disabled", false},
	}
	for _, tc := range cases {
		t.Run(tc.cli+"/"+string(tc.scope), func(t *testing.T) {
			homeDir, _, wsDir := fixture(t)
			writeSkill(t, tc.skillDir(homeDir, wsDir), "pdf", "")
			file := tc.file(homeDir, wsDir)
			if tc.seed != "" {
				writeFile(t, file, tc.seed)
			}
			row := rowOf(t, tc.cli, homeDir, wsDir, "pdf", tc.scope)
			if row.Enabled == nil || !*row.Enabled {
				t.Fatalf("a skill nobody switched off must read enabled, got %+v", row.Enabled)
			}
			res := flip(t, tc.cli, homeDir, wsDir, row, false)
			if res.Enabled || res.File != file {
				t.Fatalf("off: %+v, want disabled in %s", res, file)
			}
			if got := readFile(t, file); !strings.Contains(got, tc.wantOff) || (tc.seed != "" && !strings.Contains(got, strings.TrimSpace(strings.SplitN(tc.seed, "\n", 2)[0]))) {
				t.Fatalf("off wrote:\n%s", got)
			}
			if (res.Note != "") != tc.machineNot {
				t.Fatalf("note %q, machine-only=%v", res.Note, tc.machineNot)
			}
			if after := rowOf(t, tc.cli, homeDir, wsDir, "pdf", tc.scope); after.Status != StatusDisabled || *after.Enabled {
				t.Fatalf("the report must read the switch back: %+v", after)
			}
			if res := flip(t, tc.cli, homeDir, wsDir, row, true); !res.Enabled {
				t.Fatalf("on: %+v", res)
			}
			if got := readFile(t, file); got != tc.seed {
				t.Fatalf("on must restore the user's bytes:\n%q\nwant\n%q", got, tc.seed)
			}
		})
	}
}

// A rule the user wrote in a stronger layer still decides, and the result
// names that file instead of claiming the switch moved.
func TestSwitchNamesTheLayerThatStillDecides(t *testing.T) {
	homeDir, _, wsDir := fixture(t)
	writeSkill(t, filepath.Join(homeDir, ".claude", "skills"), "pdf", "")
	writeFile(t, filepath.Join(wsDir, ".claude", "settings.json"), `{"skillOverrides": {"pdf": "off"}}`)
	row := rowOf(t, "claude-code", homeDir, wsDir, "pdf", Machine)
	if row.Status != StatusDisabled {
		t.Fatalf("the project layer turns it off: %+v", row)
	}
	res := flip(t, "claude-code", homeDir, wsDir, row, true)
	if res.Enabled || !strings.Contains(res.Note, filepath.Join(".claude", "settings.json")) {
		t.Fatalf("want the project file named as the decider, got %+v", res)
	}

	// OpenCode: a glob the user wrote keeps denying after PiCode's key goes.
	homeDir, _, wsDir = fixture(t)
	writeSkill(t, filepath.Join(homeDir, ".config", "opencode", "skills"), "pdf", "")
	writeFile(t, filepath.Join(homeDir, ".config", "opencode", "opencode.json"), `{"permission": {"skill": {"p*": "deny"}}}`)
	row = rowOf(t, "opencode", homeDir, wsDir, "pdf", Machine)
	if res := flip(t, "opencode", homeDir, wsDir, row, true); res.Enabled || res.Note == "" {
		t.Fatalf("a glob still denies: %+v", res)
	}
}

// Omp's workspace list replaces the machine list: the first workspace write
// carries the machine's entries, or they would come back on there.
func TestOmpWorkspaceSwitchKeepsTheMachineList(t *testing.T) {
	homeDir, _, wsDir := fixture(t)
	writeSkill(t, filepath.Join(wsDir, ".omp", "skills"), "pdf", "")
	writeFile(t, filepath.Join(homeDir, ".omp", "agent", "config.yml"), "skills:\n  ignoredSkills: [\"legacy*\"]\n")
	row := rowOf(t, "omp", homeDir, wsDir, "pdf", Workspace)
	flip(t, "omp", homeDir, wsDir, row, false)
	d, err := clisettings.OpenDoc(filepath.Join(wsDir, ".omp", "config.yml"), clisettings.FormatYAML)
	if err != nil {
		t.Fatal(err)
	}
	got, _, _ := d.Strings("skills", "ignoredSkills")
	if !reflect.DeepEqual(got, []string{"legacy*", "pdf"}) {
		t.Fatalf("workspace list %v, want the machine's entries plus pdf", got)
	}
}

// OpenCode's one-rule form covers every skill; PiCode does not rewrite it.
func TestOpenCodeRefusesTheOneRuleForm(t *testing.T) {
	homeDir, _, wsDir := fixture(t)
	writeSkill(t, filepath.Join(homeDir, ".config", "opencode", "skills"), "pdf", "")
	writeFile(t, filepath.Join(homeDir, ".config", "opencode", "opencode.json"), `{"permission": {"skill": "allow"}}`)
	row := rowOf(t, "opencode", homeDir, wsDir, "pdf", Machine)
	if _, err := SetEnabled(context.Background(), ToggleReq{CLI: "opencode", Home: homeDir, Workspace: wsDir, Row: row}); err == nil {
		t.Fatal("the one-rule form must be refused by name")
	}
}

// Muse writes its own settings: PiCode runs the vendor's verb with the
// skill's SKILL.md and the scope Muse names it by.
func TestMuseRunsItsOwnVerb(t *testing.T) {
	homeDir, _, wsDir := fixture(t)
	writeSkill(t, filepath.Join(wsDir, ".agents", "skills"), "pdf", "")
	row := rowOf(t, "muse", homeDir, wsDir, "pdf", Workspace)
	var got []string
	run := func(_ context.Context, home string, args ...string) error {
		if home != homeDir {
			t.Errorf("muse ran with HOME=%s, want %s", home, homeDir)
		}
		got = args
		// What `muse skills disable` writes, measured on 1.3.0.
		writeFile(t, filepath.Join(homeDir, ".config", "muse", "settings.json"),
			`{"schema_version":1,"skills":{"activation":{"projects":{"`+wsDir+`":{".agents/skills/pdf/SKILL.md":"off"}}}}}`)
		return nil
	}
	res, err := SetEnabled(context.Background(), ToggleReq{CLI: "muse", Home: homeDir, Workspace: wsDir, Row: row, Muse: run})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"skills", "disable", filepath.Join(row.Dir, "SKILL.md"), "--scope", "project", "--json", "--workspace", wsDir}
	if !reflect.DeepEqual(got, want) || res.Enabled {
		t.Fatalf("args %v res %+v", got, res)
	}
}

// Pi and agy have no switch: no row carries one, and a write is refused.
func TestNoSwitchWhereTheCLIHasNone(t *testing.T) {
	homeDir, _, wsDir := fixture(t)
	writeSkill(t, filepath.Join(homeDir, ".agents", "skills"), "pdf", "")
	for _, cli := range []string{"pi", "agy"} {
		rep, err := Read(Query{CLI: cli, Home: homeDir, Workspace: wsDir})
		if err != nil {
			t.Fatal(err)
		}
		for _, r := range rep.Rows {
			if r.Enabled != nil {
				t.Fatalf("%s row %s carries a switch", cli, r.Name)
			}
			if _, err := SetEnabled(context.Background(), ToggleReq{CLI: cli, Home: homeDir, Row: r}); err != ErrNoSwitch {
				t.Fatalf("%s: want ErrNoSwitch, got %v", cli, err)
			}
		}
	}
}

// TestEveryToggleIsARealSettingsKey, extended to skills (docs/plans/skills.md
// slice 3): the machine file a switch writes is a file the Settings pane
// declares for that CLI, so the two panes edit the same document.
func TestEverySkillSwitchWritesADeclaredSettingsFile(t *testing.T) {
	homeDir := t.TempDir()
	declared := clisettings.UserFiles(homeDir)
	for _, s := range specs {
		if s.Switch == "" {
			continue
		}
		var file string
		switch s.Switch {
		case SwitchClaude:
			file = filepath.Join(homeDir, ".claude", "settings.json")
		case SwitchCodex:
			file = codexFile(homeDir)
		case SwitchOpenCode:
			file = openCodeUserFile(homeDir)
		case SwitchOmp:
			file = filepath.Join(homeDir, ".omp", "agent", "config.yml")
		case SwitchGrok, SwitchHermes:
			file, _, _ = nameListFile(s.Switch, homeDir)
		case SwitchMuse:
			file = museFile(homeDir)
		}
		found := false
		for _, f := range declared[s.CLI] {
			if f == file {
				found = true
			}
		}
		if !found {
			t.Errorf("%s switch writes %s, which Settings does not declare (%v)", s.CLI, file, declared[s.CLI])
		}
	}
}

// Regressions from the adversarial review of slice 3 (2026-09-24): each one
// corrupted or deleted a real config file, or misread the switch.
func TestSwitchReviewRegressions(t *testing.T) {
	t.Run("codex header with a comment keeps every table after the element", func(t *testing.T) {
		homeDir, _, wsDir := fixture(t)
		writeSkill(t, filepath.Join(homeDir, ".agents", "skills"), "pdf", "")
		row := rowOf(t, "codex", homeDir, wsDir, "pdf", Machine)
		seed := "[[skills.config]]\npath = \"" + filepath.Join(row.Dir, "SKILL.md") + "\"\nenabled = false\n\n[[skills.config]] # keep\nname = \"other\"\nenabled = false\n\n[mcp_servers.github] # work\ncommand = \"gh-mcp\"\n"
		cfg := filepath.Join(homeDir, ".codex", "config.toml")
		writeFile(t, cfg, seed)
		flip(t, "codex", homeDir, wsDir, row, true)
		got := readFile(t, cfg)
		want := "[[skills.config]] # keep\nname = \"other\"\nenabled = false\n\n[mcp_servers.github] # work\ncommand = \"gh-mcp\"\n"
		if got != want {
			t.Fatalf("got\n%s\nwant\n%s", got, want)
		}
	})
	t.Run("hermes indentless list: the last entry goes, skills stays a map", func(t *testing.T) {
		homeDir, _, wsDir := fixture(t)
		writeSkill(t, filepath.Join(homeDir, ".hermes", "skills"), "pdf", "")
		cfg := filepath.Join(homeDir, ".hermes", "config.yaml")
		writeFile(t, cfg, "model: x\nskills:\n  disabled:\n  - pdf\n  trusted: true\ntoolsets:\n- web\n")
		row := rowOf(t, "hermes", homeDir, wsDir, "pdf", Machine)
		if row.Status != StatusDisabled {
			t.Fatalf("status %s", row.Status)
		}
		if res := flip(t, "hermes", homeDir, wsDir, row, true); !res.Enabled {
			t.Fatalf("%+v", res)
		}
		if got, want := readFile(t, cfg), "model: x\nskills:\n  trusted: true\ntoolsets:\n- web\n"; got != want {
			t.Fatalf("got %q want %q", got, want)
		}
	})
	t.Run("codex: an entry with both path and name is ignored; a name-only entry can be lifted", func(t *testing.T) {
		homeDir, _, wsDir := fixture(t)
		writeSkill(t, filepath.Join(homeDir, ".agents", "skills"), "pdf", "")
		cfg := filepath.Join(homeDir, ".codex", "config.toml")
		dir := filepath.Join(homeDir, ".agents", "skills", "pdf")
		writeFile(t, cfg, "[[skills.config]]\npath = \""+filepath.Join(dir, "SKILL.md")+"\"\nname = \"pdf\"\nenabled = false\n")
		if row := rowOf(t, "codex", homeDir, wsDir, "pdf", Machine); row.Status != StatusLoaded {
			t.Fatalf("both path and name: Codex ignores it, got %s", row.Status)
		}
		writeFile(t, cfg, "[[skills.config]]\nname = \"pdf\"\nenabled = false\n")
		row := rowOf(t, "codex", homeDir, wsDir, "pdf", Machine)
		if res := flip(t, "codex", homeDir, wsDir, row, true); !res.Enabled || res.Note != "" {
			t.Fatalf("%+v", res)
		}
	})
	t.Run("a no-op never deletes the user's empty file", func(t *testing.T) {
		homeDir, _, wsDir := fixture(t)
		writeSkill(t, filepath.Join(wsDir, ".claude", "skills"), "pdf", "")
		local := filepath.Join(wsDir, ".claude", "settings.local.json")
		writeFile(t, local, "{}\n")
		writeFile(t, filepath.Join(wsDir, ".claude", "settings.json"), `{"skillOverrides": {"pdf": "off"}}`)
		row := rowOf(t, "claude-code", homeDir, wsDir, "pdf", Workspace)
		flip(t, "claude-code", homeDir, wsDir, row, true)
		if got := readFile(t, local); got != "{}\n" {
			t.Fatalf("the user's file changed: %q", got)
		}
	})
	t.Run("opencode merges user then project in the user's key order", func(t *testing.T) {
		homeDir, repo, wsDir := fixture(t)
		writeSkill(t, filepath.Join(homeDir, ".config", "opencode", "skills"), "foo", "")
		writeFile(t, filepath.Join(homeDir, ".config", "opencode", "opencode.json"), `{"permission": {"skill": {"*": "deny", "foo": "allow"}}}`)
		writeFile(t, filepath.Join(wsDir, "opencode.json"), `{"permission": {"skill": {"*": "deny"}}}`)
		if row := rowOf(t, "opencode", homeDir, wsDir, "foo", Machine); row.Status != StatusLoaded {
			t.Fatalf("foo's allow stays after the merged *: got %s", row.Status)
		}
		// A rule at the repository root reaches a workspace in a subfolder.
		writeFile(t, filepath.Join(repo, "opencode.json"), `{"permission": {"skill": {"foo": "deny"}}}`)
		if row := rowOf(t, "opencode", homeDir, wsDir, "foo", Machine); row.Status != StatusDisabled {
			t.Fatalf("the root file denies foo: got %s", row.Status)
		}
	})
}
