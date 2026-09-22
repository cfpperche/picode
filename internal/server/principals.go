package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/cfpperche/picode/internal/clilaunch"
	"github.com/cfpperche/picode/internal/grant"
	"github.com/cfpperche/picode/internal/store"
)

// principalView is one workspace actor (ADR-0160): an agents row. Kind/id/key
// match grant.Principal. CLI agents carry cli and the interactive terminal.
type principalView struct {
	Kind        string `json:"kind"`
	ID          string `json:"id"`
	Key         string `json:"key"`
	Name        string `json:"name"`
	WorkspaceID string `json:"workspaceId"`
	CLI         string `json:"cli,omitempty"`
	AgentID     string `json:"agentId,omitempty"`
	TerminalID  string `json:"terminalId,omitempty"`
}

func agentPrincipal(a store.Agent) principalView {
	p := grant.Principal{Kind: grant.KindAgent, ID: a.ID}
	v := principalView{
		Kind:        string(p.Kind),
		ID:          p.ID,
		Key:         p.Key(),
		Name:        a.Name,
		WorkspaceID: a.WorkspaceID,
		CLI:         a.CLI,
		AgentID:     a.ID,
	}
	if a.TerminalID != nil {
		v.TerminalID = *a.TerminalID
	}
	return v
}

func handleListWorkspacePrincipals(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		wsID := r.PathValue("id")
		if _, err := deps.Store.GetWorkspace(wsID); err != nil {
			writeErr(w, storeStatus(err), err.Error())
			return
		}
		agents, err := deps.Store.ListAgents(wsID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		out := make([]principalView, 0, len(agents))
		for _, a := range agents {
			out = append(out, agentPrincipal(a))
		}
		writeJSON(w, http.StatusOK, out)
	}
}

func handleAddWorkspacePrincipal(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		wsID := r.PathValue("id")
		if wsID == store.FreeWorkspaceID {
			writeErr(w, http.StatusBadRequest, "A managed CLI belongs to a workspace.")
			return
		}
		wk, err := deps.Store.GetWorkspace(wsID)
		if err != nil {
			writeErr(w, storeStatus(err), err.Error())
			return
		}
		var req struct {
			CLI        string `json:"cli"`
			Name       string `json:"name"`
			TerminalID string `json:"terminalId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		cli := strings.TrimSpace(req.CLI)
		entry, ok := clilaunch.Find(cli)
		if !ok || !entry.Launchable() {
			writeErr(w, http.StatusBadRequest, "Unknown CLI.")
			return
		}
		agent, err := deps.Store.AddAgentWithCLI(wsID, cli, strings.TrimSpace(req.Name), "")
		if err != nil {
			writeErr(w, storeStatus(err), err.Error())
			return
		}
		terminalID := strings.TrimSpace(req.TerminalID)
		if agent.IsPi() && terminalID == "" {
			writeJSON(w, http.StatusCreated, agentPrincipal(agent))
			return
		}
		if terminalID == "" {
			bound, err := attachAgentTerminal(deps, wk.ID, wk.Path, agent)
			if err != nil {
				_ = deps.Store.DeleteAgent(agent.ID)
				writeErr(w, storeStatus(err), err.Error())
				return
			}
			writeJSON(w, http.StatusCreated, agentPrincipal(bound))
			return
		}
		tm, err := deps.Store.GetTerminal(terminalID)
		if err != nil {
			_ = deps.Store.DeleteAgent(agent.ID)
			writeErr(w, storeStatus(err), err.Error())
			return
		}
		if tm.WorkspaceID != wk.ID {
			_ = deps.Store.DeleteAgent(agent.ID)
			writeErr(w, http.StatusBadRequest, "That terminal belongs to another workspace.")
			return
		}
		launch, err := deps.Store.TerminalLaunch(terminalID)
		if err != nil {
			_ = deps.Store.DeleteAgent(agent.ID)
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if launch == nil {
			if err := deps.Store.SetTerminalLaunch(terminalID, agent.CLI, managedCLILaunchOverrides(agent.CLI)); err != nil {
				_ = deps.Store.DeleteAgent(agent.ID)
				writeErr(w, storeStatus(err), err.Error())
				return
			}
		} else if launch.CLI != agent.CLI {
			_ = deps.Store.DeleteAgent(agent.ID)
			writeErr(w, http.StatusBadRequest, "That terminal is launched as a different CLI.")
			return
		} else {
			filled := fillManagedCLITools(launch.Overrides, agent.CLI)
			if launch.Overrides.Tools == nil && filled.Tools != nil {
				if err := deps.Store.SetTerminalLaunch(terminalID, agent.CLI, filled); err != nil {
					_ = deps.Store.DeleteAgent(agent.ID)
					writeErr(w, storeStatus(err), err.Error())
					return
				}
			}
		}
		bound, err := deps.Store.UpdateAgent(agent.ID, store.AgentPatch{TerminalID: &terminalID})
		if err != nil {
			_ = deps.Store.DeleteAgent(agent.ID)
			writeErr(w, storeStatus(err), err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, agentPrincipal(bound))
	}
}

func handleDeleteManagedCLI(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if err := deps.Store.DeleteAgent(id); err != nil {
			writeErr(w, storeStatus(err), err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
