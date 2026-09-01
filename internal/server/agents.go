package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/gitinfo"
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
	mux.HandleFunc("POST /api/agents/{id}/abort", handleAgentAbort(deps))
	mux.HandleFunc("POST /api/agents/{id}/prompt", handleAgentPrompt(deps))
	mux.HandleFunc("POST /api/agents/{id}/ui", handleAgentUI(deps))
}

// agentRunMode reports how an agent is currently running.
type agentRunMode string

const (
	modeStopped     agentRunMode = "stopped"
	modeInteractive agentRunMode = "interactive"
	modeManaged     agentRunMode = "managed"
)

func (deps Deps) agentHome(agent store.Agent) (store.Workspace, string, error) {
	wk, err := deps.Store.GetWorkspace(agent.WorkspaceID)
	if err != nil {
		return store.Workspace{}, "", err
	}
	cwd := store.AgentCwd(wk, agent)
	_ = os.MkdirAll(cwd, 0o755)
	return wk, cwd, nil
}

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

// agentInteractive is runMode narrowed to the one question the inbox
// needs (ADR-0037): is this agent currently in a TUI/tmux session with
// no delivery loop watching it? Managed and stopped both answer false —
// managed drains immediately, stopped is the legitimate park-and-wake
// case (the queue drains on the next managed start).
func (deps Deps) agentInteractive(ctx context.Context, agentID string) bool {
	if deps.Runtime.Get(agentID) != nil {
		return false
	}
	if !deps.Tmux.Available() {
		return false
	}
	has, err := deps.Tmux.HasSession(ctx, tmux.SessionName(agentID))
	return err == nil && has
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
		wk, cwd, err := deps.agentHome(agent)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if err := deps.Runtime.Start(agentID, cwd); err != nil {
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

func handleAgentAbort(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		agentID := r.PathValue("id")
		ma := deps.Runtime.Get(agentID)
		if ma == nil {
			writeErr(w, http.StatusConflict, "agent is not running")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
		defer cancel()
		if err := ma.Abort(ctx); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
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
		snap := ma.Snapshot()
		ev := map[string]any{
			"type": "snapshot", "mode": "managed",
			"streaming": snap.Streaming, "waiting": snap.Waiting,
		}
		if snap.Dialog != nil {
			ev["dialog"] = snap.Dialog
		}
		writeWSJSON(ws, map[string]any{"event": ev})

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
			Name             *string `json:"name"`
			Provider         *string `json:"provider"`
			Model            *string `json:"model"`
			Thinking         *string `json:"thinking"`
			OpMode           *string `json:"opMode"`
			SessionPath      *string `json:"sessionPath"`
			ExtraPrompt      *string `json:"extraPrompt"`
			PackagesIsolated *bool   `json:"packagesIsolated"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		agent, err := deps.Store.UpdateAgent(id, store.AgentPatch{
			Name: req.Name, Provider: req.Provider, Model: req.Model,
			Thinking: req.Thinking, OpMode: req.OpMode, SessionPath: req.SessionPath, ExtraPrompt: req.ExtraPrompt,
			PackagesIsolated: req.PackagesIsolated,
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
			if err := deps.Tmux.NewSessionEnv(r.Context(), name, store.AgentCwd(wk, agent), agent.SpawnEnv(), deps.AgentCmd, deps.spawnFlags(agent)...); err != nil {
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
			if err := deps.Tmux.NewSessionEnv(r.Context(), name, store.AgentCwd(wk, agent), agent.SpawnEnv(), deps.AgentCmd, deps.spawnFlags(agent)...); err != nil {
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
			if err := deps.Runtime.Start(id, store.AgentCwd(wk, agent)); err != nil {
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
			low := strings.ToLower(err.Error())
			if strings.Contains(low, "already compacted") || strings.Contains(low, "too small") {
				writeJSON(w, http.StatusOK, map[string]any{"ok": true, "already": true})
				return
			}
			writeErr(w, http.StatusBadRequest, err.Error())
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

func handleListFreeAgents(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("free") != "1" {
			writeErr(w, http.StatusBadRequest, "pass ?free=1")
			return
		}
		agents, err := deps.Store.ListAgents(store.FreeWorkspaceID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		out := make([]agentView, 0, len(agents))
		for _, a := range agents {
			mode := deps.runMode(r, a.ID)
			cwd := ""
			if a.WorkPath != nil {
				cwd = *a.WorkPath
			}
			out = append(out, agentView{Agent: a, Running: mode != modeStopped, Mode: string(mode), Git: gitinfo.Inspect(cwd)})
		}
		writeJSON(w, http.StatusOK, out)
	}
}

func handleAddFreeAgent(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name     string `json:"name"`
			Path     string `json:"path"`
			Provider string `json:"provider"`
			Model    string `json:"model"`
			Thinking string `json:"thinking"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		dir, err := resolveAgentWorkDir(deps, req.Path, req.Name)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		agent, err := deps.Store.AddAgent(store.FreeWorkspaceID, req.Name, dir)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		agent, err = patchNewAgent(deps, agent, req.Provider, req.Model, req.Thinking)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, agentView{Agent: agent, Mode: string(modeStopped)})
	}
}

func handleAddWorkspaceAgent(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		wsID := r.PathValue("id")
		if wsID == store.FreeWorkspaceID {
			writeErr(w, http.StatusBadRequest, "use POST /api/agents for free agents")
			return
		}
		if _, err := deps.Store.GetWorkspace(wsID); errors.Is(err, store.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "workspace not found")
			return
		} else if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		var req struct {
			Name     string `json:"name"`
			WorkPath string `json:"workPath"`
			Provider string `json:"provider"`
			Model    string `json:"model"`
			Thinking string `json:"thinking"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		// An empty workPath keeps the agent on the workspace folder, which is
		// what store.AgentCwd does with "". A path sent explicitly goes through
		// the same resolver free agents use — including its MkdirAll, so the
		// two creation paths cannot drift apart. This is what lets a workspace
		// hold agents in sibling worktrees (ADR-0022).
		work := ""
		if strings.TrimSpace(req.WorkPath) != "" {
			resolved, rerr := resolveAgentWorkDir(deps, req.WorkPath, req.Name)
			if rerr != nil {
				writeErr(w, http.StatusBadRequest, rerr.Error())
				return
			}
			work = resolved
		}
		agent, err := deps.Store.AddAgent(wsID, req.Name, work)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		agent, err = patchNewAgent(deps, agent, req.Provider, req.Model, req.Thinking)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, agentView{Agent: agent, Mode: string(modeStopped)})
	}
}

func patchNewAgent(deps Deps, agent store.Agent, provider, model, thinking string) (store.Agent, error) {
	if provider == "" && model == "" && thinking == "" {
		return agent, nil
	}
	return deps.Store.UpdateAgent(agent.ID, store.AgentPatch{
		Provider: &provider, Model: &model, Thinking: &thinking,
	})
}

func handleDeleteAgent(deps Deps) http.HandlerFunc {
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
		_, cwd, err := deps.agentCwd(agent)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		preview := deps.previewCleanup(cwd, map[string]bool{agent.ID: true})
		deps.stopAgent(r.Context(), id)
		if err := deps.Store.DeleteAgent(id); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		deps.applyCleanup(preview, queryFlag(r, "sessions"), queryFlag(r, "work"))
		w.WriteHeader(http.StatusNoContent)
	}
}

func handleAgentOpen(deps Deps) http.HandlerFunc {
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
		wk, cwd, err := deps.agentHome(agent)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		name := tmux.SessionName(agent.ID)
		deps.Runtime.Stop(agent.ID)
		if has, err := deps.Tmux.HasSession(r.Context(), name); err == nil && has {
			_ = deps.Store.SetAgentRuntime(agent.ID, store.StatusRunning)
			writeJSON(w, http.StatusOK, map[string]any{"running": true, "alreadyRunning": true, "session": name})
			return
		}
		if _, err := exec.LookPath(deps.AgentCmd); err != nil {
			writeErr(w, http.StatusServiceUnavailable,
				"pi is not installed or not on PATH — install it with: npm install -g @earendil-works/pi-coding-agent")
			return
		}
		if err := deps.Tmux.NewSessionEnv(r.Context(), name, cwd, agent.SpawnEnv(), deps.AgentCmd, deps.spawnFlags(agent)...); err != nil {
			_ = deps.Store.SetAgentRuntime(agent.ID, store.StatusStopped)
			writeErr(w, http.StatusInternalServerError, "start agent: "+err.Error())
			return
		}
		_ = deps.Store.SetAgentRuntime(agent.ID, store.StatusRunning)
		_ = deps.Store.AppendEvent("agent_started", &agent.ID, &wk.ID, map[string]string{"session": name})
		writeJSON(w, http.StatusCreated, map[string]any{"running": true, "session": name})
	}
}

func handleAgentClose(deps Deps) http.HandlerFunc {
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
		deps.Runtime.Stop(id)
		if deps.Tmux.Available() {
			if err := deps.Tmux.KillSession(r.Context(), tmux.SessionName(id)); err != nil {
				writeErr(w, http.StatusInternalServerError, "stop agent: "+err.Error())
				return
			}
		}
		_ = deps.Store.SetAgentRuntime(id, store.StatusStopped)
		wid := agent.WorkspaceID
		_ = deps.Store.AppendEvent("agent_stopped", &agent.ID, &wid, nil)
		writeJSON(w, http.StatusOK, map[string]any{"running": false})
	}
}

func resolveAgentWorkDir(deps Deps, path, name string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		root := deps.DataDir
		if root == "" {
			h, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			root = filepath.Join(h, ".picode")
		}
		path = filepath.Join(root, "work", slugDir(name))
	}
	if strings.HasPrefix(path, "~/") {
		h, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		path = filepath.Join(h, path[2:])
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return "", err
	}
	st, err := os.Stat(abs)
	if err != nil {
		return "", err
	}
	if !st.IsDir() {
		return "", errors.New("work path is not a directory")
	}
	return abs, nil
}

func slugDir(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(name)) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		} else if r == ' ' || r == '_' {
			b.WriteByte('-')
		}
	}
	s := strings.Trim(b.String(), "-")
	if s == "" {
		s = "agent"
	}
	return s
}
