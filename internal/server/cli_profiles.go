package server

import (
	"net/http"
	"os"

	"github.com/cfpperche/picode/internal/clilaunch"
	"github.com/cfpperche/picode/internal/store"
)

func cliIntegrationPrepared(dir string, cli clilaunch.CLI) bool {
	if !interceptWired(dir, cli.ID, cli.Command) {
		return false
	}
	files := append(cliIntegrationPlan(cli.ID, dir, hookScriptPath(dir)).Files, hookScriptPath(dir))
	for _, p := range files {
		if st, err := os.Stat(p); err != nil || st.IsDir() || st.Size() == 0 {
			return false
		}
	}
	return true
}

func registerCLIProfileRoutes(mux Registrar, deps Deps) {
	mux.HandleFunc("GET /api/clis/profiles", func(w http.ResponseWriter, r *http.Request) {
		rows, err := deps.Store.CLIProfiles()
		if err != nil {
			writeErr(w, 500, err.Error())
			return
		}
		writeJSON(w, 200, map[string]any{"profiles": rows})
	})
	mux.HandleFunc("PUT /api/clis/profiles/{id}", func(w http.ResponseWriter, r *http.Request) {
		var p store.CLIProfile
		if !readCLIJSON(w, r, &p) {
			return
		}
		p.ID = r.PathValue("id")
		if err := deps.Store.SetCLIProfile(p); err != nil {
			writeErr(w, 400, err.Error())
			return
		}
		writeJSON(w, 200, map[string]any{"saved": true})
	})
	mux.HandleFunc("DELETE /api/clis/profiles/{id}", func(w http.ResponseWriter, r *http.Request) {
		if err := deps.Store.DeleteCLIProfile(r.PathValue("id")); err != nil {
			writeStoreErr(w, err)
			return
		}
		w.WriteHeader(204)
	})
}
