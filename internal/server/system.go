package server

import (
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/cfpperche/picode/internal/catalog"
	"github.com/cfpperche/picode/internal/share"
)

// systemReport drives the System page: host and network facts plus the
// infrastructure PiCode itself depends on. tmux is the one hard requirement;
// mkcert and tailscale are optional. No agent CLI is probed here — which
// CLIs are present, and their versions, is `GET /api/clis` (ADR-0179).
type systemReport struct {
	Tmux struct {
		Installed bool `json:"installed"`
		// Version is the tmux in use: the running server's, or the installed
		// binary's when no server answers. Never the client binary's while an
		// older server still owns the panes (ADR-0164).
		Version            string `json:"version,omitempty"`
		ExtendedKeysFormat string `json:"extendedKeysFormat,omitempty"`
	} `json:"tmux"`
	Mkcert struct {
		Installed bool `json:"installed"`
	} `json:"mkcert"`
	Tailscale struct {
		Installed bool   `json:"installed"`
		IP        string `json:"ip,omitempty"`
	} `json:"tailscale"`
	Host struct {
		Name string `json:"name"`
		OS   string `json:"os"`
		Arch string `json:"arch"`
		WSL  bool   `json:"wsl"`
	} `json:"host"`
	Network struct {
		Bind      string   `json:"bind"`
		Port      int      `json:"port,omitempty"`
		HTTPS     bool     `json:"https"`
		LAN       []string `json:"lan"`
		Tailscale string   `json:"tailscale,omitempty"`
	} `json:"network"`
	Warnings []string `json:"warnings"`
}

func handleSystem(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var rep systemReport

		if deps.Tmux.Available() {
			rep.Tmux.Installed = true
			// The version in use, not `tmux -V`: after an upgrade the new client
			// talks to the old server until it exits, and only the server's
			// version says which tmux owns the panes (ADR-0164).
			if v := deps.Tmux.VersionInUse(r.Context()); v != "" {
				rep.Tmux.Version = v
			}
			// Best effort: requires a running tmux server.
			if f, err := deps.Tmux.ExtendedKeysFormat(r.Context()); err == nil {
				rep.Tmux.ExtendedKeysFormat = f
				if f != "xterm" {
					rep.Warnings = append(rep.Warnings,
						"tmux extended-keys-format is \""+f+"\"; PiCode sets \"xterm\" on attach so Shift+Enter reaches your agents")
				}
			}
		} else {
			rep.Warnings = append(rep.Warnings,
				"tmux is not installed — agents and terminals need tmux 3.5+ to keep running after you close the browser")
		}

		if _, err := exec.LookPath("mkcert"); err == nil {
			rep.Mkcert.Installed = true
		}
		if _, err := exec.LookPath("tailscale"); err == nil {
			rep.Tailscale.Installed = true
			if out, err := exec.Command("tailscale", "ip", "-4").Output(); err == nil {
				rep.Tailscale.IP = strings.TrimSpace(string(out))
			}
		}

		rep.Host.OS = runtime.GOOS
		rep.Host.Arch = runtime.GOARCH
		rep.Host.WSL = runningOnWSL()
		if name, err := os.Hostname(); err == nil {
			rep.Host.Name = name
		}

		rep.Network.Bind = deps.BindHost
		if rep.Network.Bind == "" {
			rep.Network.Bind = "0.0.0.0"
		}
		rep.Network.HTTPS = !deps.Insecure
		if deps.PortSnapshot != nil {
			rep.Network.Port = deps.PortSnapshot().Current
		}
		rep.Network.Tailscale = rep.Tailscale.IP
		rep.Network.LAN = []string{}
		for _, ip := range share.ReachableIPv4() {
			if ip != "" && ip != rep.Network.Tailscale {
				rep.Network.LAN = append(rep.Network.LAN, ip)
			}
		}

		if len(rep.Warnings) == 0 {
			rep.Warnings = []string{}
		}
		writeJSON(w, http.StatusOK, rep)
	}
}

func runningOnWSL() bool {
	b, err := os.ReadFile("/proc/version")
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(string(b)), "microsoft")
}

func handleCatalog(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rep, err := loadCatalog(deps)
		if err != nil {
			writeErr(w, http.StatusServiceUnavailable, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, rep)
	}
}

// loadCatalog is pi's catalog with everything a pane shows beside it — the
// llama models, and how many agents and automations name each provider — so
// "the catalog" has one definition here. The credential roster reads pi's
// provider list from the same place (ADR-0169).
func loadCatalog(deps Deps) (catalog.Report, error) {
	rep, err := catalog.Load(deps.AgentCmd)
	if err != nil {
		return rep, err
	}
	attachLlamaModels(&rep)
	attachProviderRefs(deps, &rep)
	return rep, nil
}

// attachProviderRefs fills how many agents and automations name each
// provider, so Sign out can say what it breaks (Zapier names the blast
// radius on its connections page). A store error leaves the counts at zero
// rather than failing the catalog every consumer depends on.
func attachProviderRefs(deps Deps, rep *catalog.Report) {
	if deps.Store == nil {
		return
	}
	refs, err := deps.Store.CountProviderRefs()
	if err != nil || len(refs) == 0 {
		return
	}
	for i := range rep.Providers {
		r := refs[rep.Providers[i].ID]
		rep.Providers[i].Agents = r.Agents
		rep.Providers[i].Automations = r.Automations
	}
}
