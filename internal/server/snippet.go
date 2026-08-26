package server

import (
	"encoding/json"
	"net/http"

	"github.com/cfpperche/picode/internal/snippet"
	"github.com/cfpperche/picode/internal/store"
)

func registerSnippet(mux *http.ServeMux, deps Deps) {
	mux.HandleFunc("POST /api/agents/{id}/snippet", handleAgentSnippet(deps))
}

func handleAgentSnippet(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		agent, err := deps.Store.GetAgent(id)
		if err != nil {
			writeErr(w, statusForStore(err), err.Error())
			return
		}
		var req struct {
			Lang string `json:"lang"`
			Code string `json:"code"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		wk, err := deps.Store.GetWorkspace(agent.WorkspaceID)
		if err != nil {
			writeErr(w, statusForStore(err), err.Error())
			return
		}
		cwd := store.AgentCwd(wk, agent)
		res, err := snippet.Run(cwd, req.Lang, req.Code)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, res)
	}
}
