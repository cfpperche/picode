package server

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/cfpperche/picode/internal/store"
)

func handleAgentUI(deps Deps) http.HandlerFunc {
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
			ID        string `json:"id"`
			Value     string `json:"value"`
			Confirmed *bool  `json:"confirmed"`
			Cancelled bool   `json:"cancelled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ID == "" {
			writeErr(w, http.StatusBadRequest, "id is required")
			return
		}
		if err := ma.ReplyUI(req.ID, req.Value, req.Confirmed, req.Cancelled); err != nil {
			writeErr(w, http.StatusConflict, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	}
}
