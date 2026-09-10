// Browser surface consent toggle (ADR-0115): flips the session's input
// mirror so the proxy starts (or stops) forwarding mouse/keyboard/touch.
// The mirror lives beside the session file, written by this daemon or by
// the pi-browser-capture extension (TUI path); absence means off.

package server

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/cfpperche/picode/internal/store"
)

func handleBrowserInput(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if _, err := deps.Store.GetAgent(id); errors.Is(err, store.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "agent not found")
			return
		} else if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		ma := deps.Runtime.Get(id)
		if ma == nil {
			writeErr(w, http.StatusConflict, "The agent is not running.")
			return
		}
		var req struct {
			On *bool `json:"on"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.On == nil {
			writeErr(w, http.StatusBadRequest, "on is required")
			return
		}
		if err := ma.WriteInputConsent(r.Context(), *req.On); err != nil {
			writeErr(w, http.StatusConflict, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "on": *req.On})
	}
}
