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

func TestParseGrok(t *testing.T) {
	raw, err := os.ReadFile("testdata/grok-models-cache.json")
	if err != nil {
		t.Fatal(err)
	}
	rows, err := parseGrok(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("rows = %d; the hidden model must be left out", len(rows))
	}
	a := rows[1]
	if a.Selector != "grok-4.7" || a.Name != "Grok 4.7" || a.Provider != "xai" || a.Context != 500000 || !a.Reasoning ||
		strings.Join(a.Thinking, ",") != "low,medium,high,xhigh" {
		t.Fatalf("row 1 = %+v", a)
	}
	if _, err := parseGrok([]byte("not json")); err == nil {
		t.Error("a file PiCode cannot read must be an error that says so")
	}
}

// Grok is never run: the reader reads the file Grok keeps, from GROK_HOME
// when set, and says so plainly when Grok has not written it yet.
func TestGrokReadsItsOwnFileAndRunsNothing(t *testing.T) {
	root := t.TempDir()
	t.Setenv("GROK_HOME", root)
	t.Setenv("PATH", t.TempDir()) // no grok binary to run
	Forget("grok")
	if _, err := Read(t.Context(), "grok", "", true); err == nil || !strings.Contains(err.Error(), "sign in and open Grok once") {
		t.Fatalf("no file: err = %v", err)
	}
	raw, err := os.ReadFile("testdata/grok-models-cache.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "models_cache.json"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	rep, err := Read(t.Context(), "grok", "", true)
	if err != nil {
		t.Fatal(err)
	}
	if rep.CLI != "grok" || len(rep.Models) != 2 {
		t.Fatalf("report = %+v", rep)
	}
}

func TestParseClaude(t *testing.T) {
	raw, err := os.ReadFile("testdata/claude-list-models.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	rows, err := parseClaude(raw)
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, r := range rows {
		ids = append(ids, r.Selector)
	}
	if strings.Join(ids, ",") != "opus[1m],claude-fable-5-1,sonnet,haiku" {
		t.Fatalf("selectors = %v; the default row is a mirror and must be left out", ids)
	}
	if rows[0].Name != "Opus (1M context)" || strings.Join(rows[0].Thinking, ",") != "low,medium,high,xhigh,max" || !rows[0].Reasoning {
		t.Fatalf("row 0 = %+v", rows[0])
	}
	if rows[3].Reasoning || len(rows[3].Thinking) != 0 {
		t.Fatalf("haiku states no effort levels: %+v", rows[3])
	}
	disabled := `{"type":"control_response","response":{"subtype":"success","request_id":"p","response":{"models":[{"value":"cc-update-required-1","displayName":"Fable (disabled)","disabled":true},{"value":"sonnet","displayName":"Sonnet"}]}}}`
	if rows, err := parseClaude([]byte(disabled)); err != nil || len(rows) != 1 || rows[0].ID != "sonnet" {
		t.Fatalf("a disabled placeholder must never be offered: %+v %v", rows, err)
	}
	old := `{"type":"control_response","response":{"subtype":"error","request_id":"p","error":"Unsupported control request subtype: list_models"}}`
	if _, err := parseClaude([]byte(old)); err == nil || !strings.Contains(err.Error(), "update it") {
		t.Fatalf("an old Claude Code must be named: %v", err)
	}
	if _, err := parseClaude([]byte("not json")); err == nil {
		t.Error("output PiCode cannot read must be an error that says so")
	}
}

func TestClaudeExtrasComeFromTheOwnersConfigReadOnly(t *testing.T) {
	cfg := filepath.Join(t.TempDir(), ".claude.json")
	body := `{"additionalModelOptionsCache":[{"value":"claude-fable-5-1[1m]","label":"Fable","description":"Fable 5.1"},{"value":"sonnet","label":"dup"}],"other":1}`
	if err := os.WriteFile(cfg, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	before, _ := os.Stat(cfg)
	rows := addClaudeExtras([]Model{{ID: "sonnet", Selector: "sonnet"}}, cfg)
	if len(rows) != 2 || rows[1].Selector != "claude-fable-5-1[1m]" || rows[1].Name != "Fable" {
		t.Fatalf("rows = %+v", rows)
	}
	after, _ := os.Stat(cfg)
	if !after.ModTime().Equal(before.ModTime()) {
		t.Error("the owner's config must never be written")
	}
	if got := addClaudeExtras([]Model{{ID: "sonnet"}}, filepath.Join(t.TempDir(), "missing")); len(got) != 1 {
		t.Errorf("no config adds nothing: %+v", got)
	}
}
