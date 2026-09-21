package catalog

// omp's models.yml through the GUI's verbs (the owner's amendment to
// ADR-0169): upsert creates and merges node-level, the key lives inside the
// definition and is never read back into any serialized shape, remove owns
// only what it created, and a file omp would reject is never written into.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func ompHome(t *testing.T) string {
	t.Helper()
	old := ompTestDir
	ompTestDir = t.TempDir()
	t.Cleanup(func() { ompTestDir = old })
	return ompTestDir
}

func ompDef() CustomDefinition {
	yes := true
	return CustomDefinition{
		BaseURL: "https://api.cheaperinference.com/v1",
		API:     APIOpenAICompletions,
		Compat:  map[string]bool{"supportsDeveloperRole": true, "supportsReasoningEffort": false, "supportsUsageInStreaming": true},
		Models: []CustomModel{
			{ID: "m1", Reasoning: &yes, ContextWindow: intPtr(128000), MaxTokens: intPtr(8192), Name: "Model One", Input: []string{"text", "image"}},
		},
	}
}

func TestOMPUpsertCreatesFileWithKeyInside(t *testing.T) {
	home := ompHome(t)
	if err := OMPUpsertCustomProvider("cheaperinference", ompDef(), "ci_live_secret123"); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(home, "models.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "apiKey: ci_live_secret123") {
		t.Fatalf("key is not in the file:\n%s", raw)
	}
	if info, err := os.Stat(filepath.Join(home, "models.yml")); err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("models.yml perms = %v, want 0600", info.Mode())
	}
	defs := OMPLoadCustomDefinitions()
	row, ok := defs["cheaperinference"]
	if !ok {
		t.Fatalf("definition missing: %v", defs)
	}
	if row.BaseURL != "https://api.cheaperinference.com/v1" || row.API != APIOpenAICompletions {
		t.Fatalf("row = %+v", row)
	}
	if !row.Keyed {
		t.Fatal("row does not report the saved key")
	}
	if len(row.Definitions) != 1 || row.Definitions[0].ID != "m1" || row.Definitions[0].ContextWindow == nil || *row.Definitions[0].ContextWindow != 128000 {
		t.Fatalf("definitions = %+v", row.Definitions)
	}
	// The compat map carries only what omp accepts: supportsUsageInStreaming
	// is pi's Responses switch and must not be invented here.
	if row.Compat["supportsUsageInStreaming"] {
		t.Fatalf("pi-only compat flag written: %v", row.Compat)
	}
	if !row.Compat["supportsDeveloperRole"] || row.Compat["supportsReasoningEffort"] {
		t.Fatalf("compat = %v", row.Compat)
	}
	// The roster shape never carries the key; the server-side reader does.
	if _, ok := OMPCustomAPIKey("cheaperinference"); !ok {
		t.Fatal("saved key is not readable server-side")
	}
}

func TestOMPUpsertPreservesUnknownFieldsAndNeighbors(t *testing.T) {
	home := ompHome(t)
	seed := `# hand-tuned gateways
providers:
  cheaperinference:
    baseUrl: https://api.cheaperinference.com/v1
    api: openai-completions
    headers:
      X-Team: platform # per-team routing
    models:
      - id: m1
        contextWindow: 128000
        samplingParams:
          temperature: 0.2
  handrolled:
    baseUrl: https://gw.example.com/v1
    api: openai-completions
    models:
      - id: g1
`
	if err := os.WriteFile(filepath.Join(home, "models.yml"), []byte(seed), 0o600); err != nil {
		t.Fatal(err)
	}
	def := ompDef()
	if err := OMPUpsertCustomProvider("cheaperinference", def, ""); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(filepath.Join(home, "models.yml"))
	out := string(raw)
	for _, want := range []string{
		"# hand-tuned gateways", // the file's own comment survives
		"X-Team: platform",      // unknown provider field survives
		"temperature: 0.2",      // unknown model field survives
		"# per-team routing",    // value-line comment survives
		"handrolled",            // untouched neighbor survives
		"contextWindow: 128000", // managed number rewritten
		"name: Model One",       // managed name written
		"reasoning: true",       // reasoning flag written
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("lost %q after upsert:\n%s", want, out)
		}
	}
	if strings.Contains(out, "apiKey") {
		t.Fatalf("blank key wrote an apiKey key:\n%s", out)
	}
}

func TestOMPUpsertBlankKeyKeepsStoredCredential(t *testing.T) {
	home := ompHome(t)
	if err := OMPUpsertCustomProvider("gw", ompDef(), "first-key"); err != nil {
		t.Fatal(err)
	}
	def := ompDef()
	def.Compat = map[string]bool{}
	if err := OMPUpsertCustomProvider("gw", def, ""); err != nil {
		t.Fatal(err)
	}
	if key, ok := OMPCustomAPIKey("gw"); !ok || key != "first-key" {
		t.Fatalf("key = %q, %v; want the stored credential kept", key, ok)
	}
	_ = home
}

func TestOMPUpertIsIdempotent(t *testing.T) {
	home := ompHome(t)
	if err := OMPUpsertCustomProvider("gw", ompDef(), "k"); err != nil {
		t.Fatal(err)
	}
	first, err := os.ReadFile(filepath.Join(home, "models.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if err := OMPUpsertCustomProvider("gw", ompDef(), "k"); err != nil {
		t.Fatal(err)
	}
	second, _ := os.ReadFile(filepath.Join(home, "models.yml"))
	if string(first) != string(second) {
		t.Fatalf("same upsert changed the file:\nfirst:\n%s\nsecond:\n%s", first, second)
	}
}

func TestOMPRemoveCustomProvider(t *testing.T) {
	home := ompHome(t)
	if err := OMPUpsertCustomProvider("gw", ompDef(), "k"); err != nil {
		t.Fatal(err)
	}
	if err := OMPUpsertCustomProvider("other", ompDef(), "k2"); err != nil {
		t.Fatal(err)
	}
	if err := OMPRemoveCustomProvider("nope"); err == nil {
		t.Fatal("removing an absent id succeeded")
	}
	if err := OMPRemoveCustomProvider("anthropic"); err == nil {
		t.Fatal("removing a built-in id succeeded")
	}
	if err := OMPRemoveCustomProvider("gw"); err != nil {
		t.Fatal(err)
	}
	if _, ok := OMPLoadCustomDefinitions()["gw"]; ok {
		t.Fatal("removed definition is still loaded")
	}
	if _, ok := OMPLoadCustomDefinitions()["other"]; !ok {
		t.Fatal("the neighbor went with it")
	}
	raw, _ := os.ReadFile(filepath.Join(home, "models.yml"))
	if strings.Contains(string(raw), "apiKey: k\n") && strings.Contains(string(raw), "gw") {
		t.Fatalf("gw residue left behind:\n%s", raw)
	}
}

func TestOMPUpsertRefusals(t *testing.T) {
	home := ompHome(t)
	if err := os.WriteFile(filepath.Join(home, "models.yml"), []byte("providers: {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name string
		id   string
		def  CustomDefinition
	}{
		{"builtin id", "openai", ompDef()},
		{"bad id", "Nope!", ompDef()},
		{"empty id", "  ", ompDef()},
		{"no base url", "gw", CustomDefinition{API: APIOpenAICompletions, Models: []CustomModel{{ID: "m"}}}},
		{"bad api", "gw", CustomDefinition{BaseURL: "https://x.com/v1", API: "pi-messages", Models: []CustomModel{{ID: "m"}}}},
		{"no models", "gw", CustomDefinition{BaseURL: "https://x.com/v1", API: APIOpenAICompletions}},
	}
	for _, tc := range cases {
		if err := OMPUpsertCustomProvider(tc.id, tc.def, "k"); err == nil {
			t.Fatalf("%s: upsert succeeded", tc.name)
		}
	}
	// pi's wider thinking-format union is pi's wire code; omp documents five.
	withFormat := ompDef()
	withFormat.ThinkingFormat = "deepseek"
	if err := OMPUpsertCustomProvider("gw", withFormat, "k"); err == nil {
		t.Fatal("omp upsert accepted pi's deepseek thinking format")
	}
	withFormat.ThinkingFormat = "zai"
	if err := OMPUpsertCustomProvider("gw", withFormat, "k"); err != nil {
		t.Fatalf("omp's own zai format refused: %v", err)
	}
}

func TestOMPUpsertRefusesForeignRootKeys(t *testing.T) {
	home := ompHome(t)
	seed := "providers:\n  gw:\n    baseUrl: https://x.com/v1\nother-top-level: 1\n"
	path := filepath.Join(home, "models.yml")
	if err := os.WriteFile(path, []byte(seed), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := OMPUpsertCustomProvider("new", ompDef(), "k"); err == nil {
		t.Fatal("upsert into a file omp would reject succeeded")
	}
	raw, _ := os.ReadFile(path)
	if string(raw) != seed {
		t.Fatalf("the refused write changed the file:\n%s", raw)
	}
}

func TestOMPLoadSkipsBuiltinsBrokenAndKeyless(t *testing.T) {
	home := ompHome(t)
	seed := `providers:
  anthropic:
    baseUrl: https://proxy.example.com/v1
    api: anthropic-messages
  broken: just a scalar, not a mapping
  keyed:
    baseUrl: https://a.example.com/v1
    apiKey: sk-test-123
    models:
      - id: m
  keyless:
    baseUrl: https://b.example.com/v1
    models:
      - id: m
    headers: 3
`
	if err := os.WriteFile(filepath.Join(home, "models.yml"), []byte(seed), 0o600); err != nil {
		t.Fatal(err)
	}
	defs := OMPLoadCustomDefinitions()
	if _, ok := defs["anthropic"]; ok {
		t.Fatal("a built-in override loaded as a custom definition")
	}
	if _, ok := defs["broken"]; ok {
		t.Fatal("a broken entry loaded")
	}
	if !defs["keyed"].Keyed {
		t.Fatal("a keyed definition reads as keyless")
	}
	// A hand-written scalar where an object belongs (headers: 3) cannot break
	// the row — it reads as absent.
	if defs["keyless"].Keyed {
		t.Fatal("keyless definition reads as keyed")
	}
	if defs["keyless"].BaseURL != "https://b.example.com/v1" {
		t.Fatalf("keyless = %+v", defs["keyless"])
	}
}

func TestOMPLoadReadsThinkingLevelMapForPrefill(t *testing.T) {
	home := ompHome(t)
	seed := `providers:
  gw:
    baseUrl: https://x.com/v1
    api: openai-completions
    models:
      - id: m1
        reasoning: true
        thinkingLevelMap:
          low: low
          xhigh: high
`
	if err := os.WriteFile(filepath.Join(home, "models.yml"), []byte(seed), 0o600); err != nil {
		t.Fatal(err)
	}
	defs := OMPLoadCustomDefinitions()
	defs0 := defs["gw"]
	if len(defs0.Definitions) != 1 {
		t.Fatalf("definitions = %+v", defs0.Definitions)
	}
	m := defs0.Definitions[0]
	if len(m.ThinkingLevels) != 2 || m.ThinkingLevels[0] != "low" || m.ThinkingLevels[1] != "xhigh" {
		t.Fatalf("thinking levels = %v", m.ThinkingLevels)
	}
	if m.ThinkingLevelMap["xhigh"] != "high" {
		t.Fatalf("level map = %v", m.ThinkingLevelMap)
	}
}

func TestOMPModelsPath(t *testing.T) {
	home := ompHome(t)
	if got, want := OMPModelsPath(), filepath.Join(home, "models.yml"); got != want {
		t.Fatalf("path = %q, want %q", got, want)
	}
	// The env override loses to the test seam, like clisession's OmpTestRoot.
	t.Setenv("PI_CODING_AGENT_DIR", "/somewhere/else")
	if got, want := OMPModelsPath(), filepath.Join(home, "models.yml"); got != want {
		t.Fatalf("seam path = %q, want %q", got, want)
	}
	old := ompTestDir
	ompTestDir = ""
	defer func() { ompTestDir = old }()
	fakeHome := t.TempDir()
	t.Setenv("HOME", fakeHome)
	t.Setenv("PI_CODING_AGENT_DIR", "")
	if got, want := OMPModelsPath(), filepath.Join(fakeHome, ".omp", "agent", "models.yml"); got != want {
		t.Fatalf("default path = %q, want %q", got, want)
	}
	t.Setenv("PI_CODING_AGENT_DIR", "/somewhere/else")
	if got, want := OMPModelsPath(), "/somewhere/else/models.yml"; got != want {
		t.Fatalf("override path = %q, want %q", got, want)
	}
}

func TestOMPValidateCustomID(t *testing.T) {
	if err := OMPValidateCustomID("llama.cpp"); err == nil {
		t.Fatal("omp's llama.cpp id accepted as custom")
	}
	if err := OMPValidateCustomID("cheaperinference"); err != nil {
		t.Fatalf("plain id refused: %v", err)
	}
}
