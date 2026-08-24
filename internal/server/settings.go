package server

import (
	"encoding/json"
	"net"
	"net/http"
	"strconv"

	"github.com/cfpperche/picode/internal/config"
)

// PortSnapshot reports the live port state to the UI.
type PortSnapshot struct {
	Current    int    `json:"current"`    // actually bound port
	Configured string `json:"configured"` // configured port/range string
	URL        string `json:"url"`        // best-guess user-facing URL
}

func registerServerRoutes(mux *http.ServeMux, deps Deps) {
	mux.HandleFunc("GET /api/server", handleServerInfo(deps))
	mux.HandleFunc("PUT /api/server/port", handlePortChange(deps))
}

func handleServerInfo(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		if deps.PortSnapshot == nil {
			writeErr(w, http.StatusServiceUnavailable, "port state unavailable")
			return
		}
		writeJSON(w, http.StatusOK, deps.PortSnapshot())
	}
}

// handlePortChange applies a new port (specific, e.g. "8446") chosen in the
// Settings UI. Validates, test-binds, persists, and signals the main loop to
// rebind. The response reaches the client on the OLD listener before it
// shuts down — the UI then reconnects to the new port (ADR-0007).
func handlePortChange(deps Deps) http.HandlerFunc {
	var req struct {
		Port string `json:"port"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.Rebind == nil || deps.PortSnapshot == nil {
			writeErr(w, http.StatusServiceUnavailable, "port management unavailable")
			return
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON body")
			return
		}

		p, err := config.ParsePort(req.Port)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := p.Validate(); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		// UI changes must be a single specific port (ranges are an env-var
		// affordance for headless operation; see ADR-0007).
		if p.Min != p.Max {
			writeErr(w, http.StatusBadRequest, "the UI sets one specific port; ranges are set via PICODE_PORT")
			return
		}

		// Occupied right now? Fail fast without dropping the current server.
		if probe, lerr := net.Listen("tcp", net.JoinHostPort(deps.BindHost, strconv.Itoa(p.Min))); lerr == nil {
			_ = probe.Close()
		} else {
			writeErr(w, http.StatusConflict, "port "+strconv.Itoa(p.Min)+" is already in use")
			return
		}

		if serr := deps.Store.SetSetting(config.PortSettingKey, p.String()); serr != nil {
			writeErr(w, http.StatusInternalServerError, serr.Error())
			return
		}
		_ = deps.Store.AppendEvent("server_port_changed", nil, nil, map[string]string{
			"from": strconv.Itoa(deps.PortSnapshot().Current), "to": p.String(),
		})

		deps.Rebind() // main loop binds the new port, then shuts this one down

		writeJSON(w, http.StatusAccepted, map[string]any{
			"moving": true,
			"port":   p.Min,
			"note":   "reconnecting on the new port",
		})
	}
}
