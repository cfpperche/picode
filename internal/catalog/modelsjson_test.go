package catalog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// readModels returns the top-level object; readProviderRows the object pi's
// schema expects under "providers".
func readTop(t *testing.T, home string) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(home, ".pi", "agent", "models.json"))
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func readProviderRows(t *testing.T, home string) map[string]any {
	t.Helper()
	providers, ok := readTop(t, home)["providers"].(map[string]any)
	if !ok {
		t.Fatalf("models.json has no providers object: written shape must match pi's schema")
	}
	return providers
}

func TestUpsertCustomProviderCreatesFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	yes := true
	err := UpsertCustomProvider("cheaperinference", CustomDefinition{
		BaseURL: "https://api.cheaperinference.com/v1",
		API:     APIOpenAICompletions,
		Compat:  map[string]bool{"supportsDeveloperRole": false, "supportsReasoningEffort": !yes},
		Models:  []CustomModel{{ID: "gpt-5.4", ContextWindow: intPtr(400000), MaxTokens: intPtr(32000)}},
	})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(home, ".pi", "agent", "models.json")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode %v, want 0600", info.Mode().Perm())
	}
	// pi rejects anything else: the wrapper is the schema.
	prov, ok := readProviderRows(t, home)["cheaperinference"].(map[string]any)
	if !ok {
		t.Fatalf("provider missing: %s", rawOf(t, readTop(t, home)))
	}
	if prov["baseUrl"] != "https://api.cheaperinference.com/v1" || prov["api"] != "openai-completions" {
		t.Fatalf("%s", rawOf(t, prov))
	}
	models, _ := prov["models"].([]any)
	if len(models) != 1 || models[0].(map[string]any)["id"] != "gpt-5.4" {
		t.Fatalf("%s", rawOf(t, prov))
	}
}

func TestUpsertPreservesOtherProviders(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	path := filepath.Join(home, ".pi", "agent", "models.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	before := `{"providers":{"ollama":{"baseUrl":"http://localhost:11434/v1","api":"openai-completions","apiKey":"ollama","models":[{"id":"llama3.1:8b"}]}}}`
	if err := os.WriteFile(path, []byte(before), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := UpsertCustomProvider("my-gateway", CustomDefinition{
		BaseURL: "https://gw.example.com/v1", API: APIOpenAICompletions,
		Models: []CustomModel{{ID: "m1"}},
	}); err != nil {
		t.Fatal(err)
	}
	rows := readProviderRows(t, home)
	ollama, ok := rows["ollama"].(map[string]any)
	if !ok {
		t.Fatal("ollama dropped")
	}
	// JSON-value equality, not bytes: the untouched provider survives whole.
	if ollama["apiKey"] != "ollama" || ollama["baseUrl"] != "http://localhost:11434/v1" {
		t.Fatalf("%s", rawOf(t, ollama))
	}
	models, _ := ollama["models"].([]any)
	if len(models) != 1 || models[0].(map[string]any)["id"] != "llama3.1:8b" {
		t.Fatalf("%s", rawOf(t, ollama))
	}
	if _, ok := rows["my-gateway"]; !ok {
		t.Fatal("my-gateway missing")
	}
}

func TestUpsertMergesUnknownFields(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	path := filepath.Join(home, ".pi", "agent", "models.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	before := `{"providers":{"my-gateway":{"baseUrl":"https://old.example.com","api":"openai-completions","headers":{"X-Trace":"keepme"},"models":[{"id":"m1","samplingParams":{"top_k":5}},{"id":"m2"}]}}}`
	if err := os.WriteFile(path, []byte(before), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := UpsertCustomProvider("my-gateway", CustomDefinition{
		BaseURL: "https://new.example.com/v1", API: APIOpenAICompletions,
		Models: []CustomModel{{ID: "m1", MaxTokens: intPtr(8192)}},
	}); err != nil {
		t.Fatal(err)
	}
	prov := readProviderRows(t, home)["my-gateway"].(map[string]any)
	if prov["baseUrl"] != "https://new.example.com/v1" {
		t.Fatalf("%s", rawOf(t, prov))
	}
	headers, _ := prov["headers"].(map[string]any)
	if headers["X-Trace"] != "keepme" {
		t.Fatalf("headers dropped: %s", rawOf(t, prov))
	}
	// m2 was removed by the form; m1 keeps its samplingParams.
	models, _ := prov["models"].([]any)
	if len(models) != 1 {
		t.Fatalf("models = %s", rawOf(t, models))
	}
	m1 := models[0].(map[string]any)
	if m1["maxTokens"].(float64) != 8192 {
		t.Fatalf("%s", rawOf(t, m1))
	}
	sp, _ := m1["samplingParams"].(map[string]any)
	if sp == nil || sp["top_k"].(float64) != 5 {
		t.Fatalf("samplingParams dropped: %s", rawOf(t, m1))
	}
}

func TestUpsertThinkingLevels(t *testing.T) {
	yes, no := true, false

	// Decision table: the form's selection becomes a thinkingLevelMap, a
	// non-reasoning model drops the managed keys, and a hand-set key that the
	// form does not manage survives either way.
	t.Run("selection becomes a map with hidden levels as null", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)
		if err := UpsertCustomProvider("gw", CustomDefinition{
			BaseURL: "https://api.example.com/v1", API: APIOpenAICompletions,
			Models: []CustomModel{{ID: "m", Reasoning: &yes, ThinkingLevels: []string{"max", "high"}}},
		}); err != nil {
			t.Fatal(err)
		}
		m := firstModel(t, home, "gw")
		if m["reasoning"] != true {
			t.Fatalf("reasoning = %s", rawOf(t, m))
		}
		levelMap, _ := m["thinkingLevelMap"].(map[string]any)
		want := map[string]any{"minimal": nil, "low": nil, "medium": nil, "high": "high", "xhigh": nil, "max": "max"}
		if len(levelMap) != len(want) {
			t.Fatalf("map = %s", rawOf(t, levelMap))
		}
		for k, v := range want {
			if levelMap[k] != v {
				t.Fatalf("map[%s] = %v, want %v (%s)", k, levelMap[k], v, rawOf(t, levelMap))
			}
		}
	})

	t.Run("hand-set keys outside the form survive", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)
		seed(t, home, `{"providers":{"gw":{"baseUrl":"https://api.example.com/v1","api":"openai-completions","models":[{"id":"m","reasoning":true,"thinkingLevelMap":{"off":"none","high":"high","future":"x"}}]}}}`)
		if err := UpsertCustomProvider("gw", CustomDefinition{
			BaseURL: "https://api.example.com/v1", API: APIOpenAICompletions,
			Models: []CustomModel{{ID: "m", Reasoning: &yes, ThinkingLevels: []string{"high", "max"}}},
		}); err != nil {
			t.Fatal(err)
		}
		levelMap, _ := firstModel(t, home, "gw")["thinkingLevelMap"].(map[string]any)
		if levelMap["off"] != "none" || levelMap["future"] != "x" {
			t.Fatalf("unmanaged keys dropped: %s", rawOf(t, levelMap))
		}
		if levelMap["max"] != "max" {
			t.Fatalf("selection not applied: %s", rawOf(t, levelMap))
		}
	})

	t.Run("switching to non-reasoning drops the managed levels only", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)
		seed(t, home, `{"providers":{"gw":{"baseUrl":"https://api.example.com/v1","api":"openai-completions","models":[{"id":"m","reasoning":true,"thinkingLevelMap":{"off":"none","high":"high","max":"max"}}]}}}`)
		if err := UpsertCustomProvider("gw", CustomDefinition{
			BaseURL: "https://api.example.com/v1", API: APIOpenAICompletions,
			Models: []CustomModel{{ID: "m", Reasoning: &no}},
		}); err != nil {
			t.Fatal(err)
		}
		m := firstModel(t, home, "gw")
		if m["reasoning"] != false {
			t.Fatalf("reasoning = %s", rawOf(t, m))
		}
		levelMap, _ := m["thinkingLevelMap"].(map[string]any)
		if len(levelMap) != 1 || levelMap["off"] != "none" {
			t.Fatalf("managed keys must go, off must stay: %s", rawOf(t, levelMap))
		}
	})

	t.Run("a model without a map keeps none", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)
		if err := UpsertCustomProvider("gw", CustomDefinition{
			BaseURL: "https://api.example.com/v1", API: APIOpenAICompletions,
			Models: []CustomModel{{ID: "m"}},
		}); err != nil {
			t.Fatal(err)
		}
		if _, ok := firstModel(t, home, "gw")["thinkingLevelMap"]; ok {
			t.Fatal("an untouched model must not grow a map")
		}
	})

	t.Run("unknown level is refused and writes nothing", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)
		err := UpsertCustomProvider("gw", CustomDefinition{
			BaseURL: "https://api.example.com/v1", API: APIOpenAICompletions,
			Models: []CustomModel{{ID: "m", Reasoning: &yes, ThinkingLevels: []string{"turbo"}}},
		})
		if err == nil || !strings.Contains(err.Error(), "unsupported thinking level") {
			t.Fatalf("err = %v", err)
		}
		if _, statErr := os.Stat(filepath.Join(home, ".pi", "agent", "models.json")); !os.IsNotExist(statErr) {
			t.Fatal("a refused definition must not touch the file")
		}
	})

	t.Run("re-saving the same definition is byte-identical", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)
		def := CustomDefinition{
			BaseURL: "https://api.example.com/v1", API: APIOpenAICompletions,
			Models: []CustomModel{{ID: "m", Reasoning: &yes, ThinkingLevels: []string{"high", "max"}}},
		}
		if err := UpsertCustomProvider("gw", def); err != nil {
			t.Fatal(err)
		}
		first := rawOf(t, readTop(t, home))
		if err := UpsertCustomProvider("gw", def); err != nil {
			t.Fatal(err)
		}
		if second := rawOf(t, readTop(t, home)); second != first {
			t.Fatalf("not idempotent:\n%s\n%s", first, second)
		}
	})
}

// firstModel returns the provider's first model row as a decoded object.
func firstModel(t *testing.T, home, id string) map[string]any {
	t.Helper()
	prov, _ := readProviderRows(t, home)[id].(map[string]any)
	models, _ := prov["models"].([]any)
	if len(models) == 0 {
		t.Fatalf("no models under %s: %s", id, rawOf(t, prov))
	}
	m, _ := models[0].(map[string]any)
	return m
}

// seed writes a hand-edited models.json before the upsert runs.
func seed(t *testing.T, home, body string) {
	t.Helper()
	path := filepath.Join(home, ".pi", "agent", "models.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestUpsertCustomProviderValidation(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	yes := true
	cases := []struct {
		name string
		id   string
		def  CustomDefinition
	}{
		{"empty id", "", CustomDefinition{BaseURL: "https://x.com/v1", API: APIOpenAICompletions, Models: []CustomModel{{ID: "m"}}}},
		{"uppercase id", "MyGW", CustomDefinition{BaseURL: "https://x.com/v1", API: APIOpenAICompletions, Models: []CustomModel{{ID: "m"}}}},
		{"dot in id", "my.gw", CustomDefinition{BaseURL: "https://x.com/v1", API: APIOpenAICompletions, Models: []CustomModel{{ID: "m"}}}},
		{"builtin id", "anthropic", CustomDefinition{BaseURL: "https://x.com/v1", API: APIOpenAICompletions, Models: []CustomModel{{ID: "m"}}}},
		{"llama.cpp id", "llama.cpp", CustomDefinition{BaseURL: "https://x.com/v1", API: APIOpenAICompletions, Models: []CustomModel{{ID: "m"}}}},
		{"no scheme", "ok", CustomDefinition{BaseURL: "ftp://x.com/v1", API: APIOpenAICompletions, Models: []CustomModel{{ID: "m"}}}},
		{"bad api", "ok", CustomDefinition{BaseURL: "https://x.com/v1", API: "ollama-native", Models: []CustomModel{{ID: "m"}}}},
		{"no models", "ok", CustomDefinition{BaseURL: "https://x.com/v1", API: APIOpenAICompletions}},
		{"dup models", "ok", CustomDefinition{BaseURL: "https://x.com/v1", API: APIOpenAICompletions, Models: []CustomModel{{ID: "m"}, {ID: "m"}}}},
		{"space in model id", "ok", CustomDefinition{BaseURL: "https://x.com/v1", API: APIOpenAICompletions, Models: []CustomModel{{ID: "openai gpt"}}}},
		{"zero context", "ok", CustomDefinition{BaseURL: "https://x.com/v1", API: APIOpenAICompletions, Models: []CustomModel{{ID: "m", ContextWindow: intPtr(0)}}}},
		{"negative max", "ok", CustomDefinition{BaseURL: "https://x.com/v1", API: APIOpenAICompletions, Models: []CustomModel{{ID: "m", MaxTokens: intPtr(-1)}}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := UpsertCustomProvider(tc.id, tc.def); err == nil {
				t.Fatal("expected error")
			}
		})
	}
	// Sanity: the dotted "llama.cpp" is refused by the id pattern; a dash
	// spelling is a legitimate custom name.
	if err := UpsertCustomProvider("ok", CustomDefinition{BaseURL: "https://x.com/v1", API: APIOpenAICompletions, Compat: map[string]bool{"supportsDeveloperRole": !yes}, Models: []CustomModel{{ID: "m"}}}); err != nil {
		t.Fatal(err)
	}
}

func TestRemoveCustomProvider(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	path := filepath.Join(home, ".pi", "agent", "models.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	before := `{"providers":{"my-gw":{"baseUrl":"https://x.com/v1","models":[{"id":"m"}]},"hand-override":{"baseUrl":"https://proxy.example.com"}}}`
	if err := os.WriteFile(path, []byte(before), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := RemoveCustomProvider("my-gw"); err != nil {
		t.Fatal(err)
	}
	rows := readProviderRows(t, home)
	if _, ok := rows["my-gw"]; ok {
		t.Fatal("my-gw still present")
	}
	if _, ok := rows["hand-override"]; !ok {
		t.Fatal("hand-override dropped")
	}
	if err := RemoveCustomProvider("my-gw"); err == nil {
		t.Fatal("expected error for absent provider")
	}
	if err := RemoveCustomProvider("anthropic"); err == nil {
		t.Fatal("expected error for built-in")
	}
}

func TestUpsertRefusesUnwrappedFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	path := filepath.Join(home, ".pi", "agent", "models.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	// pi rejects this file; a merge would report success while the gateway
	// never loads. The write must be refused, not mixed into junk.
	before := `{"cheaperinference":{"baseUrl":"https://x.com/v1","models":[{"id":"m"}]}}`
	if err := os.WriteFile(path, []byte(before), 0o600); err != nil {
		t.Fatal(err)
	}
	err := UpsertCustomProvider("my-gw", CustomDefinition{
		BaseURL: "https://new.example.com/v1", API: APIOpenAICompletions,
		Models: []CustomModel{{ID: "m"}},
	})
	if err == nil {
		t.Fatal("expected refusal for unwrapped file")
	}
	after, _ := os.ReadFile(path)
	if string(after) != before {
		t.Fatalf("file changed on refusal:\n%s", after)
	}
}

func TestLoadCustomDefinitions(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if got := LoadCustomDefinitions(); len(got) != 0 {
		t.Fatalf("missing file should mean none, got %d", len(got))
	}
	path := filepath.Join(home, ".pi", "agent", "models.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := LoadCustomDefinitions(); len(got) != 0 {
		t.Fatalf("invalid file should mean none, got %d", len(got))
	}
	// A hand-written file without the wrapper is invalid for pi; PiCode
	// reads none of it and never silently adopts the stray entries.
	if err := os.WriteFile(path, []byte(`{"cheaperinference":{"baseUrl":"https://x.com/v1"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := LoadCustomDefinitions(); len(got) != 0 {
		t.Fatalf("unwrapped file should mean none, got %d", len(got))
	}
	body := `{"providers":{
	  "cheaperinference": {"baseUrl":"https://api.cheaperinference.com/v1","api":"anthropic-messages","models":[{"id":"claude-opus-5"}]},
	  "anthropic": {"baseUrl":"https://proxy.example.com"},
	  "no-url": {"api":"openai-completions"},
	  "defaulted": {"baseUrl":"https://d.example.com/v1"}
	}}`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	got := LoadCustomDefinitions()
	if len(got) != 2 {
		t.Fatalf("got %d (%v), want 2", len(got), rawOf(t, got))
	}
	ci := got["cheaperinference"]
	if ci.API != APIAnthropicMessages || len(ci.Models) != 1 {
		t.Fatalf("%s", rawOf(t, ci))
	}
	if got["defaulted"].API != APIOpenAICompletions {
		t.Fatalf("api default not applied")
	}
}

// rawOf is a test-only JSON dump for failure messages.
func rawOf(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		return "<unmarshalable>"
	}
	s := string(b)
	if len(s) > 400 {
		s = s[:400] + "…"
	}
	return s
}

func intPtr(i int) *int { return &i }
