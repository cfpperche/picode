package server

// Inbox replies land directly in the agent's running TUI (ADR-0060,
// superseding the ADR-0059 transient burst). The TUI never stops being the
// writer: a generated receiver extension inside pi submits the reply through
// `pi.sendUserMessage`, or — when no receiver has said hello — the daemon
// types the reply into the pane with a tmux bracketed paste. Durable proof is
// unchanged: the exact session JSONL must gain the reply as a new user row.

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/cfpperche/picode/internal/rpc"
	"github.com/cfpperche/picode/internal/session"
	"github.com/cfpperche/picode/internal/store"
)

const (
	// Kept verbatim from ADR-0059: mutation conflicts read the same before
	// and after the burst removal.
	replyControlConflict = "This agent's terminal is changing; try the reply again when it is ready"
	replyControlBusy     = "This agent's terminal is already changing. Wait for it to finish and try again."
)

// Timing knobs are vars so tests can shrink the windows.
var (
	// How long a receiver hello counts as "a live receiver is inside this
	// agent's TUI". The extension re-hellos every 5 minutes; the margin
	// absorbs a missed beat without mistaking a dead TUI for a receiver.
	receiverHelloTTL = 10 * time.Minute

	// The extension channel answers fast: consumed file → ack. Past that,
	// reconcile against the JSONL row once and reopen on silence.
	receiverAckWait = 20 * time.Second

	// Both channels finally trust only the JSONL row. A mid-turn queue can
	// delay it until pi processes the message, so the background window is
	// generous before an item reopens.
	replyRowWait = 10 * time.Minute

	// Boot grace: a receiver may submit a file the crashed daemon never
	// saw an ack for. Probe, breathe, probe again before failing a task.
	bootRowGrace = 2 * time.Second
)

// AgentControls serializes terminal/session mutations against replies. One
// rule: a mutation and a reply send never interleave on one agent.
type AgentControls struct {
	mu   sync.Mutex
	held map[string]int
}

func newAgentControls() *AgentControls {
	return &AgentControls{held: map[string]int{}}
}

// check returns a conflict error when a mutation guard is held.
func (g *AgentControls) check(agentID string) error {
	if g == nil {
		return nil
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.held[agentID] > 0 {
		return errors.New(replyControlConflict)
	}
	return nil
}

// BeginMutation blocks new replies while a pane/session mutation runs. The
// returned release is idempotent.
func (g *AgentControls) BeginMutation(agentID string) func() {
	if g == nil {
		return func() {}
	}
	g.mu.Lock()
	g.held[agentID]++
	g.mu.Unlock()
	return g.release(agentID)
}

// TryBeginMutation is the exclusive form used by forced TUI recovery: a
// duplicate click must not kill the fresh pane started by the first request.
func (g *AgentControls) TryBeginMutation(agentID string) (func(), error) {
	if g == nil {
		return func() {}, nil
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.held[agentID] > 0 {
		return nil, errors.New(replyControlBusy)
	}
	g.held[agentID] = 1
	return g.release(agentID), nil
}

func (g *AgentControls) release(agentID string) func() {
	var once sync.Once
	return func() {
		once.Do(func() {
			g.mu.Lock()
			if g.held[agentID] <= 1 {
				delete(g.held, agentID)
			} else {
				g.held[agentID]--
			}
			g.mu.Unlock()
		})
	}
}

// replyAck is the receiver's verdict on one reply file.
type replyAck struct {
	OK     bool
	Reason string
}

// TuiReplies owns reply delivery state: receiver hellos, the one synchronous
// send per agent, ack channels for in-flight reply files, and the shared
// mutation guards.
type TuiReplies struct {
	Controls *AgentControls

	mu    sync.Mutex
	hello map[string]time.Time
	// session is the JSONL a terminal's receiver said it was showing at its
	// last hello (ADR-0089 amendment). Agents keep their session on the
	// agent record; a terminal has no such record, so the hello is the one
	// place the daemon learns which conversation an Ask would land in.
	session           map[string]string
	connection        map[string]string
	connectionProcess map[string]string
	// pid is the process that sent the last hello. A terminal's directory
	// is watched by every pi that inherited its id — a nested `pi -p` or a
	// print-mode run loads the same receiver — so a reply file names this
	// pid and only that process consumes it (2026-09-11).
	pid    map[string]int
	active map[string]bool
	acks   map[string]chan replyAck
}

// NewTuiReplies creates the shared receiver registry for routes and background delivery.
func NewTuiReplies() *TuiReplies {
	return &TuiReplies{
		Controls:   newAgentControls(),
		hello:      map[string]time.Time{},
		session:    map[string]string{},
		connection: map[string]string{},
		pid:        map[string]int{},
		active:     map[string]bool{},
		acks:       map[string]chan replyAck{},
	}
}

// Hello records a live receiver inside this agent's TUI.
func (t *TuiReplies) Hello(agentID string) {
	if t == nil || strings.TrimSpace(agentID) == "" {
		return
	}
	t.mu.Lock()
	t.hello[agentID] = time.Now()
	t.mu.Unlock()
}

// HelloSession records a hello together with the session file the receiver
// reported. An empty session still counts as a hello — the receiver is alive
// — but leaves nothing to ask into until the next one names a file.
func (t *TuiReplies) HelloSession(key, sessionPath string) { t.HelloConnection(key, sessionPath, "") }
func (t *TuiReplies) HelloConnection(key, sessionPath, connection string) {
	t.helloConnectionProcess(key, sessionPath, connection, "")
}
func (t *TuiReplies) helloConnectionProcess(key, sessionPath, connection, process string) {
	t.helloReceiver(key, sessionPath, connection, process, 0)
}

// helloReceiver records a hello. A sessionless hello (a nested `pi -p`, a
// print-mode run: the terminal id and the reply directory are inherited
// environment, so every pi in a terminal watches the same reply files)
// refreshes liveness but must not take over the recorded conversation — the
// reply file is addressed to the process that named a session (ADR-0060's
// 2026-09-11 amendment). A session-bearing hello always wins.
func (t *TuiReplies) helloReceiver(key, sessionPath, connection, process string, pid int) {
	if t == nil || strings.TrimSpace(key) == "" {
		return
	}
	sessionPath = strings.TrimSpace(sessionPath)
	t.mu.Lock()
	adopt := sessionPath != "" || t.session[key] == ""
	t.hello[key] = time.Now()
	if adopt {
		t.session[key] = sessionPath
		if t.pid == nil {
			t.pid = map[string]int{}
		}
		if pid > 0 {
			t.pid[key] = pid
		} else {
			delete(t.pid, key)
		}
		if t.connection == nil {
			t.connection = map[string]string{}
		}
		if t.connectionProcess == nil {
			t.connectionProcess = map[string]string{}
		}
		t.connection[key] = connection
		t.connectionProcess[key] = process
	}
	t.mu.Unlock()
}

// receiverSession is the session file a fresh receiver last reported, or ""
// when the hello is stale or never carried one.
func (t *TuiReplies) receiverSession(key string) string {
	if t == nil {
		return ""
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	at, ok := t.hello[key]
	if !ok || time.Since(at) > receiverHelloTTL {
		return ""
	}
	return t.session[key]
}

// receiverFresh reports whether this agent's TUI recently announced a
// receiver. A daemon restart clears the registry; the extension re-hellos
// within five minutes, and until then the paste fallback carries replies.
func (t *TuiReplies) receiverFresh(agentID string) bool {
	if t == nil {
		return false
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	at, ok := t.hello[agentID]
	return ok && time.Since(at) <= receiverHelloTTL
}

// receiverPID is the process the last session-bearing hello came from, or 0
// when it did not name one.
func (t *TuiReplies) receiverPID(key string) int {
	if t == nil {
		return 0
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.pid[key]
}

// registerAck wires a nonce to its ack channel until the returned func runs.
func (t *TuiReplies) registerAck(nonce string) (<-chan replyAck, func()) {
	ch := make(chan replyAck, 1)
	t.mu.Lock()
	t.acks[nonce] = ch
	t.mu.Unlock()
	return ch, func() {
		t.mu.Lock()
		delete(t.acks, nonce)
		t.mu.Unlock()
	}
}

// resolveAck hands an inbound ack to the waiting send, if any. Unknown
// nonces (daemon restarted mid-flight) are dropped; boot reconciliation
// already settled those tasks against the JSONL.
func (t *TuiReplies) resolveAck(nonce string, ack replyAck) {
	if t == nil || strings.TrimSpace(nonce) == "" {
		return
	}
	t.mu.Lock()
	ch := t.acks[nonce]
	delete(t.acks, nonce)
	t.mu.Unlock()
	if ch != nil {
		ch <- ack
	}
}

type replyPreflight struct {
	Pending       bool
	Managed       bool
	TmuxAvailable bool
	SessionExists bool
	SessionSafe   bool
}

func replyRefusal(in replyPreflight) string {
	switch {
	case in.Pending:
		return "This agent already has a reply on its way to the terminal. Try again in a moment."
	case in.Managed:
		return "This agent is no longer in its terminal."
	case !in.TmuxAvailable:
		return "Terminal integration is unavailable on this machine."
	case !in.SessionExists:
		return "The agent terminal is no longer running."
	case !in.SessionSafe:
		return "The terminal session could not be identified safely. Open the TUI and try again."
	default:
		return ""
	}
}

// DeliverReply answers an Inbox item by sending the reply into the agent's
// running TUI. The item parks done immediately; every failure path reopens it
// with the response preserved for prefill before returning the error.
func (deps Deps) DeliverReply(ctx context.Context, itemID, verb, text string) (agentID string, err error) {
	it, err := deps.Store.GetInboxItem(itemID)
	if err != nil {
		return "", err
	}
	if it.SourceKind != store.InboxFromAgent || strings.TrimSpace(it.SourceID) == "" {
		return "", fmt.Errorf("this item has no agent terminal")
	}
	agentID = it.SourceID
	agent, err := deps.Store.GetAgent(agentID)
	if err != nil {
		return "", err
	}
	// A reply racing an explicit pane/session mutation gets a deterministic,
	// retryable refusal before it inspects transitional process state.
	if err := deps.Replies.Controls.check(agentID); err != nil {
		return "", err
	}
	_, cwd, err := deps.agentHome(agent)
	if err != nil {
		return "", err
	}

	available := deps.Tmux != nil && deps.Tmux.Available()
	hasSession := false
	if available {
		hasSession, _ = deps.Tmux.HasSession(ctx, deps.agentSession(agentID))
	}
	sessionPath, sessionOK := deps.resolveReplySession(agent, it, cwd)
	deps.Replies.mu.Lock()
	pending := deps.Replies.active[agentID]
	deps.Replies.mu.Unlock()
	if reason := replyRefusal(replyPreflight{
		Pending: pending,
		Managed: deps.Runtime.Active(agentID), TmuxAvailable: available,
		SessionExists: hasSession, SessionSafe: sessionOK,
	}); reason != "" {
		return "", errors.New(reason)
	}

	// One synchronous send per agent. The guard covers preflight through
	// dispatch only — the background JSONL wait never holds it, so a delete
	// or restart is never blocked behind a queued turn.
	deps.Replies.mu.Lock()
	if deps.Replies.active[agentID] {
		deps.Replies.mu.Unlock()
		return "", errors.New("This agent already has a reply on its way to the terminal. Try again in a moment.")
	}
	deps.Replies.active[agentID] = true
	deps.Replies.mu.Unlock()
	defer func() {
		deps.Replies.mu.Lock()
		delete(deps.Replies.active, agentID)
		deps.Replies.mu.Unlock()
	}()

	_, task, err := deps.Store.RespondAndPark(itemID, verb, text)
	if err != nil {
		return "", err
	}
	// The send now owns this exact task: delivering, until the durable row
	// settles it. Nothing else may claim or drain it.
	if _, err := deps.Store.ClaimTask(agentID, task.ID); err != nil {
		note := "The reply could not start. Send it again from this item."
		_ = deps.Store.EndInboxReply(task.ID, store.TaskFailed, err.Error(), note)
		return "", err
	}

	baseline := rpc.CaptureDeliveryBaseline(sessionPath)
	settle := deps.inboxSettle()
	if deps.Replies.receiverFresh(agentID) {
		err = deps.deliverViaReceiver(agentID, sessionPath, task, baseline, settle)
	} else {
		err = deps.deliverViaPaste(ctx, agentID, sessionPath, task, baseline, settle)
	}
	if err != nil {
		note := "The reply could not be delivered to the terminal. Send it again from this item."
		_ = deps.Store.EndInboxReply(task.ID, store.TaskFailed, err.Error(), note)
		return agentID, err
	}
	return agentID, nil
}

// deliverySettle is what a delivered or lost message means to its owner: an
// Inbox reply settles or reopens its item (ADR-0060); an Inspector ask
// (ADR-0078 stage 3) records its task. rowWait is how long the durable JSONL
// row may take to appear; zero means no session file is known, so a paste is
// recorded as typed the moment tmux accepts it.
type deliverySettle struct {
	delivered func(task store.Task)
	failed    func(task store.Task, reason string)
	rowWait   time.Duration
}

// inboxSettle keeps ADR-0060's contract: the row settles the item, silence
// reopens it with the response preserved for prefill.
func (deps Deps) inboxSettle() deliverySettle {
	return deliverySettle{
		delivered: deps.finishDelivered,
		failed: func(task store.Task, reason string) {
			_ = deps.Store.EndInboxReply(task.ID, store.TaskFailed, reason,
				"The reply never reached the session. Send it again from this item.")
		},
		rowWait: replyRowWait,
	}
}

// deliverViaReceiver hands the reply to the receiver extension through a
// one-shot file and waits briefly for its ack. The ack means the TUI owns the
// message and renders it (queued mid-turn, per the owner's decision); the
// durable JSONL row, reconciled in the background, remains the truth that
// keeps the item done — silence reopens it.
func (deps Deps) deliverViaReceiver(agentID, sessionPath string, task store.Task, baseline rpc.DeliveryBaseline, settle deliverySettle) error {
	nonce, err := newReplyNonce()
	if err != nil {
		return err
	}
	file, err := writeReplyFile(deps.DataDir, agentID, replyFile{
		Nonce: nonce, SessionPath: sessionPath, Payload: task.Payload, CreatedAt: time.Now().UTC(),
		PID: deps.Replies.receiverPID(agentID),
	})
	if err != nil {
		return err
	}
	ack, done := deps.Replies.registerAck(nonce)
	defer done()

	deadline := time.After(receiverAckWait)
	tick := time.NewTicker(500 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case a := <-ack:
			_ = os.Remove(file)
			if !a.OK {
				return errors.New(a.Reason)
			}
			// The TUI owns the message now. The row decides whether the
			// item stays done; a TUI that dies before processing reopens it.
			// rowWait <= 0: caller treats the ack as proof (snip-run /
			// interactive prompt) and does not wait on JSONL.
			if settle.rowWait <= 0 {
				settle.delivered(task)
				return nil
			}
			go deps.awaitReplyRow(task, sessionPath, baseline, settle)
			return nil
		case <-deadline:
			_ = os.Remove(file)
			// No ack: the receiver may have died with the file unread. The
			// durable row decides; silence reopens the item.
			if rpc.UserMessageAfter(baseline, task.Payload) {
				settle.delivered(task)
				return nil
			}
			return errors.New("the terminal did not pick up the reply")
		case <-tick.C:
			if rpc.UserMessageAfter(baseline, task.Payload) {
				_ = os.Remove(file)
				settle.delivered(task)
				return nil
			}
		}
	}
}

// deliverViaPaste types the reply into the pane for legacy TUIs without a
// receiver: bracketed paste (the editor inserts it wholesale, no keybindings
// fire) plus Enter. pi queues the submit natively while a turn is streaming.
// Tradeoffs the owner accepted: the paste can land in an open draft, and the
// pane's current session cannot be verified — the JSONL row proof still
// gates whether the item stays done.
func (deps Deps) deliverViaPaste(ctx context.Context, agentID, sessionPath string, task store.Task, baseline rpc.DeliveryBaseline, settle deliverySettle) error {
	if err := deps.Tmux.PasteText(ctx, deps.agentSession(agentID), task.Payload); err != nil {
		return err
	}
	if settle.rowWait <= 0 {
		// No session file to read: the paste is the proof there is.
		settle.delivered(task)
		return nil
	}
	go deps.awaitReplyRow(task, sessionPath, baseline, settle)
	return nil
}

// awaitReplyRow closes the loop in the background: the reply counts as
// delivered when the exact session gains the user row, and the item reopens
// with a truthful reason if the terminal never processes it.
func (deps Deps) awaitReplyRow(task store.Task, sessionPath string, baseline rpc.DeliveryBaseline, settle deliverySettle) {
	deadline := time.After(settle.rowWait)
	tick := time.NewTicker(2 * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-deadline:
			settle.failed(task, "the terminal never processed the reply")
			return
		case <-tick.C:
			if rpc.UserMessageAfter(baseline, task.Payload) {
				settle.delivered(task)
				return
			}
		}
	}
}

func (deps Deps) finishDelivered(task store.Task) {
	_ = deps.Store.SettleReplyTask(task.ID)
}

// ReconcilePendingReplies runs once at daemon startup, after the store is
// open: every pending reply task is settled against its session JSONL (with a
// short grace for a receiver that submitted right before the crash), leftover
// reply files are removed, and unanswered items reopen with the response
// preserved for prefill.
func ReconcilePendingReplies(st *store.Store, dataDir string) {
	tasks, err := st.PendingReplyTasks()
	if err != nil {
		return
	}
	finishDelivered := func(task store.Task) {
		_ = st.SettleReplyTask(task.ID)
	}
	for _, task := range tasks {
		itemID := strings.TrimPrefix(strings.TrimPrefix(task.Source, "inbox-tui:"), "inbox-burst:")
		sessionPath := ""
		if it, err := st.GetInboxItem(itemID); err == nil {
			sessionPath = it.SessionPath
		}
		present := sessionPath != "" && store.ReplyRowPresent(sessionPath, task.Payload, task.CreatedAt)
		if !present {
			// A receiver may have submitted the file right before the crash:
			// breathe once, then probe again before declaring failure.
			time.Sleep(bootRowGrace)
			present = sessionPath != "" && store.ReplyRowPresent(sessionPath, task.Payload, task.CreatedAt)
		}
		if present {
			finishDelivered(task)
			continue
		}
		_ = os.RemoveAll(replyDir(dataDir, task.AgentID))
		_ = st.EndInboxReply(task.ID, store.TaskFailed,
			"PiCode restarted before the terminal confirmed the reply",
			"PiCode restarted before the reply was confirmed. Send it again from this item.")
	}
}

// replyFile is the one-shot handoff between the daemon and the receiver
// extension inside the TUI.
type replyFile struct {
	SetupConnection string    `json:"setupConnection,omitempty"`
	AttentionOnly   bool      `json:"attentionOnly,omitempty"`
	Nonce           string    `json:"nonce"`
	SessionPath     string    `json:"sessionPath"`
	Payload         string    `json:"payload"`
	CreatedAt       time.Time `json:"createdAt"`
	// PID addresses the one process that may consume this file: the receiver
	// that said hello, not merely any pi that inherited the terminal id.
	// Absent when the hello did not name one (an older receiver).
	PID int `json:"pid,omitempty"`
}

func replyDir(dataDir, agentID string) string {
	return filepath.Join(dataDir, "tui-inbox", agentID)
}

func writeReplyFile(dataDir, agentID string, f replyFile) (string, error) {
	dir := replyDir(dataDir, agentID)
	if strings.TrimSpace(dataDir) == "" || strings.TrimSpace(agentID) == "" {
		return "", fmt.Errorf("reply drop directory unknown")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	raw, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, f.Nonce+".json")
	return path, os.WriteFile(path, raw, 0o600)
}

func newReplyNonce() (string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}

func (deps Deps) resolveReplySession(agent store.Agent, it store.InboxItem, cwd string) (string, bool) {
	// A reply must target the session that filed this exact item. Falling
	// back to the agent's current or latest session can deliver an answer
	// into a different conversation.
	path := strings.TrimSpace(it.SessionPath)
	if path == "" || !safeSessionPath(path, session.AgentDir(agent.ID), session.Dir(cwd)) {
		return "", false
	}
	st, err := os.Stat(path)
	return path, err == nil && !st.IsDir() && st.Size() > 0
}

func handleTuiHello(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Session    string `json:"session"`
			Connection string `json:"connection"`
			PID        int    `json:"pid"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		agentID := r.PathValue("id")
		if _, err := deps.Store.GetAgent(agentID); err != nil {
			writeErr(w, http.StatusNotFound, "no such agent")
			return
		}
		if req.PID > 0 {
			if ma := deps.Runtime.Get(agentID); ma != nil && ma.PID() != req.PID {
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}
		deps.Replies.helloReceiver(agentID, req.Session, req.Connection, fmt.Sprint(req.PID), req.PID)
		w.WriteHeader(http.StatusNoContent)
	}
}

func handleTuiAck(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Nonce  string `json:"nonce"`
			OK     bool   `json:"ok"`
			Reason string `json:"reason"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Nonce) == "" {
			writeErr(w, http.StatusBadRequest, "nonce required")
			return
		}
		deps.Replies.resolveAck(req.Nonce, replyAck{OK: req.OK, Reason: req.Reason})
		w.WriteHeader(http.StatusNoContent)
	}
}

func (t *TuiReplies) receiverConnection(key, session string, process ...string) string {
	if t == nil {
		return ""
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.session[key] != session || time.Since(t.hello[key]) > receiverHelloTTL {
		return ""
	}
	if len(process) > 0 && t.connectionProcess[key] != process[0] {
		return ""
	}
	return t.connection[key]
}
