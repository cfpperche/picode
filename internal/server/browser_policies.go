package server

// The grants editor's read/write surface (slice 4, ADR-0128 + ADR-0134):
// the page lists every managed agent with its effective grant, and saves
// one grant at a time through browser.Save. Saving is a settings write, so
// it announces itself on the feed (setting.updated) — an open editor
// refetches on that, it does not poll.

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/cfpperche/picode/internal/browser"
)

func registerBrowserPolicyRoutes(mux Registrar, deps Deps) {
	mux.HandleFunc("GET /api/browser/policies", handleBrowserPolicies(deps))
	mux.HandleFunc("POST /api/browser/policy", handleBrowserPolicySave(deps))
}

// handleBrowserPolicies lists every managed agent with its effective grant:
// the saved one when there is one, the ADR-0134 default otherwise. `saved`
// distinguishes "the owner wrote this" from "this is just the default".
func handleBrowserPolicies(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.Store == nil {
			writeErr(w, http.StatusServiceUnavailable, "store is not open")
			return
		}
		agents, err := deps.Store.ListAllAgents()
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		type grant struct {
			AgentID   string   `json:"agentId"`
			Name      string   `json:"name"`
			Workspace string   `json:"workspaceId"`
			Tier      string   `json:"tier"`
			Domains   []string `json:"domains"`
			Saved     bool     `json:"saved"`
		}
		out := make([]grant, 0, len(agents))
		for _, a := range agents {
			p := browser.Resolve(deps.Store, a.ID)
			_, saved, _ := deps.Store.GetSetting(browser.SettingPrefix + a.ID)
			domains := p.Domains
			if domains == nil {
				domains = []string{}
			}
			out = append(out, grant{
				AgentID:   a.ID,
				Name:      a.Name,
				Workspace: a.WorkspaceID,
				Tier:      p.Tier,
				Domains:   domains,
				Saved:     saved,
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{"policies": out})
	}
}

// handleBrowserPolicySave writes one agent's grant. The destination rule is
// the host table of browser.AllowsOrigin, so entries arrive as bare hosts
// ("example.com", "*.example.com") — anything else would never match.
func handleBrowserPolicySave(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.Store == nil {
			writeErr(w, http.StatusServiceUnavailable, "store is not open")
			return
		}
		var req struct {
			Agent   string   `json:"agent"`
			Tier    string   `json:"tier"`
			Domains []string `json:"domains"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "bad request: "+err.Error())
			return
		}
		req.Agent = strings.TrimSpace(req.Agent)
		if req.Agent == "" {
			writeErr(w, http.StatusBadRequest, "an agent id is required")
			return
		}
		known := false
		agents, err := deps.Store.ListAllAgents()
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		for _, a := range agents {
			if a.ID == req.Agent {
				known = true
				break
			}
		}
		if !known {
			writeErr(w, http.StatusNotFound, "unknown agent: "+req.Agent)
			return
		}
		domains := make([]string, 0, len(req.Domains))
		for _, d := range req.Domains {
			if d = strings.TrimSpace(strings.ToLower(d)); d != "" {
				domains = append(domains, d)
			}
		}
		if err := browser.Save(deps.Store, req.Agent, browser.Policy{Tier: req.Tier, Domains: domains}); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		p := browser.Resolve(deps.Store, req.Agent)
		writeJSON(w, http.StatusOK, map[string]any{"agentId": req.Agent, "tier": p.Tier, "domains": p.Domains, "saved": true})
	}
}
