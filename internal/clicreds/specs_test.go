package clicreds

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// clis is the nine CLIs PiCode ships, in the order the surface lists them.
var clis = []string{"pi", "claude-code", "codex", "grok", "hermes", "opencode", "muse", "agy", "omp"}

var envName = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)

// TestDeclarationShape holds every row to the contract clicreds.go describes:
// a provider has at least one kind, an env var name exists only for a declared
// kind, a Native names a known parser, and a per-account directory is declared
// only where the CLI has the variable that moves its whole home.
func TestDeclarationShape(t *testing.T) {
	specs := Declarations()
	if len(specs) != len(clis) {
		t.Fatalf("%d declarations, want %d", len(specs), len(clis))
	}
	seen := map[string]bool{}
	for _, spec := range specs {
		if spec.CLI == "" || spec.Name == "" {
			t.Fatalf("declaration without a CLI or name: %+v", spec)
		}
		if seen[spec.CLI] {
			t.Fatalf("%s is declared twice", spec.CLI)
		}
		seen[spec.CLI] = true
		if len(spec.Providers) == 0 {
			t.Fatalf("%s declares no provider", spec.CLI)
		}
		providers := map[string]bool{}
		for _, p := range spec.Providers {
			if p.Provider == "" {
				t.Fatalf("%s has a provider row without an id", spec.CLI)
			}
			if providers[p.Provider] {
				t.Fatalf("%s declares %s twice", spec.CLI, p.Provider)
			}
			providers[p.Provider] = true
			if len(p.Kinds) == 0 {
				t.Fatalf("%s/%s declares no kind", spec.CLI, p.Provider)
			}
			kinds := map[string]bool{}
			for _, kind := range p.Kinds {
				if kind != KindAPIKey && kind != KindOAuth {
					t.Fatalf("%s/%s has unknown kind %q", spec.CLI, p.Provider, kind)
				}
				if kinds[kind] {
					t.Fatalf("%s/%s repeats kind %q", spec.CLI, p.Provider, kind)
				}
				kinds[kind] = true
			}
			for kind, name := range p.Env {
				if !kinds[kind] {
					t.Fatalf("%s/%s passes %s in %s but does not declare that kind", spec.CLI, p.Provider, kind, name)
				}
				if !envName.MatchString(name) {
					t.Fatalf("%s/%s names %q, which is not an environment variable", spec.CLI, p.Provider, name)
				}
			}
			if p.Native == nil {
				continue
			}
			n := p.Native
			if n.Path == "" {
				t.Fatalf("%s/%s declares a Native with no path", spec.CLI, p.Provider)
			}
			if !contains(formats, n.Format) {
				t.Fatalf("%s/%s names unknown format %q", spec.CLI, p.Provider, n.Format)
			}
			if (n.DirEnv == "") != (n.VendorDir == "") {
				t.Fatalf("%s/%s declares DirEnv %q with VendorDir %q — a home variable needs its directory and vice versa",
					spec.CLI, p.Provider, n.DirEnv, n.VendorDir)
			}
			if n.DirEnv != "" && n.CredFile == "" {
				t.Fatalf("%s/%s can be relocated but names no credential file", spec.CLI, p.Provider)
			}
		}
	}
	for _, cli := range clis {
		if !seen[cli] {
			t.Fatalf("%s has no declaration", cli)
		}
	}
}

// TestProvidersForOrder pins declaration order — the pane renders it and the
// Detect walk depends on it.
func TestProvidersForOrder(t *testing.T) {
	cases := map[string][]string{
		"pi":          {"anthropic", "openai-codex", "xai", "google", "meta-ai", "openrouter", "github-copilot", "opencode", "zai", "kimi-coding", "llama.cpp"},
		"omp":         {"anthropic", "openai-codex", "xai", "google", "meta-ai", "openrouter", "github-copilot", "opencode", "zai", "kimi-coding", "llama.cpp"},
		"hermes":      {"openai-codex", "anthropic", "xai", "google", "openrouter", "github-copilot", "opencode", "zai", "kimi-coding"},
		"opencode":    {"anthropic", "openai-codex", "xai", "google", "openrouter", "github-copilot", "opencode", "zai", "kimi-coding", "meta-ai"},
		"claude-code": {"anthropic"},
		"codex":       {"openai-codex"},
		"grok":        {"xai"},
		"muse":        {"meta-ai"},
		"agy":         {"google"},
	}
	for cli, want := range cases {
		if got := ProvidersFor(cli); !equal(got, want) {
			t.Fatalf("ProvidersFor(%q) = %v, want %v", cli, got, want)
		}
	}
	if got := ProvidersFor("no-such-cli"); got != nil {
		t.Fatalf("ProvidersFor of an unknown CLI = %v, want nil", got)
	}
}

// TestCanUseAndEnvVar checks the projection the pane and step 2 read.
func TestCanUseAndEnvVar(t *testing.T) {
	if kinds, ok := CanUse("claude-code", "anthropic"); !ok || !equal(kinds, []string{KindAPIKey, KindOAuth}) {
		t.Fatalf("claude-code/anthropic = %v %v", kinds, ok)
	}
	if _, ok := CanUse("claude-code", "openai-codex"); ok {
		t.Fatal("claude-code claims a provider it cannot reach")
	}
	if _, ok := CanUse("grok", "xai-oauth"); ok {
		t.Fatal("a CLI's own id must not answer for the vault's")
	}
	if name, ok := EnvVar("claude-code", "anthropic", KindOAuth); !ok || name != "CLAUDE_CODE_OAUTH_TOKEN" {
		t.Fatalf("claude-code anthropic oauth env = %q %v", name, ok)
	}
	if name, ok := EnvVar("codex", "openai-codex", KindAPIKey); !ok || name != "OPENAI_API_KEY" {
		t.Fatalf("codex api key env = %q %v", name, ok)
	}
	if _, ok := EnvVar("pi", "openai-codex", KindOAuth); ok {
		t.Fatal("pi's ChatGPT login has no environment variable and must not claim one")
	}
	if name, ok := EnvVar("pi", "llama.cpp", KindAPIKey); !ok || name != "LLAMA_API_KEY" {
		t.Fatalf("pi llama.cpp env = %q %v", name, ok)
	}
	if name, ok := EnvVar("omp", "llama.cpp", KindAPIKey); !ok || name != "LLAMA_CPP_API_KEY" {
		t.Fatalf("omp llama.cpp env = %q %v", name, ok)
	}
}

// TestNativeDeclarations pins the paths, the home variables and what a
// per-account directory seeds — the facts step 2 builds on.
func TestNativeDeclarations(t *testing.T) {
	cases := []struct {
		cli, provider string
		want          Native
	}{
		{"claude-code", "anthropic", Native{
			Path: "$HOME/.claude/.credentials.json", Format: "claude",
			DirEnv: "CLAUDE_CONFIG_DIR", VendorDir: ".claude", CredFile: ".credentials.json",
			Seed: []string{".claude.json", "settings.json", "mcp.json", "plugins", "skills", "projects"},
		}},
		{"codex", "openai-codex", Native{
			Path: "$HOME/.codex/auth.json", Format: "codex",
			DirEnv: "CODEX_HOME", VendorDir: ".codex", CredFile: "auth.json",
			Seed: []string{"config.toml", "skills", "memories"},
		}},
		{"grok", "xai", Native{
			Path: "$HOME/.grok/auth.json", Format: "grok",
			DirEnv: "GROK_HOME", VendorDir: ".grok", CredFile: "auth.json",
		}},
		{"hermes", "openai-codex", Native{
			Path: "$HOME/.hermes/auth.json", Format: "hermes",
			DirEnv: "HERMES_HOME", VendorDir: ".hermes", CredFile: "auth.json",
		}},
		{"pi", "anthropic", Native{
			Path: "$HOME/.pi/agent/auth.json", Format: "pi",
			DirEnv: "PI_CODING_AGENT_DIR", VendorDir: ".pi/agent", CredFile: "auth.json",
		}},
		{"opencode", "anthropic", Native{
			Path: "$XDG_DATA_HOME/opencode/auth.json", Format: "opencode", CredFile: "auth.json",
		}},
		{"muse", "meta-ai", Native{
			Path: "$HOME/.config/muse/auth.json", Format: "muse", CredFile: "auth.json",
		}},
		{"agy", "google", Native{
			Path: "$HOME/.gemini/antigravity-cli/antigravity-oauth-token", Format: "agy",
			CredFile: "antigravity-oauth-token",
		}},
	}
	for _, tc := range cases {
		spec, ok := For(tc.cli)
		if !ok {
			t.Fatalf("%s is not declared", tc.cli)
		}
		var found *Native
		for _, p := range spec.Providers {
			if p.Provider == tc.provider {
				found = p.Native
			}
		}
		if found == nil {
			t.Fatalf("%s/%s declares no native store", tc.cli, tc.provider)
		}
		if !sameNative(*found, tc.want) {
			t.Fatalf("%s/%s native = %+v, want %+v", tc.cli, tc.provider, *found, tc.want)
		}
	}
}

// TestOmpHasNoNative is the deliberate gap: Omp's credentials live in a live
// SQLite database PiCode does not read, so no row may point at a file.
func TestOmpHasNoNative(t *testing.T) {
	spec, ok := For("omp")
	if !ok {
		t.Fatal("omp is not declared")
	}
	for _, p := range spec.Providers {
		if p.Native != nil {
			t.Fatalf("omp/%s declares a native store %+v", p.Provider, p.Native)
		}
	}
	// Even with Omp's own directory and a file that looks like a login in
	// place, Detect answers nothing: there is no reader to reach it.
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", "")
	if err := os.MkdirAll(filepath.Join(home, ".omp", "agent"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".omp", "agent", "auth.json"),
		[]byte(`{"anthropic":{"type":"api_key","key":"omp-key"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if login, ok := Detect("omp"); ok {
		t.Fatalf("Detect(omp) = %+v, want no login", login)
	}
	if login, ok := Detect("no-such-cli"); ok {
		t.Fatalf("Detect of an unknown CLI = %+v, want no login", login)
	}
}

// TestDetectReadsTheDeclaredFile walks each CLI whose store PiCode can read
// through Detect itself — the declaration's path, expandPath and the reader
// together, which is the only way the wiring can be wrong.
func TestDetectReadsTheDeclaredFile(t *testing.T) {
	cases := []struct {
		cli, provider, kind, where, raw string
	}{
		{"pi", "llama.cpp", KindAPIKey, ".pi/agent/auth.json",
			`{"llama.cpp":{"type":"api_key","key":"local-key"}}`},
		{"claude-code", "anthropic", KindOAuth, ".claude/.credentials.json",
			`{"claudeAiOauth":{"accessToken":"a","refreshToken":"r","expiresAt":1799100000000}}`},
		{"codex", "openai-codex", KindOAuth, ".codex/auth.json",
			`{"auth_mode":"chatgpt","tokens":{"access_token":"a","refresh_token":"r","account_id":"acct"}}`},
		{"grok", "xai", KindOAuth, ".grok/auth.json",
			`{"https://auth.x.ai::c1":{"key":"a","refresh_token":"r","expires_at":"2027-01-02T03:04:05Z","email":"who@example.com"}}`},
		{"hermes", "xai", KindOAuth, ".hermes/auth.json",
			`{"version":2,"providers":{"xai-oauth":{"tokens":{"access_token":"a","refresh_token":"r"},"auth_mode":"oidc"}}}`},
		{"opencode", "zai", KindAPIKey, ".local/share/opencode/auth.json",
			`{"zai-coding-plan":{"type":"api","key":"zhipu-key"}}`},
		{"muse", "meta-ai", KindAPIKey, ".config/muse/auth.json",
			`{"schema_version":1,"providers":{"meta":{"api_key":"meta-key"}}}`},
		{"agy", "google", KindOAuth, ".gemini/antigravity-cli/antigravity-oauth-token",
			`{"token":{"access_token":"a","refresh_token":"r","expiry":"2027-01-02T03:04:05Z"},"auth_method":"consumer"}`},
	}
	for _, tc := range cases {
		t.Run(tc.cli, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("XDG_DATA_HOME", "")
			path := filepath.Join(home, filepath.FromSlash(tc.where))
			if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(tc.raw), 0o600); err != nil {
				t.Fatal(err)
			}
			login, ok := Detect(tc.cli)
			if !ok {
				t.Fatalf("Detect(%s) found nothing in %s", tc.cli, path)
			}
			if login.Provider != tc.provider || login.Kind != tc.kind {
				t.Fatalf("Detect(%s) = %s/%s, want %s/%s", tc.cli, login.Provider, login.Kind, tc.provider, tc.kind)
			}
			var shape map[string]any
			if err := json.Unmarshal(login.Cred, &shape); err != nil {
				t.Fatalf("Detect(%s) cred is not JSON: %v", tc.cli, err)
			}
			if shape["type"] != tc.kind {
				t.Fatalf("Detect(%s) cred type = %v, want %s", tc.cli, shape["type"], tc.kind)
			}
		})
	}
}

// TestNotesWhereARowCannotBeInjected: a row whose login PiCode cannot pass to
// the CLI says so, so no pane can render a control that does nothing.
func TestNotesWhereARowCannotBeInjected(t *testing.T) {
	for _, tc := range []struct{ cli, provider string }{
		{"muse", "meta-ai"},
		{"agy", "google"},
		{"omp", "openai-codex"},
		{"hermes", "openai-codex"},
		{"opencode", "anthropic"},
		{"grok", "xai"},
	} {
		spec, ok := For(tc.cli)
		if !ok {
			t.Fatalf("%s is not declared", tc.cli)
		}
		for _, p := range spec.Providers {
			if p.Provider == tc.provider && p.Note == "" {
				t.Fatalf("%s/%s has no Note", tc.cli, tc.provider)
			}
		}
	}
	// Every omp row that offers a subscription carries the SQLite note: there
	// is no Path behind any of them.
	spec, _ := For("omp")
	for _, p := range spec.Providers {
		if contains(p.Kinds, KindOAuth) && !strings.Contains(p.Note, "SQLite") {
			t.Fatalf("omp/%s offers oauth with no note: %q", p.Provider, p.Note)
		}
	}
}

// TestDeclarationsIsTheTable keeps the accessor and the table one object.
func TestDeclarationsIsTheTable(t *testing.T) {
	specs := Declarations()
	if len(specs) == 0 {
		t.Fatal("no declarations")
	}
	ids := make([]string, 0, len(specs))
	for _, spec := range specs {
		ids = append(ids, spec.CLI)
	}
	want := append([]string(nil), clis...)
	sort.Strings(want)
	got := append([]string(nil), ids...)
	sort.Strings(got)
	if !equal(got, want) {
		t.Fatalf("declared CLIs = %v, want %v", got, want)
	}
	if spec, ok := For("grok"); !ok || spec.Name != "Grok" {
		t.Fatalf("For(grok) = %+v %v", spec, ok)
	}
	if _, ok := For("no-such-cli"); ok {
		t.Fatal("For accepted an unknown CLI")
	}
}

func contains(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func sameNative(a, b Native) bool {
	return a.Path == b.Path && a.Format == b.Format && a.DirEnv == b.DirEnv &&
		a.VendorDir == b.VendorDir && a.CredFile == b.CredFile && equal(a.Seed, b.Seed)
}
