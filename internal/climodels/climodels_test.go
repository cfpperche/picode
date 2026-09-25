package climodels

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

// The fixture is the shape omp 18.2.8 answered with on this machine
// (`omp models --json --kind all`, 2026-09-22): a `models` array whose rows
// carry provider, id, name, selector and contextWindow, and whose chat rows
// carry no `kind` at all.
const ompFixture = `{"models":[
 {"provider":"openai","id":"gpt-5-nano","name":"GPT-5 Nano","selector":"openai/gpt-5-nano","contextWindow":400000,"thinking":["low","medium","high"]},
 {"provider":"openai","id":"gpt-4o-mini","name":"GPT-4o mini","selector":"openai/gpt-4o-mini","contextWindow":128000},
 {"provider":"local","id":"kokoro","name":"Kokoro","kind":"tts","selector":"local/kokoro"}
]}`

func TestParseOmp(t *testing.T) {
	rep, err := parseOmp([]byte(ompFixture), "/work")
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Models) != 3 {
		t.Fatalf("got %d models", len(rep.Models))
	}
	if rep.Dir != "/work" {
		t.Errorf("the report must name the directory it was read in, got %q", rep.Dir)
	}
	// A row with no kind is a chat model; the pane's filter must not show a
	// blank chip for the majority of the catalog.
	if rep.Models[0].Kind != "chat" {
		t.Errorf("a row with no kind is %q, want chat", rep.Models[0].Kind)
	}
	if rep.Models[2].Kind != "tts" {
		t.Errorf("an explicit kind is %q", rep.Models[2].Kind)
	}
	if len(rep.Kinds) != 2 || rep.Kinds[0] != "chat" || rep.Kinds[1] != "tts" {
		t.Errorf("kinds = %v, want [chat tts]", rep.Kinds)
	}
	if rep.Models[0].Selector != "openai/gpt-5-nano" {
		t.Errorf("selector = %q", rep.Models[0].Selector)
	}
}

// An empty catalog is an answer, not a failure: omp returns zero models both
// when a project disabled every provider and when none has a credential
// (measured 2026-09-22 — 55 models in one folder, 0 in the next). The reader
// must not turn either into an error, and must not invent a reason.
func TestEmptyCatalogIsAnAnswer(t *testing.T) {
	rep, err := parseOmp([]byte(`{"models":[]}`), "")
	if err != nil {
		t.Fatal(err)
	}
	if rep.Models == nil {
		t.Fatal("an empty catalog must serialise as [], not null")
	}
	if len(rep.Models) != 0 || len(rep.Kinds) != 0 {
		t.Errorf("got %d models, %d kinds", len(rep.Models), len(rep.Kinds))
	}
}

// A selector the vendor omitted is rebuilt the way the vendor builds one, so a
// row is never offered without the string a save would write.
func TestSelectorIsAlwaysPresent(t *testing.T) {
	rep, err := parseOmp([]byte(`{"models":[{"provider":"zai","id":"glm-5.3-flash"}]}`), "")
	if err != nil {
		t.Fatal(err)
	}
	if rep.Models[0].Selector != "zai/glm-5.3-flash" {
		t.Errorf("selector = %q", rep.Models[0].Selector)
	}
}

func TestOnlyDeclaredCLIsAnswer(t *testing.T) {
	for _, cli := range []string{"omp", "pi", "codex", "opencode", "muse"} {
		if !Supports(cli) {
			t.Errorf("%s has a reader", cli)
		}
	}
	// Measured 2026-09-23: Claude Code, Grok and Antigravity can list but
	// rewrite the owner's files on every read; Hermes has no public way.
	for _, cli := range []string{"claude-code", "grok", "hermes", "agy"} {
		if Supports(cli) {
			t.Errorf("%s must not have a reader", cli)
		}
	}
	if _, err := Read(t.Context(), "grok", "", false); err == nil {
		t.Error("a CLI with no reader must be refused by name, not probed")
	}
}

func TestGarbageIsNamed(t *testing.T) {
	if _, err := parseOmp([]byte("not json"), ""); err == nil {
		t.Error("output PiCode cannot read must be an error that says so")
	}
}

// The pane asks for a Models tab only for a CLI the server can answer for; this
// is the seam between the two lists (the TestJSListMatchesTheCatalog pattern).
func TestJSListMatchesTheReaders(t *testing.T) {
	body, err := os.ReadFile("../../web/shared/domain/cliModels.js")
	if err != nil {
		t.Fatal(err)
	}
	match := regexp.MustCompile(`MODELS_CLIS = \[([^\]]*)\]`).FindSubmatch(body)
	if match == nil {
		t.Fatal("MODELS_CLIS not found in web/shared/domain/cliModels.js")
	}
	var js []string
	for _, part := range strings.Split(string(match[1]), ",") {
		if id := strings.Trim(strings.TrimSpace(part), `"`); id != "" {
			js = append(js, id)
		}
	}
	if strings.Join(js, ",") != strings.Join(Panes(), ",") {
		t.Fatalf("UI list %v, panes %v", js, Panes())
	}
}

// The price pair is read, the time-of-day multiplier is not.
func TestCostIsTheVendorsPair(t *testing.T) {
	rep, err := parseOmp([]byte(`{"models":[{"provider":"deepseek","id":"deepseek-flash","cost":{"input":0.3,"output":1.2,"cacheRead":0.006,"timeBased":{"offPeakMultiplier":0.5}},"input":["text","image"]}]}`), "")
	if err != nil {
		t.Fatal(err)
	}
	c := rep.Models[0].Cost
	if c == nil || c.Input != 0.3 || c.Output != 1.2 {
		t.Fatalf("cost = %+v", c)
	}
	if strings.Join(rep.Models[0].Input, ",") != "text,image" {
		t.Errorf("input = %v", rep.Models[0].Input)
	}
}

// A cached answer is valid only while the files that decide it are unchanged:
// a write to either config layer must never be answered from before it.
func TestFingerprintMovesWithTheFiles(t *testing.T) {
	h := t.TempDir()
	old := home
	home = func() string { return h }
	t.Cleanup(func() { home = old })
	t.Setenv("PI_CODING_AGENT_DIR", "")
	ws := t.TempDir()
	a := fingerprint("omp", "", ws)
	cfg := filepath.Join(h, ".omp", "agent", "config.yml")
	if err := os.MkdirAll(filepath.Dir(cfg), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfg, []byte("disabledProviders: [groq]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	b := fingerprint("omp", "", ws)
	if a == b {
		t.Fatal("creating the global config did not move the fingerprint")
	}
	proj := filepath.Join(ws, ".omp", "config.yml")
	if err := os.MkdirAll(filepath.Dir(proj), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(proj, []byte("disabledProviders: [openai]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	c := fingerprint("omp", "", ws)
	if c == b {
		t.Fatal("writing the workspace config did not move the fingerprint")
	}
	later := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(proj, later, later); err != nil {
		t.Fatal(err)
	}
	if fingerprint("omp", "", ws) == c {
		t.Fatal("a touched workspace config did not move the fingerprint")
	}
	if fingerprint("omp", "", "") == fingerprint("omp", "", ws) {
		t.Fatal("the machine answer and a workspace answer must not share a fingerprint")
	}
}

// The UI offers a model list beside a CLI's model field only for a CLI the
// server can ask: MODEL_READERS in cliModels.js is Supported().
func TestJSReadersMatchTheServer(t *testing.T) {
	body, err := os.ReadFile("../../web/shared/domain/cliModels.js")
	if err != nil {
		t.Fatal(err)
	}
	match := regexp.MustCompile(`MODEL_READERS = \[([^\]]*)\]`).FindSubmatch(body)
	if match == nil {
		t.Fatal("MODEL_READERS not found in web/shared/domain/cliModels.js")
	}
	var js []string
	for _, part := range strings.Split(string(match[1]), ",") {
		if id := strings.Trim(strings.TrimSpace(part), `"`); id != "" {
			js = append(js, id)
		}
	}
	if strings.Join(js, ",") != strings.Join(Supported(), ",") {
		t.Fatalf("UI readers %v, server readers %v", js, Supported())
	}
}
