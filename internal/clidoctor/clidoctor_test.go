package clidoctor

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The listing is shaped like omp 18.2.8's own `omp config list --json`
// (2026-09-22): a map of dotted key to {value, type, description}, with
// `redacted` on a credential.
const listing = `{
 "modelRoles": {"value": {"default": "openai/gpt-5"}, "type": "record", "description": ""},
 "composer.shape": {"value": "minimal", "type": "string", "description": "Visual layout"},
 "tools.approvalMode": {"value": "yolo", "type": "enum", "description": "Default approval behavior"},
 "disabledProviders": {"value": ["groq"], "type": "array", "description": ""},
 "retry.fallbackChains": {"value": {}, "type": "record", "description": ""},
 "images.urls.credentials": {"value": "abc", "type": "string", "description": "", "redacted": true},
 "providers.openai.apiKey": {"value": "sk-live", "type": "string", "description": ""}
}`

type fixture struct {
	home, dir string
	tracked   map[string]bool
}

func (f fixture) paths() Paths {
	return Paths{
		Home:    f.home,
		Dir:     f.dir,
		List:    func(context.Context, string) ([]byte, error) { return []byte(listing), nil },
		Tracked: func(dir, file string) bool { return f.tracked[file] },
	}
}

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func setup(t *testing.T, global, project string) fixture {
	t.Helper()
	t.Setenv("PI_CODING_AGENT_DIR", "")
	f := fixture{home: t.TempDir(), dir: t.TempDir(), tracked: map[string]bool{}}
	if global != "" {
		write(t, filepath.Join(f.home, ".omp", "agent", "config.yml"), global)
	}
	if project != "" {
		write(t, filepath.Join(f.dir, ".omp", "config.yml"), project)
	}
	return f
}

func finding(rep Report, id string) (Finding, bool) {
	for _, f := range rep.Findings {
		if f.ID == id {
			return f, true
		}
	}
	return Finding{}, false
}

func entry(rep Report, key string) Entry {
	for _, e := range rep.Entries {
		if e.Key == key {
			return e
		}
	}
	return Entry{}
}

// The decision table: each row is a file state and the finding it must (or
// must not) produce.
func TestFindings(t *testing.T) {
	for _, tc := range []struct {
		name, global, project string
		setup                 func(t *testing.T, f fixture)
		want, absent          []string
	}{
		{name: "a clean pair of files says nothing but the approval default",
			global: "modelRoles:\n  default: openai/gpt-5\n", want: []string{"approval-unset"},
			absent: []string{"unknown-keys", "written-twice", "flat-keys", "quarantined"}},
		{name: "an explicit approval mode, even yolo, is the user's choice",
			global: "tools:\n  approvalMode: yolo\n", absent: []string{"approval-unset"}},
		{name: "a key omp does not list is unknown — retired keys fall out of the vendor's own list",
			global: "tinyModel: foo/bar\ntotallyUnknownKey: 1\n", want: []string{"unknown-keys"}},
		{name: "a role name under a record setting is not unknown",
			global: "modelRoles:\n  review: openai/gpt-5\nretry:\n  fallbackChains:\n    openai/*: []\n", absent: []string{"unknown-keys"}},
		{name: "a key written flat and nested is written twice",
			global: "composer.shape: band\ncomposer:\n  shape: minimal\n", want: []string{"written-twice"}},
		{name: "a key written only flat is read by omp and missed by PiCode's rows",
			global: "composer.shape: band\n", want: []string{"flat-keys"}, absent: []string{"written-twice", "unknown-keys"}},
		{name: "a quarantined copy is reported with its file",
			global: "modelRoles: {}\n",
			setup: func(t *testing.T, f fixture) {
				write(t, filepath.Join(f.home, ".omp", "agent", "config.yml.broken-1790098763721-1-x"), "oops: [\n")
			}, want: []string{"quarantined"}},
		{name: "a file that does not parse is named before omp moves it",
			global: "oops: [\n", want: []string{"unparseable"}},
		{name: "the older project json is reported",
			setup: func(t *testing.T, f fixture) { write(t, filepath.Join(f.dir, ".omp", "settings.json"), "{}") },
			want:  []string{"legacy-project-json"}},
		{name: "Pi settings with no omp folder are named",
			setup: func(t *testing.T, f fixture) { write(t, filepath.Join(f.dir, ".pi", "settings.json"), "{}") },
			want:  []string{"pi-only"}},
		{name: "a committed .env is a warning",
			setup: func(t *testing.T, f fixture) { f.tracked[".env"] = true }, want: []string{"env-tracked"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := setup(t, tc.global, tc.project)
			if tc.setup != nil {
				tc.setup(t, f)
			}
			rep, err := Read(context.Background(), "omp", f.paths())
			if err != nil {
				t.Fatal(err)
			}
			for _, id := range tc.want {
				if _, ok := finding(rep, id); !ok {
					t.Errorf("missing finding %s; got %+v", id, rep.Findings)
				}
			}
			for _, id := range tc.absent {
				if got, ok := finding(rep, id); ok {
					t.Errorf("unexpected finding %s: %s", id, got.Text)
				}
			}
		})
	}
}

func TestUnknownKeysNameTheKeys(t *testing.T) {
	f := setup(t, "tinyModel: foo/bar\nnope:\n  deep:\n    er: 1\n", "")
	rep, err := Read(context.Background(), "omp", f.paths())
	if err != nil {
		t.Fatal(err)
	}
	got, _ := finding(rep, "unknown-keys")
	if strings.Join(got.Keys, ",") != "nope,tinyModel" {
		t.Errorf("keys = %v, want the top of each unknown subtree", got.Keys)
	}
}

// Where a value comes from is read from the two files omp layers; the project
// file wins where both set a key.
func TestSourceIsTheLayerThatSetsIt(t *testing.T) {
	f := setup(t, "disabledProviders: [ollama]\ncomposer:\n  shape: minimal\n", "disabledProviders: [groq]\n")
	rep, err := Read(context.Background(), "omp", f.paths())
	if err != nil {
		t.Fatal(err)
	}
	if s := entry(rep, "disabledProviders").Source; s != "project" {
		t.Errorf("disabledProviders source = %q", s)
	}
	if s := entry(rep, "composer.shape").Source; s != "user" {
		t.Errorf("composer.shape source = %q", s)
	}
	if s := entry(rep, "tools.approvalMode").Source; s != "" {
		t.Errorf("an unset key has no file source, got %q", s)
	}
}

// A credential never reaches the browser: what omp marks redacted stays
// masked, and so does any key whose name says it is a secret.
func TestCredentialsAreMasked(t *testing.T) {
	f := setup(t, "", "")
	rep, err := Read(context.Background(), "omp", f.paths())
	if err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"images.urls.credentials", "providers.openai.apiKey"} {
		e := entry(rep, k)
		if !e.Redacted || e.Value == "abc" || e.Value == "sk-live" {
			t.Errorf("%s leaked: %+v", k, e)
		}
	}
}

func TestOnlyDeclaredCLIsHaveChecks(t *testing.T) {
	if _, err := Read(context.Background(), "codex", Paths{}); err == nil {
		t.Error("a CLI with no checks must be refused")
	}
}

// Measured against omp 18.2.8's own list: these are credentials, and these are not.
func TestSecretKeysAreTheCredentialsOnly(t *testing.T) {
	for _, k := range []string{"auth.broker.token", "hindsight.apiToken", "mnemopi.llmApiKey", "mnemopi.embeddingApiKey", "searxng.basicPassword", "searxng.token", "images.urls.credentials"} {
		if !secretish.MatchString(k) {
			t.Errorf("%s is a credential and must be masked", k)
		}
	}
	for _, k := range []string{"compaction.maxTokens", "compaction.reserveTokens", "composer.tokenRate", "display.showTokenUsage", "share.redactSecrets", "memories.fallbackTokenLimit"} {
		if secretish.MatchString(k) {
			t.Errorf("%s is not a credential and must stay readable", k)
		}
	}
}

func TestJSListMatchesTheChecks(t *testing.T) {
	body, err := os.ReadFile("../../web/shared/domain/cliDoctor.js")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), `DOCTOR_CLIS = ["omp"]`) || strings.Join(Supported(), ",") != "omp" {
		t.Fatalf("the UI list and Supported() disagree: %v", Supported())
	}
}
