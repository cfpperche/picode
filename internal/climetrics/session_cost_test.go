package climetrics

import (
	"path/filepath"
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
