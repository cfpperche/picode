package server

import (
	"net/http"

	"github.com/cfpperche/picode/internal/catalog"
	"github.com/cfpperche/picode/internal/session"
)

func handleWorkspaceStatus(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		wk, agent, err := loadWS(deps, r.PathValue("id"))
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		path := ""
		if agent.SessionPath != nil {
			path = *agent.SessionPath
		}
		win := 0
		if agent.Provider != nil && agent.Model != nil && *agent.Provider != "" && *agent.Model != "" {
			if rep, err := catalog.Load(deps.AgentCmd); err == nil {
				for _, p := range rep.Providers {
					if p.ID != *agent.Provider {
						continue
					}
					for _, m := range p.Models {
						if m.ID == *agent.Model {
							win = session.ParseContextWindow(m.Context)
							break
						}
					}
				}
			}
		}
		writeJSON(w, http.StatusOK, session.BuildBar(wk.Path, path, win))
	}
}
