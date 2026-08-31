package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/cfpperche/picode/internal/catalog"
	"github.com/cfpperche/picode/internal/pisettings"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/usage"
)

func registerSlashOps(mux *http.ServeMux, deps Deps) {
	mux.HandleFunc("POST /api/agents/{id}/trust", handleAgentTrust(deps))
	mux.HandleFunc("PUT /api/providers/{id}", handleProviderLogin)
	mux.HandleFunc("DELETE /api/providers/{id}", handleProviderLogout(deps))
	mux.HandleFunc("POST /api/providers/{id}/accounts/{aid}/activate", handleAccountActivate)
	mux.HandleFunc("PATCH /api/providers/{id}/accounts/{aid}", handleAccountRename)
	mux.HandleFunc("DELETE /api/providers/{id}/accounts/{aid}", handleAccountDelete)
	mux.HandleFunc("GET /api/providers/{id}/usage", handleProviderUsage)
}

func handleProviderUsage(w http.ResponseWriter, r *http.Request) {
	rep := usage.Fetch(r.Context(), r.PathValue("id"))
	writeJSON(w, http.StatusOK, rep)
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

func handleProviderLogin(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		Key string `json:"key"`
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	var err error
	if id == "llama.cpp" {
		err = catalog.PutLlama(req.URL, req.Key)
	} else {
		err = catalog.PutAPIKey(id, req.Key)
	}
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "signedIn": true})
}

func handleAccountActivate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	aid := r.PathValue("aid")
	if err := catalog.ActivateAccount(id, aid); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "active": aid})
}

func handleAccountRename(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	aid := r.PathValue("aid")
	var req struct {
		Label string `json:"label"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := catalog.RenameAccount(id, aid, req.Label); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "account": aid, "label": strings.TrimSpace(req.Label)})
}

func handleAccountDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	aid := r.PathValue("aid")
	if err := catalog.RemoveAccount(id, aid); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "removed": aid})
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
