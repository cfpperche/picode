package rpc

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

// The fake re-executes the test binary as a minimal pi-rpc double
// (cross-platform; runs on windows CI too). Enabled by PICODE_FAKE_RPC=1
// in init() below.

func init() {
	if os.Getenv("PICODE_FAKE_RPC_HOLDER") == "1" {
		// The grandchild in the Close regression test: holds the inherited
		// stdout pipe open and ignores the world, like the real pi behind
		// the intercept wrapper does.
		time.Sleep(60 * time.Second)
		os.Exit(0)
	}
	if os.Getenv("PICODE_FAKE_RPC_WRAPPER") == "1" {
		// The intercept wrapper double: spawns a grandchild that inherits
		// the client's stdout pipe, then serves the protocol itself — the
		// exact shape ~/.picode/bin/pi gives managed runs.
		fakeWrapperMain()
	}
	if os.Getenv("PICODE_FAKE_RPC") == "1" {
		fakeMain()
		os.Exit(0)
	}
}

// fakeWrapperMain spawns the holder grandchild (inheriting stdout), records
// its pid for the test, and then behaves as the plain rpc double.
func fakeWrapperMain() {
	holder := exec.Command(os.Args[0])
	holder.Env = append(os.Environ(), "PICODE_FAKE_RPC_HOLDER=1")
	holder.Stdout = os.Stdout // the whole point: hold the client's stdout pipe
	holder.Stdin = os.Stdin
	if err := holder.Start(); err == nil {
		if path := os.Getenv("PICODE_HOLDER_PID_FILE"); path != "" {
			_ = os.WriteFile(path, []byte(strconv.Itoa(holder.Process.Pid)), 0o600)
		}
	}
	fakeMain()
	os.Exit(0)
}

// fakeMain speaks a minimal subset of pi's rpc protocol on stdio.
func fakeMain() {
	dec := json.NewDecoder(os.Stdin)
	enc := json.NewEncoder(os.Stdout)
	for {
		var req map[string]any
		if err := dec.Decode(&req); err != nil {
			return // stdin closed
		}
		id, _ := req["id"].(string)
		typ, _ := req["type"].(string)

		switch typ {
		case "get_state":
			_ = enc.Encode(map[string]any{
				"id": id, "type": "response", "command": "get_state", "success": true,
				"data": map[string]any{
					"model":               map[string]any{"id": "fake-model", "displayName": "Fake Model"},
					"isStreaming":         false,
					"isCompacting":        false,
					"pendingMessageCount": 0,
					"sessionFile":         os.Getenv("PICODE_FAKE_SESSION"),
				},
			})
		case "prompt", "steer", "follow_up":
			msg, _ := req["message"].(string)
			if typ == "prompt" && strings.HasPrefix(msg, "ASK:") {
				fakeAsk(enc, dec, id, typ, msg)
				break
			}
			if typ == "prompt" && len(msg) >= 10 && msg[:10] == "/mcp-auth " {
				if msg == "/mcp-auth already" {
					_ = enc.Encode(map[string]any{
						"id": id, "type": "response", "command": typ, "success": true,
					})
					break
				}
				_ = enc.Encode(map[string]any{
					"type": "extension_ui_request", "id": "ui-auth",
					"method": "input", "title": "Complete OAuth\nhttps://example.test/oauth\nPaste the URL",
				})
				var reply map[string]any
				if err := dec.Decode(&reply); err != nil {
					return
				}
				_ = enc.Encode(map[string]any{
					"id": id, "type": "response", "command": typ, "success": true,
				})
				break
			}
			_ = enc.Encode(map[string]any{"type": "agent_start"})
			_ = enc.Encode(map[string]any{"type": "message_update", "assistantMessageEvent": map[string]any{
				"type": "text_delta", "contentIndex": 0, "delta": "hello from fake",
			}})
			_ = enc.Encode(map[string]any{"type": "agent_end", "messages": []map[string]any{
				{"role": "user", "content": []map[string]any{{"type": "text", "text": msg}}},
				{"role": "assistant", "content": []map[string]any{{"type": "text", "text": "hello from fake"}}},
			}})
			_ = enc.Encode(map[string]any{"type": "agent_settled"})
			_ = enc.Encode(map[string]any{
				"id": id, "type": "response", "command": typ, "success": true,
			})
		case "fail_me":
			_ = enc.Encode(map[string]any{
				"id": id, "type": "response", "command": typ, "success": false, "error": "as requested",
			})
		case "bash":
			cmd, _ := req["command"].(string)
			_ = enc.Encode(map[string]any{"type": "bash_execution_update", "id": id, "delta": "fake: " + cmd + "\n"})
			_ = enc.Encode(map[string]any{
				"id": id, "type": "response", "command": "bash", "success": true,
				"data": map[string]any{"output": "fake: " + cmd + "\n", "exitCode": 0, "cancelled": false, "truncated": false},
			})
		case "abort":
			_ = enc.Encode(map[string]any{
				"id": id, "type": "response", "command": "abort", "success": true,
			})
		case "abort_bash":
			_ = enc.Encode(map[string]any{
				"id": id, "type": "response", "command": "abort_bash", "success": true,
			})
		case "die":
			_ = enc.Encode(map[string]any{
				"id": id, "type": "response", "command": typ, "success": true,
			})
			os.Exit(0)
		default:
			_ = enc.Encode(map[string]any{
				"id": id, "type": "response", "command": typ, "success": false, "error": "unsupported",
			})
		}
	}
}

func fakeAsk(enc *json.Encoder, dec *json.Decoder, id, typ, msg string) {
	switch msg {
	case "ASK:notify":
		_ = enc.Encode(map[string]any{
			"type": "extension_ui_request", "id": "ui-note", "method": "notify",
			"message": "Heads up", "notifyType": "info",
		})
		_ = enc.Encode(map[string]any{
			"id": id, "type": "response", "command": typ, "success": true,
		})
		return
	case "ASK:timeout":
		_ = enc.Encode(map[string]any{
			"type": "extension_ui_request", "id": "ui-to", "method": "confirm",
			"title": "Allow this?", "message": "Times out", "timeout": 40,
		})
		// Still block so the process stays up; the server timer clears waiting.
	case "ASK:select":
		_ = enc.Encode(map[string]any{
			"type": "extension_ui_request", "id": "ui-sel", "method": "select",
			"title": "Pick one", "options": []string{"Allow", "Block"},
		})
	default: // ASK:confirm
		_ = enc.Encode(map[string]any{
			"type": "extension_ui_request", "id": "ui-ask", "method": "confirm",
			"title": "Allow this?", "message": "The agent needs a yes or no.",
		})
	}
	for {
		var reply map[string]any
		if err := dec.Decode(&reply); err != nil {
			return
		}
		rtype, _ := reply["type"].(string)
		if rtype == "extension_ui_response" {
			_ = enc.Encode(map[string]any{
				"id": id, "type": "response", "command": typ, "success": true,
			})
			return
		}
		rid, _ := reply["id"].(string)
		_ = enc.Encode(map[string]any{
			"id": rid, "type": "response", "command": rtype, "success": true,
		})
	}
}

func startClient(t *testing.T) *Client {
	t.Helper()
	t.Setenv("PICODE_FAKE_RPC", "1")
	c, err := Start(os.Args[0], []string{"--mode", "rpc"}, t.TempDir())
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(c.Close)
	return c
}

func TestMCPAuthUI(t *testing.T) {
	c := startClient(t)
	got := make(chan Event, 1)
	unsub := c.Subscribe(func(e Event) {
		if e.EventType() == "extension_ui_request" {
			got <- e
		}
	})
	defer unsub()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	errCh := make(chan error, 1)
	go func() {
		_, err := c.Send(ctx, Command{Type: "prompt", Body: map[string]any{"message": "/mcp-auth docs"}})
		errCh <- err
	}()
	select {
	case ev := <-got:
		var body map[string]any
		if err := json.Unmarshal([]byte(ev), &body); err != nil {
			t.Fatal(err)
		}
		if body["method"] != "input" {
			t.Fatalf("%v", body)
		}
		id, _ := body["id"].(string)
		if err := c.SendRaw(map[string]any{"type": "extension_ui_response", "id": id, "value": "http://127.0.0.1/cb"}); err != nil {
			t.Fatal(err)
		}
	case <-ctx.Done():
		t.Fatal("no ui request")
	}
	if err := <-errCh; err != nil {
		t.Fatal(err)
	}
}

func TestSendGetState(t *testing.T) {
	c := startClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := c.Send(ctx, Command{Type: "get_state"})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	var data struct {
		Model struct {
			ID string `json:"id"`
		} `json:"model"`
	}
	if err := json.Unmarshal(res.Data, &data); err != nil {
		t.Fatalf("decode data: %v (raw %s)", err, res.Data)
	}
	if data.Model.ID != "fake-model" {
		t.Errorf("model id = %q", data.Model.ID)
	}
}

func TestEventsFanOut(t *testing.T) {
	c := startClient(t)
	got := make(chan string, 8)
	unsub := c.Subscribe(func(e Event) {
		got <- e.EventType()
	})
	defer unsub()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := c.Send(ctx, Command{Type: "prompt", Body: map[string]any{"message": "hi"}}); err != nil {
		t.Fatalf("Send prompt: %v", err)
	}

	seen := map[string]bool{}
	for i := 0; i < 4; i++ { // start, update, end, settled
		select {
		case ev := <-got:
			seen[ev] = true
		case <-time.After(3 * time.Second):
			t.Fatalf("event %d not received; seen=%v", i, seen)
		}
	}
	if !seen["agent_start"] || !seen["message_update"] || !seen["agent_end"] || !seen["agent_settled"] {
		t.Errorf("events seen = %v", seen)
	}
}

func TestErrorResponse(t *testing.T) {
	c := startClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := c.Send(ctx, Command{Type: "fail_me"})
	if err == nil {
		t.Fatal("expected error response")
	}
}

func TestProcessExitFailsPendingAndDone(t *testing.T) {
	c := startClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := c.Send(ctx, Command{Type: "die"}); err != nil {
		t.Fatalf("die: %v", err)
	}
	select {
	case <-c.Done():
	case <-time.After(3 * time.Second):
		t.Fatal("Done not closed after process exit")
	}

	// Sends after exit fail, not hang.
	sendCtx, sendCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer sendCancel()
	if _, err := c.Send(sendCtx, Command{Type: "get_state"}); err == nil {
		t.Fatal("Send after exit should fail")
	}
}

// closeWithin fails the test if Close does not return before timeout — the
// shape of the 2026-09-05 pi-diff hang, where Stop never came back.
func closeWithin(t *testing.T, c *Client, timeout time.Duration) {
	t.Helper()
	done := make(chan struct{})
	go func() { c.Close(); close(done) }()
	select {
	case <-done:
	case <-time.After(timeout):
		t.Fatal("Close hung: the process tree survived the kill and is holding the pipe")
	}
}

// TestCloseKillsGrandchildHoldingStdout pins the stop-agent hang: a real
// deployment starts the real pi BEHIND a shell intercept wrapper, and the
// wrapper's child inherits the client's stdout pipe. Killing only the
// direct child leaves that grandchild alive, pump never sees EOF, and
// Close — reached from Runtime.Stop by both "Stop agent" and "Open
// terminal" — waits forever. The kill must reach the whole group, and the
// pipe must not be the only thing Close waits on.
//
//	Conditions                    | Action expected
//	------------------------------|----------------------------------
//	wrapper + grandchild on pipe  | both die, Close returns
//	process already self-exited   | Close returns immediately
//	second Close (Stop races All) | returns immediately, no panic
func TestCloseKillsGrandchildHoldingStdout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("process-group kill and grandchild liveness are unix-only")
	}
	dir := t.TempDir()
	pidFile := dir + "/holder.pid"
	c, err := Start(os.Args[0], []string{"--mode", "rpc"}, dir,
		"PICODE_FAKE_RPC=1", "PICODE_FAKE_RPC_WRAPPER=1", "PICODE_HOLDER_PID_FILE="+pidFile)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	var holderPID int
	for i := 0; i < 50 && holderPID == 0; i++ { // wait for the grandchild to exist
		if b, readErr := os.ReadFile(pidFile); readErr == nil {
			holderPID, _ = strconv.Atoi(strings.TrimSpace(string(b)))
		}
		time.Sleep(20 * time.Millisecond)
	}
	if holderPID == 0 {
		c.Close()
		t.Fatal("wrapper double never reported the grandchild pid")
	}
	closeWithin(t, c, 5*time.Second)
	deadline := time.Now().Add(3 * time.Second)
	for {
		if !processAlive(holderPID) {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("grandchild %d survived Close: it still holds the stdout pipe", holderPID)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func TestCloseAfterProcessExited(t *testing.T) {
	c := startClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := c.Send(ctx, Command{Type: "die"}); err != nil {
		t.Fatalf("die: %v", err)
	}
	select {
	case <-c.Done():
	case <-time.After(3 * time.Second):
		t.Fatal("Done not closed after process exit")
	}
	closeWithin(t, c, 3*time.Second) // group kill on a dead group must be a no-op
}

func TestCloseIdempotent(t *testing.T) {
	c := startClient(t)
	closeWithin(t, c, 5*time.Second)
	finished := make(chan struct{})
	go func() { c.Close(); close(finished) }() // Stop racing StopAll: second Close
	select {
	case <-finished:
	case <-time.After(3 * time.Second):
		t.Fatal("second Close blocked")
	}
}
