package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/cfpperche/picode/internal/clilaunch"
	"github.com/cfpperche/picode/internal/grant"
	"github.com/cfpperche/picode/internal/store"
)

// principalView is one workspace actor (ADR-0159): a Pi agent or a bound
// Agent CLI terminal. Kind/id/key match grant.Principal.
type principalView struct {
	Kind         string `json:"kind"`
	ID           string `json:"id"`
	Key          string `json:"key"`
	Name         string `json:"name"`
	WorkspaceID  string `json:"workspaceId"`
	CLI          string `json:"cli,omitempty"`
	ManagedCLIID string `json:"managedCliId,omitempty"`
	AgentID      string `json:"agentId,omitempty"`
}

func agentPrincipal(a store.Agent) principalView {
	p := grant.Principal{Kind: grant.KindAgent, ID: a.ID}
	return principalView{
		Kind:        string(p.Kind),
		ID:          p.ID,
		Key:         p.Key(),
		Name:        a.Name,
		WorkspaceID: a.WorkspaceID,
		AgentID:     a.ID,
	}
}

func managedCLIPrincipal(m store.ManagedCLI) principalView {
	p := m.Principal()
	return principalView{
		Kind:         string(p.Kind),
		ID:           p.ID,
		Key:          p.Key(),
		Name:         m.Name,
		WorkspaceID:  m.WorkspaceID,
		CLI:          m.CLI,
		ManagedCLIID: m.ID,
	}
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
		managed, err := deps.Store.ListManagedCLIs(wsID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		out := make([]principalView, 0, len(agents)+len(managed))
		for _, a := range agents {
			out = append(out, agentPrincipal(a))
		}
		for _, m := range managed {
			out = append(out, managedCLIPrincipal(m))
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
		name := strings.TrimSpace(req.Name)
		if name == "" {
			name = entry.Name
		}
		terminalID := strings.TrimSpace(req.TerminalID)
		created := false
		if terminalID == "" {
			tm, err := deps.Store.CreateTerminalIn(wsID, name, wk.Path)
			if err != nil {
				writeErr(w, storeStatus(err), err.Error())
				return
			}
			if err := deps.Store.SetTerminalLaunch(tm.ID, cli, managedCLILaunchOverrides(cli)); err != nil {
				_ = deps.Store.DeleteTerminal(tm.ID)
				writeErr(w, storeStatus(err), err.Error())
				return
			}
			terminalID = tm.ID
			created = true
		} else if launch, err := deps.Store.TerminalLaunch(terminalID); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		} else if launch == nil {
			if err := deps.Store.SetTerminalLaunch(terminalID, cli, managedCLILaunchOverrides(cli)); err != nil {
				writeErr(w, storeStatus(err), err.Error())
				return
			}
		} else {
			filled := fillManagedCLITools(launch.Overrides, cli)
			if launch.Overrides.Tools == nil && filled.Tools != nil {
				if err := deps.Store.SetTerminalLaunch(terminalID, cli, filled); err != nil {
					writeErr(w, storeStatus(err), err.Error())
					return
				}
			}
		}
		m, err := deps.Store.AddManagedCLI(wsID, cli, name, terminalID)
		if err != nil {
			if created {
				_ = deps.Store.DeleteTerminal(terminalID)
			}
			writeErr(w, storeStatus(err), err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, managedCLIPrincipal(m))
	}
}

func handleDeleteManagedCLI(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if err := deps.Store.RemoveManagedCLI(id); err != nil {
			writeErr(w, storeStatus(err), err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
