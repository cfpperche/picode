package server

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/clilaunch"
	"github.com/cfpperche/picode/internal/clisession"
	"github.com/cfpperche/picode/internal/session"
	"github.com/cfpperche/picode/internal/store"
)

// Agent history (ADR-0205): a removed agent stays restorable while its
// transcript is on disk. The history is not a table of its own — it is the
// exit records (ADR-0194) whose session a clisession.Locator still finds.
// Restoring creates the agent again from the frozen setup and resumes that
// session; forgetting takes the exit out of the history and leaves the
// catalog and every file alone (a Pi session file may be deleted on
// request, since Pi's files are the only ones PiCode manages).

func registerAgentHistoryRoutes(mux Registrar, deps Deps) {
	mux.HandleFunc("GET /api/agent-history", handleAgentHistory(deps))
	mux.HandleFunc("POST /api/agent-history/{id}/restore", handleRestoreAgent(deps))
	mux.HandleFunc("POST /api/agent-history/{id}/forget", handleForgetAgentHistory(deps))
}

// historyEntry is one restorable agent: its exit, the session found on
// disk, and what a restore would meet.
type historyEntry struct {
	Exit    store.AgentExit    `json:"exit"`
	Session clisession.Summary `json:"session"`
	// Folder is where the agent's conversation ran and where a restore
	// starts it; FolderExists says whether it is still there.
	Folder       string `json:"folder"`
	FolderExists bool   `json:"folderExists"`
	// WorkspaceExists is false when the agent's workspace was removed; a
	// restore then needs another workspace.
	WorkspaceExists bool `json:"workspaceExists"`
	// CanDeleteFile: forgetting may also delete the transcript (Pi only).
	CanDeleteFile bool `json:"canDeleteFile"`
}

// locateExit finds the transcript an exit points at, or nil.
func locateExit(l *clisession.Locator, ex store.AgentExit) *clisession.Summary {
	cli := ex.CLI
	if cli == "" {
		cli = store.CLIPi
	}
	var sum *clisession.Summary
	var err error
	if cli == store.CLIPi {
		sum, err = l.Locate(cli, "", ex.Sessions.PiSessionPath, ex.Sessions.Cwd)
	} else {
		sum, err = l.Locate(cli, ex.Sessions.CLISessionID, ex.Sessions.CLISessionPath, ex.Sessions.Cwd)
	}
	if err != nil {
		return nil
	}
	return sum
}

// historyFolder is where a restored agent works: the folder its session
// ran in, else the one the exit recorded.
func historyFolder(ex store.AgentExit, sum *clisession.Summary) string {
	first := ""
	if sum != nil {
		first = sum.Cwd
	}
	for _, f := range []string{first, ex.Sessions.Cwd, ex.Config.WorkPath} {
		if f = strings.TrimSpace(f); f != "" {
			return f
		}
	}
	return ""
}

// underDir reports whether p is root or inside it.
func underDir(root, p string) bool {
	if root == "" || p == "" {
		return false
	}
	rel, err := filepath.Rel(filepath.Clean(root), filepath.Clean(p))
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func dirExists(p string) bool {
	if p == "" {
		return false
	}
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}

func handleAgentHistory(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		exits, err := deps.Store.AgentHistoryCandidates(0)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		live := map[string]bool{}
		if wss, err := deps.Store.ListWorkspaces(); err == nil {
			for _, ws := range wss {
				live[ws.ID] = true
			}
		}
		live[store.FreeWorkspaceID] = true
		l := clisession.NewAgentHistoryLocator(deps.DataDir)
		out := []historyEntry{}
		for _, ex := range exits {
			sum := locateExit(l, ex)
			if sum == nil {
				continue
			}
			folder := historyFolder(ex, sum)
			out = append(out, historyEntry{
				Exit:            ex,
				Session:         *sum,
				Folder:          folder,
				FolderExists:    dirExists(folder),
				WorkspaceExists: live[ex.WorkspaceID],
				CanDeleteFile:   ex.CLI == store.CLIPi && session.UnderRoot(session.Root(), sum.Path),
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{"entries": out})
	}
}

// restoreRequest: where the agent comes back (its own workspace when
// empty) and under which name (its own when empty).
type restoreRequest struct {
	WorkspaceID string `json:"workspaceId"`
	Name        string `json:"name"`
	// Undo is the removal toast's Undo: the agent comes back even with no
	// conversation to resume (it may never have had one), and a work
	// folder the removal purged is made again.
	Undo bool `json:"undo"`
}

// restoredOverrides rebuilds a CLI agent's launch from its exit. Values of
// environment variables were never recorded (secrets); their names come
// back in the answer so the person can set them again. Arguments are the
// resumed session's, applied at start from the pinned session.
func restoredOverrides(l *store.ExitLaunch) (clilaunch.Overrides, []string) {
	ov := clilaunch.Overrides{}
	if l == nil {
		return ov, nil
	}
	if l.Executable != "" {
		exe := l.Executable
		ov.Executable = &exe
	}
	if l.Path != nil {
		p := append([]string{}, l.Path...)
		ov.Path = &p
	}
	if l.Tools != nil {
		t := append([]string{}, l.Tools...)
		ov.Tools = &t
	}
	ov.Integration = l.Integration
	if len(l.RemovedEnv) > 0 {
		ov.Env = map[string]*string{}
		for _, k := range l.RemovedEnv {
			ov.Env[k] = nil
		}
	}
	return ov, append([]string{}, l.EnvKeys...)
}

// handleRestoreAgent brings a removed agent back (ADR-0205's table):
//
//	restored already, or its id taken → 409
//	undo (the removal toast)          → as below, and with no transcript
//	                                    the agent comes back without one
//	transcript not on disk           → 410
//	workspace gone, none chosen      → 409 workspace_gone
//	folder gone                      → 409 folder_gone
//	CLI not launchable               → the launch pre-flight's answer
//	Pi                               → agent + its session file
//	any other CLI                    → agent + terminal, session pinned;
//	                                   the client starts it with resume
func handleRestoreAgent(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req restoreRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<10)).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
			writeErr(w, http.StatusBadRequest, "invalid request body")
			return
		}
		id := r.PathValue("id")
		unlock := terminalLock(deps, "exit:"+id)
		defer unlock()
		ex, err := deps.Store.GetAgentExit(id)
		if err != nil {
			writeExitErr(w, err)
			return
		}
		if ex.UndoneAt != nil {
			writeErr(w, http.StatusConflict, "This agent was already brought back.")
			return
		}
		sum := locateExit(clisession.NewAgentHistoryLocator(deps.DataDir), ex)
		if sum == nil && !req.Undo {
			writeErr(w, http.StatusGone, "Its conversation is no longer on disk, so there is nothing to resume.")
			return
		}
		// It comes back as itself (ADR-0205): what still names the id — its
		// Pi session folder, automations, pins — points at it again.
		if _, err := deps.Store.GetAgent(ex.AgentID); err == nil {
			writeErr(w, http.StatusConflict, "An agent with its id is already here.")
			return
		}
		wsID := strings.TrimSpace(req.WorkspaceID)
		if wsID == "" {
			wsID = ex.WorkspaceID
		}
		wk, err := deps.Store.GetWorkspace(wsID)
		if errors.Is(err, store.ErrNotFound) {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "Its workspace was removed. Choose where to bring it back.", "code": "workspace_gone"})
			return
		} else if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		folder := historyFolder(ex, sum)
		if folder == "" && req.Undo {
			folder = store.AgentCwd(wk, store.Agent{ID: ex.AgentID})
		}
		if !dirExists(folder) && req.Undo && underDir(filepath.Join(deps.DataDir, "work"), folder) {
			// The removal purged the agent's private folder; Undo brings
			// back the agent, and it needs somewhere to work.
			_ = os.MkdirAll(folder, 0o755)
		}
		if !dirExists(folder) {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "The folder it worked in is gone: " + folder, "code": "folder_gone"})
			return
		}
		name := strings.TrimSpace(req.Name)
		if name == "" {
			name = ex.AgentName
		}
		cli := ex.CLI
		if cli == "" {
			cli = store.CLIPi
		}
		// The agent works where its conversation ran. In a workspace whose
		// folder that is, it needs no folder of its own.
		work := folder
		if !store.IsFree(wk) && filepath.Clean(folder) == filepath.Clean(wk.Path) {
			work = ""
		}
		var ov *clilaunch.Overrides
		var envKeys []string
		if cli != store.CLIPi || (ex.Config.Launch != nil && ex.Config.Launch.HasOverrides) {
			o, keys := restoredOverrides(ex.Config.Launch)
			ov, envKeys = &o, keys
		}
		if status, err := checkAgentLaunch(deps, cli, folder, ov); err != nil {
			writeErr(w, status, err.Error())
			return
		}
		agent, status, err := newLaunchAgentAs(deps, ex.AgentID, wk.ID, folder, cli, name, work, ov)
		if err != nil {
			writeErr(w, status, err.Error())
			return
		}
		resume := false
		if agent.IsPi() {
			c := ex.Config
			patch := store.AgentPatch{Checklist: &c.Checklist}
			if sum != nil {
				patch.SessionPath = &sum.Path
			}
			for _, f := range []struct {
				dst **string
				v   string
			}{{&patch.Provider, ex.Provider}, {&patch.Model, ex.Model}, {&patch.Thinking, c.Thinking}, {&patch.OpMode, c.OpMode}, {&patch.ExtraPrompt, c.ExtraPrompt}} {
				if f.v != "" {
					v := f.v
					*f.dst = &v
				}
			}
			if c.PackagesIsolated {
				iso := true
				patch.PackagesIsolated = &iso
			}
			agent, err = deps.Store.UpdateAgent(agent.ID, patch)
		} else if agent.TerminalID != nil && sum != nil {
			err = deps.Store.SetTerminalLastSession(*agent.TerminalID, store.TerminalLastSession{
				CLI: cli, SessionID: sum.ID, Path: sum.Path, Cwd: folder, Name: sum.Name,
				UpdatedAt: sum.UpdatedAt, Preview: sum.Preview, ResumeArgs: sum.ResumeArgs,
			})
			resume = err == nil
		}
		// The agent's own skills come back with it; their cached folders are
		// never swept, so the launch finds them.
		if err == nil && len(ex.Config.Skills) > 0 {
			if withSkills, e := deps.Store.SetAgentSkills(agent.ID, ex.Config.Skills); e != nil {
				err = e
			} else {
				agent = withSkills
			}
		}
		if err != nil {
			_ = deps.Store.DeleteAgent(agent.ID)
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if _, err := deps.Store.MarkAgentExitUndone(ex.ID, agent.ID); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{
			"agent":      agentView{Agent: agent, Mode: string(modeStopped)},
			"terminalId": deref(agent.TerminalID),
			"resume":     resume,
			"envKeys":    envKeys,
		})
	}
}

// handleForgetAgentHistory takes an exit out of the history. With
// deleteFile a Pi transcript is deleted too — only under Pi's sessions
// root and only when no living agent is bound to it.
func handleForgetAgentHistory(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			DeleteFile bool `json:"deleteFile"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<10)).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
			writeErr(w, http.StatusBadRequest, "invalid request body")
			return
		}
		id := r.PathValue("id")
		unlock := terminalLock(deps, "exit:"+id)
		defer unlock()
		ex, err := deps.Store.GetAgentExit(id)
		if err != nil {
			writeExitErr(w, err)
			return
		}
		if req.DeleteFile {
			path := ex.Sessions.PiSessionPath
			if ex.CLI != store.CLIPi || path == "" || !session.UnderRoot(session.Root(), path) {
				writeErr(w, http.StatusBadRequest, "Only a Pi session file can be deleted from here; other CLIs keep their own files.")
				return
			}
			if agents, err := deps.Store.ListAllAgents(); err == nil {
				for _, a := range agents {
					if a.SessionPath != nil && filepath.Clean(*a.SessionPath) == filepath.Clean(path) {
						writeErr(w, http.StatusConflict, "\""+a.Name+"\" is using this conversation now.")
						return
					}
				}
			}
			if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
				writeErr(w, http.StatusInternalServerError, err.Error())
				return
			}
		}
		ex, err = deps.Store.ForgetAgentExit(id)
		if err != nil {
			writeExitErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, ex)
	}
}

// piSessionFallback is the newest session in a Pi agent's private folder,
// for an agent that never bound one (lazy binding), so its exit still
// points at the conversation.
func piSessionFallback(agent store.Agent) string {
	if !agent.IsPi() || (agent.SessionPath != nil && *agent.SessionPath != "") {
		return ""
	}
	list, err := session.ListDir(session.AgentDir(agent.ID))
	if err != nil {
		return ""
	}
	best := ""
	var at time.Time
	for _, s := range list {
		t, err := time.Parse(time.RFC3339Nano, s.UpdatedAt)
		if err != nil {
			t, _ = time.Parse(time.RFC3339, s.UpdatedAt)
		}
		if best == "" || t.After(at) {
			best, at = s.Path, t
		}
	}
	return best
}
