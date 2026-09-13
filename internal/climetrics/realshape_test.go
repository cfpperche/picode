package climetrics

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/clisession"
)

// These tests parse lines copied from real stores (see testdata/README.md)
// rather than dicts built to match the author's model of the format. The
// three bugs they pin all shipped with every hand-built test green.

func realReq() Request {
	// The fixture lines carry their own timestamps (2026-08/09); an all-time
	// window keeps the tests independent of the day they run on.
	to := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	return Request{To: to, Loc: time.UTC}
}

func copyFixture(t *testing.T, name, dst string) {
	t.Helper()
	src := filepath.Join("testdata", name)
	b, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, b, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestRealShapeClaudeCode(t *testing.T) {
	root := withClaudeRoot(t)
	// The parent transcript and one subagent transcript, as Claude Code
	// lays them out: same project dir, same sessionId, two files.
	copyFixture(t, "claude-parent.jsonl", filepath.Join(root, "-home-goat-picode", "f33c16ba.jsonl"))
	copyFixture(t, "claude-agent-a1.jsonl", filepath.Join(root, "-home-goat-picode", "agent-a1.jsonl"))

	w, err := ClaudeCodeMeter{}.Meter(realReq())
	if err != nil {
		t.Fatal(err)
	}
	tu := w.Stats.Turns

	// The error rides on the *user* turn that returns the tool result. The
	// deployed dashboard read 0 here against 456 on disk.
	if tu.Errors != 1 {
		t.Fatalf("errors = %d, want 1 — tool_result.is_error lives on the user turn", tu.Errors)
	}
	if w.Coverage.Signals[SigErrors] != StateReported {
		t.Fatalf("errors coverage = %v, want reported now that a result was inspected", w.Coverage.Signals[SigErrors])
	}
	// No "aborted" stop exists in this format; the old parser counted
	// stop_sequence as one. A refusal is its own count.
	if tu.Aborted != 0 {
		t.Fatalf("aborted = %d, want 0", tu.Aborted)
	}
	if tu.Refusals != 1 {
		t.Fatalf("refusals = %d, want 1", tu.Refusals)
	}
	// Two files, one session: the subagent transcript folds into its parent.
	if got := w.Stats.Current.Sessions; got != 1 {
		t.Fatalf("sessions = %d, want 1 — agent-*.jsonl carries the parent's sessionId", got)
	}
	if len(w.Stats.TopSessions) != 1 || w.Stats.TopSessions[0].Path != "f33c16ba-4143-4e91-9b78-d097d228bd0c" {
		t.Fatalf("topSessions = %+v, want the session id as identity", w.Stats.TopSessions)
	}
	// And the subagent's turn still counts toward the session's work.
	if tu.Assistant != 3 {
		t.Fatalf("assistant turns = %d, want 3 (parent asst + refusal + subagent asst)", tu.Assistant)
	}
	// Priced: the parent file carries the snapshot, so the one session is
	// priced — not "1 of 2" with the subagent file as an unpriced session.
	if w.Coverage.Signals[SigCost] != StateReported {
		t.Fatalf("cost coverage = %v, note %q", w.Coverage.Signals[SigCost], w.Coverage.Note)
	}
}

func TestRealShapeCodex(t *testing.T) {
	root := withCodexRoot(t)
	copyFixture(t, "codex-rollout.jsonl", filepath.Join(root, "2026", "09", "06", "rollout-x.jsonl"))

	w, err := CodexMeter{}.Meter(realReq())
	if err != nil {
		t.Fatal(err)
	}
	// One prompt: the user.text item. The AGENTS.md injection beside it is
	// also role "user" and counting it reported 620 prompts against 455;
	// the event_msg twin is not counted either, since codex-tui sessions
	// never emit one and relying on it zeroed every interactive prompt.
	if w.Stats.Turns.User != 1 {
		t.Fatalf("user turns = %d, want 1", w.Stats.Turns.User)
	}
	if w.Stats.Turns.Assistant != 1 || w.Stats.Tokens.Input == 0 {
		t.Fatalf("turns = %+v tokens = %+v", w.Stats.Turns, w.Stats.Tokens)
	}
	if len(w.Stats.Tools) != 1 {
		t.Fatalf("tools = %+v, want the one function_call", w.Stats.Tools)
	}
	// Codex has no per-turn error state, so evidence cannot claim one.
	if w.Coverage.Signals[SigErrors] != StateNotReported {
		t.Fatalf("errors coverage = %v", w.Coverage.Signals[SigErrors])
	}
	if w.Coverage.Signals[SigLimits] != StateReported {
		t.Fatalf("limits coverage = %v, want reported: the fixture carries rate_limits", w.Coverage.Signals[SigLimits])
	}
}

// TestCoverageIsEvidenceNotAssertion is the structural fix: a CLI that is
// capable of a signal but showed none of it in an active window reports
// not-reported, never a "reported" beside a counter nothing incremented.
func TestCoverageIsEvidenceNotAssertion(t *testing.T) {
	root := withClaudeRoot(t)
	// An active window with assistant turns but no tool results at all.
	writeTranscript(t, root, "-repo", "s.jsonl", []map[string]any{
		assistant(day(1), "opus", 10, 5, 0, nil),
	})
	w, err := ClaudeCodeMeter{}.Meter(req(7))
	if err != nil {
		t.Fatal(err)
	}
	if got := w.Coverage.Signals[SigErrors]; got != StateNotReported {
		t.Fatalf("errors = %v with no tool result inspected, want not-reported", got)
	}
	if got := w.Coverage.Signals[SigTools]; got != StateNotReported {
		t.Fatalf("tools = %v with no tool_use seen, want not-reported", got)
	}
	if got := w.Coverage.Signals[SigTokens]; got != StateReported {
		t.Fatalf("tokens = %v, want reported: usage was seen", got)
	}
	// A signal the CLI cannot record stays not-reported however busy it is.
	if got := w.Coverage.Signals[SigLimits]; got != StateNotReported {
		t.Fatalf("limits = %v", got)
	}
	// And Hermes, which never populated errors before, no longer claims them
	// on a window with no finish_reason to inspect.
	_ = clisession.HermesTestDB
}
