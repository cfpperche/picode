package catalog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

const sample = `provider      model                                               context  max-out  thinking  images
anthropic     claude-sonnet-4-5                                   1M       64K      yes       yes
openai-codex  gpt-5.4                                             272K     128K     yes       yes
opencode      gemini-3-flash                                      1.0M     65.5K    yes       yes
`

func TestParseListModels(t *testing.T) {
	rows := ParseListModels(sample)
	if len(rows) != 3 {
		t.Fatalf("rows = %d, want 3", len(rows))
	}
	if rows[0].provider != "anthropic" || rows[0].model != "claude-sonnet-4-5" || !rows[0].thinking {
		t.Fatalf("row0 = %+v", rows[0])
	}
	if rows[1].provider != "openai-codex" || rows[1].model != "gpt-5.4" {
		t.Fatalf("row1 = %+v", rows[1])
	}
}

func TestSupportedThinkingMatchesTUI(t *testing.T) {
	zai := SupportedThinking(true, map[string]any{
		"off": nil, "minimal": nil, "low": "low", "medium": nil, "high": "high", "xhigh": nil, "max": "max",
	})
	if got := stringsJoin(zai); got != "low,high,max" {
		t.Fatalf("zai = %s", got)
	}
	xai := SupportedThinking(true, map[string]any{
		"off": nil, "minimal": nil, "low": "low", "medium": "medium", "high": "high", "xhigh": "xhigh", "max": nil,
	})
	if got := stringsJoin(xai); got != "low,medium,high,xhigh" {
		t.Fatalf("xai = %s", got)
	}
	anthropic := SupportedThinking(true, map[string]any{
		"off": nil, "xhigh": "xhigh", "max": "max",
	})
	if got := stringsJoin(anthropic); got != "minimal,low,medium,high,xhigh,max" {
		t.Fatalf("anthropic = %s", got)
	}
	if got := stringsJoin(SupportedThinking(false, nil)); got != "off" {
		t.Fatalf("no reasoning = %s", got)
	}
}

func stringsJoin(s []string) string {
	out := ""
	for i, v := range s {
		if i > 0 {
			out += ","
		}
		out += v
	}
	return out
}

func TestLoginMethod(t *testing.T) {
	if loginMethod("xai") != LoginBoth {
		t.Fatal("xai")
	}
	if loginMethod("openai-codex") != LoginOAuth {
		t.Fatal("codex")
	}
	if loginMethod("groq") != LoginAPIKey {
		t.Fatal("groq")
	}
	if loginMethod("kimi-coding") != LoginBoth {
		t.Fatal("kimi")
	}
}

func TestParseListModelsSkipsJunk(t *testing.T) {
	if n := len(ParseListModels("not a table\n\n")); n != 0 {
		t.Fatalf("got %d", n)
	}
}

func TestPutAPIKey(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := PutAPIKey("xai", "sk-test"); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(home, ".pi", "agent", "auth.json"))
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]map[string]string
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if m["xai"]["type"] != "api_key" || m["xai"]["key"] != "sk-test" {
		t.Fatalf("%s", raw)
	}
}

func TestPutLlama(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := PutLlama("http://127.0.0.1:8080/v1", ""); err != nil {
		t.Fatal(err)
	}
	if LlamaURL() != "http://127.0.0.1:8080" {
		t.Fatalf("%s", LlamaURL())
	}
	if LlamaKey() != "" {
		t.Fatal("key leaked empty")
	}
}

func TestRemoveAuthKeepsOtherKeys(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := filepath.Join(home, ".pi", "agent")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "auth.json")
	if err := os.WriteFile(path, []byte(`{"xai":{"type":"api"},"openai":{"type":"api"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := RemoveAuth("xai"); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if _, ok := m["xai"]; ok {
		t.Fatal("xai still present")
	}
	if _, ok := m["openai"]; !ok {
		t.Fatal("openai dropped")
	}
}
