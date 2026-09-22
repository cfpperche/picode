package climetrics

import (
	"math"
	"testing"

	"github.com/cfpperche/picode/internal/pricing"
)

// testPrices is a table whose rates make the arithmetic readable:
// $1 per 1000 uncached input or output tokens, a tenth of that cached.
func testPrices(t *testing.T) *pricing.Table {
	t.Helper()
	tab, err := pricing.Parse([]byte(`{
	 "opus": {"input_cost_per_token": 1, "output_cost_per_token": 1},
	 "claude-opus-5": {"input_cost_per_token": 0.001, "output_cost_per_token": 0.001,
	   "cache_read_input_token_cost": 0.0001, "cache_creation_input_token_cost": 0.001},
	 "gpt-5.6": {"input_cost_per_token": 0.001, "output_cost_per_token": 0.001,
	   "cache_read_input_token_cost": 0.0001}
	}`))
	if err != nil {
		t.Fatal(err)
	}
	return tab
}

func near(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestCodexIsEstimatedAtListPrice(t *testing.T) {
	root := withCodexRoot(t)
	turn := codexTurn(day(1), 1000, 100)
	turn["payload"].(map[string]any)["info"].(map[string]any)["last_token_usage"].(map[string]any)["cached_input_tokens"] = 500
	writeRollout(t, root, "2026-09-06", []map[string]any{
		{"timestamp": day(1), "type": "session_meta", "payload": map[string]any{"id": "s1", "cwd": "/repo"}},
		{"timestamp": day(1), "type": "turn_context", "payload": map[string]any{"model": "gpt-5.6"}},
		turn,
		{"timestamp": day(1), "type": "turn_context", "payload": map[string]any{"model": "gpt-unlisted"}},
		codexTurn(day(1), 1000, 100),
	})
	r := req(7)
	r.Prices = testPrices(t)
	w, err := CodexMeter{}.Meter(r)
	if err != nil {
		t.Fatal(err)
	}
	// 500 uncached + 100 out at $0.001, 500 cached at $0.0001.
	want := 0.6 + 0.05
	if !near(w.Stats.Current.Cost, want) || !near(w.Stats.Current.Estimated, want) {
		t.Fatalf("cost = %v estimated = %v, want both %v", w.Stats.Current.Cost, w.Stats.Current.Estimated, want)
	}
	if w.Coverage.Signals[SigCost] != StateEstimated {
		t.Fatalf("cost state = %v, want estimated", w.Coverage.Signals[SigCost])
	}
	for _, m := range w.Stats.ByModel {
		if m.Model == "gpt-5.6" && !near(m.Estimated, want) {
			t.Fatalf("model row = %+v", m)
		}
	}
	if acc := w.Coverage.Note; acc == "" {
		t.Fatal("no note naming the estimate")
	}

	// No table: nothing is estimated, and Codex says it prices nothing.
	w, _ = CodexMeter{}.Meter(req(7))
	if w.Stats.Current.Cost != 0 || w.Coverage.Signals[SigCost] != StateNotReported {
		t.Fatalf("without a table: cost %v state %v", w.Stats.Current.Cost, w.Coverage.Signals[SigCost])
	}
}

func TestClaudeCodeEstimatesOnlySessionsWithoutASnapshot(t *testing.T) {
	root := withClaudeRoot(t)
	mk := func(sid string, in int64) map[string]any {
		a := assistant(day(1), "claude-opus-5", in, 0, 0, nil)
		a["sessionId"] = sid
		return a
	}
	// A priced session: Claude Code's own $1.00 stands, and its subagent —
	// which never carries a snapshot — is not estimated on top of it.
	writeTranscript(t, root, "-repo", "priced.jsonl", []map[string]any{
		mk("priced", 1000),
		costState("claude-opus-5", 1000, 0, 0, 1.00, 0, 0, 0, 0, 0),
	})
	writeTranscript(t, root, "-repo/priced/subagents", "agent-a.jsonl", []map[string]any{mk("priced", 500)})
	// A live session with no snapshot: 2000 input at $0.001 = $2.00.
	writeTranscript(t, root, "-repo", "live.jsonl", []map[string]any{mk("live", 2000)})

	r := req(7)
	r.Prices = testPrices(t)
	w, err := ClaudeCodeMeter{}.Meter(r)
	if err != nil {
		t.Fatal(err)
	}
	if !near(w.Stats.Current.Cost, 3.00) || !near(w.Stats.Current.Estimated, 2.00) {
		t.Fatalf("cost = %v estimated = %v, want 3.00 with 2.00 estimated", w.Stats.Current.Cost, w.Stats.Current.Estimated)
	}
	if w.Coverage.Signals[SigCost] != StateReported {
		t.Fatalf("cost state = %v, want reported (every turn priced, some by estimate)", w.Coverage.Signals[SigCost])
	}
}

func TestAFamilyNameIsNeverEstimated(t *testing.T) {
	root := withClaudeRoot(t)
	writeTranscript(t, root, "-repo", "s.jsonl", []map[string]any{assistant(day(1), "opus", 1000, 0, 0, nil)})
	r := req(7)
	r.Prices = testPrices(t) // lists "opus" at $1 a token — never used
	w, _ := ClaudeCodeMeter{}.Meter(r)
	if w.Stats.Current.Cost != 0 || w.Coverage.Signals[SigCost] != StateNotReported {
		t.Fatalf("cost = %v state = %v, want unpriced", w.Stats.Current.Cost, w.Coverage.Signals[SigCost])
	}
}
