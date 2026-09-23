package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/cfpperche/picode/internal/apps"
	"github.com/cfpperche/picode/internal/store"
)

// Inbox routes (ADR-0037). Localhost trust model (ADR-0007): no auth;
// the mitigations are mandatory provenance on every item and bodies
// rendered as markdown by the host, never HTML.

func registerInboxRoutes(mux Registrar, deps Deps) {
	mux.HandleFunc("POST /api/inbox", handleCreateInboxItem(deps))
	mux.HandleFunc("GET /api/inbox", handleListInbox(deps))
	mux.HandleFunc("GET /api/inbox/{id}", handleGetInboxItem(deps))
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
	SessionPath string   `json:"sessionPath"`
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
		// An agent's question names the conversation it came from so a
		// receiver can answer into exactly that one (ADR-0060). Pi's own
		// inbox extension sends the path; an `ask_human` over MCP cannot
		// know it, so the session the agent's receiver last reported stands
		// in — the one it is showing while it asks. resolveReplySession
		// still checks the path before anything is delivered to it.
		if req.SourceKind == store.InboxFromAgent && strings.TrimSpace(req.SessionPath) == "" {
			req.SessionPath = deps.Replies.receiverSession(req.SourceID)
		}
		it, err := deps.Store.CreateInboxItem(store.InboxItemParams{
			Kind: req.Kind, SourceKind: req.SourceKind, SourceID: req.SourceID,
			WorkspaceID: req.WorkspaceID, SessionPath: req.SessionPath, Reason: req.Reason, Title: req.Title,
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
		f := store.InboxFilter{State: r.URL.Query().Get("state"), Kind: r.URL.Query().Get("kind")}
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

// handleGetInboxItem is one item by id — what `picode mcp inbox` polls
// while `ask_human` waits for the human (ADR-0154, N1). A poll with
// ?wait=1 marks an asker as waiting, so the answer is recorded for it to read
// instead of also being typed into its terminal (agent_answer.go).
func handleGetInboxItem(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if queryFlag(r, "wait") {
			deps.Replies.askPolled(r.PathValue("id"))
		}
		it, err := deps.Store.GetInboxItem(r.PathValue("id"))
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, it)
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
		id := r.PathValue("id")
		// A question filed by pi in an Agent CLI terminal (sourceKind
		// "terminal", ADR-0089's amendment) is answered through that
		// terminal's receiver — the task queue is an agent's. Ignore is the
		// exception: it sends nothing, so it closes the item locally like an
		// agent's ignore does, however the terminal looks right now.
		if it, err := deps.Store.GetInboxItem(id); err == nil && it.SourceKind == store.InboxFromTerminal && req.Verb != store.VerbIgnore {
			if _, err := deps.AnswerTerminalQuestion(id, req.Verb, req.Text); err != nil {
				if errors.Is(err, store.ErrNotFound) {
					writeErr(w, http.StatusConflict, "terminal no longer exists — reply not delivered; the item stays open")
					return
				}
				writeErr(w, http.StatusConflict, err.Error())
				return
			}
			it, _ = deps.Store.GetInboxItem(id)
			writeJSON(w, http.StatusOK, it)
			return
		}
		// An agent's question or approval goes through the door that agent
		// listens on — the same rule the Inbox app applies, so the mobile
		// Reply and the desktop one never disagree.
		if it, err := deps.Store.GetInboxItem(id); err == nil && it.SourceKind == store.InboxFromAgent &&
			req.Verb != store.VerbIgnore && it.State != store.InboxDone &&
			(it.Kind == store.InboxQuestion || it.Kind == store.InboxApproval) {
			if _, err := deps.AnswerAgentQuestion(r.Context(), id, req.Verb, req.Text); err != nil {
				if errors.Is(err, errAnswerInvalid) {
					writeErr(w, http.StatusBadRequest, err.Error())
					return
				}
				if strings.Contains(err.Error(), "agent no longer exists") {
					writeErr(w, http.StatusConflict, "agent no longer exists — reply not delivered; the item stays open")
					return
				}
				writeErr(w, http.StatusConflict, err.Error())
				return
			}
			it, _ = deps.Store.GetInboxItem(id)
			writeJSON(w, http.StatusOK, it)
			return
		}
		deliverable := func(agentID string) bool { return !deps.agentInteractive(r.Context(), agentID) }
		it, err := deps.Store.RespondAndForward(r.PathValue("id"), req.Verb, req.Text, deliverable)
		if err != nil {
			if errors.Is(err, store.ErrNoReplyChannel) {
				// ADR-0154's ask door, one row further: a question whose
				// source has no delivery channel at all (a guest CLI with
				// neither agent nor terminal identity) is answered by
				// recording the response on the item — the same rule the
				// terminal branch above applies to a non-pi terminal — so a
				// polling asker (`picode inbox ask --wait`) can read it and
				// the human's Reply always closes the item. The refusal
				// predates the polling askers: it left the item open forever
				// with the asker hanging on a poll that could never end. The
				// appended note names who still must be told another way,
				// because a plain ask and an unmanaged pi do not poll.
				if done, rerr := deps.Store.RespondInboxItem(id, req.Verb, req.Text); rerr == nil {
					_ = deps.Store.AnnotateInboxItem(id, apps.InboxAnswerRecordedNote)
					writeJSON(w, http.StatusOK, done)
					return
				}
				writeErr(w, http.StatusConflict, "this question came from a session PiCode has no reply channel for — answer it in its terminal; the item stays open")
				return
			}
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
