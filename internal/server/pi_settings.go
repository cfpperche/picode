package server

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/cfpperche/picode/internal/pisettings"
	"github.com/cfpperche/picode/internal/store"
)

func registerPiSettingsRoutes(mux *http.ServeMux, deps Deps) {
	mux.HandleFunc("GET /api/pi-settings", handleGetPiSettings(deps))
	mux.HandleFunc("PUT /api/pi-settings", handlePutPiSettings(deps))
}

type piSettingsReport struct {
	Global    pisettings.Layer  `json:"global"`
	Project   *pisettings.Layer `json:"project,omitempty"`
	Agent     *store.Agent      `json:"agent,omitempty"`
	Effective pisettings.Layer  `json:"effective"`
}

func handleGetPiSettings(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		global, err := pisettings.Load(pisettings.UserFile())
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		rep := piSettingsReport{Global: global, Effective: global}
		id := r.URL.Query().Get("agentId")
		if id == "" {
			writeJSON(w, http.StatusOK, rep)
			return
		}
		agent, err := deps.Store.GetAgent(id)
		if err != nil {
			writeErr(w, statusForStore(err), err.Error())
			return
		}
		rep.Agent = &agent
		wk, err := deps.Store.GetWorkspace(agent.WorkspaceID)
		if err == nil && !store.IsFree(wk) {
			layer, err := pisettings.Load(pisettings.ProjectFile(wk.Path))
			if err != nil {
				writeErr(w, http.StatusInternalServerError, err.Error())
				return
			}
			rep.Project = &layer
		}
		writeJSON(w, http.StatusOK, rep)
	}
}

type piSettingsPut struct {
	AgentID string           `json:"agentId"`
	Layer   string           `json:"layer"`
	Patch   pisettings.Patch `json:"patch"`
}

func handlePutPiSettings(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req piSettingsPut
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid json")
			return
		}
		if req.Layer != "global" {
			writeErr(w, http.StatusBadRequest, "only global writes in this phase")
			return
		}
		if err := pisettings.Apply(pisettings.UserFile(), req.Patch); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		if req.AgentID != "" && deps.Runtime != nil {
			if ma := deps.Runtime.Get(req.AgentID); ma != nil {
				ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
				defer cancel()
				liveApply(ctx, ma, req.Patch)
			}
		}
		global, err := pisettings.Load(pisettings.UserFile())
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, piSettingsReport{Global: global, Effective: global})
	}
}

func liveApply(ctx context.Context, ma interface {
	SetAutoCompaction(context.Context, bool) error
	SetSteeringMode(context.Context, string) error
	SetFollowUpMode(context.Context, string) error
}, p pisettings.Patch) {
	if p.CompactionEnabled != nil {
		_ = ma.SetAutoCompaction(ctx, *p.CompactionEnabled)
	}
	if p.SteeringMode != nil {
		_ = ma.SetSteeringMode(ctx, *p.SteeringMode)
	}
	if p.FollowUpMode != nil {
		_ = ma.SetFollowUpMode(ctx, *p.FollowUpMode)
	}
}
