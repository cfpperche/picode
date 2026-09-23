package server

import (
	"errors"
	"net/http"
	"strings"

	"github.com/cfpperche/picode/internal/pisettings"
	"github.com/cfpperche/picode/internal/skills"
)

// registerSkillRoutes serves the Skills tab (ADR-0196, docs/plans/skills.md).
// Slice 1 is read-only: what one CLI loads, from which folder, and why.
func registerSkillRoutes(mux Registrar, deps Deps) {
	mux.HandleFunc("GET /api/skills/report", handleSkillsReport(deps))
}

// skillsTrusted answers a CLI's own trust record where PiCode can read it:
// Pi's trust.json. Every other CLI keeps trust where PiCode does not read,
// and the report says the skill loads once the folder is trusted there.
func skillsTrusted(cli, workspace string) *bool {
	if cli != "pi" {
		return nil
	}
	t := pisettings.Trusted(workspace)
	return &t
}

func handleSkillsReport(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cli := strings.TrimSpace(r.URL.Query().Get("cli"))
		if cli == "" {
			cli = "pi"
		}
		q := skills.Query{CLI: cli, Trusted: skillsTrusted}
		if id := strings.TrimSpace(r.URL.Query().Get("workspace")); id != "" && deps.Store != nil {
			ws, err := deps.Store.GetWorkspace(id)
			if err != nil {
				writeErr(w, http.StatusNotFound, "workspace not found")
				return
			}
			q.Workspace = ws.Path
		}
		rep, err := skills.Read(q)
		if errors.Is(err, skills.ErrUnknownCLI) {
			writeErr(w, http.StatusBadRequest, "unknown cli "+cli+"; skills are declared for: "+strings.Join(skills.CLIs(), ", "))
			return
		}
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, rep)
	}
}
