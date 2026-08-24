package server

import (
	"encoding/json"
	"net/http"

	"github.com/cfpperche/picode/internal/presence"
	"github.com/cfpperche/picode/internal/share"
)

func registerDeviceRoutes(mux *http.ServeMux, deps *Deps) {
	mux.HandleFunc("POST /api/devices/ping", handleDevicePing(deps))
	mux.HandleFunc("GET /api/devices", handleDeviceList(deps))
}

func (d *Deps) presence() *presence.Registry {
	if d.Presence == nil {
		d.Presence = presence.New(share.ReachableIPv4())
	}
	return d.Presence
}

func handleDevicePing(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID   string `json:"id"`
			Host bool   `json:"host"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ID == "" {
			writeErr(w, http.StatusBadRequest, "id required")
			return
		}
		dev := deps.presence().Ping(req.ID, r.UserAgent(), r.RemoteAddr, req.Host)
		writeJSON(w, http.StatusOK, dev)
	}
}

func handleDeviceList(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, deps.presence().List())
	}
}
