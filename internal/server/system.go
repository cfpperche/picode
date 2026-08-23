package server

import (
	"net/http"
	"os"
	"os/exec"
	"strings"
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

		if host, err := os.Hostname(); err == nil {
			rep.Host = host
		}

		if len(rep.Warnings) == 0 {
			rep.Warnings = []string{}
		}
		writeJSON(w, http.StatusOK, rep)
	}
}
