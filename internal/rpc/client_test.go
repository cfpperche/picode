package rpc

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"
)

// The fake re-executes the test binary as a minimal pi-rpc double
// (cross-platform; runs on windows CI too). Enabled by PICODE_FAKE_RPC=1
// in init() below.

func init() {
	if os.Getenv("PICODE_FAKE_RPC") == "1" {
		fakeMain()
		os.Exit(0)
	}
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
					"model":       map[string]any{"id": "fake-model", "displayName": "Fake Model"},
					"isStreaming": false,
				},
			})
		case "prompt", "steer", "follow_up":
			msg, _ := req["message"].(string)
			if typ == "prompt" && len(msg) >= 10 && msg[:10] == "/mcp-auth " {
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
	for i := 0; i < 3; i++ {
		select {
		case ev := <-got:
			seen[ev] = true
		case <-time.After(3 * time.Second):
			t.Fatalf("event %d not received; seen=%v", i, seen)
		}
	}
	if !seen["agent_start"] || !seen["message_update"] || !seen["agent_settled"] {
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
