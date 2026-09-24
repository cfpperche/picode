package climetrics

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// An exit prices the sessions it points at (ADR-0194): the CLI's own record,
// and "not measured" — never zero — for a CLI whose sessions are not files.
func TestMeterSessionFile(t *testing.T) {
	cc, ok := MeterSessionFile("claude-code", filepath.Join("testdata", "claude-parent.jsonl"), nil)
	if !ok || cc.Tokens == 0 || cc.Turns == 0 {
		t.Fatalf("claude-code fixture = %+v, %v", cc, ok)
	}
	cx, ok := MeterSessionFile("codex", filepath.Join("testdata", "codex-rollout.jsonl"), nil)
	if !ok || cx.Tokens == 0 {
		t.Fatalf("codex fixture = %+v, %v", cx, ok)
	}
	// The models each assistant turn names (an exit's model for a terminal
	// CLI). Codex names it in turn_context, which this fixture predates; the
	// line below is the shape a 2026-09-23 rollout carries.
	if cc.Models["claude-opus-5"] != 1 || cc.Models["claude-fable-5-1"] != 1 {
		t.Fatalf("claude-code models = %v", cc.Models)
	}
	raw, err := os.ReadFile(filepath.Join("testdata", "codex-rollout.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.SplitN(string(raw), "\n", 2)
	ctx := `{"timestamp":"2026-09-06T00:32:35.000Z","type":"turn_context","payload":{"cwd":"/home/goat/picode","model":"gpt-6-astra"}}`
	withCtx := filepath.Join(t.TempDir(), "rollout.jsonl")
	if err := os.WriteFile(withCtx, []byte(lines[0]+"\n"+ctx+"\n"+lines[1]), 0o644); err != nil {
		t.Fatal(err)
	}
	if cm, ok := MeterSessionFile("codex", withCtx, nil); !ok || cm.Models["gpt-6-astra"] == 0 {
		t.Fatalf("codex models = %v, %v", cm.Models, ok)
	}
	// Without a price table nothing is estimated; Codex never writes a price.
	if cx.Estimated != 0 || cx.Cost != 0 {
		t.Fatalf("codex without prices must not invent a cost: %+v", cx)
	}
	for _, c := range []struct{ cli, path string }{
		{"grok", filepath.Join("testdata", "claude-parent.jsonl")},
		{"claude-code", filepath.Join("testdata", "missing.jsonl")},
		{"claude-code", "testdata"},
	} {
		if _, ok := MeterSessionFile(c.cli, c.path, nil); ok {
			t.Errorf("%s %s: measured, want not measured", c.cli, c.path)
		}
	}
}
