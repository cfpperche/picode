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

// systemReport drives the UI's setup guidance (ADR-0003: helpful
// dependency UX instead of hard failures).
type systemReport struct {
	Tmux struct {
		Installed          bool   `json:"installed"`
		Version            string `json:"version,omitempty"`
		ExtendedKeysFormat string `json:"extendedKeysFormat,omitempty"`
	} `json:"tmux"`
	Pi struct {
		Installed bool   `json:"installed"`
		Version   string `json:"version,omitempty"`
	} `json:"pi"`
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
			if v, err := deps.Tmux.Version(); err == nil {
				rep.Tmux.Version = v
			}
			// Best effort: requires a running tmux server.
			if f, err := deps.Tmux.ExtendedKeysFormat(r.Context()); err == nil {
				rep.Tmux.ExtendedKeysFormat = f
				if f != "csi-u" {
					rep.Warnings = append(rep.Warnings,
						"tmux extended-keys-format is \""+f+"\"; Pi recommends \"csi-u\" so keys like Shift+Enter reach your agents")
				}
			}
		} else {
			rep.Warnings = append(rep.Warnings,
				"tmux is not installed — agents need tmux 3.5+ to keep running after you close the browser")
		}

		if _, err := exec.LookPath(deps.AgentCmd); err == nil {
			rep.Pi.Installed = true
			if out, err := exec.Command(deps.AgentCmd, "--version").Output(); err == nil {
				rep.Pi.Version = strings.TrimSpace(string(out))
			}
		} else {
			rep.Warnings = append(rep.Warnings,
				"pi is not installed — install it with: npm install -g @earendil-works/pi-coding-agent")
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
		rep, err := catalog.Load(deps.AgentCmd)
		if err != nil {
			writeErr(w, http.StatusServiceUnavailable, err.Error())
			return
		}
		attachLlamaModels(&rep)
		writeJSON(w, http.StatusOK, rep)
	}
}
