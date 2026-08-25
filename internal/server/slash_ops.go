package server

import (
	"net/http"

	"github.com/cfpperche/picode/internal/catalog"
	"github.com/cfpperche/picode/internal/pisettings"
	"github.com/cfpperche/picode/internal/store"
)

func registerSlashOps(mux *http.ServeMux, deps Deps) {
	mux.HandleFunc("POST /api/agents/{id}/trust", handleAgentTrust(deps))
	mux.HandleFunc("DELETE /api/providers/{id}", handleProviderLogout(deps))
}

func handleAgentTrust(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		agent, err := deps.Store.GetAgent(r.PathValue("id"))
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		wk, err := deps.Store.GetWorkspace(agent.WorkspaceID)
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		cwd := store.AgentCwd(wk, agent)
		if pisettings.Trusted(cwd) {
			writeJSON(w, http.StatusOK, map[string]any{"trusted": true, "cwd": cwd, "already": true})
			return
		}
		if err := pisettings.Set(cwd, true); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"trusted": true, "cwd": cwd, "already": false})
	}
}

func handleProviderLogout(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if err := catalog.RemoveAuth(id); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"id": id, "signedIn": false})
	}
}
