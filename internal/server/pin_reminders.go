package server

import (
	"encoding/json"
	"net/http"

	"github.com/cfpperche/picode/internal/store"
)

// Pin reminders (ADR-0100): one cadence per pin in v2. The rule is read
// through GET /api/pins/{id} (and the list's summary) so the picker and the
// sidebar need no extra round trip; these two routes write it.
func registerPinReminders(mux Registrar, deps Deps) {
	mux.HandleFunc("PUT /api/pins/{id}/reminder", handleSetPinReminder(deps))
	mux.HandleFunc("DELETE /api/pins/{id}/reminder", handleDeletePinReminder(deps))
}

type pinReminderReq struct {
	Kind        string `json:"kind"`                  // once | interval | cron
	At          string `json:"at,omitempty"`          // once: RFC 3339, any offset
	IntervalMin int    `json:"intervalMin,omitempty"` // interval
	Cron        string `json:"cron,omitempty"`        // cron: 5 fields
	Anchor      string `json:"anchor,omitempty"`      // interval: schedule (default) | completion
	TZ          string `json:"tz"`                    // IANA zone of the wall clock
	Enabled     *bool  `json:"enabled,omitempty"`
}

func handleSetPinReminder(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if !pinIDOK.MatchString(id) {
			writeErr(w, http.StatusBadRequest, "invalid pin")
			return
		}
		var req pinReminderReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		rem, err := deps.Store.SetPinReminder(id, store.PinReminderParams{
			Kind: req.Kind, At: req.At, IntervalMin: req.IntervalMin, Cron: req.Cron, Anchor: req.Anchor, TZ: req.TZ, Enabled: req.Enabled,
		})
		if err != nil {
			writePinErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, rem)
	}
}

func handleDeletePinReminder(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if !pinIDOK.MatchString(id) {
			writeErr(w, http.StatusBadRequest, "invalid pin")
			return
		}
		if err := deps.Store.DeletePinReminder(id); err != nil {
			writePinErr(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
