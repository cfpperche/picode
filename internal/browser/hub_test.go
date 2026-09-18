package browser

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

// readCommand takes the next command off an attached stream.
func readCommand(t *testing.T, events <-chan Command) Command {
	t.Helper()
	select {
	case cmd := <-events:
		return cmd
	case <-time.After(2 * time.Second):
		t.Fatal("no command arrived on the stream")
		return Command{}
	}
}

func TestDispatchWithoutShellFailsFast(t *testing.T) {
	h := New()
	if h.Connected() {
		t.Fatal("a fresh hub reports a shell")
	}
	if _, err := h.Dispatch(context.Background(), Command{Method: "Page.captureScreenshot", Tier: "read"}); !errors.Is(err, ErrNoShell) {
		t.Fatalf("err = %v, want ErrNoShell", err)
	}
}

func TestDispatchRoundTrip(t *testing.T) {
	h := New()
	events, detach := h.Attach()
	defer detach()
	if !h.Connected() {
		t.Fatal("an attached hub reports no shell")
	}

	type answer struct {
		out json.RawMessage
		err error
	}
	done := make(chan answer, 1)
	go func() {
		out, err := h.Dispatch(context.Background(), Command{
			Method:  "Network.getResponseBody",
			Params:  json.RawMessage(`{"requestId":"1"}`),
			Tier:    "read",
			Domains: []string{"example.com"},
		})
		done <- answer{out, err}
	}()

	cmd := readCommand(t, events)
	if cmd.ID == "" {
		t.Fatal("command has no id")
	}
	if cmd.Method != "Network.getResponseBody" || cmd.Tier != "read" || string(cmd.Params) != `{"requestId":"1"}` {
		t.Fatalf("command = %+v", cmd)
	}
	if len(cmd.Domains) != 1 || cmd.Domains[0] != "example.com" {
		t.Fatalf("domains = %v", cmd.Domains)
	}
	if err := h.Complete(Result{ID: cmd.ID, Output: json.RawMessage(`{"body":"hi"}`)}); err != nil {
		t.Fatalf("complete: %v", err)
	}
	got := <-done
	if got.err != nil || string(got.out) != `{"body":"hi"}` {
		t.Fatalf("dispatch = %q, %v", got.out, got.err)
	}
}

func TestNewestStreamIsTheActiveOne(t *testing.T) {
	h := New()
	old, detachOld := h.Attach()
	defer detachOld()
	newer, detachNew := h.Attach()
	defer detachNew()

	go func() {
		_, _ = h.Dispatch(context.Background(), Command{Method: "Page.captureScreenshot", Tier: "read"})
	}()

	cmd := readCommand(t, newer)
	if err := h.Complete(Result{ID: cmd.ID}); err != nil {
		t.Fatal(err)
	}
	select {
	case c := <-old:
		t.Fatalf("the abandoned stream still received %+v", c)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestDetachLeavesTheLineEmpty(t *testing.T) {
	h := New()
	_, detach := h.Attach()
	detach()
	if h.Connected() {
		t.Fatal("a detached hub reports a shell")
	}
	if _, err := h.Dispatch(context.Background(), Command{Method: "Page.captureScreenshot"}); !errors.Is(err, ErrNoShell) {
		t.Fatalf("err = %v, want ErrNoShell", err)
	}
	// Taking the line back works (a reconnecting shell).
	events, detachAgain := h.Attach()
	defer detachAgain()
	go func() { _, _ = h.Dispatch(context.Background(), Command{Method: "Page.captureScreenshot"}) }()
	if cmd := readCommand(t, events); cmd.Method != "Page.captureScreenshot" {
		t.Fatalf("cmd = %+v", cmd)
	}
}

func TestShellErrorReachesTheCaller(t *testing.T) {
	h := New()
	events, detach := h.Attach()
	defer detach()
	done := make(chan error, 1)
	go func() {
		_, err := h.Dispatch(context.Background(), Command{Method: "Runtime.evaluate", Tier: "read"})
		done <- err
	}()
	cmd := readCommand(t, events)
	if err := h.Complete(Result{ID: cmd.ID, Error: "Runtime.evaluate needs the act tier; this agent has read"}); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err == nil || err.Error() != "Runtime.evaluate needs the act tier; this agent has read" {
		t.Fatalf("err = %v", err)
	}
}

func TestTimeoutForgetsTheCommand(t *testing.T) {
	h := New()
	h.Timeout = 20 * time.Millisecond
	events, detach := h.Attach()
	defer detach()
	_, err := h.Dispatch(context.Background(), Command{Method: "Page.captureScreenshot", Tier: "read"})
	if err == nil || !strings.Contains(err.Error(), "did not answer") {
		t.Fatalf("err = %v, want a timeout", err)
	}
	// The late answer has nowhere to go and says so instead of vanishing.
	cmd := readCommand(t, events)
	if err := h.Complete(Result{ID: cmd.ID}); !errors.Is(err, ErrUnknownResult) {
		t.Fatalf("late complete = %v, want ErrUnknownResult", err)
	}
}

func TestCanceledContextStopsWaiting(t *testing.T) {
	h := New()
	h.Timeout = 5 * time.Second
	events, detach := h.Attach()
	defer detach()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := h.Dispatch(ctx, Command{Method: "Page.captureScreenshot"})
		done <- err
	}()
	readCommand(t, events)
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}

func TestCompleteUnknownResult(t *testing.T) {
	h := New()
	if err := h.Complete(Result{ID: "nope"}); !errors.Is(err, ErrUnknownResult) {
		t.Fatalf("err = %v, want ErrUnknownResult", err)
	}
}

func TestBusyStreamRefusesMore(t *testing.T) {
	h := New()
	// Every dispatch times out with nobody reading, so the commands pile up on
	// the stream until its backlog is full.
	h.Timeout = 1 * time.Millisecond
	_, detach := h.Attach()
	defer detach()
	var lastErr error
	for i := 0; i <= backlog; i++ {
		_, lastErr = h.Dispatch(context.Background(), Command{Method: "Page.captureScreenshot"})
	}
	if !errors.Is(lastErr, ErrBusy) {
		t.Fatalf("err = %v, want ErrBusy after %d undrained commands", lastErr, backlog)
	}
}

func TestCommandTimeoutOverridesTheHub(t *testing.T) {
	h := New()
	h.Timeout = 5 * time.Second
	_, detach := h.Attach() // a shell that never answers
	defer detach()
	start := time.Now()
	_, err := h.Dispatch(context.Background(), Command{Kind: "computer", Method: "screenshot", Timeout: 20 * time.Millisecond})
	if err == nil || !strings.Contains(err.Error(), "did not answer") {
		t.Fatalf("err = %v, want a timeout", err)
	}
	if time.Since(start) > time.Second {
		t.Fatal("the per-command timeout was not honoured")
	}
}

func TestKindAndPrincipalTravelOnTheWire(t *testing.T) {
	body, err := json.Marshal(Command{ID: "1", Kind: "computer", Method: "screenshot", Principal: "agent-1", Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"kind":"computer"`, `"principal":"agent-1"`} {
		if !strings.Contains(string(body), want) {
			t.Errorf("frame %s lacks %s", body, want)
		}
	}
	if strings.Contains(strings.ToLower(string(body)), "timeout") {
		t.Errorf("the timeout travelled: %s", body)
	}
	body, _ = json.Marshal(Command{ID: "2", Method: "Page.captureScreenshot", Tier: "read"})
	if strings.Contains(string(body), "kind") || strings.Contains(string(body), "principal") {
		t.Errorf("a browser frame grew fields it does not carry: %s", body)
	}
}
