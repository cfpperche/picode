package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"os/exec"

	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

// registerAgentRoutes wires managed-mode control (ADR-0006): one live pi
// process per agent — starting managed mode stops the interactive session
// and vice versa.
func registerAgentRoutes(mux *http.ServeMux, deps Deps) {
	mux.HandleFunc("POST /api/agents/{id}/managed/start", handleManagedStart(deps))
	mux.HandleFunc("POST /api/agents/{id}/managed/stop", handleManagedStop(deps))
}

// agentRunMode reports how an agent is currently running.
type agentRunMode string

const (
	modeStopped     agentRunMode = "stopped"
	modeInteractive agentRunMode = "interactive"
	modeManaged     agentRunMode = "managed"
)

func (deps Deps) runMode(r *http.Request, agentID string) agentRunMode {
	if ma := deps.Runtime.Get(agentID); ma != nil {
		return modeManaged
	}
	if deps.Tmux.Available() {
		if has, err := deps.Tmux.HasSession(r.Context(), tmux.SessionName(agentID)); err == nil && has {
			return modeInteractive
		}
	}
	return modeStopped
}

func handleManagedStart(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		agentID := r.PathValue("id")
		agent, err := deps.Store.GetAgent(agentID)
		if errors.Is(err, store.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "agent not found")
			return
		}
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}

		// ADR-0006: exclusive run mode — stop interactive first.
		if deps.runMode(r, agentID) == modeInteractive {
			if err := deps.Tmux.KillSession(r.Context(), tmux.SessionName(agentID)); err != nil {
				writeErr(w, http.StatusInternalServerError, "stop interactive: "+err.Error())
				return
			}
		}
		if deps.Runtime.Get(agentID) != nil {
			writeJSON(w, http.StatusOK, map[string]any{"mode": modeManaged, "alreadyRunning": true})
			return
		}

		if _, err := exec.LookPath(deps.AgentCmd); err != nil {
			writeErr(w, http.StatusServiceUnavailable,
				"pi is not installed or not on PATH — install it with: npm install -g @earendil-works/pi-coding-agent")
			return
		}
		wk, err := deps.Store.GetWorkspace(agent.WorkspaceID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if err := deps.Runtime.Start(agentID, wk.Path); err != nil {
			writeErr(w, http.StatusInternalServerError, "start managed: "+err.Error())
			return
		}
		_ = deps.Store.SetAgentRuntime(agentID, store.StatusRunning)
		_ = deps.Store.AppendEvent("agent_managed_started", &agentID, &wk.ID, nil)
		writeJSON(w, http.StatusCreated, map[string]any{"mode": modeManaged})
	}
}

func handleManagedStop(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		agentID := r.PathValue("id")
		if _, err := deps.Store.GetAgent(agentID); errors.Is(err, store.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "agent not found")
			return
		}
		if !deps.Runtime.Stop(agentID) {
			writeJSON(w, http.StatusOK, map[string]any{"mode": modeStopped, "alreadyStopped": true})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"mode": modeStopped})
	}
}

// agentWS streams managed-agent events over WebSocket and accepts
// enqueue commands from the client:
//
//	server -> client : {"event": {...pi rpc event..., "agentId"}}
//	client -> server : {"type":"enqueue","kind":"prompt|steer|follow_up","payload":"..."}
//
// First message on connect is a snapshot: {"event":{"type":"snapshot",
// "mode":"managed","streaming":bool}} or an error envelope when the agent
// is not running in managed mode.
func agentWS(deps Deps) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		agentID := r.URL.Query().Get("agent")
		ma := deps.Runtime.Get(agentID)
		if ma == nil {
			// Accept the upgrade, then deliver a friendly status and close.
			ws, err := upgraderUpgrade(w, r)
			if err != nil {
				return
			}
			defer ws.Close()
			writeWSJSON(ws, map[string]any{"event": map[string]any{
				"type": "status", "mode": "stopped", "streaming": false,
			}})
			return
		}

		ws, err := upgraderUpgrade(w, r)
		if err != nil {
			return
		}
		defer ws.Close()

		// Snapshot first.
		writeWSJSON(ws, map[string]any{"event": map[string]any{
			"type": "snapshot", "mode": "managed", "streaming": ma.Snapshot().Streaming,
		}})

		events, unsub := ma.Subscribe()
		defer unsub()

		writeDone := make(chan struct{})
		go func() { // events -> client
			defer close(writeDone)
			for msg := range events {
				if !writeWSRaw(ws, msg) {
					return
				}
			}
		}()

		// client -> enqueue commands (read until error/close).
		for {
			msgType, data, err := ws.ReadMessage()
			if err != nil {
				break
			}
			if msgType != 1 { // text
				continue
			}
			var req struct {
				Type    string `json:"type"`
				Kind    string `json:"kind"`
				Payload string `json:"payload"`
				Source  string `json:"source"`
			}
			if err := json.Unmarshal(data, &req); err != nil || req.Type != "enqueue" {
				continue
			}
			if req.Kind == "" {
				req.Kind = store.TaskPrompt
			}
			if req.Source == "" {
				req.Source = "user"
			}
			task, err := deps.Store.EnqueueTask(agentID, req.Kind, req.Payload, req.Source)
			if err != nil {
				writeWSJSON(ws, map[string]any{"event": map[string]any{
					"type": "enqueue_rejected", "error": err.Error(),
				}})
				continue
			}
			writeWSJSON(ws, map[string]any{"event": map[string]any{
				"type": "enqueue_accepted", "taskId": task.ID, "kind": task.Kind,
			}})
		}

		unsub()
		<-writeDone
	})
}
