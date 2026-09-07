package server

// Stage 3 of ADR-0078: the Inspector asks a running agent to do a Git
// action, through the channel that already carries prompts to that agent.
// Nothing here runs git. A managed agent gets a prompt task the runtime
// delivers (pi's follow_up while a turn streams); a TUI agent gets
// ADR-0060's door — the receiver extension when it said hello and the
// current session file is known, else a bracketed paste into its pane. The
// request carries the rail's pinned root as the equality precondition: the
// agent must work in the same repository the rail shows.

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"

	"github.com/cfpperche/picode/internal/gitgraph"
	"github.com/cfpperche/picode/internal/rpc"
	"github.com/cfpperche/picode/internal/session"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

func registerAgentAskRoutes(mux Registrar, deps Deps) {
	mux.HandleFunc("POST /api/agents/{id}/ask", handleAgentAsk(deps))
}

// askMaxLen bounds one ask. The prompts the Inspector builds run a few
// hundred characters; a commit message rides inside them.
const askMaxLen = 5000

// askSource marks the task's provenance in the queue.
const askSource = "inspector"

// askResult tells the client which door the prompt took, so its note can be
// honest: "asked", "queued after the turn", or "sent to the terminal".
type askResult struct {
	Mode   string `json:"mode"`   // managed | interactive
	Via    string `json:"via"`    // queue | receiver | paste
	Busy   bool   `json:"busy"`   // managed: a turn was streaming, so the prompt follows it
	Proof  bool   `json:"proof"`  // interactive: a session row will confirm the delivery
	TaskID string `json:"taskId"` // the queued task, for provenance
}

var (
	errAskStopped = errors.New("the agent is not running")
	errAskBusy    = errors.New("the agent is already receiving a message")
	errAskNoTmux  = errors.New("terminal integration is unavailable")
)

// askName is how refusals name the agent: its own name, else its workspace.
func askName(agent store.Agent, wk store.Workspace) string {
	if n := strings.TrimSpace(agent.Name); n != "" && n != "default" {
		return n
	}
	if n := strings.TrimSpace(wk.Name); n != "" {
		return n
	}
	return "This agent"
}

// sameRepository is the ask's precondition: both folders share one git
// common dir (worktrees count as one repository), or — outside git — they
// are the same folder.
func sameRepository(a, b string) bool {
	ka, kb := gitgraph.Key(a), gitgraph.Key(b)
	if ka != "" && kb != "" {
		return ka == kb
	}
	ca := canonDir(a)
	return ca != "" && ca == canonDir(b)
}

func handleAgentAsk(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		agent, err := deps.Store.GetAgent(id)
		if errors.Is(err, store.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "agent not found")
			return
		} else if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
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
		wk, cwd, err := deps.agentHome(agent)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		name := askName(agent, wk)
		if !sameRepository(cwd, req.Root) {
			writeJSON(w, http.StatusConflict, map[string]any{
				"error": name + " is not working in this repository.", "reason": "moved",
			})
			return
		}
		switch deps.runMode(r, id) {
		case modeManaged:
			busy := false
			if ma := deps.Runtime.Get(id); ma != nil {
				snap := ma.Snapshot()
				busy = snap.Streaming || snap.Waiting
			}
			task, err := deps.Store.EnqueueTask(id, store.TaskPrompt, text, askSource)
			if err != nil {
				writeErr(w, http.StatusBadRequest, err.Error())
				return
			}
			writeJSON(w, http.StatusAccepted, askResult{Mode: "managed", Via: "queue", Busy: busy, TaskID: task.ID})
		case modeInteractive:
			res, err := deps.askTUI(r.Context(), agent, cwd, text)
			switch {
			case err == nil:
				writeJSON(w, http.StatusOK, res)
			case errors.Is(err, errAskBusy):
				writeJSON(w, http.StatusConflict, map[string]any{
					"error": name + "'s terminal is already receiving a message. Try again in a moment.", "reason": "busy",
				})
			case errors.Is(err, errAskStopped):
				writeJSON(w, http.StatusConflict, map[string]any{
					"error": name + " is not running anymore. Use the terminal instead.", "reason": "stopped",
				})
			case errors.Is(err, errAskNoTmux):
				writeErr(w, http.StatusServiceUnavailable, "Terminal integration is unavailable on this machine.")
			default:
				writeErr(w, http.StatusBadGateway, name+"'s terminal did not take the message: "+err.Error())
			}
		default:
			writeJSON(w, http.StatusConflict, map[string]any{
				"error": name + " is not running. Use the terminal instead.", "reason": "stopped",
			})
		}
	}
}

// askTUI sends one prompt into an interactive agent's TUI the way Inbox
// replies travel (ADR-0060): receiver extension first when it is fresh and
// the session is known, bracketed paste otherwise. One send per agent at a
// time, never during a session mutation.
func (deps Deps) askTUI(ctx context.Context, agent store.Agent, cwd, text string) (askResult, error) {
	agentID := agent.ID
	if deps.Replies == nil {
		return askResult{}, errAskNoTmux
	}
	if err := deps.Replies.Controls.check(agentID); err != nil {
		return askResult{}, errAskBusy
	}
	if deps.Tmux == nil || !deps.Tmux.Available() {
		return askResult{}, errAskNoTmux
	}
	if has, err := deps.Tmux.HasSession(ctx, tmux.SessionName(agentID)); err != nil || !has {
		return askResult{}, errAskStopped
	}
	sessionPath := deps.askSession(agent, cwd)

	deps.Replies.mu.Lock()
	if deps.Replies.active[agentID] {
		deps.Replies.mu.Unlock()
		return askResult{}, errAskBusy
	}
	deps.Replies.active[agentID] = true
	deps.Replies.mu.Unlock()
	defer func() {
		deps.Replies.mu.Lock()
		delete(deps.Replies.active, agentID)
		deps.Replies.mu.Unlock()
	}()

	task, err := deps.Store.EnqueueTask(agentID, store.TaskPrompt, text, askSource)
	if err != nil {
		return askResult{}, err
	}
	if _, err := deps.Store.ClaimTask(agentID, task.ID); err != nil {
		return askResult{}, err
	}
	settle := deliverySettle{
		delivered: func(t store.Task) { _ = deps.Store.FinishTask(t.ID, store.TaskDelivered, "") },
		failed:    func(t store.Task, reason string) { _ = deps.Store.FinishTask(t.ID, store.TaskFailed, reason) },
	}
	if sessionPath != "" {
		settle.rowWait = replyRowWait
	}
	baseline := rpc.CaptureDeliveryBaseline(sessionPath)
	via := "paste"
	if sessionPath != "" && deps.Replies.receiverFresh(agentID) {
		via = "receiver"
		err = deps.deliverViaReceiver(agentID, sessionPath, task, baseline, settle)
	} else {
		err = deps.deliverViaPaste(ctx, agentID, sessionPath, task, baseline, settle)
	}
	if err != nil {
		settle.failed(task, err.Error())
		return askResult{}, err
	}
	return askResult{Mode: "interactive", Via: via, Proof: sessionPath != "", TaskID: task.ID}, nil
}

// askSession is the TUI's current session file when PiCode knows it and it
// sits where this agent's sessions live. The receiver refuses any other
// target, and the durable proof reads it; without one the paste is the
// proof there is.
func (deps Deps) askSession(agent store.Agent, cwd string) string {
	if agent.SessionPath == nil {
		return ""
	}
	path := strings.TrimSpace(*agent.SessionPath)
	if path == "" || !safeSessionPath(path, session.AgentDir(agent.ID), session.Dir(cwd)) {
		return ""
	}
	st, err := os.Stat(path)
	if err != nil || st.IsDir() || st.Size() == 0 {
		return ""
	}
	return path
}
