package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/cfpperche/picode/internal/mcp"
	"github.com/cfpperche/picode/internal/store"
)

// Hub fans out events to subscribers (one per managed agent).
type Hub struct {
	mu   sync.Mutex
	subs map[int]chan []byte
	next int
}

func NewHub() *Hub { return &Hub{subs: map[int]chan []byte{}} }

// Subscribe returns a buffered channel of JSON messages and an unsubscribe.
func (h *Hub) Subscribe() (<-chan []byte, func()) {
	ch := make(chan []byte, 256)
	h.mu.Lock()
	id := h.next
	h.next++
	h.subs[id] = ch
	h.mu.Unlock()
	return ch, func() {
		h.mu.Lock()
		delete(h.subs, id)
		h.mu.Unlock()
	}
}

// Broadcast delivers to every subscriber, dropping for slow consumers.
func (h *Hub) Broadcast(msg []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, ch := range h.subs {
		select {
		case ch <- msg:
		default: // slow consumer: drop rather than block the agent
		}
	}
}

// ManagedAgent is one live `pi --mode rpc` process plus its delivery loop.
type ManagedAgent struct {
	AgentID string
	Path    string

	client *Client
	hub    *Hub
	store  *store.Store

	cancel context.CancelFunc
	done   chan struct{}

	mu        sync.Mutex
	streaming bool
	lastErr   error
	settledCh chan struct{} // closed+replaced when agent_settled arrives
	waiting   *UIDialog     // blocking extension_ui_request, if any
}

// Runtime owns all managed agents.
type Runtime struct {
	AgentCmd string // "pi" (ADR-0003)
	DataDir  string // ~/.picode — MCP live snapshots when set

	mu     sync.Mutex
	agents map[string]*ManagedAgent
	store  *store.Store
	onExit func(agentID string)

	authMu   sync.Mutex
	authJobs map[string]*mcpAuthJob
}

// NewRuntime builds a runtime. onExit (optional) fires when a managed
// agent's process dies on its own.
func NewRuntime(agentCmd string, st *store.Store, onExit func(string)) *Runtime {
	return &Runtime{
		AgentCmd: agentCmd,
		agents:   map[string]*ManagedAgent{},
		store:    st,
		onExit:   onExit,
		authJobs: map[string]*mcpAuthJob{},
	}
}

// Start launches the managed agent: spawns `pi --mode rpc` in path and
// begins consuming the task queue (delivery engine).
func (r *Runtime) Start(agentID, path string) error {
	r.mu.Lock()
	if _, exists := r.agents[agentID]; exists {
		r.mu.Unlock()
		return fmt.Errorf("rpc: agent %s already managed", agentID)
	}
	r.mu.Unlock()

	args := []string{"--mode", "rpc"}
	if r.store != nil {
		if a, err := r.store.GetAgent(agentID); err == nil {
			args = append(args, a.CLIFlags()...)
		}
	}
	mcp.ClearLive(r.DataDir, agentID)
	var extraEnv []string
	if liveArgs, liveEnv := mcp.AttachLive(r.DataDir, agentID); len(liveArgs) > 0 {
		args = append(args, liveArgs...)
		extraEnv = liveEnv
	}
	client, err := Start(r.AgentCmd, args, path, extraEnv...)
	if err != nil {
		return err
	}

	_, cancel := context.WithCancel(context.Background())
	ma := &ManagedAgent{
		AgentID:   agentID,
		Path:      path,
		client:    client,
		hub:       NewHub(),
		store:     r.store,
		cancel:    cancel,
		done:      make(chan struct{}),
		settledCh: closedChan(), // settled until a prompt is accepted
	}

	r.mu.Lock()
	r.agents[agentID] = ma
	r.mu.Unlock()

	go ma.pumpEvents()
	go ma.deliverLoop()
	return nil
}

// Stop terminates a managed agent (idempotent).
func (r *Runtime) Stop(agentID string) bool {
	r.mu.Lock()
	ma := r.agents[agentID]
	delete(r.agents, agentID)
	r.mu.Unlock()
	if ma == nil {
		return false
	}
	mcp.ClearLive(r.DataDir, agentID)
	ma.cancel()
	ma.client.Close()
	<-ma.done
	return true
}

// Get returns the managed agent, if running.
func (r *Runtime) Get(agentID string) *ManagedAgent {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.agents[agentID]
}

// StopAll terminates every managed agent (server shutdown).
func (r *Runtime) StopAll() {
	r.CloseMCPAuth()
	r.mu.Lock()
	ids := make([]string, 0, len(r.agents))
	for id := range r.agents {
		ids = append(ids, id)
	}
	r.mu.Unlock()
	for _, id := range ids {
		r.Stop(id)
	}
}

func closedChan() chan struct{} {
	ch := make(chan struct{})
	close(ch)
	return ch
}

// markSettled swaps the settled broadcast channel.
func (ma *ManagedAgent) markSettled() {
	ma.mu.Lock()
	ma.streaming = false
	ma.settledCh = closedChan()
	ch := ma.settledCh
	ma.mu.Unlock()
	_ = ch
}

// settledChannel returns the current wait-for-settled channel.
func (ma *ManagedAgent) settledChannel() <-chan struct{} {
	ma.mu.Lock()
	defer ma.mu.Unlock()
	return ma.settledCh
}

// pumpEvents forwards rpc events to the hub and tracks streaming state.
func (ma *ManagedAgent) pumpEvents() {
	unsub := ma.client.Subscribe(func(ev Event) {
		switch ev.EventType() {
		case "agent_start":
			ma.mu.Lock()
			ma.streaming = true
			// fresh wait channel: not settled anymore
			ma.settledCh = make(chan struct{})
			ma.mu.Unlock()
		case "agent_settled":
			ma.markSettled()
		case "extension_ui_request":
			ma.noteUIRequest(ev)
		}
		// Envelope for WS consumers: {"agentId":..., "event":{...}}
		env, _ := json.Marshal(map[string]any{"agentId": ma.AgentID, "event": json.RawMessage(ev)})
		ma.hub.Broadcast(env)
	})
	<-ma.client.Done()
	unsub()

	ma.mu.Lock()
	ma.streaming = false
	ma.waiting = nil
	ma.lastErr = fmt.Errorf("process exited")
	ma.mu.Unlock()

	ma.hub.Broadcast(mustEnvelope(ma.AgentID, map[string]any{"type": "exit"}))
	ma.cancel()
	if r := ma.store; r != nil {
		_ = r.SetAgentRuntime(ma.AgentID, store.StatusStopped)
		_ = r.AppendEvent("agent_process_exit", &ma.AgentID, nil, nil)
	}
	close(ma.done)
}

// deliverLoop is the task delivery engine: claim → send → finish.
func (ma *ManagedAgent) deliverLoop() {
	for {
		select {
		case <-ma.done:
			return
		default:
		}

		task, err := ma.store.ClaimNextTask(ma.AgentID)
		if err != nil { // queue empty
			select {
			case <-time.After(500 * time.Millisecond):
			case <-ma.done:
				return
			}
			continue
		}

		if err := ma.deliver(task); err != nil {
			_ = ma.store.FinishTask(task.ID, store.TaskFailed, err.Error())
			_ = ma.store.AppendEvent("task_failed", &ma.AgentID, nil,
				map[string]string{"taskId": task.ID, "error": err.Error()})
			ma.hub.Broadcast(mustEnvelope(ma.AgentID, map[string]any{
				"type": "task_failed", "taskId": task.ID, "error": err.Error(),
			}))
			continue
		}
		_ = ma.store.FinishTask(task.ID, store.TaskDelivered, "")
		_ = ma.store.AppendEvent("task_delivered", &ma.AgentID, nil,
			map[string]string{"taskId": task.ID, "kind": task.Kind})
		ma.hub.Broadcast(mustEnvelope(ma.AgentID, map[string]any{
			"type": "task_delivered", "taskId": task.ID, "kind": task.Kind,
		}))
	}
}

// deliver maps a task kind to its rpc command and waits for acceptance.
// prompt waits until settled (rpc rejects a concurrent prompt). steer and
// follow_up are live-queue commands — send them while the turn is running.
func (ma *ManagedAgent) deliver(task store.Task) error {
	kind := EffectiveTurnKind(task.Kind, ma.isBusy())
	if kind == store.TaskPrompt {
		select {
		case <-ma.settledChannel():
		case <-ma.done:
			return fmt.Errorf("agent stopped")
		case <-time.After(10 * time.Minute):
			return fmt.Errorf("timed out waiting for agent to settle")
		}
	}

	body := map[string]any{"message": task.Payload}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	_, err := ma.client.Send(ctx, Command{Type: kind, Body: body})
	return err
}

func (ma *ManagedAgent) isBusy() bool {
	ma.mu.Lock()
	defer ma.mu.Unlock()
	return ma.streaming || ma.waiting != nil
}

// EffectiveTurnKind maps a composer kind onto the rpc command to send.
// A prompt while the agent is busy becomes follow_up (do not error).
func EffectiveTurnKind(kind string, busy bool) string {
	switch kind {
	case store.TaskSteer, store.TaskFollowUp:
		return kind
	default:
		if busy {
			return store.TaskFollowUp
		}
		return store.TaskPrompt
	}
}

// SetSessionName sets the display name of the live session.
func (ma *ManagedAgent) SetSessionName(ctx context.Context, name string) error {
	_, err := ma.client.Send(ctx, Command{Type: "set_session_name", Body: map[string]any{"name": name}})
	return err
}

// Abort interrupts the current turn (RPC abort). The process stays up.
func (ma *ManagedAgent) Abort(ctx context.Context) error {
	_, err := ma.client.Send(ctx, Command{Type: "abort"})
	return err
}

// Compact asks pi to summarize older turns (RPC compact).
func (ma *ManagedAgent) Compact(ctx context.Context) (Response, error) {
	return ma.client.Send(ctx, Command{Type: "compact"})
}

// SetAutoCompaction toggles live auto-compact (RPC).
func (ma *ManagedAgent) SetAutoCompaction(ctx context.Context, enabled bool) error {
	_, err := ma.client.Send(ctx, Command{Type: "set_auto_compaction", Body: map[string]any{"enabled": enabled}})
	return err
}

// SetSteeringMode sets live steering delivery (RPC).
func (ma *ManagedAgent) SetSteeringMode(ctx context.Context, mode string) error {
	_, err := ma.client.Send(ctx, Command{Type: "set_steering_mode", Body: map[string]any{"mode": mode}})
	return err
}

// SetFollowUpMode sets live follow-up delivery (RPC).
func (ma *ManagedAgent) SetFollowUpMode(ctx context.Context, mode string) error {
	_, err := ma.client.Send(ctx, Command{Type: "set_follow_up_mode", Body: map[string]any{"mode": mode}})
	return err
}

// Fork starts a new session from entryId (RPC fork).
func (ma *ManagedAgent) Fork(ctx context.Context, entryID string) (Response, error) {
	return ma.client.Send(ctx, Command{Type: "fork", Body: map[string]any{"entryId": entryID}})
}

// Clone duplicates the current branch (RPC clone).
func (ma *ManagedAgent) Clone(ctx context.Context) (Response, error) {
	return ma.client.Send(ctx, Command{Type: "clone"})
}

// GetState returns live sessionFile and related fields.
func (ma *ManagedAgent) GetState(ctx context.Context) (Response, error) {
	return ma.client.Send(ctx, Command{Type: "get_state"})
}

// GetCommands lists slash commands the live pi process knows (extensions,
// skills, templates). Used by the composer picker (ADR-0029).
func (ma *ManagedAgent) GetCommands(ctx context.Context) (Response, error) {
	return ma.client.Send(ctx, Command{Type: "get_commands"})
}

// SendBash runs a shell command in the agent cwd (RPC bash). Output
// streams as bash_execution_update events; the response carries the
// final result. The next prompt folds it into context (pi behavior).
func (ma *ManagedAgent) SendBash(ctx context.Context, command string) (Response, error) {
	return ma.client.Send(ctx, Command{Type: "bash", Body: map[string]any{"command": command}})
}

// AbortBash stops a running direct bash command (RPC abort_bash).
func (ma *ManagedAgent) AbortBash(ctx context.Context) error {
	_, err := ma.client.Send(ctx, Command{Type: "abort_bash"})
	return err
}

// SendPrompt delivers a one-off prompt outside the queue (UI "send now").
func (ma *ManagedAgent) SendPrompt(message string) error {
	return ma.SendTurn("prompt", message, nil)
}

// SendPromptCtx is SendPrompt with a caller deadline (slash commands that wait on UI).
func (ma *ManagedAgent) SendPromptCtx(ctx context.Context, message string) error {
	select {
	case <-ma.settledChannel():
	case <-ma.done:
		return fmt.Errorf("agent stopped")
	case <-ctx.Done():
		return ctx.Err()
	}
	_, err := ma.client.Send(ctx, Command{Type: "prompt", Body: map[string]any{"message": message}})
	return err
}

// SendTurn sends prompt/steer/follow_up, optionally with images.
// Images go on the live RPC call — they are not stored in the task table.
// steer / follow_up (and a prompt while busy) do not wait for settled.
func (ma *ManagedAgent) SendTurn(kind, message string, images []map[string]any) error {
	kind = EffectiveTurnKind(kind, ma.isBusy())
	if kind == store.TaskPrompt {
		select {
		case <-ma.settledChannel():
		case <-ma.done:
			return fmt.Errorf("agent stopped")
		case <-time.After(10 * time.Minute):
			return fmt.Errorf("timed out waiting for agent to settle")
		}
	}
	body := map[string]any{"message": message}
	if len(images) > 0 {
		body["images"] = images
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	_, err := ma.client.Send(ctx, Command{Type: kind, Body: body})
	return err
}

// WatchEvents listens to raw RPC events (extension UI, etc.).
func (ma *ManagedAgent) WatchEvents(fn func(Event)) func() {
	return ma.client.Subscribe(fn)
}

// UIDialog is a blocking extension_ui_request (select/confirm/input/editor).
type UIDialog struct {
	ID          string   `json:"id"`
	Method      string   `json:"method"`
	Title       string   `json:"title,omitempty"`
	Message     string   `json:"message,omitempty"`
	Options     []string `json:"options,omitempty"`
	Placeholder string   `json:"placeholder,omitempty"`
	Prefill     string   `json:"prefill,omitempty"`
	Timeout     int      `json:"timeout,omitempty"`
}

func isDialogMethod(m string) bool {
	switch m {
	case "select", "confirm", "input", "editor":
		return true
	}
	return false
}

func (ma *ManagedAgent) noteUIRequest(ev Event) {
	var raw struct {
		ID          string          `json:"id"`
		Method      string          `json:"method"`
		Title       string          `json:"title"`
		Message     string          `json:"message"`
		Placeholder string          `json:"placeholder"`
		Prefill     string          `json:"prefill"`
		Timeout     int             `json:"timeout"`
		Options     json.RawMessage `json:"options"`
	}
	if err := json.Unmarshal(ev, &raw); err != nil || raw.ID == "" || !isDialogMethod(raw.Method) {
		return
	}
	d := &UIDialog{
		ID: raw.ID, Method: raw.Method, Title: raw.Title, Message: raw.Message,
		Placeholder: raw.Placeholder, Prefill: raw.Prefill, Timeout: raw.Timeout,
	}
	if len(raw.Options) > 0 {
		_ = json.Unmarshal(raw.Options, &d.Options)
	}
	ma.mu.Lock()
	ma.waiting = d
	ma.mu.Unlock()
	ma.armTimeout(d.ID, d.Timeout)
}

func (ma *ManagedAgent) armTimeout(id string, ms int) {
	if ms <= 0 {
		return
	}
	time.AfterFunc(time.Duration(ms)*time.Millisecond, func() {
		ma.mu.Lock()
		if ma.waiting == nil || ma.waiting.ID != id {
			ma.mu.Unlock()
			return
		}
		ma.waiting = nil
		ma.mu.Unlock()
		ma.hub.Broadcast(mustEnvelope(ma.AgentID, map[string]any{
			"type": "extension_ui_timeout", "id": id,
		}))
	})
}

// ReplyUI answers an extension_ui_request. confirmed is used for method=confirm.
func (ma *ManagedAgent) ReplyUI(id, value string, confirmed *bool, cancelled bool) error {
	ma.mu.Lock()
	if ma.waiting == nil || ma.waiting.ID != id {
		ma.mu.Unlock()
		return fmt.Errorf("the agent is not asking that")
	}
	ma.waiting = nil
	ma.mu.Unlock()
	body := map[string]any{"type": "extension_ui_response", "id": id}
	if cancelled {
		body["cancelled"] = true
	} else if confirmed != nil {
		body["confirmed"] = *confirmed
	} else {
		body["value"] = value
	}
	return ma.client.SendRaw(body)
}

// Subscribe exposes the event hub (WS consumers).
func (ma *ManagedAgent) Subscribe() (<-chan []byte, func()) {
	return ma.hub.Subscribe()
}

// Snapshot describes the managed agent for UIs.
type Snapshot struct {
	AgentID   string    `json:"agentId"`
	Streaming bool      `json:"streaming"`
	Waiting   bool      `json:"waiting"`
	Dialog    *UIDialog `json:"dialog,omitempty"`
}

// Snapshot returns streaming + waiting state.
func (ma *ManagedAgent) Snapshot() Snapshot {
	ma.mu.Lock()
	defer ma.mu.Unlock()
	s := Snapshot{AgentID: ma.AgentID, Streaming: ma.streaming, Waiting: ma.waiting != nil}
	if ma.waiting != nil {
		d := *ma.waiting
		if len(d.Options) > 0 {
			d.Options = append([]string(nil), d.Options...)
		}
		s.Dialog = &d
	}
	return s
}

func mustEnvelope(agentID string, payload map[string]any) []byte {
	payload["agentId"] = agentID
	b, err := json.Marshal(map[string]any{"event": payload})
	if err != nil {
		return []byte(`{"event":{"type":"error"}}`)
	}
	return b
}
