package server

import (
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/cfpperche/picode/internal/catalog"
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
	Host     string   `json:"host"`
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

		if host, err := os.Hostname(); err == nil {
			rep.Host = host
		}

		if len(rep.Warnings) == 0 {
			rep.Warnings = []string{}
		}
		writeJSON(w, http.StatusOK, rep)
	}
}

func handleCatalog(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rep, err := catalog.Load(deps.AgentCmd)
		if err != nil {
			writeErr(w, http.StatusServiceUnavailable, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, rep)
	}
}

// handleMCP reports adapter/config presence only (ADR-0009: no manager).
func handleMCP(w http.ResponseWriter, _ *http.Request) {
	home, _ := os.UserHomeDir()
	candidates := []string{
		filepath.Join(home, ".pi", "mcp.json"),
		filepath.Join(home, ".pi", "agent", "mcp.json"),
		filepath.Join(home, ".config", "pi-mcp-adapter", "config.json"),
	}
	found := ""
	for _, p := range candidates {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			found = p
			break
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"configured": found != "",
		"path":       found,
	})
}
