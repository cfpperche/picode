package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/cfpperche/picode/internal/gitinfo"
	"github.com/cfpperche/picode/internal/osopen"
	"github.com/cfpperche/picode/internal/rpc"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

// workspaceView is a workspace plus its agents (ADR-0011). Agent is a
// pointer on purpose: a workspace can be empty (ADR-0027), and a zero-value
// object here would read as a truthy agent with an empty id in the UI.
type workspaceView struct {
	store.Workspace
	Agent      *agentView    `json:"agent,omitempty"` // first agent; kept for older clients
	Agents     []agentView   `json:"agents"`
	Git        *gitinfo.Info `json:"git,omitempty"`
	HasFavicon bool          `json:"hasFavicon"`
	// The menu's ways out of PiCode: the repository's web page (nil for a
	// plain folder or a remote with no web host) and, under WSL, the path
	// Windows uses for the folder (Copy path offers both).
	Remote  *gitinfo.Remote `json:"remote,omitempty"`
	WinPath string          `json:"winPath,omitempty"`
}

type agentView struct {
	MissionID string `json:"missionId,omitempty"`
	store.Agent
	Running bool          `json:"running"`
	Mode    string        `json:"mode"` // stopped | interactive | managed (ADR-0006)
	Git     *gitinfo.Info `json:"git,omitempty"`
	// Live run state from the managed runtime's snapshot (ADR-0044): the
	// mobile Now screen learns who is streaming or blocked on a dialog
	// from the workspace list alone, with no socket per agent. Dialog is
	// the open prompt itself, so the phone can answer it through
	// POST /api/agents/{id}/ui without ever subscribing.
	Streaming         bool           `json:"streaming"`
	Waiting           bool           `json:"waiting"`
	Dialog            *rpc.UIDialog  `json:"dialog,omitempty"`
	Terminal          map[string]any `json:"terminal,omitempty"`
	LegacyInteractive bool           `json:"legacyInteractive,omitempty"`
	// ForkedFrom names the agent this one was forked from (Fork agent…).
	ForkedFrom *forkOrigin `json:"forkedFrom,omitempty"`
}

func asAgentView(a store.Agent, running bool) agentView {
	return agentView{Agent: a, Running: running}
}

// liveState is the runtime snapshot for a managed agent; zero for anything
// else (stopped, or in a tmux TUI, which has no event channel).
func (deps Deps) liveState(agentID string) (streaming, waiting bool, dialog *rpc.UIDialog) {
	ma := deps.Runtime.Get(agentID)
	if ma == nil {
		return false, false, nil
	}
	s := ma.Snapshot()
	return s.Streaming, s.Waiting, s.Dialog
}

func registerWorkspaceRoutes(mux Registrar, deps Deps) {
	mux.HandleFunc("GET /api/workspaces", handleList(deps))
	mux.HandleFunc("GET /api/workspaces/{id}/stats", handleWorkspaceStats(deps))
	mux.HandleFunc("POST /api/workspaces", handleAdd(deps))
	registerSidebarOrderRoutes(mux, deps)
	mux.HandleFunc("DELETE /api/workspaces/{id}", handleRemove(deps))
	mux.HandleFunc("PATCH /api/workspaces/{id}", handleRenameWorkspace(deps))
	mux.HandleFunc("GET /api/workspaces/{id}/cleanup", handleWorkspaceCleanup(deps))
	mux.HandleFunc("GET /api/workspaces/{id}/favicon", handleWorkspaceFavicon(deps))
	mux.HandleFunc("POST /api/workspaces/{id}/open", handleOpen(deps))
	mux.HandleFunc("POST /api/workspaces/{id}/close", handleClose(deps))
	mux.HandleFunc("GET /api/workspaces/{id}/status", handleWorkspaceStatus(deps))
	mux.HandleFunc("GET /api/workspaces/{id}/sessions/transcript", handleSessionTranscript(deps))
	mux.HandleFunc("GET /api/workspaces/{id}/sessions", handleListSessions(deps))
	mux.HandleFunc("POST /api/workspaces/{id}/sessions/new", handleNewSession(deps))
	mux.HandleFunc("POST /api/workspaces/{id}/sessions/resume", handleResumeSession(deps))
	mux.HandleFunc("POST /api/workspaces/{id}/sessions/rename", handleRenameSession(deps))
	mux.HandleFunc("POST /api/agents/{id}/tasks", handleEnqueueTask(deps))
	mux.HandleFunc("GET /api/agents/{id}/tasks", handleListTasks(deps))
	mux.HandleFunc("POST /api/workspaces/{id}/agents", handleAddWorkspaceAgent(deps))
	mux.HandleFunc("GET /api/workspaces/{id}/principals", handleListWorkspacePrincipals(deps))
	mux.HandleFunc("POST /api/workspaces/{id}/principals", handleAddWorkspacePrincipal(deps))
	mux.HandleFunc("DELETE /api/managed-clis/{id}", handleDeleteAgent(deps)) // the alias stops the CLI first, like the agent route
	mux.HandleFunc("GET /api/agents", handleListFreeAgents(deps))
	mux.HandleFunc("POST /api/agents", handleAddFreeAgent(deps))
	mux.HandleFunc("DELETE /api/agents/{id}", handleDeleteAgent(deps))
	mux.HandleFunc("GET /api/agents/{id}/cleanup", handleAgentCleanup(deps))
	mux.HandleFunc("POST /api/agents/{id}/open", handleAgentOpen(deps))
	mux.HandleFunc("POST /api/agents/{id}/close", handleAgentClose(deps))
	registerAgentRoutes(mux, deps)
}

func (deps Deps) view(r *http.Request, w store.Workspace) (workspaceView, error) {
	agents, err := deps.Store.ListAgents(w.ID)
	if err != nil {
		return workspaceView{}, err
	}
	views := make([]agentView, 0, len(agents))
	origins := forkOrigins(deps)
	for _, a := range agents {
		mode := deps.runMode(r, a.ID)
		// Each agent carries its own git facts, read from its EFFECTIVE
		// directory: an agent with a workPath may sit in a different repo (or
		// branch, via a worktree) than the workspace it belongs to, and the
		// sidebar line is about the agent, not its container.
		st, wt, dl := deps.liveState(a.ID)
		views = append(views, agentView{MissionID: deps.Store.AgentMissionID(a.ID), Agent: a, Running: mode != modeStopped, Mode: string(mode),
			Git: gitinfo.Inspect(store.AgentCwd(w, a)), Streaming: st, Waiting: wt, Dialog: dl, Terminal: deps.agentTerminalView(r, a), LegacyInteractive: a.TerminalID != nil && deps.agentSession(a.ID) == tmux.SessionName(a.ID), ForkedFrom: origins[a.ID]})
	}
	var first *agentView
	if len(views) > 0 {
		first = &views[0]
	}
	v := workspaceView{Workspace: w, Agent: first, Agents: views, Git: gitinfo.Inspect(w.Path), HasFavicon: workspaceHasFavicon(w.Path)}
	if v.Git != nil {
		v.Remote = gitinfo.RemoteOf(w.Path)
	}
	v.WinPath, _ = osopen.WindowsPath(w.Path)
	return v, nil
}

func handleList(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ws, err := deps.Store.ListWorkspaces()
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		views := make([]workspaceView, 0, len(ws))
		for _, wk := range ws {
			v, err := deps.view(r, wk)
			if err != nil {
				writeErr(w, http.StatusInternalServerError, err.Error())
				return
			}
			views = append(views, v)
		}
		writeJSON(w, http.StatusOK, views)
	}
}

func handleAdd(deps Deps) http.HandlerFunc {
	var req struct {
		Name string `json:"name"`
		Path string `json:"path"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		// The workspace starts empty (ADR-0027); agents come from
		// POST /api/workspaces/{id}/agents. An idempotent re-add answers
		// with the existing workspace and whatever agents it really has.
		wk, err := deps.Store.AddWorkspace(req.Name, req.Path)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		v, err := deps.view(r, wk)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, v)
	}
}

func handleRemove(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		// The optional body names the face that removed it (ADR-0194).
		req, err := readExitRequest(w, r)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		unlockWorkspace := terminalLock(deps, "workspace:"+id)
		defer unlockWorkspace()
		if id == store.FreeWorkspaceID {
			writeErr(w, http.StatusBadRequest, "cannot remove the free workspace")
			return
		}
		wk, err := deps.Store.GetWorkspace(id)
		if errors.Is(err, store.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "workspace not found")
			return
		}
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		// Opt-in local deletion (ADR-0035): validated before anything is
		// stopped or removed, so a refused delete leaves the workspace whole.
		deleteFiles := queryFlag(r, "files")
		if deleteFiles {
			if err := checkFolderDeletable(wk.Path, r.URL.Query().Get("confirm")); err != nil {
				writeErr(w, http.StatusBadRequest, err.Error())
				return
			}
		}
		agents, err := deps.Store.ListAgents(wk.ID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		dying := map[string]bool{}
		for _, a := range agents {
			dying[a.ID] = true
			unlock := terminalLock(deps, "agent:"+a.ID)
			defer unlock()
			if err := deps.stopAgentLocked(r.Context(), a.ID); err != nil {
				writeErr(w, 500, err.Error())
				return
			}
		}
		// The workspace's terminals go with it (ADRs 0026/0070): stop each
		// session before cleanup. Failure leaves the records available for
		// retry; successful removal cascades their settings in one transaction.
		terms, err := deps.Store.ListWorkspaceTerminals(wk.ID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if deps.Tmux != nil && deps.Tmux.Available() {
			for _, t := range terms {
				unlock := terminalLock(deps, t.ID)
				defer unlock()
				if err := deps.Tmux.KillSession(r.Context(), tmux.ShellSessionName(t.ID)); err != nil {
					writeErr(w, 500, "Could not stop a workspace terminal. Try removing the workspace again.")
					return
				}
			}
		}
		for _, t := range terms {
			if deps.DataDir != "" {
				if err := cleanCLILaunches(deps.DataDir, t.ID, ""); err != nil {
					writeErr(w, 500, "Could not remove terminal launch files. Try again.")
					return
				}
			}
			if deps.TermStates != nil {
				deps.TermStates.Drop(t.ID)
			}
			if deps.TermRuntimes != nil {
				deps.TermRuntimes.Drop(t.ID)
			}
		}
		preview := deps.previewCleanup(wk.Path, dying)
		// Its agents end with it: each gets an unasked exit (ADR-0194).
		exitIn := store.ExitInput{
			Meter:             exitMeter(),
			PiSessionFallback: piSessionFallback,
			Origin:            exitOriginOf(req),
			SessionsPurged:    queryFlag(r, "sessions"),
			WorkPurged:        queryFlag(r, "work") && preview.LastOccupant && preview.CanPurgeWork,
		}
		removed, _, err := deps.Store.RemoveWorkspaceWithExits(wk.ID, exitIn)
		if err != nil || !removed {
			writeErr(w, http.StatusInternalServerError, "remove failed")
			return
		}
		deps.applyCleanup(preview, queryFlag(r, "sessions"), queryFlag(r, "work"))
		if deleteFiles {
			// The record is gone either way; a failed delete reports what is
			// left on disk instead of pretending. The remote repository (if
			// any) is never touched — this is a local rm only.
			if err := os.RemoveAll(wk.Path); err != nil {
				writeErr(w, http.StatusInternalServerError,
					"workspace removed, but deleting the folder failed: "+err.Error())
				return
			}
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// checkFolderDeletable is the server-side gate for deleting a workspace's
// folder: the typed confirmation must match the folder's name, and a few
// paths are never deletable no matter what was typed.
func checkFolderDeletable(path, confirm string) error {
	abs, err := filepath.Abs(strings.TrimSpace(path))
	if err != nil || abs == "" {
		return errors.New("workspace path is not usable")
	}
	base := filepath.Base(abs)
	if abs == string(filepath.Separator) || base == string(filepath.Separator) || base == "." {
		return errors.New("refusing to delete the filesystem root")
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		if h, err := filepath.Abs(home); err == nil && h == abs {
			return errors.New("refusing to delete the home folder")
		}
	}
	if strings.TrimSpace(confirm) != base {
		return errors.New("type the folder name (" + base + ") to confirm deleting local data")
	}
	return nil
}

func handleOpen(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		wk, err := deps.Store.GetWorkspace(id)
		if errors.Is(err, store.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "workspace not found")
			return
		}
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		agent, err := deps.Store.DefaultAgent(wk.ID)
		if errors.Is(err, store.ErrNotFound) {
			writeErr(w, http.StatusConflict, "workspace has no agents — add one first")
			return
		}
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}

		already, err := deps.openAgentTUI(r.Context(), agent.ID, false)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		code := http.StatusCreated
		if already {
			code = http.StatusOK
		}
		writeJSON(w, code, map[string]any{"running": true, "alreadyRunning": already, "session": deps.agentSession(agent.ID)})
	}
}

func handleClose(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		wk, err := deps.Store.GetWorkspace(id)
		if errors.Is(err, store.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "workspace not found")
			return
		}
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		agent, err := deps.Store.DefaultAgent(wk.ID)
		if errors.Is(err, store.ErrNotFound) {
			writeErr(w, http.StatusConflict, "workspace has no agents — add one first")
			return
		}
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		unlockAgent := terminalLock(deps, "agent:"+agent.ID)
		defer unlockAgent()
		release := deps.Replies.Controls.BeginMutation(agent.ID)
		defer release()
		if err := deps.stopAgentInteractive(r.Context(), agent.ID); err != nil {
			writeErr(w, http.StatusInternalServerError, "stop agent: "+err.Error())
			return
		}
		_ = deps.Store.SetAgentRuntime(agent.ID, store.StatusStopped)
		_ = deps.Store.AppendEvent("agent_stopped", &agent.ID, &wk.ID, nil)
		writeJSON(w, http.StatusOK, map[string]any{"running": false})
	}
}

// handleEnqueueTask queues a prompt/steer/follow_up for an agent.
// Delivery engine lands with the M2 RPC bridge; until then tasks stay
// `queued` (documented in docs/handoff/open/communication.md).
func handleEnqueueTask(deps Deps) http.HandlerFunc {
	var req struct {
		Kind    string `json:"kind"`
		Payload string `json:"payload"`
		Source  string `json:"source"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		agentID := r.PathValue("id")
		if _, err := deps.Store.GetAgent(agentID); errors.Is(err, store.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "agent not found")
			return
		} else if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if req.Kind == "" {
			req.Kind = store.TaskPrompt
		}
		task, err := deps.Store.EnqueueTask(agentID, req.Kind, req.Payload, req.Source)
		if errors.Is(err, store.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "agent not found")
			return
		}
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, task)
	}
}

func handleListTasks(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		agentID := r.PathValue("id")
		if _, err := deps.Store.GetAgent(agentID); errors.Is(err, store.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "agent not found")
			return
		} else if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		tasks, err := deps.Store.ListTasks(agentID, 50)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, tasks)
	}
}

// handleRenameWorkspace changes the name on the card (the Settings dialog's
// General section). Only the name: the folder is the workspace's identity.
func handleRenameWorkspace(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name string `json:"name"`
		}
		dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<10))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		ws, err := deps.Store.RenameWorkspace(r.PathValue("id"), req.Name)
		switch {
		case errors.Is(err, store.ErrNotFound):
			writeErr(w, http.StatusNotFound, "workspace not found")
		case err != nil && strings.Contains(err.Error(), "name is"):
			writeErr(w, http.StatusBadRequest, strings.TrimPrefix(err.Error(), "store: "))
		case err != nil:
			writeErr(w, http.StatusInternalServerError, err.Error())
		default:
			writeJSON(w, http.StatusOK, ws)
		}
	}
}
