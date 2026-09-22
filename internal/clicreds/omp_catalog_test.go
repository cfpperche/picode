package clicreds

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ompRulesFixture is the shape of Omp's rules.json this reader depends on,
// cut to one provider per case parseOMPRules distinguishes.
const ompRulesFixture = `{
  "version": 1,
  "auth": {"providers": [
    {"id": "anthropic", "name": "Anthropic (Claude Pro/Max)", "login": {"kind": "oauth-code"}},
    {"id": "meta", "name": "Meta Model API", "login": {"kind": "api-key"}},
    {"id": "cerebras", "name": "Cerebras", "login": {"kind": "api-key"}},
    {"id": "devin", "name": "Devin", "login": {"kind": "oauth-code"}},
    {"id": "zai-coding-plan", "name": "Z.AI (GLM Coding Plan · Sign in)", "login": {"kind": "oauth-code"}},
    {"id": "tavily", "name": "Tavily", "login": {"kind": "api-key"}, "env": {"vars": ["TAVILY_API_KEY"]}},
    {"id": "groq", "name": "Groq"},
    {"id": "local", "name": "Local models"},
    {"id": "cerebras", "name": "Cerebras again", "login": {"kind": "api-key"}}
  ]},
  "providers": {
    "cerebras": {"id": "cerebras", "envVars": ["CEREBRAS_API_KEY"]},
    "devin": {"id": "devin", "envVars": ["DEVIN_API_KEY"]},
    "groq": {"id": "groq", "envVars": ["GROQ_API_KEY"]}
  }
}`

func TestParseOMPRules(t *testing.T) {
	declared := []Provider{{Provider: "anthropic"}, {Provider: "meta-ai"}}
	got := parseOMPRules([]byte(ompRulesFixture), declared)
	type want struct {
		name  string
		kinds []string
		env   string
		note  string
	}
	wants := map[string]want{
		"cerebras":        {name: "Cerebras", kinds: []string{KindAPIKey}, env: "CEREBRAS_API_KEY"},
		"devin":           {name: "Devin", kinds: []string{KindAPIKey, KindOAuth}, env: "DEVIN_API_KEY", note: noteOmpStore},
		"zai-coding-plan": {name: "Z.AI (GLM Coding Plan · Sign in)", kinds: []string{KindOAuth}, note: noteOmpLogin},
		"tavily":          {name: "Tavily", kinds: []string{KindAPIKey}, env: "TAVILY_API_KEY"},
		"groq":            {name: "Groq", kinds: []string{KindAPIKey}, env: "GROQ_API_KEY"},
	}
	order := []string{"cerebras", "devin", "zai-coding-plan", "tavily", "groq"}
	if len(got) != len(order) {
		t.Fatalf("got %d providers %+v, want %v", len(got), got, order)
	}
	for i, p := range got {
		// Omp's /login order is kept; declared ids (anthropic), their Omp
		// aliases (meta), providers nothing can reach (local) and repeats are
		// skipped.
		if p.Provider != order[i] {
			t.Fatalf("got[%d] = %s, want %s", i, p.Provider, order[i])
		}
		w := wants[p.Provider]
		if p.Name != w.name || strings.Join(p.Kinds, ",") != strings.Join(w.kinds, ",") ||
			p.Env[KindAPIKey] != w.env || p.Note != w.note || p.Native != nil {
			t.Fatalf("%s = %+v, want %+v", p.Provider, p, w)
		}
	}
	if parseOMPRules([]byte("not json"), declared) != nil {
		t.Fatal("a file that does not parse must add nothing")
	}
}

// TestForOmpAppendsItsCatalog: the declaration comes first, untouched, and
// the installed package's other providers follow; without the file the
// declaration stands alone.
func TestForOmpAppendsItsCatalog(t *testing.T) {
	t.Cleanup(resetOMPCatalog)
	declared := len(catalogSpec(t, "omp").Providers)

	t.Setenv(ompRulesEnv, filepath.Join(t.TempDir(), "missing.json"))
	resetOMPCatalog()
	if spec, _ := For("omp"); len(spec.Providers) != declared {
		t.Fatalf("no rules.json: %d providers, want the %d declared", len(spec.Providers), declared)
	}

	path := filepath.Join(t.TempDir(), "rules.json")
	if err := os.WriteFile(path, []byte(ompRulesFixture), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(ompRulesEnv, path)
	resetOMPCatalog()
	spec, _ := For("omp")
	if len(spec.Providers) != declared+5 {
		t.Fatalf("with rules.json: %d providers, want %d", len(spec.Providers), declared+5)
	}
	if spec.Providers[0].Provider != "anthropic" || spec.Providers[declared].Provider != "cerebras" {
		t.Fatalf("order = %s … %s, want the declaration first", spec.Providers[0].Provider, spec.Providers[declared].Provider)
	}
	if name, ok := EnvVar("omp", "cerebras", KindAPIKey); !ok || name != "CEREBRAS_API_KEY" {
		t.Fatalf("EnvVar(omp, cerebras) = %q, %v", name, ok)
	}
	// Only omp reads Omp's file.
	if _, ok := CanUse("opencode", "cerebras"); ok {
		t.Fatal("opencode must not gain Omp's providers")
	}
	found := false
	for _, s := range Declarations() {
		if s.CLI == "omp" {
			found = len(s.Providers) == declared+5
		}
	}
	if !found {
		t.Fatal("Declarations() must carry Omp's catalog too")
	}
}

func catalogSpec(t *testing.T, cli string) Spec {
	t.Helper()
	for _, s := range catalog {
		if s.CLI == cli {
			return s
		}
	}
	t.Fatalf("%s is not declared", cli)
	return Spec{}
}

// TestLocateOMPRules walks an npm layout: the bin symlink resolves to
// <pkg>/dist/cli.js, whose package.json names Omp, and rules.json sits in the
// package's own node_modules.
func TestLocateOMPRules(t *testing.T) {
	t.Setenv(ompRulesEnv, "")
	root := t.TempDir()
	pkg := filepath.Join(root, "lib", "node_modules", "@oh-my-pi", "pi-coding-agent")
	rules := filepath.Join(pkg, "node_modules", "@oh-my-pi", "pi-catalog", "src", "compat", "rules.json")
	for _, dir := range []string{filepath.Join(pkg, "dist"), filepath.Dir(rules), filepath.Join(root, "bin")} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	must := func(path, body string, mode os.FileMode) {
		if err := os.WriteFile(path, []byte(body), mode); err != nil {
			t.Fatal(err)
		}
	}
	must(filepath.Join(pkg, "package.json"), `{"name":"@oh-my-pi/pi-coding-agent"}`, 0o644)
	must(filepath.Join(pkg, "dist", "cli.js"), "#!/usr/bin/env node\n", 0o755)
	must(rules, ompRulesFixture, 0o644)
	if err := os.Symlink(filepath.Join(pkg, "dist", "cli.js"), filepath.Join(root, "bin", "omp")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", filepath.Join(root, "bin"))
	if got := locateOMPRules(); got != rules {
		t.Fatalf("locateOMPRules = %q, want %q", got, rules)
	}
	t.Setenv("PATH", t.TempDir())
	if got := locateOMPRules(); got != "" {
		t.Fatalf("no omp on PATH: %q, want empty", got)
	}
}
