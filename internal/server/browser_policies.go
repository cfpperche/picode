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
			Kind      string   `json:"kind"`
			AgentID   string   `json:"agentId,omitempty"`
			TermID    string   `json:"termId,omitempty"`
			Name      string   `json:"name"`
			Workspace string   `json:"workspaceId,omitempty"`
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
				Kind:      "agent",
				AgentID:   a.ID,
				Name:      a.Name,
				Workspace: a.WorkspaceID,
				Tier:      p.Tier,
				Domains:   domains,
				Saved:     saved,
			})
		}
		// ADR-0143: a CLI running in a terminal is a principal too, with its
		// own row — the human decides what it may do instead of the tool
		// silently reading the tab on screen.
		if terms, err := deps.Store.ListTerminals(); err == nil {
			for _, t := range terms {
				key := browser.TerminalPrefix + t.ID
				p := browser.Resolve(deps.Store, key)
				_, saved, _ := deps.Store.GetSetting(browser.SettingPrefix + key)
				domains := p.Domains
				if domains == nil {
					domains = []string{}
				}
				out = append(out, grant{
					Kind:      "terminal",
					TermID:    t.ID,
					Name:      t.Name,
					Workspace: t.WorkspaceID,
					Tier:      p.Tier,
					Domains:   domains,
					Saved:     saved,
				})
			}
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
			Term    string   `json:"term"`
			Tier    string   `json:"tier"`
			Domains []string `json:"domains"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "bad request: "+err.Error())
			return
		}
		req.Agent = strings.TrimSpace(req.Agent)
		req.Term = strings.TrimSpace(req.Term)
		// ADR-0143: one principal per request — a managed agent, or a terminal.
		// The key carries the difference (browser.TerminalPrefix), so a term id
		// can never widen an agent's grant or the other way round.
		key := req.Agent
		if req.Term != "" {
			if req.Agent != "" {
				writeErr(w, http.StatusBadRequest, "send either an agent or a term, not both")
				return
			}
			if _, err := deps.Store.GetTerminal(req.Term); err != nil {
				writeErr(w, http.StatusNotFound, "unknown terminal: "+req.Term)
				return
			}
			key = browser.TerminalPrefix + req.Term
		}
		if key == "" {
			writeErr(w, http.StatusBadRequest, "an agent id is required")
			return
		}
		if req.Agent != "" {
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
		}
		domains := make([]string, 0, len(req.Domains))
		for _, d := range req.Domains {
			if d = strings.TrimSpace(strings.ToLower(d)); d != "" {
				domains = append(domains, d)
			}
		}
		if err := browser.Save(deps.Store, key, browser.Policy{Tier: req.Tier, Domains: domains}); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		p := browser.Resolve(deps.Store, key)
		writeJSON(w, http.StatusOK, map[string]any{"agentId": req.Agent, "termId": req.Term, "tier": p.Tier, "domains": p.Domains, "saved": true})
	}
}
