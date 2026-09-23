package server

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cfpperche/picode/internal/credentials"
)

// Omp's roster carries the providers of its installed package (clicreds'
// omp_catalog.go): the pane lists them with Omp's own name, a key for one is
// accepted into the vault, and a launch passes it in the variable Omp reads.
func TestOMPRosterCarriesItsCatalog(t *testing.T) {
	rules := filepath.Join(t.TempDir(), "rules.json")
	if err := os.WriteFile(rules, []byte(`{
	  "auth": {"providers": [
	    {"id": "cerebras", "name": "Cerebras", "login": {"kind": "api-key"}},
	    {"id": "zai-coding-plan", "name": "Z.AI (GLM Coding Plan · Sign in)", "login": {"kind": "oauth-code"}}
	  ]},
	  "providers": {"cerebras": {"envVars": ["CEREBRAS_API_KEY"]}}
	}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PICODE_OMP_RULES", rules)
	ts, _, _, _ := credentialsServer(t)

	row := ompRosterRow(t, ts, "cerebras")
	if row == nil || row["name"] != "Cerebras" {
		t.Fatalf("cerebras row = %+v, want Omp's name", row)
	}
	if env, _ := row["env"].(map[string]any); env["api_key"] != "CEREBRAS_API_KEY" {
		t.Fatalf("cerebras env = %+v", row["env"])
	}
	if plan := ompRosterRow(t, ts, "zai-coding-plan"); plan == nil || plan["note"] == "" || plan["signin"] != "terminal" {
		t.Fatalf("a sign-in-only provider the OAuth engine does not know signs in through omp's own /login: %+v", plan)
	}
	// Another CLI's roster is untouched.
	roster := cliRequest(t, ts, "GET", "/api/credentials?cli=opencode", nil, 200)
	for _, p := range roster["providers"].([]any) {
		if p.(map[string]any)["id"] == "cerebras" {
			t.Fatal("opencode's roster gained Omp's catalog")
		}
	}

	cliRequest(t, ts, "POST", "/api/credentials", map[string]any{"provider": "cerebras", "key": "csk-test-1234567890"}, 201)
	file, err := credentials.Default().Load()
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, kv := range ompCredentialEnv(&file, map[string]string{}) {
		got[kv[0]] = kv[1]
	}
	if got["CEREBRAS_API_KEY"] != "csk-test-1234567890" {
		t.Fatalf("launch env = %+v, want CEREBRAS_API_KEY", got)
	}
}

// Omp opens pi's Add flow (add.kind "provider"), and each provider says how
// its account sign-in runs: PiCode's browser flow where the OAuth engine
// knows it, the CLI's own /login in a terminal otherwise, nothing where the
// provider takes only a key. Other guests open a sign-in of their own.
func TestOMPRosterSaysHowEachSigninRuns(t *testing.T) {
	ts, _, _, _ := credentialsServer(t)
	roster := cliRequest(t, ts, "GET", "/api/credentials?cli=omp", nil, 200)
	if add, _ := roster["add"].(map[string]any); add["kind"] != "provider" {
		t.Fatalf("omp add = %+v, want the provider dialog", roster["add"])
	}
	cases := map[string]string{
		"anthropic":   "browser", // the OAuth engine signs it in
		"kimi-coding": "browser",
		"google":      "", // key only
		"openrouter":  "", // declared key-only for omp
	}
	for id, want := range cases {
		row := ompRosterRow(t, ts, id)
		if row == nil {
			t.Fatalf("%s missing from the omp roster", id)
		}
		got, _ := row["signin"].(string)
		if got != want {
			t.Fatalf("%s signin = %q, want %q", id, got, want)
		}
	}
	// Every guest CLI now opens a sign-in of its own; OpenCode's is its own
	// server's flow (ADR-0201), not omp's picker.
	other := cliRequest(t, ts, "GET", "/api/credentials?cli=opencode", nil, 200)
	if add, _ := other["add"].(map[string]any); add["kind"] != "opencode" {
		t.Fatalf("opencode add = %+v, want OpenCode's own dialog", other["add"])
	}
}
