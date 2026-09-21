package server

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/cfpperche/picode/internal/rpc"
	"github.com/cfpperche/picode/internal/store"
)

const (
	tuiDeliverSnippet = "snippet"
	tuiDeliverPrompt  = "prompt"
)

// deliverToInteractiveAgent pastes (or receiver-delivers) one prompt into a
// running Pi TUI. It is not askTUI: provenance is source (never
// "inspector"), there is no git-root check, HTTP does not wait on JSONL,
// and the JSON is {ok, typed, text} matching a terminal snip-run.
func (deps Deps) deliverToInteractiveAgent(ctx context.Context, agent store.Agent, text, source string) (int, map[string]any) {
	text = strings.TrimRight(text, "\n")
	if strings.TrimSpace(text) == "" {
		return http.StatusBadRequest, map[string]any{"error": "message or file is required"}
	}
	if deps.Replies == nil {
		return http.StatusServiceUnavailable, map[string]any{"error": "Need tmux to send to a terminal."}
	}
	if err := deps.Replies.Controls.check(agent.ID); err != nil {
		return http.StatusConflict, map[string]any{
			"error":  "This agent's terminal is already receiving a message. Try again in a moment.",
			"reason": "busy",
		}
	}
	if deps.Tmux == nil || !deps.Tmux.Available() {
		return http.StatusServiceUnavailable, map[string]any{"error": "Need tmux to send to a terminal."}
	}
	has, err := deps.Tmux.HasSession(ctx, deps.agentSession(agent.ID))
	if err != nil || !has {
		return http.StatusConflict, map[string]any{"error": "agent is not running", "reason": "stopped"}
	}

	deps.Replies.mu.Lock()
	if deps.Replies.active[agent.ID] {
		deps.Replies.mu.Unlock()
		return http.StatusConflict, map[string]any{
			"error":  "This agent's terminal is already receiving a message. Try again in a moment.",
			"reason": "busy",
		}
	}
	deps.Replies.active[agent.ID] = true
	deps.Replies.mu.Unlock()
	defer func() {
		deps.Replies.mu.Lock()
		delete(deps.Replies.active, agent.ID)
		deps.Replies.mu.Unlock()
	}()

	if strings.TrimSpace(source) == "" {
		source = tuiDeliverSnippet
	}
	task, err := deps.Store.EnqueueTask(agent.ID, store.TaskPrompt, text, source)
	if err != nil {
		return http.StatusBadRequest, map[string]any{"error": err.Error()}
	}
	if _, err := deps.Store.ClaimTask(agent.ID, task.ID); err != nil {
		return http.StatusBadRequest, map[string]any{"error": err.Error()}
	}
	cwd := ""
	if _, home, err := deps.agentHome(agent); err == nil {
		cwd = home
	}
	sessionPath := deps.askSession(agent, cwd)
	settle := deliverySettle{
		delivered: func(t store.Task) { _ = deps.Store.FinishTask(t.ID, store.TaskDelivered, "") },
		failed:    func(t store.Task, reason string) { _ = deps.Store.FinishTask(t.ID, store.TaskFailed, reason) },
	}
	baseline := rpc.CaptureDeliveryBaseline(sessionPath)
	if sessionPath != "" && deps.Replies.receiverFresh(agent.ID) {
		err = deps.deliverViaReceiver(agent.ID, sessionPath, task, baseline, settle)
	} else {
		err = deps.deliverViaPaste(ctx, agent.ID, sessionPath, task, baseline, settle)
	}
	if err != nil {
		settle.failed(task, err.Error())
		if errors.Is(err, errPayloadStaged) {
			return http.StatusBadGateway, map[string]any{"error": errPayloadStaged.Error(), "reason": "staged"}
		}
		return http.StatusBadGateway, map[string]any{"error": "the terminal did not take the message: " + err.Error()}
	}
	return http.StatusOK, map[string]any{"ok": true, "typed": true, "text": text}
}
