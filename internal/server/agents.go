package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os/exec"
	"time"

	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

// registerAgentRoutes wires managed-mode control (ADR-0006): one live pi
// process per agent — starting managed mode stops the interactive session
// and vice versa.
func registerAgentRoutes(mux *http.ServeMux, deps Deps) {
	mux.HandleFunc("POST /api/agents/{id}/managed/start", handleManagedStart(deps))
	mux.HandleFunc("POST /api/agents/{id}/managed/stop", handleManagedStop(deps))
	mux.HandleFunc("PATCH /api/agents/{id}", handlePatchAgent(deps))
	mux.HandleFunc("POST /api/agents/{id}/login", handleAgentLogin(deps))
	mux.HandleFunc("POST /api/agents/{id}/command", handleAgentCommand(deps))
	mux.HandleFunc("POST /api/agents/{id}/compact", handleAgentCompact(deps))
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

func handlePatchAgent(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		var req struct {
			Name        *string `json:"name"`
			Provider    *string `json:"provider"`
			Model       *string `json:"model"`
			Thinking    *string `json:"thinking"`
			OpMode      *string `json:"opMode"`
			SessionPath *string `json:"sessionPath"`
			ExtraPrompt *string `json:"extraPrompt"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		agent, err := deps.Store.UpdateAgent(id, store.AgentPatch{
			Name: req.Name, Provider: req.Provider, Model: req.Model,
			Thinking: req.Thinking, OpMode: req.OpMode, SessionPath: req.SessionPath, ExtraPrompt: req.ExtraPrompt,
		})
		if errors.Is(err, store.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "agent not found")
			return
		}
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, agent)
	}
}

// handleAgentLogin starts the interactive TUI (if needed) and types
// `/login [provider]` — credentials stay in pi (ADR-0009).
func handleAgentLogin(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		var req struct {
			Provider string `json:"provider"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)

		agent, err := deps.Store.GetAgent(id)
		if errors.Is(err, store.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "agent not found")
			return
		}
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		wk, err := deps.Store.GetWorkspace(agent.WorkspaceID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if !deps.Tmux.Available() {
			writeErr(w, http.StatusServiceUnavailable, "tmux is not installed")
			return
		}
		deps.Runtime.Stop(id)
		name := tmux.SessionName(id)
		has, err := deps.Tmux.HasSession(r.Context(), name)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if !has {
			if _, err := exec.LookPath(deps.AgentCmd); err != nil {
				writeErr(w, http.StatusServiceUnavailable, "pi is not installed or not on PATH")
				return
			}
			if err := deps.Tmux.NewSession(r.Context(), name, wk.Path, deps.AgentCmd, agent.CLIFlags()...); err != nil {
				writeErr(w, http.StatusInternalServerError, "start agent: "+err.Error())
				return
			}
			_ = deps.Store.SetAgentRuntime(id, store.StatusRunning)
		}
		cmd := "/login"
		if req.Provider != "" {
			cmd = "/login " + req.Provider
		}
		if err := deps.Tmux.SendKeys(r.Context(), name, cmd, "Enter"); err != nil {
			writeErr(w, http.StatusInternalServerError, "send /login: "+err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "session": name, "command": cmd})
	}
}

// handleAgentCommand types a native pi slash command into the interactive TUI.
func handleAgentCommand(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		var req struct {
			Text string `json:"text"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Text == "" {
			writeErr(w, http.StatusBadRequest, "text required")
			return
		}
		if req.Text[0] != '/' {
			writeErr(w, http.StatusBadRequest, "only slash commands")
			return
		}
		agent, err := deps.Store.GetAgent(id)
		if errors.Is(err, store.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "agent not found")
			return
		}
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		wk, err := deps.Store.GetWorkspace(agent.WorkspaceID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if !deps.Tmux.Available() {
			writeErr(w, http.StatusServiceUnavailable, "tmux is not installed")
			return
		}
		deps.Runtime.Stop(id)
		name := tmux.SessionName(id)
		has, err := deps.Tmux.HasSession(r.Context(), name)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if !has {
			if err := deps.Tmux.NewSession(r.Context(), name, wk.Path, deps.AgentCmd, agent.CLIFlags()...); err != nil {
				writeErr(w, http.StatusInternalServerError, "start agent: "+err.Error())
				return
			}
			_ = deps.Store.SetAgentRuntime(id, store.StatusRunning)
		}
		if err := deps.Tmux.SendKeys(r.Context(), name, req.Text, "Enter"); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "session": name, "command": req.Text})
	}
}

func handleAgentCompact(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		agent, err := deps.Store.GetAgent(id)
		if errors.Is(err, store.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "agent not found")
			return
		}
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		wk, err := deps.Store.GetWorkspace(agent.WorkspaceID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if deps.runMode(r, id) == modeInteractive {
			_ = deps.Tmux.KillSession(r.Context(), tmux.SessionName(id))
		}
		if deps.Runtime.Get(id) == nil {
			if err := deps.Runtime.Start(id, wk.Path); err != nil {
				writeErr(w, http.StatusInternalServerError, err.Error())
				return
			}
			_ = deps.Store.SetAgentRuntime(id, store.StatusRunning)
			select {
			case <-time.After(600 * time.Millisecond):
			case <-r.Context().Done():
				return
			}
		}
		ma := deps.Runtime.Get(id)
		if ma == nil {
			writeErr(w, http.StatusServiceUnavailable, "agent is not running in chat")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
		defer cancel()
		res, err := ma.Compact(ctx)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if !res.Success {
			msg := res.Error
			if msg == "" {
				msg = "compact failed"
			}
			writeErr(w, http.StatusBadRequest, msg)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": json.RawMessage(res.Data)})
	}
}
