package server

import (
	"encoding/json"
	"github.com/cfpperche/picode/internal/auth"
	"net/http"

	"github.com/cfpperche/picode/internal/presence"
	"github.com/cfpperche/picode/internal/share"
)

func registerDeviceRoutes(mux Registrar, deps *Deps) {
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
			Kind string `json:"kind"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ID == "" {
			writeErr(w, http.StatusBadRequest, "id required")
			return
		}
		sess := ""
		if p := auth.From(r); p != nil {
			sess = p.Session.ID
		}
		dev := deps.presence().PingSession(req.ID, r.UserAgent(), r.RemoteAddr, req.Host, req.Kind, sess)
		writeJSON(w, http.StatusOK, dev)
	}
}

func handleDeviceList(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, deps.presence().List())
	}
}
