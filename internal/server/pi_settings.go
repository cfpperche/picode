package server

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/cfpperche/picode/internal/pisettings"
	"github.com/cfpperche/picode/internal/store"
)

func registerPiSettingsRoutes(mux Registrar, deps Deps) {
	mux.HandleFunc("GET /api/pi-settings", handleGetPiSettings(deps))
	mux.HandleFunc("PUT /api/pi-settings", handlePutPiSettings(deps))
}

type piWritable struct {
	Global  bool `json:"global"`
	Project bool `json:"project"`
	Agent   bool `json:"agent"`
}

type piSettingsReport struct {
	Global    pisettings.Layer  `json:"global"`
	Project   *pisettings.Layer `json:"project,omitempty"`
	Agent     *store.Agent      `json:"agent,omitempty"`
	Effective pisettings.Layer  `json:"effective"`
	Writable  piWritable        `json:"writable"`
}

func handleGetPiSettings(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rep, err := piSettingsLoad(deps, r.URL.Query().Get("agentId"))
		if err != nil {
			writeErr(w, statusForStore(err), err.Error())
			return
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
		path, code, msg := piSettingsPath(deps, req.Layer, req.AgentID)
		if code != 0 {
			writeErr(w, code, msg)
			return
		}
		if err := pisettings.Apply(path, req.Patch); err != nil {
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
		rep, err := piSettingsLoad(deps, req.AgentID)
		if err != nil {
			writeErr(w, statusForStore(err), err.Error())
			return
		}
		writeJSON(w, http.StatusOK, rep)
	}
}

func piSettingsLoad(deps Deps, agentID string) (piSettingsReport, error) {
	global, err := pisettings.Load(pisettings.UserFile())
	if err != nil {
		return piSettingsReport{}, err
	}
	rep := piSettingsReport{
		Global:    global,
		Effective: global,
		Writable:  piWritable{Global: true},
	}
	if agentID == "" || deps.Store == nil {
		return rep, nil
	}
	agent, err := deps.Store.GetAgent(agentID)
	if err != nil {
		return piSettingsReport{}, err
	}
	rep.Agent = &agent
	rep.Writable.Agent = true
	wk, err := deps.Store.GetWorkspace(agent.WorkspaceID)
	if err != nil || store.IsFree(wk) {
		return rep, nil
	}
	layer, err := pisettings.Load(pisettings.ProjectFile(wk.Path))
	if err != nil {
		return piSettingsReport{}, err
	}
	rep.Project = &layer
	rep.Writable.Project = pisettings.Trusted(wk.Path)
	return rep, nil
}

func piSettingsPath(deps Deps, layer, agentID string) (string, int, string) {
	switch layer {
	case "global":
		return pisettings.UserFile(), 0, ""
	case "project":
		if agentID == "" || deps.Store == nil {
			return "", http.StatusBadRequest, "agent is required"
		}
		agent, err := deps.Store.GetAgent(agentID)
		if err != nil {
			return "", statusForStore(err), err.Error()
		}
		wk, err := deps.Store.GetWorkspace(agent.WorkspaceID)
		if err != nil {
			return "", statusForStore(err), err.Error()
		}
		if store.IsFree(wk) {
			return "", http.StatusBadRequest, "unbound agents have no workspace settings"
		}
		if !pisettings.Trusted(wk.Path) {
			return "", http.StatusConflict, "This folder is not trusted. Run /trust in the terminal."
		}
		return pisettings.ProjectFile(wk.Path), 0, ""
	default:
		return "", http.StatusBadRequest, "layer must be global or project"
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
