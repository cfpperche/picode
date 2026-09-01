package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/cfpperche/picode/internal/store"
)

// Inbox routes (ADR-0037). Localhost trust model (ADR-0007): no auth;
// the mitigations are mandatory provenance on every item and bodies
// rendered as markdown by the host, never HTML.

func registerInboxRoutes(mux *http.ServeMux, deps Deps) {
	mux.HandleFunc("POST /api/inbox", handleCreateInboxItem(deps))
	mux.HandleFunc("GET /api/inbox", handleListInbox(deps))
	mux.HandleFunc("POST /api/inbox/{id}/respond", handleRespondInbox(deps))
	mux.HandleFunc("POST /api/inbox/{id}/state", handleInboxState(deps))
	mux.HandleFunc("DELETE /api/inbox/{id}", handleDeleteInboxItem(deps))
	mux.HandleFunc("DELETE /api/inbox", handleClearDoneInbox(deps))
}

type inboxCreateReq struct {
	Kind        string   `json:"kind"`
	SourceKind  string   `json:"sourceKind"`
	SourceID    string   `json:"sourceId"`
	WorkspaceID string   `json:"workspaceId"`
	Reason      string   `json:"reason"`
	Title       string   `json:"title"`
	Body        string   `json:"body"`
	Blocking    bool     `json:"blocking"`
	Allowed     []string `json:"allowedResponses"`
}

func handleCreateInboxItem(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req inboxCreateReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		it, err := deps.Store.CreateInboxItem(store.InboxItemParams{
			Kind: req.Kind, SourceKind: req.SourceKind, SourceID: req.SourceID,
			WorkspaceID: req.WorkspaceID, Reason: req.Reason, Title: req.Title,
			Body: req.Body, Blocking: req.Blocking, Allowed: req.Allowed,
		})
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, it)
	}
}

func handleListInbox(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f := store.InboxFilter{State: r.URL.Query().Get("state")}
		if v := r.URL.Query().Get("blocking"); v != "" {
			b := v == "1" || v == "true"
			f.Blocking = &b
		}
		if queryFlag(r, "snoozed") {
			f.IncludeSnoozed = true
		}
		items, err := deps.Store.ListInboxItems(f)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
	}
}

func handleRespondInbox(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Verb string `json:"verb"`
			Text string `json:"text"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if (req.Verb == store.VerbRespond || req.Verb == store.VerbEdit) && strings.TrimSpace(req.Text) == "" {
			writeErr(w, http.StatusBadRequest, "text is required for a "+req.Verb)
			return
		}
		deliverable := func(agentID string) bool { return !deps.agentInteractive(r.Context(), agentID) }
		it, err := deps.Store.RespondAndForward(r.PathValue("id"), req.Verb, req.Text, deliverable)
		if err != nil {
			if errors.Is(err, store.ErrAgentInteractive) {
				writeErr(w, http.StatusConflict, "agent is running in an interactive terminal — reply not delivered automatically; the item stays open")
				return
			}
			if errors.Is(err, store.ErrNotFound) {
				// The item exists but its agent is gone (annotated), or the
				// item id itself is unknown — the message distinguishes.
				if strings.Contains(err.Error(), "agent no longer exists") {
					writeErr(w, http.StatusConflict, "agent no longer exists — reply not delivered; the item stays open")
					return
				}
				writeErr(w, http.StatusNotFound, "no such inbox item")
				return
			}
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, it)
	}
}

func handleInboxState(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			State        string  `json:"state"`
			SnoozedUntil *string `json:"snoozedUntil"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		it, err := deps.Store.SetInboxItemState(r.PathValue("id"), req.State, req.SnoozedUntil)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				writeErr(w, http.StatusNotFound, "no such inbox item")
				return
			}
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, it)
	}
}

// handleDeleteInboxItem permanently removes one item. The Inbox app
// itself never calls this (it uses deps.Store directly, like every
// other action) — this route exists for pi-inbox/scripts/automation, the
// same role the other four routes already fill. CRUD without a delete
// leg is its own footgun.
func handleDeleteInboxItem(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := deps.Store.DeleteInboxItem(r.PathValue("id")); err != nil {
			writeErr(w, statusForStore(err), err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// handleClearDoneInbox bulk-deletes done items. Deliberately requires an
// explicit ?state=done: a bare DELETE /api/inbox must never be able to
// mean "delete everything" by accident.
func handleClearDoneInbox(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != store.InboxDone {
			writeErr(w, http.StatusBadRequest, "DELETE /api/inbox requires ?state=done")
			return
		}
		n, err := deps.Store.DeleteDoneInboxItems()
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"deleted": n})
	}
}
