package server

import (
	"net/http"

	"github.com/cfpperche/picode/internal/clilaunch"
	"github.com/cfpperche/picode/internal/clisession"
)

// registerCLISessionRoutes adds the multi-CLI sessions index (ADR-0079
// phase 2). Read-only: pi's management surface (delete, auto-clean,
// resume, in-use guards) keeps living on its own endpoints.
func registerCLISessionRoutes(mux Registrar, deps Deps) {
	mux.HandleFunc("GET /api/clis/{cli}/sessions", handleCLISessions(deps))
}

// handleCLISessions answers GET /api/clis/{cli}/sessions?cwd=<folder>.
// Decision table (covered in cli_sessions_test.go):
//
//	unknown catalog CLI                     → 404
//	known CLI, nothing on disk              → 200, empty list
//	cwd filter set                          → only sessions in that folder
//	malformed or undeclared files           → skipped, never a 500
//	session cwd matches a PiCode workspace  → tagged workspaceId/workspace
func handleCLISessions(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cli, ok := clilaunch.Find(r.PathValue("cli"))
		if !ok {
			writeErr(w, http.StatusNotFound, "Unknown CLI.")
			return
		}
		src, ok := clisession.Get(cli.ID)
		if !ok {
			writeErr(w, http.StatusNotFound, "Unknown CLI.")
			return
		}
		rows, err := src.List(r.URL.Query().Get("cwd"))
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		views := make([]cliSessionView, 0, len(rows))
		var total int64
		wss, err := deps.Store.ListWorkspaces()
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		for _, s := range rows {
			total += s.Size
			v := cliSessionView{Summary: s}
			for _, wk := range wss {
				if wk.Path != "" && wk.Path == s.Cwd {
					v.WorkspaceID = wk.ID
					v.Workspace = wk.Name
					break
				}
			}
			views = append(views, v)
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"cli":        cli.ID,
			"sessions":   views,
			"count":      len(views),
			"totalBytes": total,
		})
	}
}

// cliSessionView adds the PiCode workspace context (if any) to a session.
type cliSessionView struct {
	clisession.Summary
	WorkspaceID string `json:"workspaceId,omitempty"`
	Workspace   string `json:"workspace,omitempty"`
}
