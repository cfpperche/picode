// Package rpc implements a client for pi's RPC mode (JSONL over stdio)
// and the managed-agent runtime built on it (ADR-0006).
//
// Protocol (docs: pi rpc.md): commands in on stdin, one JSON object per
// line; responses and events out on stdout, one JSON object per line.
// Responses carry `type:"response"` + command + success; events carry
// their own `type` (`message_update`, `tool_execution_start`, ...).
// Framing is strict `\n` — bufio.Scanner's default line split is compliant
// (splits on \n only, tolerates a trailing \r).
package rpc

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
)

// Response is the reply to a command.
type Response struct {
	ID      string          `json:"id,omitempty"`
	Type    string          `json:"type"`
	Command string          `json:"command"`
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data,omitempty"`
	Error   string          `json:"error,omitempty"`
}

// Event is any non-response stdout line, forwarded verbatim.
type Event json.RawMessage

// EventType extracts the event's type field ("" when absent).
func (e Event) EventType() string {
	var head struct {
		Type string `json:"type"`
	}
	_ = json.Unmarshal(e, &head)
	return head.Type
}

// Client is a live `pi --mode rpc` subprocess.
type Client struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	done   chan struct{}

	writeMu sync.Mutex // serializes stdin writes

	mu        sync.Mutex
	pending   map[string]chan Response
	listeners map[int]func(Event)
	nextSub   int
	exitErr   error
}

// Start launches the rpc process (command run with args, cwd) and begins
// pumping stdout until the process exits or Close is called.
func Start(command string, args []string, cwd string) (*Client, error) {
	cmd := exec.Command(command, args...)
	cmd.Dir = cwd
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("rpc: stdin: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("rpc: stdout: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("rpc: start %s: %w", command, err)
	}
	c := &Client{
		cmd:       cmd,
		stdin:     stdin,
		stdout:    stdout,
		done:      make(chan struct{}),
		pending:   map[string]chan Response{},
		listeners: map[int]func(Event){},
	}
	go c.pump()
	return c, nil
}

// pump reads stdout lines forever, dispatching responses to waiters and
// events to listeners. Runs until EOF/process exit.
func (c *Client) pump() {
	defer close(c.done)
	scanner := bufio.NewScanner(c.stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024) // large events (messages) allowed
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var head struct {
			Type string `json:"type"`
			ID   string `json:"id"`
		}
		if err := json.Unmarshal(line, &head); err != nil {
			continue // not JSON — ignore
		}
		if head.Type == "response" {
			c.mu.Lock()
			ch := c.pending[head.ID]
			delete(c.pending, head.ID)
			c.mu.Unlock()
			if ch == nil {
				continue
			}
			var res Response
			if err := json.Unmarshal(line, &res); err == nil {
				ch <- res
			} else {
				ch <- Response{ID: head.ID, Type: "response", Success: false, Error: "unparseable response"}
			}
			continue
		}
		// scanner.Bytes() is reused across scans — copy before handing out.
		ev := Event(append([]byte(nil), line...))
		c.mu.Lock()
		subs := make([]func(Event), 0, len(c.listeners))
		for _, fn := range c.listeners {
			subs = append(subs, fn)
		}
		c.mu.Unlock()
		for _, fn := range subs {
			fn(ev)
		}
	}
	_ = c.cmd.Wait()

	// Fail any still-pending waiters so Send callers don't hang.
	c.mu.Lock()
	c.exitErr = fmt.Errorf("rpc: process exited")
	for id, ch := range c.pending {
		ch <- Response{ID: id, Type: "response", Success: false, Error: "rpc process exited"}
		delete(c.pending, id)
	}
	c.mu.Unlock()
}

var idCounter atomic.Int64

// newID builds a unique correlation id for commands.
func newID() string {
	return fmt.Sprintf("picode-%d-%d", os.Getpid(), idCounter.Add(1))
}

// Command is a typed rpc command builder.
type Command struct {
	Type string
	Body map[string]any
}

// Send writes a command and waits for its response.
func (c *Client) Send(ctx context.Context, cmd Command) (Response, error) {
	id := newID()
	body := map[string]any{"id": id, "type": cmd.Type}
	for k, v := range cmd.Body {
		body[k] = v
	}
	line, err := json.Marshal(body)
	if err != nil {
		return Response{}, fmt.Errorf("rpc: encode command: %w", err)
	}

	ch := make(chan Response, 1)
	c.mu.Lock()
	c.pending[id] = ch
	c.mu.Unlock()

	c.writeMu.Lock()
	_, werr := c.stdin.Write(append(line, '\n'))
	c.writeMu.Unlock()
	if werr != nil {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return Response{}, fmt.Errorf("rpc: write command: %w", werr)
	}

	select {
	case res := <-ch:
		if !res.Success {
			return res, fmt.Errorf("rpc: %s failed: %s", cmd.Type, res.Error)
		}
		return res, nil
	case <-ctx.Done():
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return Response{}, ctx.Err()
	case <-c.done:
		c.mu.Lock()
		err := c.exitErr
		c.mu.Unlock()
		if err == nil {
			err = fmt.Errorf("rpc: process exited")
		}
		return Response{}, err
	}
}

// Subscribe registers an event listener; the returned func unsubscribes.
func (c *Client) Subscribe(fn func(Event)) func() {
	c.mu.Lock()
	id := c.nextSub
	c.nextSub++
	c.listeners[id] = fn
	c.mu.Unlock()
	return func() {
		c.mu.Lock()
		delete(c.listeners, id)
		c.mu.Unlock()
	}
}

// Done is closed when the underlying process exits.
func (c *Client) Done() <-chan struct{} { return c.done }

// Close kills the process and fails pending Send callers.
func (c *Client) Close() {
	_ = c.cmd.Process.Kill()
	<-c.done
}
