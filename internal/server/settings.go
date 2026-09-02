package server

import (
	"encoding/json"
	"net"
	"net/http"
	"strconv"

	"github.com/cfpperche/picode/internal/config"
	"github.com/cfpperche/picode/internal/share"
)

// PortSnapshot reports the live server state to the UI.
type PortSnapshot struct {
	Current    int    `json:"current"`    // actually bound port
	Configured string `json:"configured"` // configured port/range string
	URL        string `json:"url"`        // best-guess user-facing URL
	Host       string `json:"host"`       // bind host (ADR-0050)
	PublicURL  string `json:"publicUrl"`  // configured origin, "" when none
}

// serverInfo is GET /api/server: the snapshot plus what the machine
// offers, so Preferences → Server can list binds and suggest a public URL.
type serverInfo struct {
	PortSnapshot
	Interfaces  []ifaceInfo `json:"interfaces"`
	Suggestions struct {
		TailscaleIP string `json:"tailscaleIp,omitempty"`
		MagicDNS    string `json:"magicDns,omitempty"`
	} `json:"suggestions"`
}

type ifaceInfo struct {
	IP   string `json:"ip"`
	Kind string `json:"kind"` // lan | tailnet
}

func registerServerRoutes(mux *http.ServeMux, deps Deps) {
	mux.HandleFunc("GET /api/server", handleServerInfo(deps))
	mux.HandleFunc("PUT /api/server/port", handlePortChange(deps))
	mux.HandleFunc("PUT /api/server/host", handleHostChange(deps))
	mux.HandleFunc("PUT /api/server/public-url", handlePublicURLChange(deps))
}

func handleServerInfo(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		if deps.PortSnapshot == nil {
			writeErr(w, http.StatusServiceUnavailable, "port state unavailable")
			return
		}
		info := serverInfo{PortSnapshot: deps.PortSnapshot(), Interfaces: []ifaceInfo{}}
		ts := share.TailscaleIPv4()
		for _, ip := range share.ReachableIPv4() {
			kind := "lan"
			if ip == ts {
				kind = "tailnet"
			}
			info.Interfaces = append(info.Interfaces, ifaceInfo{IP: ip, Kind: kind})
		}
		info.Suggestions.TailscaleIP = ts
		info.Suggestions.MagicDNS = share.MagicDNSName()
		writeJSON(w, http.StatusOK, info)
	}
}

// handleHostChange moves the bind (ADR-0050): validated, persisted, then
// the main loop moves the listener (loopback stays whenever the bind is
// an outside address). The answer leaves on the old listener.
func handleHostChange(deps Deps) http.HandlerFunc {
	var req struct {
		Host string `json:"host"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.Rebind == nil || deps.PortSnapshot == nil {
			writeErr(w, http.StatusServiceUnavailable, "bind management unavailable")
			return
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		host, err := config.ValidateHost(req.Host)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		snap := deps.PortSnapshot()
		if host == snap.Host {
			writeJSON(w, http.StatusOK, map[string]any{"moving": false, "host": host})
			return
		}
		if ip := net.ParseIP(host); !ip.IsUnspecified() && !ip.IsLoopback() && !onLocalInterface(host) {
			writeErr(w, http.StatusBadRequest, host+" is not an address of this machine")
			return
		}
		// No probe: the old listener holds the port on an address that
		// overlaps the new one (0.0.0.0 covers every specific address).
		// The main loop moves the listener and restores the old one if
		// the new bind fails.
		if serr := deps.Store.SetSetting(config.HostSettingKey, host); serr != nil {
			writeErr(w, http.StatusInternalServerError, serr.Error())
			return
		}
		_ = deps.Store.AppendEvent("server_host_changed", nil, nil, map[string]string{"from": snap.Host, "to": host})
		deps.Rebind()
		writeJSON(w, http.StatusAccepted, map[string]any{"moving": true, "host": host, "note": "reconnecting on the new address"})
	}
}

// handlePublicURLChange records the origin other machines use (ADR-0050).
// Advisory: pairing links, server.json and the share drawer read it; no
// listener moves. Empty clears it.
func handlePublicURLChange(deps Deps) http.HandlerFunc {
	var req struct {
		URL string `json:"url"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		pub, err := config.ValidatePublicURL(req.URL, deps.Insecure)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		if serr := deps.Store.SetSetting(config.PublicURLSettingKey, pub); serr != nil {
			writeErr(w, http.StatusInternalServerError, serr.Error())
			return
		}
		if deps.Rebind != nil {
			deps.Rebind() // refresh the snapshot and server.json; the listener stays
		}
		writeJSON(w, http.StatusOK, map[string]any{"publicUrl": pub})
	}
}

func onLocalInterface(ip string) bool {
	addrs, _ := net.InterfaceAddrs()
	for _, a := range addrs {
		if n, ok := a.(*net.IPNet); ok && n.IP.String() == ip {
			return true
		}
	}
	return false
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

// publicURL is the configured origin from the live snapshot, "" when none.
func (deps Deps) publicURL() string {
	if deps.PortSnapshot == nil {
		return ""
	}
	return deps.PortSnapshot().PublicURL
}
