package server

import (
	"encoding/json"
	"net/http"

	"github.com/cfpperche/picode/internal/pikeys"
)

func registerPiKeysRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/pi-keys", handleGetPiKeys)
	mux.HandleFunc("PUT /api/pi-keys", handlePutPiKeys)
}

type piKeysReport struct {
	Actions []pikeys.Action     `json:"actions"`
	User    map[string][]string `json:"user"`
}

func handleGetPiKeys(w http.ResponseWriter, _ *http.Request) {
	user, err := pikeys.LoadUser()
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, piKeysReport{Actions: pikeys.Catalog, User: user})
}

type piKeysPut struct {
	Action string    `json:"action"`
	Keys   *[]string `json:"keys"`
	Reset  bool      `json:"reset"`
}

func handlePutPiKeys(w http.ResponseWriter, r *http.Request) {
	var req piKeysPut
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	var keys []string
	if req.Reset {
		keys = nil
	} else if req.Keys == nil {
		writeErr(w, http.StatusBadRequest, "keys or reset")
		return
	} else {
		keys = *req.Keys
	}
	if err := pikeys.Set(req.Action, keys); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	user, err := pikeys.LoadUser()
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, piKeysReport{Actions: pikeys.Catalog, User: user})
}
