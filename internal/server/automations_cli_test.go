package server

import (
	"net/http"
	"testing"
	"time"
)

// ADR-0217: the door is retried while a fresh TUI grows its composer, never
// for a refusal it will not outgrow (a draft in the composer, an error).
func TestDoorRetryable(t *testing.T) {
	for _, c := range []struct {
		status int
		reason string
		want   bool
	}{
		{http.StatusConflict, "closed", true},
		{http.StatusConflict, "unrecognized", true},
		{http.StatusConflict, "working", true},
		{http.StatusConflict, "occupied", false},
		{http.StatusConflict, "needs-you", false},
		{http.StatusServiceUnavailable, "", false},
		{http.StatusOK, "", false},
	} {
		if got := doorRetryable(c.status, c.reason); got != c.want {
			t.Errorf("%d %q = %v", c.status, c.reason, got)
		}
	}
}

// The end of a turn is an idle state set after the prompt went in; a state
// from before counts for nothing; each new needs-you tells the person once.
func TestTurnWatch(t *testing.T) {
	at := time.Now()
	tw := &turnWatch{delivered: at}
	steps := []struct {
		st   TermState
		ok   bool
		want turnStep
	}{
		{TermState{State: TermIdle, At: at.Add(-time.Second)}, true, turnGoing}, // the idle before the prompt
		{TermState{}, false, turnGoing},
		{TermState{State: TermWorking, At: at.Add(time.Second)}, true, turnGoing},
		{TermState{State: TermNeedsYou, At: at.Add(2 * time.Second)}, true, turnNeedsYou},
		{TermState{State: TermNeedsYou, At: at.Add(2 * time.Second)}, true, turnGoing}, // told already
		{TermState{State: TermWorking, At: at.Add(3 * time.Second)}, true, turnGoing},
		{TermState{State: TermNeedsYou, At: at.Add(4 * time.Second)}, true, turnNeedsYou}, // a new question
		{TermState{State: TermIdle, At: at.Add(5 * time.Second)}, true, turnEnded},
	}
	for i, s := range steps {
		if got := tw.step(s.st, s.ok); got != s.want {
			t.Fatalf("step %d: %v, want %v", i, got, s.want)
		}
	}
}

// An unconfirmed receipt counts only when the CLI's hook reports a state
// after the prompt went in; an older state proves nothing.
func TestHookSawPrompt(t *testing.T) {
	ts := NewTermStates()
	deps := Deps{TermStates: ts}
	sent := time.Now()
	ts.Set("t", TermIdle, "claude-code", sent.Add(-time.Second))
	if hookSawPrompt(deps, "t", sent, 300*time.Millisecond) {
		t.Fatal("a state from before the prompt confirmed it")
	}
	ts.Set("t", TermWorking, "claude-code", sent.Add(time.Second))
	if !hookSawPrompt(deps, "t", sent, 300*time.Millisecond) {
		t.Fatal("the hook's working state did not confirm the prompt")
	}
}
