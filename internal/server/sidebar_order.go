package server

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/cfpperche/picode/internal/store"
)

func registerSidebarOrderRoutes(mux Registrar, deps Deps) {
	mux.HandleFunc("PUT /api/workspaces/order", handleReorderWorkspaces(deps))
	mux.HandleFunc("PUT /api/workspaces/{id}/agents/order", handleReorderAgents(deps))
	mux.HandleFunc("PUT /api/workspaces/{id}/terminals/order", handleReorderTerminals(deps))
}

func handleReorderWorkspaces(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ids, ok := readOrderIDs(w, r)
		if !ok {
			return
		}
		writeOrder(w, deps.Store.ReorderWorkspaces(ids))
	}
}

func handleReorderAgents(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ids, ok := readOrderIDs(w, r)
		if !ok {
			return
		}
		writeOrder(w, deps.Store.ReorderAgents(r.PathValue("id"), ids))
	}
}

func handleReorderTerminals(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ids, ok := readOrderIDs(w, r)
		if !ok {
			return
		}
		writeOrder(w, deps.Store.ReorderTerminals(r.PathValue("id"), ids))
	}
}

func readOrderIDs(w http.ResponseWriter, r *http.Request) ([]string, bool) {
	var body struct {
		IDs []string `json:"ids"`
	}
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body")
		return nil, false
	}
	return body.IDs, true
}

func writeOrder(w http.ResponseWriter, err error) {
	if err == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	var oe *store.OrderError
	if errors.As(err, &oe) {
		writeErr(w, http.StatusBadRequest, oe.Error())
		return
	}
	if errors.Is(err, store.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	writeErr(w, http.StatusInternalServerError, err.Error())
}
