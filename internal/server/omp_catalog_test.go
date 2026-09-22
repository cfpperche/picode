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
	if plan := ompRosterRow(t, ts, "zai-coding-plan"); plan == nil || plan["note"] == "" {
		t.Fatalf("a sign-in-only provider must say where its login happens: %+v", plan)
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
