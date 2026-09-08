package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/gitgraph"
	"github.com/cfpperche/picode/internal/rpc"
	"github.com/cfpperche/picode/internal/session"
	"github.com/cfpperche/picode/internal/store"
)

// ADR-0089, amended 2026-09-08: a terminal hosting pi can be *asked* — the
// git graph's and the Inspector's third door reaches the TUI the owner
// actually works in — but only through ADR-0060's receiver, never by paste.
// The receiver inside that pi said hello within the TTL and named the session
// it is showing; the prompt travels as a one-shot reply file, the TUI submits
// it through pi.sendUserMessage, and the JSONL row is the proof. No receiver,
// no session, another repository, a message already in flight: each is a 409
// that names itself, and nothing is typed into the pane.
//
// Provenance is an event, not a task: the task queue is an agent's
// (tasks.agent_id references agents), and a terminal is not one.
func registerTerminalAskRoutes(mux Registrar, deps Deps) {
	mux.HandleFunc("POST /api/terminals/{id}/tui-hello", handleTerminalTuiHello(deps))
	mux.HandleFunc("POST /api/terminals/{id}/tui-ack", handleTuiAck(deps))
	mux.HandleFunc("POST /api/terminals/{id}/ask", handleTerminalAsk(deps))
}

// termReplyKey namespaces a terminal in the reply registry and the reply-file
// tree, so a terminal id can never collide with an agent id there.
func termReplyKey(id string) string { return "term-" + id }

// termHostsPi says whether this terminal was launched as, or is presently
// running, the pi CLI — the only CLI with a receiver.
func termHostsPi(deps Deps, id string) bool {
	if deps.Store != nil {
		if v, err := deps.Store.TerminalLaunch(id); err == nil && v != nil && strings.TrimSpace(v.CLI) == "pi" {
			return true
		}
	}
	if deps.TermRuntimes != nil {
		if rt, ok := deps.TermRuntimes.Get(id); ok && strings.TrimSpace(rt.CLI) == "pi" {
			return true
		}
	}
	return false
}

func handleTerminalTuiHello(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Session string `json:"session"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		id := r.PathValue("id")
		if _, err := deps.Store.GetTerminal(id); err != nil {
			writeErr(w, http.StatusNotFound, "no such terminal")
			return
		}
		deps.Replies.HelloSession(termReplyKey(id), req.Session)
		w.WriteHeader(http.StatusNoContent)
	}
}

var (
	errAskNoReceiver = errors.New("no receiver is listening in this terminal")
	errAskNoSession  = errors.New("the terminal has not named a session yet")
	errAskNotPi      = errors.New("this terminal is not running pi")
)

// terminalAskSession is the session a fresh receiver reported, kept only
// when it is a real file under this folder's session tree — the same bound
// askSession applies to an agent's record.
func (deps Deps) terminalAskSession(id, cwd string) (string, error) {
	key := termReplyKey(id)
	if !deps.Replies.receiverFresh(key) {
		return "", errAskNoReceiver
	}
	path := deps.Replies.receiverSession(key)
	if path == "" || !safeSessionPath(path, session.Dir(cwd)) {
		return "", errAskNoSession
	}
	return path, nil
}

// askTerminal delivers text to the pi in this terminal through its receiver.
func (deps Deps) askTerminal(id, cwd, text string) (askResult, error) {
	if deps.Replies == nil {
		return askResult{}, errAskNoTmux
	}
	if !termHostsPi(deps, id) {
		return askResult{}, errAskNotPi
	}
	if _, live := deps.TermRuntimes.Get(id); !live {
		return askResult{}, errAskStopped
	}
	sessionPath, err := deps.terminalAskSession(id, cwd)
	if err != nil {
		return askResult{}, err
	}
	key := termReplyKey(id)
	deps.Replies.mu.Lock()
	if deps.Replies.active[key] {
		deps.Replies.mu.Unlock()
		return askResult{}, errAskBusy
	}
	deps.Replies.active[key] = true
	deps.Replies.mu.Unlock()
	defer func() {
		deps.Replies.mu.Lock()
		delete(deps.Replies.active, key)
		deps.Replies.mu.Unlock()
	}()

	// A transient task: the receiver path reads its payload and hands it to
	// the settle callbacks, which record the outcome as events rather than
	// rows in a queue that belongs to agents.
	task := store.Task{ID: "term-ask-" + id + "-" + time.Now().UTC().Format("20060102T150405.000000000"), Payload: text}
	event := func(name, reason string) {
		_ = deps.Store.AppendEvent(name, nil, nil, map[string]any{
			"terminalId": id, "source": askSource, "session": sessionPath, "reason": reason,
		})
	}
	settle := deliverySettle{
		delivered: func(store.Task) { event("terminal_ask_delivered", "") },
		failed:    func(_ store.Task, reason string) { event("terminal_ask_failed", reason) },
		rowWait:   replyRowWait,
	}
	baseline := rpc.CaptureDeliveryBaseline(sessionPath)
	if err := deps.deliverViaReceiver(key, sessionPath, task, baseline, settle); err != nil {
		settle.failed(task, err.Error())
		return askResult{}, err
	}
	return askResult{Mode: "interactive", Via: "receiver", Proof: true, TaskID: task.ID}, nil
}

// DeliverTerminalReply answers an Inbox item filed by a pi running in an
// Agent CLI terminal (sourceKind "terminal", stamped from PICODE_TERM_ID by
// pi-inbox): the reply travels through that terminal's own receiver with the
// item's exact session as the destination — the ADR-0089 ask door, reversed.
// The item parks done once the preflight passes (ADR-0060's contract); every
// failure path reopens it with the response preserved for prefill. There is
// deliberately no task row — the task queue belongs to agents (ADR-0089) —
// so reopen goes through store.ReopenInboxItem and a daemon death between
// park and JSONL row is the accepted terminal-ask gap.
func (deps Deps) DeliverTerminalReply(itemID, verb, text string) (termID string, err error) {
	it, err := deps.Store.GetInboxItem(itemID)
	if err != nil {
		return "", err
	}
	if it.SourceKind != store.InboxFromTerminal || strings.TrimSpace(it.SourceID) == "" {
		return "", fmt.Errorf("this item has no terminal")
	}
	termID = it.SourceID
	term, err := deps.Store.GetTerminal(termID)
	if err != nil {
		_ = deps.Store.AnnotateInboxItem(itemID, "Reply not delivered: the terminal no longer exists.")
		return termID, fmt.Errorf("terminal no longer exists: %w", store.ErrNotFound)
	}
	name := term.Name
	if name == "" {
		name = "This terminal"
	}
	key := termReplyKey(termID)
	// Every refusal names itself and leaves the item open for a retry.
	if !termHostsPi(deps, termID) {
		return termID, fmt.Errorf("%s is not running pi, so the reply cannot be delivered", name)
	}
	if _, live := deps.TermRuntimes.Get(termID); !live {
		return termID, fmt.Errorf("%s is not running pi right now. Start it, then send the reply again", name)
	}
	if !deps.Replies.receiverFresh(key) {
		return termID, fmt.Errorf("no receiver is listening in %s. Restart pi there, then send the reply again", name)
	}
	sessionPath := strings.TrimSpace(it.SessionPath)
	if sessionPath == "" {
		return termID, errors.New("this question predates session tracking — answer it in the terminal")
	}
	if !safeSessionPath(sessionPath, session.Dir(term.Cwd)) {
		return termID, errors.New("the question's session could not be identified safely — answer it in the terminal")
	}
	if st, err := os.Stat(sessionPath); err != nil || st.IsDir() || st.Size() == 0 {
		return termID, errors.New("the question's session no longer exists — answer it in the terminal")
	}
	if shown := deps.Replies.receiverSession(key); shown != "" && shown != sessionPath {
		return termID, errors.New("the terminal is showing a different session now — answer it there, or ask again from the new session")
	}
	deps.Replies.mu.Lock()
	busy := deps.Replies.active[key]
	if !busy {
		deps.Replies.active[key] = true
	}
	deps.Replies.mu.Unlock()
	if busy {
		return termID, fmt.Errorf("%s is already receiving a message. Try again in a moment", name)
	}
	defer func() {
		deps.Replies.mu.Lock()
		delete(deps.Replies.active, key)
		deps.Replies.mu.Unlock()
	}()

	// Park done now; every failure below reopens with the response kept
	// for prefill (EndInboxReply's contract, task-less).
	if _, err := deps.Store.RespondInboxItem(itemID, verb, text); err != nil {
		return termID, err
	}
	payload := store.InboxForwardPayload(it, verb, text)
	task := store.Task{ID: "term-inbox-" + termID + "-" + time.Now().UTC().Format("20060102T150405.000000000"), Payload: payload}
	baseline := rpc.CaptureDeliveryBaseline(sessionPath)
	settle := deliverySettle{
		delivered: func(store.Task) {},
		failed: func(_ store.Task, reason string) {
			_, _ = deps.Store.ReopenInboxItem(itemID,
				"The reply never reached the terminal ("+reason+"). Send it again from this item.")
		},
		rowWait: replyRowWait,
	}
	if err := deps.deliverViaReceiver(key, sessionPath, task, baseline, settle); err != nil {
		_, _ = deps.Store.ReopenInboxItem(itemID, "The reply could not be delivered to the terminal. Send it again from this item.")
		return termID, err
	}
	return termID, nil
}

func handleTerminalAsk(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		t, err := deps.Store.GetTerminal(id)
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		var req struct {
			Text string `json:"text"`
			Root string `json:"root"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		text := strings.TrimSpace(req.Text)
		switch {
		case text == "":
			writeErr(w, http.StatusBadRequest, "text is required")
			return
		case len(text) > askMaxLen:
			writeErr(w, http.StatusBadRequest, "text is too long")
			return
		case strings.TrimSpace(req.Root) == "":
			writeErr(w, http.StatusBadRequest, "root is required")
			return
		}
		name := t.Name
		if name == "" {
			name = "This terminal"
		}
		cwd := t.Cwd
		if gitgraph.Key(cwd) == "" || !sameRepository(cwd, req.Root) {
			writeJSON(w, http.StatusConflict, map[string]any{
				"error": name + " is not working in this repository.", "reason": "moved",
			})
			return
		}
		res, err := deps.askTerminal(id, cwd, text)
		switch {
		case err == nil:
			writeJSON(w, http.StatusOK, res)
		case errors.Is(err, errAskNotPi):
			writeJSON(w, http.StatusConflict, map[string]any{
				"error": name + " is not running pi, so it cannot be asked.", "reason": "cli",
			})
		case errors.Is(err, errAskStopped):
			writeJSON(w, http.StatusConflict, map[string]any{
				"error": name + " is not running pi right now. Start it, then try again.", "reason": "stopped",
			})
		case errors.Is(err, errAskNoReceiver):
			writeJSON(w, http.StatusConflict, map[string]any{
				"error": "No receiver is listening in " + name + ". Restart pi there, then try again.", "reason": "no-receiver",
			})
		case errors.Is(err, errAskNoSession):
			writeJSON(w, http.StatusConflict, map[string]any{
				"error": name + " has not opened a session yet. Send it one message, then try again.", "reason": "no-session",
			})
		case errors.Is(err, errAskBusy):
			writeJSON(w, http.StatusConflict, map[string]any{
				"error": name + " is already receiving a message. Try again in a moment.", "reason": "busy",
			})
		case errors.Is(err, errAskNoTmux):
			writeErr(w, http.StatusServiceUnavailable, "Terminal integration is unavailable on this machine.")
		default:
			writeErr(w, http.StatusBadGateway, name+" did not take the message: "+err.Error())
		}
	}
}
