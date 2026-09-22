package clisettings

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The decision table for the model-role matrix (ADR-0181). Each row is one set
// of conditions and the action it must produce; every row below is a case here
// or in TestRoleWrites.
//
//	| The file says                          | The report does                         |
//	|----------------------------------------|-----------------------------------------|
//	| nothing                                | 15 rows, all unset, drawn at `auto`     |
//	| modelRoles.default                      | that row set here, the rest unset       |
//	| only the workspace sets a role          | the row exists on both layers           |
//	| modelTags names a role nothing assigns  | the row exists, labelled from the tag   |
//	| cycleOrder names a role nothing assigns | the row exists                          |
//	| a role id the CLI would refuse          | no row                                  |
//	| a chain keyed by a provider/model       | one row per chain, sorted               |
//	| a chain written as a map                | the row is unreadable, not a value      |

func ompHome(t *testing.T, body string) string {
	t.Helper()
	home := t.TempDir()
	path := filepath.Join(home, ".omp", "agent", "config.yml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return home
}

func fieldByKey(rep Report, key string) (Field, bool) {
	for _, f := range rep.Fields {
		if f.Key == key {
			return f, true
		}
	}
	return Field{}, false
}

func TestRoleRows(t *testing.T) {
	for _, tc := range []struct {
		name       string
		body       string
		want       []string // keys that must be rows
		absent     []string
		setHere    map[string]any
		labels     map[string]string
		unreadable []string
	}{
		{
			name: "an empty file still lists every role the CLI has",
			body: "",
			want: []string{"modelRoles.default", "modelRoles.smol", "modelRoles.judge", "cycleOrder"},
		},
		{
			name:    "an assigned role reports its value",
			body:    "modelRoles:\n  default: deepseek/deepseek-flash:max\n",
			want:    []string{"modelRoles.default"},
			setHere: map[string]any{"modelRoles.default": "deepseek/deepseek-flash:max"},
		},
		{
			name:   "a role only a tag names is still a row, labelled by the tag",
			body:   "modelTags:\n  review:\n    name: REVIEW\n    color: accent\n",
			want:   []string{"modelRoles.review"},
			labels: map[string]string{"modelRoles.review": "REVIEW"},
		},
		{
			name: "a role only the cycle names is a row",
			body: "cycleOrder: [smol, default, scout]\n",
			want: []string{"modelRoles.scout"},
		},
		{
			name:   "a role id the CLI itself would refuse is not a row",
			body:   "modelRoles:\n  \"9lives\": openai/gpt-5\n  \"two words\": openai/gpt-5\n",
			absent: []string{"modelRoles.9lives", "modelRoles.two words"},
		},
		{
			name: "chains are one row each, sorted",
			body: "retry:\n  fallbackChains:\n    \"openai/gpt-5\": [anthropic/claude-sonnet-4-6]\n    default: [\"@smol\"]\n",
			want: []string{"retry.fallbackChains.default", "retry.fallbackChains.openai/gpt-5"},
			setHere: map[string]any{
				"retry.fallbackChains.default": []any{"@smol"},
			},
		},
		{
			name:       "a chain written as a map is reported unreadable, never a value",
			body:       "retry:\n  fallbackChains:\n    default:\n      first: openai/gpt-5\n",
			want:       []string{"retry.fallbackChains.default"},
			unreadable: []string{"retry.fallbackChains.default"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rep, err := Read("omp", Paths{Home: ompHome(t, tc.body)})
			if err != nil {
				t.Fatal(err)
			}
			for _, key := range tc.want {
				if _, ok := fieldByKey(rep, key); !ok {
					t.Errorf("%s is not a row", key)
				}
			}
			for _, key := range tc.absent {
				if _, ok := fieldByKey(rep, key); ok {
					t.Errorf("%s should not be a row", key)
				}
			}
			for key, want := range tc.setHere {
				got, ok := rep.Layers[0].Values[key]
				if !ok {
					t.Errorf("%s is not set on the global layer", key)
					continue
				}
				if list, isList := want.([]any); isList {
					gotList, _ := asStrings(got)
					wantList, _ := asStrings(list)
					if strings.Join(gotList, ",") != strings.Join(wantList, ",") {
						t.Errorf("%s = %v, want %v", key, gotList, wantList)
					}
					continue
				}
				if got != want {
					t.Errorf("%s = %v, want %v", key, got, want)
				}
			}
			for key, want := range tc.labels {
				f, _ := fieldByKey(rep, key)
				if f.Tag != want {
					t.Errorf("%s tag = %q, want %q", key, f.Tag, want)
				}
			}
			for _, key := range tc.unreadable {
				if _, ok := rep.Layers[0].Values[key]; ok {
					t.Errorf("%s is unreadable and must not be offered as a value", key)
				}
				found := false
				for _, u := range rep.Layers[0].Unreadable {
					if u == key {
						found = true
					}
				}
				if !found {
					t.Errorf("%s is missing from the layer's unreadable list", key)
				}
			}
		})
	}
}

// The vendor's own order, kept: built-ins first, then the roles the file adds.
// A report whose rows moved between two reads would scroll under the cursor.
func TestRoleOrderIsTheVendorsThenTheFiles(t *testing.T) {
	home := ompHome(t, "modelRoles:\n  zulu: openai/gpt-5\n  alpha: openai/gpt-5\n")
	rep, err := Read("omp", Paths{Home: home})
	if err != nil {
		t.Fatal(err)
	}
	var roles []string
	for _, f := range rep.Fields {
		if f.Kind == KindRole {
			roles = append(roles, strings.TrimPrefix(f.Key, "modelRoles."))
		}
	}
	if len(roles) != len(ompRoleCatalog)+2 {
		t.Fatalf("got %d role rows, want %d", len(roles), len(ompRoleCatalog)+2)
	}
	for i, def := range ompRoleCatalog {
		if roles[i] != def.ID {
			t.Fatalf("row %d is %q, the vendor's own order has %q", i, roles[i], def.ID)
		}
	}
	if roles[len(roles)-2] != "alpha" || roles[len(roles)-1] != "zulu" {
		t.Errorf("the file's own roles come last, sorted: %v", roles[len(roles)-2:])
	}
}

func TestRoleWrites(t *testing.T) {
	for _, tc := range []struct {
		name    string
		body    string
		patch   Patch
		want    string
		wantErr string
	}{
		{
			name:  "a role lands inside the map the CLI already has, and the comment stays",
			body:  "# mine\nmodelRoles:\n  default: openai/gpt-5   # the one I use\nsymbolPreset: ascii\n",
			patch: Patch{Scope: "user", Set: map[string]any{"modelRoles.default": "openai/gpt-5-nano"}},
			want:  "# mine\nmodelRoles:\n  default: openai/gpt-5-nano   # the one I use\nsymbolPreset: ascii\n",
		},
		{
			name:  "a role the file does not have creates the map",
			body:  "symbolPreset: ascii\n",
			patch: Patch{Scope: "user", Set: map[string]any{"modelRoles.plan": "anthropic/claude-opus-4-8:high"}},
			want:  "symbolPreset: ascii\nmodelRoles:\n  plan: anthropic/claude-opus-4-8:high\n",
		},
		{
			name:  "handing a role back takes the empty map with it",
			body:  "modelRoles:\n  default: openai/gpt-5\nsymbolPreset: ascii\n",
			patch: Patch{Scope: "user", Reset: []string{"modelRoles.default"}},
			want:  "symbolPreset: ascii\n",
		},
		{
			name:  "a chain is a list, written where the CLI reads it",
			body:  "symbolPreset: ascii\n",
			patch: Patch{Scope: "user", Set: map[string]any{"retry.fallbackChains.default": []any{"@smol", "openai/gpt-5"}}},
			want:  "symbolPreset: ascii\nretry:\n  fallbackChains:\n    default: [\"@smol\", \"openai/gpt-5\"]\n",
		},
		{
			name:  "an existing one-line chain is spliced in place",
			body:  "retry:\n  fallbackChains:\n    default: [\"@smol\"]   # keep me\n",
			patch: Patch{Scope: "user", Set: map[string]any{"retry.fallbackChains.default": []any{"openai/gpt-5"}}},
			want:  "retry:\n  fallbackChains:\n    default: [\"openai/gpt-5\"]   # keep me\n",
		},
		{
			name:  "an empty chain is how the CLI says never fall back",
			body:  "retry:\n  fallbackChains:\n    default: [\"@smol\"]\n",
			patch: Patch{Scope: "user", Set: map[string]any{"retry.fallbackChains.default": []any{}}},
			want:  "retry:\n  fallbackChains:\n    default: []\n",
		},
		{
			name:  "a chain keyed by a model id with dots is addressed whole",
			body:  "symbolPreset: ascii\n",
			patch: Patch{Scope: "user", Set: map[string]any{"retry.fallbackChains.openai/gpt-4.1-mini": []any{"openai/gpt-5"}}},
			want:  "symbolPreset: ascii\nretry:\n  fallbackChains:\n    openai/gpt-4.1-mini: [\"openai/gpt-5\"]\n",
		},
		{
			name:  "a new role is declared by its tag, the vendor's own way, so nothing is set over an empty string",
			body:  "symbolPreset: ascii\n",
			patch: Patch{Scope: "user", Set: map[string]any{"modelTags.scout.name": "SCOUT"}},
			want:  "symbolPreset: ascii\nmodelTags:\n  scout:\n    name: SCOUT\n",
		},
		{
			name:  "a second tag joins the map the file already has, never a second map",
			body:  "modelTags:\n  review:\n    name: REVIEW\nsymbolPreset: ascii\n",
			patch: Patch{Scope: "user", Set: map[string]any{"modelTags.scout.name": "SCOUT"}},
			want:  "modelTags:\n  review:\n    name: REVIEW\n  scout:\n    name: SCOUT\nsymbolPreset: ascii\n",
		},
		{
			name:  "a chain joins an existing retry block that has no chains yet",
			body:  "retry:\n  maxRetries: 3\nsymbolPreset: ascii\n",
			patch: Patch{Scope: "user", Set: map[string]any{"retry.fallbackChains.default": []any{"@smol"}}},
			want:  "retry:\n  maxRetries: 3\n  fallbackChains:\n    default: [\"@smol\"]\nsymbolPreset: ascii\n",
		},
		{
			name:    "a tag for a name the CLI would refuse is refused too",
			body:    "symbolPreset: ascii\n",
			patch:   Patch{Scope: "user", Set: map[string]any{"modelTags.9x.name": "9X"}},
			wantErr: "not a role name",
		},
		{
			name:  "the quick-switch cycle is a list at the top level",
			body:  "symbolPreset: ascii\n",
			patch: Patch{Scope: "user", Set: map[string]any{"cycleOrder": []any{"smol", "default"}}},
			want:  "symbolPreset: ascii\ncycleOrder: [\"smol\", \"default\"]\n",
		},
		{
			name:    "a role name the CLI would refuse is refused here, by name",
			body:    "symbolPreset: ascii\n",
			patch:   Patch{Scope: "user", Set: map[string]any{"modelRoles.9lives": "openai/gpt-5"}},
			wantErr: "not a role name",
		},
		{
			name:    "a chain key this writer cannot address is refused, by name",
			body:    "symbolPreset: ascii\n",
			patch:   Patch{Scope: "user", Set: map[string]any{"retry.fallbackChains.a: b": []any{"openai/gpt-5"}}},
			wantErr: "not a fallback key",
		},
		{
			name:    "a selector with a line break is refused rather than written as a block",
			body:    "symbolPreset: ascii\n",
			patch:   Patch{Scope: "user", Set: map[string]any{"modelRoles.default": "openai/gpt-5\nrm -rf /"}},
			wantErr: "one line",
		},
		{
			name:    "a chain the file holds as a map is never overwritten",
			body:    "retry:\n  fallbackChains:\n    default:\n      first: openai/gpt-5\n",
			patch:   Patch{Scope: "user", Set: map[string]any{"retry.fallbackChains.default": []any{"openai/gpt-5"}}},
			wantErr: "shape",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home := ompHome(t, tc.body)
			path := filepath.Join(home, ".omp", "agent", "config.yml")
			err := Apply("omp", Paths{Home: home}, tc.patch)
			got, readErr := os.ReadFile(path)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("error = %v, want one naming %q", err, tc.wantErr)
				}
				if string(got) != tc.body {
					t.Errorf("a refused write changed the file:\n%s", got)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tc.want {
				t.Errorf("file is\n%q\nwant\n%q", got, tc.want)
			}
		})
	}
}

// A role written in the workspace layer goes to the workspace file — the layer
// `omp config set` cannot reach at all (measured 2026-09-22: it writes the
// global file wherever it runs, and `--scope` is not an option), which is the
// reason this pane is worth having.
func TestRoleWritesReachTheWorkspaceLayer(t *testing.T) {
	home := ompHome(t, "modelRoles:\n  default: openai/gpt-5\n")
	cwd := t.TempDir()
	if err := Apply("omp", Paths{Home: home, Cwd: cwd}, Patch{
		Scope: "project",
		Set:   map[string]any{"modelRoles.default": "anthropic/claude-sonnet-4-6"},
	}); err != nil {
		t.Fatal(err)
	}
	project, err := os.ReadFile(filepath.Join(cwd, ".omp", "config.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(project), "claude-sonnet-4-6") {
		t.Errorf("the workspace file is %q", project)
	}
	global, err := os.ReadFile(filepath.Join(home, ".omp", "agent", "config.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(global), "openai/gpt-5") {
		t.Errorf("the global file was changed: %q", global)
	}
}

// Only a CLI that declares a matrix has one. A role key sent to any other CLI
// is refused the way any undeclared key is.
func TestRolesAreOnlyForTheCLIThatDeclaresThem(t *testing.T) {
	for _, cli := range Supported() {
		rep, err := Read(cli, Paths{Home: t.TempDir()})
		if err != nil {
			t.Fatal(err)
		}
		if (rep.Roles != nil) != (cli == "omp") {
			t.Errorf("%s: roles report present = %v", cli, rep.Roles != nil)
		}
		if cli == "omp" {
			continue
		}
		err = Apply(cli, Paths{Home: t.TempDir()}, Patch{Scope: "user", Set: map[string]any{"modelRoles.default": "x/y"}})
		if err == nil {
			t.Errorf("%s accepted a model role it does not declare", cli)
		}
	}
}

// The catalog is a claim about someone else's software. It is pinned here so a
// rename in omp lands as a red test instead of a row that writes a key the CLI
// ignores — the same trade `pikeys.Catalog` makes, and the reason 18.1.5's
// removal of `designer` and 18.2.7's move of the five kind roles are visible in
// this file's history.
func TestOmpRoleCatalogIsTheVendorsOwn(t *testing.T) {
	want := []string{
		"default", "smol", "slow", "vision", "plan", "commit", "tiny", "memory", "task", "advisor",
		"image", "web", "speech", "dictation", "judge",
	}
	if len(ompRoleCatalog) != len(want) {
		t.Fatalf("the catalog has %d roles, the vendor's own list has %d", len(ompRoleCatalog), len(want))
	}
	for i, id := range want {
		if ompRoleCatalog[i].ID != id {
			t.Errorf("role %d is %q, want %q", i, ompRoleCatalog[i].ID, id)
		}
		if ompRoleCatalog[i].Tag == "" || ompRoleCatalog[i].Name == "" {
			t.Errorf("%s has no tag or no name", id)
		}
		if ompRoleCatalog[i].Section != "chat" && ompRoleCatalog[i].Section != "kind" {
			t.Errorf("%s is in section %q", id, ompRoleCatalog[i].Section)
		}
	}
}
