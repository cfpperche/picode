package server

import (
	"net/http"
	"strings"

	"github.com/cfpperche/picode/internal/pkgs"
	"github.com/cfpperche/picode/internal/store"
)

// agentLayerName is the agent's name as a pane badges it: its own, or — for the
// unnamed default agent — the workspace it belongs to (the same rule the web
// tree applies, so one agent reads the same everywhere).
func agentLayerName(a store.Agent, ws store.Workspace) string {
	if strings.TrimSpace(a.Name) != "" && a.Name != "default" {
		return a.Name
	}
	if strings.TrimSpace(ws.Name) != "" {
		return ws.Name
	}
	return a.Name
}

// handlePackageReport is the unified read (ADR-0176): one report for any CLI,
// where the CLI's own mechanism decides the scopes, the caps and the catalog.
// The Pi pane keeps its shape at GET /api/packages; this is the shape the two
// panes merge onto, so both engines answer through one interface here.
func handlePackageReport(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		driver, ok := packageReadDriver(w, r)
		if !ok {
			return
		}
		q := pkgs.Query{
			Scope:  pkgs.Scope(strings.TrimSpace(r.URL.Query().Get("scope"))),
			Vendor: strings.TrimSpace(r.URL.Query().Get("vendor")),
			Fresh:  r.URL.Query().Get("refresh") == "1",
		}
		var ws store.Workspace
		if id := strings.TrimSpace(r.URL.Query().Get("workspace")); id != "" && deps.Store != nil {
			found, err := deps.Store.GetWorkspace(id)
			if err != nil {
				writeErr(w, statusForPackageErr(err), err.Error())
				return
			}
			ws = found
			q.WorkspacePath = ws.Path
			// The folder's name, so a driver whose scope radio names it (Pi's
			// says which workspace an install lands in) can answer it here.
			q.WorkspaceName = ws.Name
		}
		// An agent that is gone contributes no agent rows (the legacy read
		// answers the machine scope the same way) — never an error.
		if id := strings.TrimSpace(r.URL.Query().Get("agent")); id != "" && deps.Store != nil {
			if a, err := deps.Store.GetAgent(id); err == nil {
				q.AgentSources = a.Packages
				q.AgentName = agentLayerName(a, ws)
				// "Only this agent's packages" is a fact about the agent row,
				// and the pane that draws its checkbox reads it here — the same
				// read the legacy route answers (ADR-0176 slice 3).
				q.AgentIsolated = a.PackagesIsolated
			}
		}
		rep, err := driver.List(r.Context(), q)
		if err != nil {
			writeErr(w, statusForPackageErr(err), err.Error())
			return
		}
		writeJSON(w, http.StatusOK, rep)
	}
}
