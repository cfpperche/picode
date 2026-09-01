package server

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/cfpperche/picode/internal/store"
)

// Active-role state published by the pi-roles extension (ADR-0033 amendment
// #2): ~/.pi/agent/roles-state/<agentId>.json, written on every mode change.
// The composer's role chip reads it through this endpoint. The file is
// ephemeral and best-effort — absent or unreadable is {"state": null}, never
// an error.

// rolesStateRoot, when set, replaces ~/.pi/agent/roles-state (tests only).
var rolesStateRoot string

func registerRolesState(mux *http.ServeMux, deps Deps) {
	mux.HandleFunc("GET /api/agents/{id}/role-state", handleAgentRoleState(deps))
}

func rolesStatePath(agentID string) string {
	root := rolesStateRoot
	if root == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		root = filepath.Join(home, ".pi", "agent", "roles-state")
	}
	return filepath.Join(root, agentID+".json")
}

// agentHasRolesPackage reports whether pi-roles is on the agent's effective
// package list. An uninstalled package leaves the state file orphaned on
// disk — without this gate the chip would outlive the extension. Fails
// open: a listing error must not hide live state.
func agentHasRolesPackage(deps Deps, agent store.Agent) bool {
	rep, err := loadPackageReport(deps, agent.WorkspaceID, agent.ID)
	if err != nil {
		return true
	}
	for _, p := range rep.Packages {
		if rep.Isolated && p.Scope != "agent" {
			continue
		}
		if strings.Contains(strings.ToLower(p.Source), "pi-roles") {
			return true
		}
	}
	return false
}

func handleAgentRoleState(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		agent, err := deps.Store.GetAgent(r.PathValue("id"))
		if err != nil {
			writeErr(w, http.StatusNotFound, "agent not found")
			return
		}
		none := map[string]any{"state": nil}
		if !agentHasRolesPackage(deps, agent) {
			writeJSON(w, http.StatusOK, none)
			return
		}
		path := rolesStatePath(agent.ID)
		if path == "" {
			writeJSON(w, http.StatusOK, none)
			return
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			writeJSON(w, http.StatusOK, none)
			return
		}
		var state map[string]any
		if err := json.Unmarshal(raw, &state); err != nil || state == nil {
			writeJSON(w, http.StatusOK, none)
			return
		}
		if v, ok := state["v"].(float64); !ok || v != 1 {
			// A future contract version is not something this build can
			// render — hide the chip rather than guess.
			writeJSON(w, http.StatusOK, none)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"state": state})
	}
}
