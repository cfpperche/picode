package climodels

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The fixtures are trimmed from the real answers measured on 2026-09-23.

func TestParseCodex(t *testing.T) {
	raw, err := os.ReadFile("testdata/codex-models.json")
	if err != nil {
		t.Fatal(err)
	}
	rows, err := parseCodex(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 {
		t.Fatalf("rows = %d; the hidden codex-auto-review must be left out", len(rows))
	}
	a := rows[0]
	if a.Selector != "gpt-6-astra" || a.Name != "GPT-6-Astra" || a.Context != 272000 || !a.Reasoning ||
		strings.Join(a.Thinking, ",") != "low,medium,high,xhigh,max,ultra" || strings.Join(a.Input, ",") != "image" {
		t.Fatalf("row 0 = %+v", a)
	}
	if _, err := parseCodex([]byte("not json")); err == nil {
		t.Error("output PiCode cannot read must be an error that says so")
	}
}

func TestParseOpencode(t *testing.T) {
	raw, err := os.ReadFile("testdata/opencode-models.txt")
	if err != nil {
		t.Fatal(err)
	}
	rows, err := parseOpencode(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 4 {
		t.Fatalf("rows = %d", len(rows))
	}
	a := rows[0]
	if a.Selector != "opencode/big-pickle" || a.Name != "Big Pickle" || a.Context != 200000 || a.MaxOut != 32000 || !a.Reasoning {
		t.Fatalf("row 0 = %+v", a)
	}
	last := rows[3]
	if len(last.Thinking) == 0 || last.Thinking[0] != "none" || !strings.Contains(last.Selector, "/") {
		t.Fatalf("variants must become ordered thinking levels: %+v", last)
	}
}

func TestParseMuse(t *testing.T) {
	raw, err := os.ReadFile("testdata/muse-model-list.json")
	if err != nil {
		t.Fatal(err)
	}
	rows, err := parseMuse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) == 0 {
		t.Fatal("no rows")
	}
	a := rows[0]
	if a.Selector != "muse-spark-1.3" || a.Provider != "meta" || a.Context != 1007997 || a.MaxOut != 128000 || a.Cost == nil || a.Cost.Input != 1.25 || a.Cost.Output != 4.25 {
		t.Fatalf("row 0 = %+v", a)
	}
	// Another currency is not shown as dollars.
	eur, _ := parseMuse([]byte(`{"models":[{"providerId":"p","modelId":"m","cost":{"input":"1","output":"2","currency":"EUR"}}]}`))
	if eur[0].Cost != nil {
		t.Fatalf("EUR priced as USD: %+v", eur[0].Cost)
	}
}

// A stand-in muse that speaks the handshake: the reader must initialize,
// then list, and close it afterwards.
func TestProbeMuseSpeaksTheHandshake(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "muse")
	body := `#!/bin/sh
read init
echo '{"jsonrpc":"2.0","method":"notice","params":{}}'
echo '{"jsonrpc":"2.0","id":1,"result":{}}'
read initialized
read list
case "$list" in *model/list*) echo '{"jsonrpc":"2.0","id":2,"result":{"models":[{"providerId":"meta","modelId":"m1","displayLabel":"M1","contextLimit":10}]}}';; esac
read eof
`
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	rep, err := probeMuse(t.Context(), script, "")
	if err != nil || len(rep.Models) != 1 || rep.Models[0].Selector != "m1" {
		t.Fatalf("rep = %+v, %v", rep, err)
	}
}

// Each new reader keys on the files that decide its answer.
func TestNewReadersFollowTheirInputs(t *testing.T) {
	h := t.TempDir()
	old := home
	home = func() string { return h }
	t.Cleanup(func() { home = old })
	for _, env := range []string{"CODEX_HOME", "XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_CACHE_HOME"} {
		t.Setenv(env, "")
	}
	cases := map[string]string{
		"codex":    ".codex/models_cache.json",
		"opencode": ".config/opencode/opencode.json",
		"muse":     ".config/muse/settings.json",
	}
	for cli, file := range cases {
		cmd := filepath.Join(t.TempDir(), "no-such-"+cli)
		before := fingerprint(cli, cmd, "")
		p := filepath.Join(h, file)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
		if fingerprint(cli, cmd, "") == before {
			t.Errorf("%s: writing %s did not move the fingerprint", cli, file)
		}
	}
	// OpenCode's answer depends on the folder: its project file is an input.
	ws := t.TempDir()
	before := fingerprint("opencode", "opencode", ws)
	if err := os.WriteFile(filepath.Join(ws, "opencode.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if fingerprint("opencode", "opencode", ws) == before {
		t.Error("a project opencode.json did not move the fingerprint")
	}
}
