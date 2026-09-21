package server

import (
	"net/http"
	"strings"

	"github.com/cfpperche/picode/internal/pkgs"
)

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
		if id := strings.TrimSpace(r.URL.Query().Get("workspace")); id != "" && deps.Store != nil {
			dir, err := packageProjectDir(deps, id)
			if err != nil {
				writeErr(w, statusForPackageErr(err), err.Error())
				return
			}
			q.WorkspacePath = dir
		}
		// An agent that is gone contributes no agent rows (the legacy read
		// answers the machine scope the same way) — never an error.
		if id := strings.TrimSpace(r.URL.Query().Get("agent")); id != "" && deps.Store != nil {
			if a, err := deps.Store.GetAgent(id); err == nil {
				q.AgentSources = a.Packages
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
