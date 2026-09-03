package server

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/cfpperche/picode/internal/store"
)

// Internal checklists (ADR-0055). pi-checklist POSTs the agent's list on
// every change (and an "absent" marker when a required plan is missing);
// the shells read one list at boot and follow agent.checklist on the feed.
func registerChecklistRoutes(mux *http.ServeMux, deps Deps) {
	mux.HandleFunc("POST /api/agents/{id}/checklist", handleSetChecklist(deps))
	mux.HandleFunc("GET /api/agents/{id}/checklist", handleGetChecklist(deps))
	mux.HandleFunc("GET /api/checklists", handleListChecklists(deps))
}

func handleSetChecklist(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			SessionID string                `json:"sessionId"`
			Items     []store.ChecklistItem `json:"items"`
			Absent    bool                  `json:"absent"`
			Blocked   bool                  `json:"blocked"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 256<<10)).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		// A refused change with no list yet is the same fact as a turn that
		// ended without one: the plan the contract asks for is not there.
		absent := req.Absent || (req.Blocked && len(req.Items) == 0)
		c, err := deps.Store.SetChecklist(r.PathValue("id"), req.SessionID, req.Items, absent)
		if errors.Is(err, store.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "agent not found")
			return
		}
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, c)
	}
}

func handleGetChecklist(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := deps.Store.GetChecklist(r.PathValue("id"))
		if errors.Is(err, store.ErrNotFound) {
			writeJSON(w, http.StatusOK, map[string]any{"checklist": nil})
			return
		}
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"checklist": c})
	}
}

func handleListChecklists(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		list, err := deps.Store.ListChecklists()
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"checklists": list})
	}
}
