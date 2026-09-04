package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/cfpperche/picode/internal/mcp"
	"github.com/cfpperche/picode/internal/session"
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

// Len is the live subscriber count. The SPA holds one /ws/agent socket
// for the selected agent only, so 0 is the closest existing proxy for
// "nobody is watching this agent" (ADR-0037's unobserved-run gate).
func (h *Hub) Len() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.subs)
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
	// onWaiting is Runtime.OnWaiting captured at spawn (ADR-0047).
	onWaiting func(agentID, agentName, title, message string)

	cancel context.CancelFunc
	done   chan struct{}

	mu            sync.Mutex
	streaming     bool
	lastErr       error
	settledCh     chan struct{} // closed+replaced when agent_settled arrives
	waiting       *UIDialog     // blocking extension_ui_request, if any
	lastFinal     string        // last assistant text from agent_end (inbox result body)
	stopRequested bool          // Runtime.Stop was called: exit is expected, no fyi
	observer      *RunObserver  // automations/burst owner watching this run
	deliveryPaths []string      // exact selected JSONL plus private dir for fresh sessions
	onState       func(agentID string, streaming, waiting bool, dialog *UIDialog)
	onUsage       func(agentID string, u Usage)
	cost          float64 // sum of usage.cost.total over assistant message_end events
}

// RunObserver is set by an owner that files its own Inbox items for the
// run (the automations engine). While one is attached the agent's default
// unobserved-result and unexpected-exit items are suppressed — one item
// per state change (ADR-0037), written by whoever owns the run.
type RunObserver struct {
	OnSpawn   func(pid int) error // process exists; owner records its crash lease before sending work
	OnStarted func()              // first agent_start for the owned run
	OnSettled func(final string)  // turn finished; final is the agent's last text ("" if none)
	OnExit    func(expected bool) // process ended; expected = Runtime.Stop asked for it
	OnCost    func(total float64) // after each assistant message: cost so far in this process
}

// Cost is the money spent by this process so far, summed from pi's own
// per-message usage (the same number the session file carries).
func (ma *ManagedAgent) Cost() float64 {
	ma.mu.Lock()
	defer ma.mu.Unlock()
	return ma.cost
}

// messageCost reads usage.cost.total from a message_end / turn_end event
// whose message is an assistant message; 0 otherwise.
func messageCost(ev Event) float64 {
	u, ok := messageUsage(ev)
	if !ok {
		return 0
	}
	return u.Cost
}

// Usage is one assistant message's token accounting as pi reports it —
// the same keys internal/session's scanUsage sums from the session file.
type Usage struct {
	Input       int     `json:"input"`
	Output      int     `json:"output"`
	CacheRead   int     `json:"cacheRead"`
	CacheWrite  int     `json:"cacheWrite"`
	TotalTokens int     `json:"totalTokens"` // context size after this message, when pi reports it
	Cost        float64 `json:"cost"`
}

// messageUsage decodes message.usage from a message_end event for an
// assistant message; ok is false for any other message.
func messageUsage(ev Event) (Usage, bool) {
	var end struct {
		Message struct {
			Role  string `json:"role"`
			Usage struct {
				Input       float64 `json:"input"`
				Output      float64 `json:"output"`
				CacheRead   float64 `json:"cacheRead"`
				CacheWrite  float64 `json:"cacheWrite"`
				TotalTokens float64 `json:"totalTokens"`
				Cost        struct {
					Total float64 `json:"total"`
				} `json:"cost"`
			} `json:"usage"`
		} `json:"message"`
	}
	if err := json.Unmarshal(ev, &end); err != nil || end.Message.Role != "assistant" {
		return Usage{}, false
	}
	u := end.Message.Usage
	return Usage{Input: int(u.Input), Output: int(u.Output), CacheRead: int(u.CacheRead), CacheWrite: int(u.CacheWrite),
		TotalTokens: int(u.TotalTokens), Cost: u.Cost.Total}, true
}

// Observe attaches (or with nil detaches) the run observer.
func (ma *ManagedAgent) Observe(o *RunObserver) {
	ma.mu.Lock()
	ma.observer = o
	ma.mu.Unlock()
}

// Observed reports whether a run is already watching this agent.
func (ma *ManagedAgent) Observed() bool { return ma.runObserver() != nil }

func (ma *ManagedAgent) runObserver() *RunObserver {
	ma.mu.Lock()
	defer ma.mu.Unlock()
	return ma.observer
}

type runtimeStart struct {
	done      chan struct{}
	cancelled bool
}

// Runtime owns all managed agents.
type Runtime struct {
	AgentCmd string // "pi" (ADR-0003)
	DataDir  string // ~/.picode — MCP live snapshots when set

	mu       sync.Mutex
	agents   map[string]*ManagedAgent
	starting map[string]*runtimeStart
	store    *store.Store
	onExit   func(agentID string)

	// OnUsage fires after every assistant message with that message's
	// token accounting, so the status bar can add it up instead of
	// rescanning the session file (ADR-0048 ephemeral agent.usage).
	OnUsage func(agentID string, u Usage)

	// OnState fires on every live-state edge of a managed agent — streaming
	// on/off, a dialog raised or answered — so the change feed can tell
	// every shell who is running without a socket per agent (ADR-0048).
	OnState func(agentID string, streaming, waiting bool, dialog *UIDialog)

	// OnWaiting fires when a managed agent raises a dialog and nobody has
	// its socket open (ADR-0047: the push notifier calls the phone).
	// Optional; called on the pump goroutine, must return fast.
	OnWaiting func(agentID, agentName, title, message string)

	authMu   sync.Mutex
	authJobs map[string]*mcpAuthJob
}

// NewRuntime builds a runtime. onExit (optional) fires when a managed
// agent's process dies on its own.
func NewRuntime(agentCmd string, st *store.Store, onExit func(string)) *Runtime {
	return &Runtime{
		AgentCmd: agentCmd,
		agents:   map[string]*ManagedAgent{},
		starting: map[string]*runtimeStart{},
		store:    st,
		onExit:   onExit,
		authJobs: map[string]*mcpAuthJob{},
	}
}

// Start launches the ordinary managed agent and begins consuming its task
// queue (delivery engine).
func (r *Runtime) Start(agentID, path string) error {
	_, err := r.start(agentID, path, true)
	return err
}

func (r *Runtime) start(agentID, path string, drain bool) (*ManagedAgent, error) {
	r.mu.Lock()
	_, running := r.agents[agentID]
	_, starting := r.starting[agentID]
	if running || starting {
		r.mu.Unlock()
		return nil, fmt.Errorf("rpc: agent %s already managed", agentID)
	}
	ticket := &runtimeStart{done: make(chan struct{})}
	r.starting[agentID] = ticket
	r.mu.Unlock()
	finishedStart := false
	defer func() {
		if finishedStart {
			return
		}
		r.mu.Lock()
		if r.starting[agentID] == ticket {
			delete(r.starting, agentID)
			close(ticket.done)
		}
		r.mu.Unlock()
	}()

	args := []string{"--mode", "rpc"}
	deliveryPaths := []string{session.AgentDir(agentID)}
	var extraEnv []string
	if r.store != nil {
		if a, err := r.store.GetAgent(agentID); err == nil {
			sid := ""
			if a.SessionPath == nil || strings.TrimSpace(*a.SessionPath) == "" {
				// No current pointer — but an earlier run's pending
				// --session-id (ADR-0039) may already have a file on
				// disk: adopt it so the chat continues where the agent's
				// TUI or previous managed run left off (ADR-0053).
				if p := r.store.ResolvePendingAgentSession(agentID); p != "" {
					a.SessionPath = &p
				} else {
					// Fresh start: mint a session id up front so pi's
					// auto-created session is attributable to this agent
					// from the moment it exists (ADR-0039).
					sid = r.store.NewPendingAgentSession(agentID)
				}
			}
			args = append(args, a.CLIFlagsForSpawn(sid)...)
			if a.SessionPath != nil && strings.TrimSpace(*a.SessionPath) != "" {
				deliveryPaths = append(deliveryPaths, *a.SessionPath)
			}
			extraEnv = append(extraEnv, a.SpawnEnv()...)
		}
	}
	if r.DataDir != "" {
		extraEnv = append(extraEnv, "PICODE_DATA="+r.DataDir) // pi packages find server.json and the token
	}
	mcp.ClearLive(r.DataDir, agentID)
	if liveArgs, liveEnv := mcp.AttachLive(r.DataDir, agentID); len(liveArgs) > 0 {
		args = append(args, liveArgs...)
		extraEnv = append(extraEnv, liveEnv...)
	}
	client, err := Start(r.AgentCmd, args, path, extraEnv...)
	if err != nil {
		return nil, err
	}

	_, cancel := context.WithCancel(context.Background())
	ma := &ManagedAgent{
		AgentID:       agentID,
		Path:          path,
		client:        client,
		hub:           NewHub(),
		store:         r.store,
		onWaiting:     r.OnWaiting,
		onState:       r.OnState,
		onUsage:       r.OnUsage,
		cancel:        cancel,
		done:          make(chan struct{}),
		settledCh:     closedChan(), // settled until a prompt is accepted
		deliveryPaths: deliveryPaths,
	}

	r.mu.Lock()
	if ticket.cancelled {
		r.mu.Unlock()
		client.Close()
		r.mu.Lock()
		if r.starting[agentID] == ticket {
			delete(r.starting, agentID)
			close(ticket.done)
		}
		finishedStart = true
		r.mu.Unlock()
		return nil, fmt.Errorf("rpc: start for agent %s was cancelled", agentID)
	}
	delete(r.starting, agentID)
	r.agents[agentID] = ma
	close(ticket.done)
	finishedStart = true
	r.mu.Unlock()

	pumpReady := make(chan struct{})
	go ma.pumpEvents(pumpReady)
	// Do not let the owner send the first prompt until the event subscription
	// is live; otherwise a fast agent_start can disappear between process
	// spawn and observer setup.
	<-pumpReady
	if drain {
		go ma.deliverLoop()
	}
	return ma, nil
}

// Stop terminates a managed agent (idempotent).
func (r *Runtime) Stop(agentID string) bool {
	r.mu.Lock()
	ma := r.agents[agentID]
	// Keep a registered process in the map until Close has joined it. Start
	// treats that entry as an exclusive lease, so a concurrent restart cannot
	// overlap the old writer during its shutdown window.
	starting := r.starting[agentID]
	if ma == nil && starting != nil {
		starting.cancelled = true
	}
	r.mu.Unlock()
	if ma == nil {
		if starting == nil {
			return false
		}
		<-starting.done
		return true
	}
	ma.mu.Lock()
	ma.stopRequested = true // expected exit: pumpEvents files no fyi
	ma.mu.Unlock()
	mcp.ClearLive(r.DataDir, agentID)
	ma.cancel()
	ma.client.Close()
	<-ma.done
	r.mu.Lock()
	if r.agents[agentID] == ma {
		delete(r.agents, agentID)
	}
	r.mu.Unlock()
	return true
}

// Get returns the managed agent, if running.
func (r *Runtime) Get(agentID string) *ManagedAgent {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.agents[agentID]
}

// Active reports a registered or still-starting RPC writer. Lifecycle gates
// use it to preserve the one-writer invariant across process-start races.
func (r *Runtime) Active(agentID string) bool {
	if r == nil {
		return false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.agents[agentID] != nil {
		return true
	}
	return r.starting[agentID] != nil
}

// StopAll terminates every managed agent (server shutdown).
func (r *Runtime) StopAll() {
	r.CloseMCPAuth()
	r.mu.Lock()
	seen := make(map[string]bool, len(r.agents)+len(r.starting))
	ids := make([]string, 0, len(r.agents)+len(r.starting))
	for id := range r.agents {
		seen[id] = true
		ids = append(ids, id)
	}
	for id := range r.starting {
		if !seen[id] {
			ids = append(ids, id)
		}
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

// announceState pushes the current live state to Runtime.OnState.
func (ma *ManagedAgent) announceState() {
	ma.mu.Lock()
	fn := ma.onState
	streaming, waiting := ma.streaming, ma.waiting != nil
	var d *UIDialog
	if ma.waiting != nil {
		c := *ma.waiting
		d = &c
	}
	ma.mu.Unlock()
	if fn != nil {
		fn(ma.AgentID, streaming, waiting, d)
	}
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

// Settled reports whether no turn is in flight (the broadcast channel
// is closed again). Ordinary clients use it for back-pressure; burst
// completion follows its generation-scoped observer instead.
func (ma *ManagedAgent) Settled() bool {
	select {
	case <-ma.settledCh:
		return true
	default:
		return false
	}
}

// settledChannel returns the current wait-for-settled channel.
func (ma *ManagedAgent) settledChannel() <-chan struct{} {
	ma.mu.Lock()
	defer ma.mu.Unlock()
	return ma.settledCh
}

// pumpEvents forwards rpc events to the hub and tracks streaming state.
func (ma *ManagedAgent) pumpEvents(ready chan<- struct{}) {
	unsub := ma.client.Subscribe(func(ev Event) {
		switch ev.EventType() {
		case "agent_start":
			ma.mu.Lock()
			ma.streaming = true
			// fresh wait channel: not settled anymore
			ma.settledCh = make(chan struct{})
			o := ma.observer
			ma.mu.Unlock()
			ma.announceState()
			if o != nil && o.OnStarted != nil {
				o.OnStarted()
			}
		case "agent_end":
			// Stash the agent's actual final message: ADR-0037 result items
			// carry the real answer, never a generated wrapper.
			if text := lastAssistantText(ev); text != "" {
				ma.mu.Lock()
				ma.lastFinal = text
				ma.mu.Unlock()
			}
		case "message_end":
			if u, ok := messageUsage(ev); ok {
				ma.mu.Lock()
				ma.cost += u.Cost
				total := ma.cost
				o := ma.observer
				onUsage := ma.onUsage
				ma.mu.Unlock()
				if onUsage != nil && (u.Cost > 0 || u.Input > 0 || u.Output > 0) {
					onUsage(ma.AgentID, u)
				}
				if o != nil && o.OnCost != nil && u.Cost > 0 {
					o.OnCost(total)
				}
			}
		case "agent_settled":
			ma.markSettled()
			ma.announceState()
			if o := ma.runObserver(); o != nil {
				if o.OnSettled != nil {
					ma.mu.Lock()
					final := ma.lastFinal
					ma.mu.Unlock()
					o.OnSettled(final)
				}
			} else {
				ma.fileUnobservedResult()
			}
		case "extension_ui_request":
			ma.noteUIRequest(ev)
		}
		// Envelope for WS consumers: {"agentId":..., "event":{...}}
		env, _ := json.Marshal(map[string]any{"agentId": ma.AgentID, "event": json.RawMessage(ev)})
		ma.hub.Broadcast(env)
	})
	close(ready)
	<-ma.client.Done()
	unsub()

	ma.mu.Lock()
	ma.streaming = false
	ma.waiting = nil
	ma.lastErr = fmt.Errorf("process exited")
	ma.mu.Unlock()
	ma.announceState()

	ma.hub.Broadcast(mustEnvelope(ma.AgentID, map[string]any{"type": "exit"}))
	ma.cancel()
	ma.mu.Lock()
	expected := ma.stopRequested
	observer := ma.observer
	ma.mu.Unlock()
	if r := ma.store; r != nil {
		_ = r.SetAgentRuntime(ma.AgentID, store.StatusStopped)
		_ = r.AppendEvent("agent_process_exit", &ma.AgentID, nil, nil)
	}
	if observer != nil {
		if observer.OnExit != nil {
			observer.OnExit(expected)
		}
	} else if !expected && ma.store != nil {
		// Unexpected death is worth an item even when watched — it fires
		// once (one item per state change, ADR-0037).
		name, wsID := ma.agentIdentity()
		_, _ = ma.store.CreateInboxItem(store.InboxItemParams{
			Kind: store.InboxFYI, SourceKind: store.InboxFromAgent,
			SourceID: ma.AgentID, WorkspaceID: wsID,
			Reason: "process exited", Title: name + " exited unexpectedly",
			Body: "The pi process died outside a requested stop. Start the agent again to resume; queued messages deliver on the next start.",
		})
	}
	close(ma.done)
}

// fileUnobservedResult files a `result` inbox item when a run settles
// with nobody watching (hub empty — ADR-0037). One SQLite write on the
// settle path, same latency class as AppendEvent; settles are rare. An
// unread result from the same agent is superseded, not duplicated.
func (ma *ManagedAgent) fileUnobservedResult() {
	if ma.store == nil || ma.hub.Len() > 0 {
		return
	}
	ma.mu.Lock()
	body := ma.lastFinal
	ma.mu.Unlock()
	if body == "" {
		body = "Run finished."
	}
	name, wsID := ma.agentIdentity()
	_, _ = ma.store.FileAgentResult(ma.AgentID, wsID, name+" finished", body, "run finished unobserved")
}

// agentIdentity resolves display name and workspace off the hot path.
func (ma *ManagedAgent) agentIdentity() (name, wsID string) {
	name = ma.AgentID
	if ma.store != nil {
		if a, err := ma.store.GetAgent(ma.AgentID); err == nil {
			if a.Name != "" {
				name = a.Name
			}
			wsID = a.WorkspaceID
		}
	}
	return name, wsID
}

// lastAssistantText extracts the final assistant message text from an
// agent_end payload ({messages:[{role, content:[{type:"text",text}]}]}).
// Best-effort: any parse miss returns "".
func lastAssistantText(ev Event) string {
	var end struct {
		Messages []struct {
			Role    string `json:"role"`
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(ev, &end); err != nil {
		return ""
	}
	for i := len(end.Messages) - 1; i >= 0; i-- {
		if end.Messages[i].Role != "assistant" {
			continue
		}
		var parts []string
		for _, c := range end.Messages[i].Content {
			if c.Type == "text" && strings.TrimSpace(c.Text) != "" {
				parts = append(parts, c.Text)
			}
		}
		if len(parts) > 0 {
			return strings.Join(parts, "\n\n")
		}
	}
	return ""
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
			// A lost delivery (pi killed during its init window) goes back
			// to the queue for a bounded retry instead of vanishing; three
			// silent attempts become an honest failure.
			if errors.Is(err, errTurnNotStarted) && task.Attempts < 3 {
				_ = ma.store.FinishTask(task.ID, store.TaskQueued, err.Error())
				select {
				case <-time.After(2 * time.Second):
				case <-ma.done:
					return
				}
				continue
			}
			_ = ma.store.FinishTask(task.ID, store.TaskFailed, err.Error())
			_ = ma.store.AppendEvent("task.failed", &ma.AgentID, nil,
				map[string]string{"taskId": task.ID, "error": err.Error()})
			ma.hub.Broadcast(mustEnvelope(ma.AgentID, map[string]any{
				"type": "task_failed", "taskId": task.ID, "error": err.Error(),
			}))
			continue
		}
		_ = ma.store.FinishTask(task.ID, store.TaskDelivered, "")
		_ = ma.store.AppendEvent("task.delivered", &ma.AgentID, nil,
			map[string]string{"taskId": task.ID, "kind": task.Kind})
		ma.hub.Broadcast(mustEnvelope(ma.AgentID, map[string]any{
			"type": "task_delivered", "taskId": task.ID, "kind": task.Kind,
		}))
	}
}

// deliverTimeout bounds how long a turn command may take to be accepted.
// Overridden in tests.
var deliverTimeout = 60 * time.Second

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
	baseline := CaptureDeliveryBaseline(ma.deliveryPaths...)
	ctx, cancel := context.WithTimeout(context.Background(), deliverTimeout)
	defer cancel()

	_, err := ma.client.Send(ctx, Command{Type: kind, Body: body})
	if err != nil && errors.Is(err, context.DeadlineExceeded) && ma.isWaitingUI() {
		// An extension command (/roles …) answers its prompt only when the
		// whole interactive flow ends. A pending dialog means the flow is
		// alive and waiting on the user — that is delivery, not failure.
		return nil
	}
	if err != nil {
		return err
	}
	// An RPC accept is not proof. pi holds early prompts in an in-memory
	// follow-up queue that does not survive its init/resume window — an
	// earlier handoff lost a live reply there while its task already read
	// "delivered". The only honest proof is the reply materializing as a
	// user message in the agent's session files.
	if kind == store.TaskFollowUp {
		if !ma.awaitReplyInSession(baseline, task.Payload, 30*time.Second) {
			return fmt.Errorf("%w: the reply never reached newly appended session bytes", errTurnNotStarted)
		}
	}
	return err
}

// errTurnNotStarted marks a delivery pi accepted but never processed —
// the task goes back to the queue for a bounded retry.
var errTurnNotStarted = errors.New("pi did not start processing the delivery")

func (ma *ManagedAgent) isWaitingUI() bool {
	ma.mu.Lock()
	defer ma.mu.Unlock()
	return ma.waiting != nil
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
// A blocking extension dialog is cancelled first so Stop works during /roles.
func (ma *ManagedAgent) Abort(ctx context.Context) error {
	ma.mu.Lock()
	d := ma.waiting
	ma.waiting = nil
	ma.mu.Unlock()
	ma.announceState()
	if d != nil {
		_ = ma.client.SendRaw(map[string]any{
			"type": "extension_ui_response", "id": d.ID, "cancelled": true,
		})
	}
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
	ctx, cancel := context.WithTimeout(context.Background(), deliverTimeout)
	defer cancel()
	_, err := ma.client.Send(ctx, Command{Type: kind, Body: body})
	if err != nil && errors.Is(err, context.DeadlineExceeded) && ma.isWaitingUI() {
		// See deliver: an extension command answers only when its
		// interactive flow ends; a pending dialog is not a failure.
		return nil
	}
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

// IsDialogMethod reports whether an extension_ui_request method blocks on a
// human answer. Passive updates (notify/setStatus/setWidget/setTitle/
// set_editor_text) are fire-and-forget decoration and never wait.
func IsDialogMethod(m string) bool {
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
	if err := json.Unmarshal(ev, &raw); err != nil || raw.ID == "" || !IsDialogMethod(raw.Method) {
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
	ma.announceState()
	ma.armTimeout(d.ID, d.Timeout)
	if ma.onWaiting != nil && ma.hub.Len() == 0 {
		name, _ := ma.agentIdentity()
		ma.onWaiting(ma.AgentID, name, d.Title, d.Message)
	}
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
		ma.announceState()
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
	ma.announceState()
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
