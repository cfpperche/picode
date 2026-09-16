package server

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/cfpperche/picode/internal/clilaunch"
	"github.com/cfpperche/picode/internal/store"
)

func cliIntegrationPrepared(dir string, cli clilaunch.CLI) bool {
	if cli.ID == "agy" {
		return agyReporterPrepared(dir)
	}
	if cli.ID == "muse" {
		return museHooksPrepared(dir)
	}
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

// agyReporterPrepared is the applied check for the wrapper-less reporter:
// the script exists and the user's title block points at it. A user-side
// `/title off` reads as not-applied (repair reinstalls), never as an error.
func agyReporterPrepared(dir string) bool {
	rep := agyTitleReporterPath(dir)
	if st, err := os.Stat(rep); err != nil || st.IsDir() || st.Size() == 0 {
		return false
	}
	if st, err := os.Stat(hookScriptPath(dir)); err != nil || st.IsDir() || st.Size() == 0 {
		return false
	}
	settings, err := agySettingsPath()
	if err != nil {
		return false
	}
	raw, err := os.ReadFile(settings)
	if err != nil {
		return false
	}
	var doc map[string]any
	if json.Unmarshal(raw, &doc) != nil {
		return false
	}
	cur, ok := doc["title"].(map[string]any)
	if !ok {
		return false
	}
	cmd, _ := cur["command"].(string)
	return cmd == rep
}

// museHooksPrepared is the applied check for the wrapper-less observer:
// the hook script exists and every installed event in the user's settings
// carries our command. A user-side removal reads as not-applied (repair
// reinstalls), never as an error.
func museHooksPrepared(dir string) bool {
	hook := museHookPath(dir)
	if st, err := os.Stat(hook); err != nil || st.IsDir() || st.Size() == 0 {
		return false
	}
	if st, err := os.Stat(hookScriptPath(dir)); err != nil || st.IsDir() || st.Size() == 0 {
		return false
	}
	settings, err := museSettingsPath()
	if err != nil {
		return false
	}
	raw, err := os.ReadFile(settings)
	if err != nil {
		return false
	}
	var doc map[string]any
	if json.Unmarshal(raw, &doc) != nil {
		return false
	}
	hooks, ok := doc["hooks"].(map[string]any)
	if !ok {
		return false
	}
	for _, ev := range museHookEvents {
		cur, present := hooks[ev]
		if !present {
			return false
		}
		ours, _, ok := museHookCommands(cur, hook)
		if !ok || !ours {
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
