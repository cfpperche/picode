package climetrics

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/clisession"
)

// day builds an RFC3339 timestamp n days before the reference instant used
// by every fixture here.
var fixtureNow = time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)

func day(n int) string { return fixtureNow.AddDate(0, 0, -n).Format(time.RFC3339) }

// writeTranscript lays down one Claude Code transcript under a fake
// ~/.claude/projects and points ClaudeProjectsRoot at it.
func writeTranscript(t *testing.T, root, project, name string, lines []map[string]any) string {
	t.Helper()
	dir := filepath.Join(root, project)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, name)
	var b strings.Builder
	for _, l := range lines {
		raw, err := json.Marshal(l)
		if err != nil {
			t.Fatal(err)
		}
		b.Write(raw)
		b.WriteByte('\n')
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func withClaudeRoot(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	old := clisession.ClaudeTestRoot
	clisession.ClaudeTestRoot = dir
	t.Cleanup(func() { clisession.ClaudeTestRoot = old })
	return dir
}

// assistant is one priced assistant turn.
func assistant(ts, model string, in, out, cacheRead int64, tools []any) map[string]any {
	content := []any{}
	content = append(content, tools...)
	return map[string]any{
		"type": "assistant", "timestamp": ts, "cwd": "/repo",
		"message": map[string]any{
			"role": "assistant", "model": model, "stop_reason": "end_turn",
			"usage": map[string]any{
				"input_tokens": in, "output_tokens": out,
				"cache_read_input_tokens": cacheRead, "cache_creation_input_tokens": 0,
			},
			"content": content,
		},
	}
}

func toolUse(name string) any {
	return map[string]any{"type": "tool_use", "name": name}
}

func toolErr() any {
	return map[string]any{"type": "tool_result", "is_error": true}
}

// costState is the cumulative snapshot Claude Code writes; units below are
// chosen so the derived rate is exactly $1 per 1000 tokens.
func costState(model string, in, out, cacheRead int64, cost float64, linesAdd, linesDel, apiMs, toolMs, wallMs int64) map[string]any {
	return map[string]any{
		"type": "cost-state", "totalCostUSD": cost,
		"totalLinesAdded": linesAdd, "totalLinesRemoved": linesDel,
		"totalAPIDuration": apiMs, "totalToolDuration": toolMs, "totalDuration": wallMs,
		"modelUsage": map[string]any{
			model: map[string]any{
				"inputTokens": in, "outputTokens": out,
				"cacheReadInputTokens": cacheRead, "cacheCreationInputTokens": 0,
				"costUSD": cost,
			},
		},
	}
}

func req(scope Scope, days int) Request {
	to := time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)
	from := to.AddDate(0, 0, -days)
	return Request{From: from, To: to, PriorFrom: from.AddDate(0, 0, -days), Loc: time.UTC, Scope: scope}
}

func TestClaudeCodePricesFromTheSessionsOwnSnapshot(t *testing.T) {
	root := withClaudeRoot(t)
	// 900 in + 100 out = 1000 units, $1.00 total => $0.001 per unit.
	writeTranscript(t, root, "-repo", "s1.jsonl", []map[string]any{
		assistant(day(1), "opus", 400, 100, 0, []any{toolUse("Bash")}),
		assistant(day(2), "opus", 400, 0, 0, nil),
		costState("opus", 800, 100, 100, 1.00, 40, 5, 3000, 1000, 9000),
	})

	w, err := ClaudeCodeMeter{}.Meter(req(ScopeMachine, 7))
	if err != nil {
		t.Fatal(err)
	}
	// Both messages are in window, so the whole $1.00 lands.
	if got := w.Stats.Current.Cost; !approx(got, 1.00) {
		t.Fatalf("cost = %v, want 1.00 (the snapshot's own total)", got)
	}
	if w.Coverage.Signals[SigCost] != StateReported {
		t.Fatalf("cost coverage = %v, want reported", w.Coverage.Signals[SigCost])
	}
	if w.Impact == nil || w.Impact.LinesAdded != 40 || w.Impact.LinesRemoved != 5 {
		t.Fatalf("impact = %+v, want 40/5", w.Impact)
	}
	// Files is a pointer precisely so it stays absent rather than reading 0.
	if w.Impact.Files != nil {
		t.Fatalf("files = %v, want absent: Claude Code never counts files", *w.Impact.Files)
	}
	if w.Timing == nil || w.Timing.APIMs != 3000 || w.Timing.ToolMs != 1000 {
		t.Fatalf("timing = %+v", w.Timing)
	}
}

func TestClaudeCodeSplitsCostAcrossTheDaysItWasSpent(t *testing.T) {
	root := withClaudeRoot(t)
	// Equal units on two different days; a cumulative reading would put the
	// whole bill on whichever day the file was last touched.
	writeTranscript(t, root, "-repo", "s1.jsonl", []map[string]any{
		assistant(day(1), "opus", 500, 0, 0, nil),
		assistant(day(3), "opus", 500, 0, 0, nil),
		costState("opus", 1000, 0, 0, 2.00, 0, 0, 0, 0, 0),
	})

	w, err := ClaudeCodeMeter{}.Meter(req(ScopeMachine, 7))
	if err != nil {
		t.Fatal(err)
	}
	byDay := map[string]float64{}
	for _, d := range w.Stats.Series {
		byDay[d.Date] = d.Cost
	}
	if len(byDay) != 2 {
		t.Fatalf("series covers %d days, want 2: %+v", len(byDay), w.Stats.Series)
	}
	for date, cost := range byDay {
		if !approx(cost, 1.00) {
			t.Fatalf("%s = %v, want 1.00 — the spend follows the tokens, not the file's mtime", date, cost)
		}
	}
}

func TestClaudeCodeCountsWindowOnly(t *testing.T) {
	root := withClaudeRoot(t)
	writeTranscript(t, root, "-repo", "s1.jsonl", []map[string]any{
		assistant(day(1), "opus", 500, 0, 0, nil),  // inside a 2-day window
		assistant(day(30), "opus", 500, 0, 0, nil), // far outside
		costState("opus", 1000, 0, 0, 2.00, 0, 0, 0, 0, 0),
	})

	w, err := ClaudeCodeMeter{}.Meter(req(ScopeMachine, 2))
	if err != nil {
		t.Fatal(err)
	}
	if got := w.Stats.Current.Cost; !approx(got, 1.00) {
		t.Fatalf("cost = %v, want 1.00 — only the in-window half of the session", got)
	}
	if got := w.Stats.Current.Messages; got != 1 {
		t.Fatalf("messages = %d, want 1", got)
	}
}

func TestClaudeCodeReportsPartialCostRatherThanAZero(t *testing.T) {
	root := withClaudeRoot(t)
	writeTranscript(t, root, "-repo", "priced.jsonl", []map[string]any{
		assistant(day(1), "opus", 1000, 0, 0, nil),
		costState("opus", 1000, 0, 0, 1.00, 0, 0, 0, 0, 0),
	})
	// A live session: real tokens, no snapshot, therefore no price.
	writeTranscript(t, root, "-repo", "live.jsonl", []map[string]any{
		assistant(day(1), "opus", 5000, 0, 0, nil),
	})

	w, err := ClaudeCodeMeter{}.Meter(req(ScopeMachine, 7))
	if err != nil {
		t.Fatal(err)
	}
	if w.Coverage.Signals[SigCost] != StatePartial {
		t.Fatalf("cost coverage = %v, want partial", w.Coverage.Signals[SigCost])
	}
	if !strings.Contains(w.Coverage.Note, "1 of 2 sessions") {
		t.Fatalf("note should name both counts, got %q", w.Coverage.Note)
	}
	// The unpriced session still contributes its tokens: silence about cost
	// is not silence about activity.
	if got := w.Stats.Tokens.Input; got != 6000 {
		t.Fatalf("input tokens = %d, want 6000 (both sessions)", got)
	}
	if got := w.Stats.Current.Cost; !approx(got, 1.00) {
		t.Fatalf("cost = %v, want 1.00 — the priced session only", got)
	}
}

func TestClaudeCodeCountsToolsAndErrors(t *testing.T) {
	root := withClaudeRoot(t)
	writeTranscript(t, root, "-repo", "s1.jsonl", []map[string]any{
		assistant(day(1), "opus", 10, 0, 0, []any{toolUse("Bash"), toolUse("Read"), toolErr()}),
		assistant(day(1), "opus", 10, 0, 0, []any{toolUse("Bash")}),
	})
	w, err := ClaudeCodeMeter{}.Meter(req(ScopeMachine, 7))
	if err != nil {
		t.Fatal(err)
	}
	calls := map[string]int{}
	for _, tb := range w.Stats.Tools {
		if tb.CLI != "claude-code" {
			t.Fatalf("tool %q lost its CLI attribution", tb.Name)
		}
		calls[tb.Name] = tb.Calls
	}
	if calls["Bash"] != 2 || calls["Read"] != 1 {
		t.Fatalf("tool calls = %+v", calls)
	}
	if w.Stats.Turns.Errors != 1 {
		t.Fatalf("errors = %d, want 1", w.Stats.Turns.Errors)
	}
}

func TestClaudeCodeHonoursPiCodeScope(t *testing.T) {
	root := withClaudeRoot(t)
	writeTranscript(t, root, "-repo", "claimed.jsonl", []map[string]any{
		assistant(day(1), "opus", 1000, 0, 0, nil),
		costState("opus", 1000, 0, 0, 1.00, 0, 0, 0, 0, 0),
	})
	writeTranscript(t, root, "-elsewhere", "unclaimed.jsonl", []map[string]any{
		{"type": "assistant", "timestamp": day(1), "cwd": "/elsewhere",
			"message": map[string]any{"role": "assistant", "model": "opus",
				"usage": map[string]any{"input_tokens": 1000}}},
		costState("opus", 1000, 0, 0, 9.00, 0, 0, 0, 0, 0),
	})

	r := req(ScopePiCode, 7)
	r.Claimed = []string{"/repo"}
	w, err := ClaudeCodeMeter{}.Meter(r)
	if err != nil {
		t.Fatal(err)
	}
	if got := w.Stats.Current.Cost; !approx(got, 1.00) {
		t.Fatalf("cost = %v, want 1.00 — /elsewhere is not a claimed folder", got)
	}

	// A worktree under a claimed folder rolls up rather than falling out.
	writeTranscript(t, root, "-repo-wt", "wt.jsonl", []map[string]any{
		{"type": "assistant", "timestamp": day(1), "cwd": "/repo/.worktrees/x",
			"message": map[string]any{"role": "assistant", "model": "opus",
				"usage": map[string]any{"input_tokens": 1000}}},
		costState("opus", 1000, 0, 0, 3.00, 0, 0, 0, 0, 0),
	})
	w, err = ClaudeCodeMeter{}.Meter(r)
	if err != nil {
		t.Fatal(err)
	}
	if got := w.Stats.Current.Cost; !approx(got, 4.00) {
		t.Fatalf("cost = %v, want 4.00 — a worktree belongs to the workspace above it", got)
	}
}

// TestClaudeCodeNeverCarriesMessageContent is the sabotage test: it plants
// a distinctive string in every place text can hide in a transcript and
// asserts none of it reaches the payload. The dashboard is an aggregate
// surface; a preview leaking here would be a privacy regression, not a
// cosmetic one.
func TestClaudeCodeNeverCarriesMessageContent(t *testing.T) {
	const secret = "SUPERSECRETPROMPTTEXT"
	root := withClaudeRoot(t)
	writeTranscript(t, root, "-repo", "s1.jsonl", []map[string]any{
		{"type": "user", "timestamp": day(1), "cwd": "/repo",
			"message": map[string]any{"role": "user", "content": secret}},
		{"type": "assistant", "timestamp": day(1), "cwd": "/repo",
			"message": map[string]any{"role": "assistant", "model": "opus",
				"usage": map[string]any{"input_tokens": 10},
				"content": []any{
					map[string]any{"type": "text", "text": secret},
					map[string]any{"type": "thinking", "thinking": secret},
					map[string]any{"type": "tool_use", "name": "Bash",
						"input": map[string]any{"command": secret}},
					map[string]any{"type": "tool_result", "content": secret},
				}}},
		costState("opus", 10, 0, 0, 0.01, 0, 0, 0, 0, 0),
	})

	w, err := ClaudeCodeMeter{}.Meter(req(ScopeMachine, 7))
	if err != nil {
		t.Fatal(err)
	}
	blob, err := json.Marshal(w.Stats)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(blob), secret) {
		t.Fatalf("message content reached the payload: %s", blob)
	}
}

// TestClaudeCodeSummaryIsAnIdentityNotAPreview guards the one string the
// adapter does keep. A session's own title is identity, the same as pi's
// session_info name; the rule ADR-0042 drew is "no message content", not
// "no strings".
func TestClaudeCodeSummaryIsAnIdentityNotAPreview(t *testing.T) {
	root := withClaudeRoot(t)
	writeTranscript(t, root, "-repo", "s1.jsonl", []map[string]any{
		{"type": "summary", "summary": "Fix the favicon"},
		assistant(day(1), "opus", 10, 0, 0, nil),
	})
	w, err := ClaudeCodeMeter{}.Meter(req(ScopeMachine, 7))
	if err != nil {
		t.Fatal(err)
	}
	if len(w.Stats.TopSessions) != 1 || w.Stats.TopSessions[0].Name != "Fix the favicon" {
		t.Fatalf("topSessions = %+v", w.Stats.TopSessions)
	}
}

func TestClaudeCodeMissingRootIsEmptyNotAnError(t *testing.T) {
	old := clisession.ClaudeTestRoot
	clisession.ClaudeTestRoot = filepath.Join(t.TempDir(), "never-installed")
	t.Cleanup(func() { clisession.ClaudeTestRoot = old })

	w, err := ClaudeCodeMeter{}.Meter(req(ScopeMachine, 7))
	if err != nil {
		t.Fatalf("a CLI that was never installed must not be an error: %v", err)
	}
	if w.Stats.Current.Messages != 0 {
		t.Fatalf("messages = %d", w.Stats.Current.Messages)
	}
	if w.Coverage.CLI != "claude-code" {
		t.Fatal("coverage row must exist even with nothing to report")
	}
}

func approx(a, b float64) bool {
	d := a - b
	return d < 1e-9 && d > -1e-9
}

// TestClaudeCodeStaysOutOfTheCredentialRoster locks in a boundary that is
// easy to undo by accident. ByProvider feeds the Providers view, which
// reports what the credentials PiCode holds have cost. Claude Code signs in
// with its own account, so an "anthropic" row here would charge a
// subscription's list-price equivalent to an API key that never paid it.
func TestClaudeCodeStaysOutOfTheCredentialRoster(t *testing.T) {
	root := withClaudeRoot(t)
	writeTranscript(t, root, "-repo", "s1.jsonl", []map[string]any{
		assistant(day(1), "opus", 1000, 0, 0, nil),
		costState("opus", 1000, 0, 0, 25.00, 0, 0, 0, 0, 0),
	})
	w, err := ClaudeCodeMeter{}.Meter(req(ScopeMachine, 7))
	if err != nil {
		t.Fatal(err)
	}
	if len(w.Stats.ByProvider) != 0 {
		t.Fatalf("byProvider = %+v, want none", w.Stats.ByProvider)
	}
	// The spend is not lost — it is just told through the CLI, not the key.
	if !approx(w.Stats.Current.Cost, 25.00) {
		t.Fatalf("cost = %v, want 25.00", w.Stats.Current.Cost)
	}
}
