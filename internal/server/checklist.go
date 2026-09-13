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
// A pi running inside a PiCode terminal (Agent CLIs, ADR-0069) has no agent
// id — it posts under PICODE_TERM_ID to the terminal routes below, and its
// terminal view carries the same checklist the agent views do.
func registerChecklistRoutes(mux Registrar, deps Deps) {
	mux.HandleFunc("POST /api/agents/{id}/checklist", handleSetChecklist(deps))
	mux.HandleFunc("GET /api/agents/{id}/checklist", handleGetChecklist(deps))
	mux.HandleFunc("GET /api/checklists", handleListChecklists(deps))
	mux.HandleFunc("POST /api/terminals/{id}/checklist", handleSetTerminalChecklist(deps))
	mux.HandleFunc("GET /api/terminals/{id}/checklist", handleGetTerminalChecklist(deps))
}

func handleSetChecklist(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			SessionID string                `json:"sessionId"`
			Items     []store.ChecklistItem `json:"items"`
			Absent    bool                  `json:"absent"`
			Blocked   bool                  `json:"blocked"`
			Reset     bool                  `json:"reset"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 256<<10)).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		// A fresh session with no checklist in its branch resets the row:
		// whatever the daemon holds is about a dead session, and silence is
		// the honest line until this session writes a plan (ADR-0055, "no
		// channel means no line").
		if req.Reset {
			c, err := deps.Store.ClearChecklist(r.PathValue("id"))
			if errors.Is(err, store.ErrNotFound) {
				writeErr(w, http.StatusNotFound, "agent not found")
				return
			}
			if err != nil {
				writeErr(w, http.StatusBadRequest, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, c)
			return
		}
		// A refused change is the same fact as a turn that ended without
		// one: the plan the contract asks for is not there. The gate only
		// refuses unplanned tasks, so an absent or blocked POST never
		// carries a current list — normalize instead of storing stale steps
		// that would render as this task's plan.
		absent := req.Absent || req.Blocked
		if absent {
			req.Items = nil
		}
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

// handleSetTerminalChecklist is handleSetChecklist for a coding-CLI
// terminal: same body, same reset/absent semantics, terminal identity.
func handleSetTerminalChecklist(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			SessionID string                `json:"sessionId"`
			Items     []store.ChecklistItem `json:"items"`
			Absent    bool                  `json:"absent"`
			Blocked   bool                  `json:"blocked"`
			Reset     bool                  `json:"reset"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 256<<10)).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		id := r.PathValue("id")
		if req.Reset {
			c, err := deps.Store.ClearTerminalChecklist(id)
			if errors.Is(err, store.ErrNotFound) {
				writeErr(w, http.StatusNotFound, "terminal not found")
				return
			}
			if err != nil {
				writeErr(w, http.StatusBadRequest, err.Error())
				return
			}
			invalidateTerminals(deps)
			writeJSON(w, http.StatusOK, c)
			return
		}
		absent := req.Absent || req.Blocked
		if absent {
			req.Items = nil
		}
		c, err := deps.Store.SetTerminalChecklist(id, req.SessionID, req.Items, absent)
		if errors.Is(err, store.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "terminal not found")
			return
		}
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		invalidateTerminals(deps)
		writeJSON(w, http.StatusOK, c)
	}
}

func handleGetTerminalChecklist(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := deps.Store.GetTerminalChecklist(r.PathValue("id"))
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

// applyTerminalChecklist folds the terminal's stored checklist into a
// terminal view, the way applyTermState folds live presence. The list
// endpoint is the one boot fetch; updates arrive as terminal.checklist.
func applyTerminalChecklist(deps Deps, view map[string]any, termID string) {
	if deps.Store == nil {
		return
	}
	c, err := deps.Store.GetTerminalChecklist(termID)
	if err != nil {
		return
	}
	view["checklist"] = c
}
