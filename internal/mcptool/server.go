// Package mcptool is PiCode's own tools spoken over the Model Context
// Protocol (ADR-0154): a stdio JSON-RPC server that a guest CLI (Claude
// Code, Codex, OpenCode, …) launches as a child, exposing the same tools the
// pi packages under packages/ give a pi agent — same names, parameters,
// answers and refusals — by calling the daemon's routes. It is a
// translator: identity comes from the inherited environment, the credential
// is the install token read per call, and every decision (grant, tier,
// audit) stays in the daemon.
package mcptool

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
)

// The protocol revisions this server knows. initialize answers the client's
// own revision when it is one of these, else the newest — a client that
// speaks a newer one still gets a server it can talk to.
var protocolVersions = []string{"2025-06-18", "2025-03-26", "2024-11-05"}

// Tool is one MCP tool: its schema for tools/list and its body for
// tools/call. InputSchema is JSON Schema, hand-mirrored from the pi
// package's typebox parameters and held equal by tests.
type Tool struct {
	Name        string
	Description string
	InputSchema map[string]any
	Call        func(ctx context.Context, args json.RawMessage) Result
}

// Content is one block of a tool result: text, or an image the model sees.
type Content struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	Data     string `json:"data,omitempty"`
	MimeType string `json:"mimeType,omitempty"`
}

// Result is what tools/call answers. A refusal or a failed action is a
// result with IsError, never a JSON-RPC error: the model reads the words.
type Result struct {
	Content []Content `json:"content"`
	IsError bool      `json:"isError,omitempty"`
}

// Text is a one-block text result.
func Text(s string) Result { return Result{Content: []Content{{Type: "text", Text: s}}} }

// Fail is a one-block error result carrying the message verbatim.
func Fail(s string) Result { return Result{Content: []Content{{Type: "text", Text: s}}, IsError: true} }

// Family is one PiCode tool family — one pi package's worth of tools.
type Family struct {
	Name         string
	Instructions string
	Tools        func(c *Caller) []Tool
}

// Families is the catalog, in the order `picode mcp` lists them.
func Families() []Family {
	return []Family{computerFamily, browserFamily, inboxFamily, checklistFamily}
}

// FamilyFor finds a family by name.
func FamilyFor(name string) (Family, bool) {
	for _, f := range Families() {
		if f.Name == strings.ToLower(strings.TrimSpace(name)) {
			return f, true
		}
	}
	return Family{}, false
}

// FamilyNames lists the catalog for help text.
func FamilyNames() []string {
	out := make([]string, 0, len(Families()))
	for _, f := range Families() {
		out = append(out, f.Name)
	}
	return out
}

// Server answers JSON-RPC over one reader and one writer.
type Server struct {
	Name         string
	Version      string
	Instructions string
	tools        []Tool
	mu           sync.Mutex // one writer for the line-oriented output
	out          io.Writer  // set by Serve; notifications go through it
	inflight     sync.Map   // request id (string) → context.CancelFunc
}

// progressKey carries the progress reporter of a tools/call in its context.
type progressKey struct{}

// Progress reports a long call's state to the client as a
// notifications/progress (the client's idle timeout counts silence, not
// wall-clock: Claude Code aborts a call that sends nothing for its idle
// window, never one that keeps reporting). It is a no-op when the client
// sent no progress token.
type Progress func(message string)

// ProgressFrom is the call's reporter, or a no-op.
func ProgressFrom(ctx context.Context) Progress {
	if p, ok := ctx.Value(progressKey{}).(Progress); ok && p != nil {
		return p
	}
	return func(string) {}
}

// WithProgress attaches a reporter (tests use it directly).
func WithProgress(ctx context.Context, p Progress) context.Context {
	return context.WithValue(ctx, progressKey{}, p)
}

// NewServer assembles the tools of the given families for one caller.
func NewServer(name, version string, families []Family, caller *Caller) *Server {
	s := &Server{Name: name, Version: version}
	var notes []string
	for _, f := range families {
		s.tools = append(s.tools, f.Tools(caller)...)
		if f.Instructions != "" {
			notes = append(notes, f.Instructions)
		}
	}
	s.Instructions = strings.Join(notes, "\n\n")
	return s
}

// Tools lists the assembled tools, by name.
func (s *Server) Tools() []Tool {
	out := append([]Tool(nil), s.tools...)
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

const (
	codeParse         = -32700
	codeInvalidReq    = -32600
	codeMethodMissing = -32601
	codeInvalidParams = -32602
)

// Serve reads newline-delimited JSON-RPC messages from r until EOF and
// writes each answer as one line to w. Notifications get no line. Nothing
// but protocol ever goes to w — logs belong on stderr.
func (s *Server) Serve(ctx context.Context, r io.Reader, w io.Writer) error {
	s.mu.Lock()
	s.out = w
	s.mu.Unlock()
	in := bufio.NewReaderSize(r, 1<<20)
	var calls sync.WaitGroup
	defer calls.Wait()
	for {
		line, err := in.ReadBytes('\n')
		if len(strings.TrimSpace(string(line))) > 0 {
			if id, ok := toolCallID(line); ok {
				// A tool call may wait a long time (ask_human): it runs on
				// its own goroutine so ping, list and a cancellation still
				// get through on the line. Its cancel is registered before
				// the goroutine starts, so a cancellation that arrives at
				// once still finds it.
				msg := append([]byte(nil), line...)
				callCtx, cancel := context.WithCancel(ctx)
				s.inflight.Store(id, cancel)
				calls.Add(1)
				go func() {
					defer calls.Done()
					defer s.inflight.Delete(id)
					defer cancel()
					_ = s.write(s.Handle(callCtx, msg))
				}()
			} else if werr := s.write(s.Handle(ctx, line)); werr != nil {
				return werr
			}
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}
}

// write sends one line; nil means nothing to send.
func (s *Server) write(out []byte) error {
	if out == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.out == nil {
		return nil
	}
	_, err := s.out.Write(append(out, '\n'))
	return err
}

// notify sends a server→client notification.
func (s *Server) notify(method string, params any) {
	_ = s.write(encode(map[string]any{"jsonrpc": "2.0", "method": method, "params": params}))
}

// toolCallID is the request id of a tools/call line, as the cancellation
// map keys it (the raw JSON of the id, so 7 and "7" stay distinct).
func toolCallID(line []byte) (string, bool) {
	var probe struct {
		Method string          `json:"method"`
		ID     json.RawMessage `json:"id"`
	}
	if json.Unmarshal(line, &probe) != nil || probe.Method != "tools/call" || len(probe.ID) == 0 {
		return "", false
	}
	return string(probe.ID), true
}

// Handle answers one message; nil means "nothing to send" (a notification,
// or a response the client sent us). Exposed for tests and for a transport
// other than stdio.
func (s *Server) Handle(ctx context.Context, msg []byte) []byte {
	var req request
	if err := json.Unmarshal(msg, &req); err != nil {
		return encode(response{JSONRPC: "2.0", ID: json.RawMessage("null"), Error: &rpcError{codeParse, "parse error: " + err.Error()}})
	}
	if req.Method == "" {
		return nil // a response or an ack from the client
	}
	if len(req.ID) == 0 || string(req.ID) == "null" {
		if req.Method == "notifications/cancelled" {
			var p struct {
				RequestID json.RawMessage `json:"requestId"`
			}
			if json.Unmarshal(req.Params, &p) == nil {
				if cancel, ok := s.inflight.Load(string(p.RequestID)); ok {
					cancel.(context.CancelFunc)()
				}
			}
		}
		return nil // notifications (initialized, cancelled, …) need no answer
	}
	res := response{JSONRPC: "2.0", ID: req.ID}
	switch req.Method {
	case "initialize":
		res.Result = s.initialize(req.Params)
	case "ping":
		res.Result = map[string]any{}
	case "tools/list":
		res.Result = s.list()
	case "tools/call":
		result, rerr := s.call(ctx, req.Params)
		if rerr != nil {
			res.Error = rerr
		} else {
			res.Result = result
		}
	default:
		res.Error = &rpcError{codeMethodMissing, "method not found: " + req.Method}
	}
	return encode(res)
}

func encode(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		// A result we cannot encode is a bug in this file, not the client's.
		b, _ = json.Marshal(response{JSONRPC: "2.0", ID: json.RawMessage("null"), Error: &rpcError{codeInvalidReq, "unencodable response: " + err.Error()}})
	}
	return b
}

func (s *Server) initialize(params json.RawMessage) map[string]any {
	var p struct {
		ProtocolVersion string `json:"protocolVersion"`
	}
	_ = json.Unmarshal(params, &p)
	version := protocolVersions[0]
	for _, v := range protocolVersions {
		if v == p.ProtocolVersion {
			version = v
		}
	}
	out := map[string]any{
		"protocolVersion": version,
		"capabilities":    map[string]any{"tools": map[string]any{"listChanged": false}},
		"serverInfo":      map[string]any{"name": s.Name, "version": s.Version},
	}
	if s.Instructions != "" {
		out["instructions"] = s.Instructions
	}
	return out
}

func (s *Server) list() map[string]any {
	tools := make([]map[string]any, 0, len(s.tools))
	for _, t := range s.Tools() {
		tools = append(tools, map[string]any{"name": t.Name, "description": t.Description, "inputSchema": t.InputSchema})
	}
	return map[string]any{"tools": tools}
}

func (s *Server) call(ctx context.Context, params json.RawMessage) (Result, *rpcError) {
	var p struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
		Meta      struct {
			ProgressToken json.RawMessage `json:"progressToken"`
		} `json:"_meta"`
	}
	if err := json.Unmarshal(params, &p); err != nil || strings.TrimSpace(p.Name) == "" {
		return Result{}, &rpcError{codeInvalidParams, "tools/call needs a tool name"}
	}
	if len(p.Meta.ProgressToken) > 0 {
		token := p.Meta.ProgressToken
		var n float64
		ctx = WithProgress(ctx, func(message string) {
			n++
			s.notify("notifications/progress", map[string]any{"progressToken": token, "progress": n, "message": message})
		})
	}
	for _, t := range s.tools {
		if t.Name == p.Name {
			args := p.Arguments
			if len(args) == 0 {
				args = json.RawMessage("{}")
			}
			return t.Call(ctx, args), nil
		}
	}
	return Result{}, &rpcError{codeInvalidParams, fmt.Sprintf("unknown tool %q", p.Name)}
}
