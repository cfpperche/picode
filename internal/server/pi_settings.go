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
		// A machine-only key on another layer would write a name pi never
		// reads there. The reason is the key, not the folder, so this answers
		// before trust or path resolution can blame something else.
		if req.Layer != "" && req.Layer != "global" {
			if bad := machineOnlyInPatch(req.Patch); bad != "" {
				writeErr(w, http.StatusBadRequest, bad+" is kept for this machine only; edit it on the This machine layer")
				return
			}
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
		rep, err := piSettingsLoad(deps, req.AgentID)
		if err != nil {
			writeErr(w, statusForStore(err), err.Error())
			return
		}
		if req.AgentID != "" && deps.Runtime != nil {
			if ma := deps.Runtime.Get(req.AgentID); ma != nil {
				ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
				defer cancel()
				liveApply(ctx, ma, livePatch(req.Patch, rep))
			}
		}
		writeJSON(w, http.StatusOK, rep)
	}
}

// livePatch is what a running agent should adopt now. An explicit set applies
// as given; a reset makes the parent layer (or Pi's own default) current, so
// the effective knobs — not the override that just left the file — are what a
// live agent must see. Compaction, steering and follow-up are the fields the
// runtime owns (ADR-0101); the rest land on the next start.
func livePatch(p pisettings.Patch, rep piSettingsReport) pisettings.Patch {
	if len(p.Reset) == 0 {
		return p
	}
	c := rep.Global.CompactionEnabled
	s := rep.Global.SteeringMode
	f := rep.Global.FollowUpMode
	if rep.Project != nil {
		if rep.Project.Has["compactionEnabled"] {
			c = rep.Project.CompactionEnabled
		}
		if rep.Project.Has["steeringMode"] {
			s = rep.Project.SteeringMode
		}
		if rep.Project.Has["followUpMode"] {
			f = rep.Project.FollowUpMode
		}
	}
	p.CompactionEnabled, p.SteeringMode, p.FollowUpMode = &c, &s, &f
	return p
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

// machineOnlyInPatch names the first machine-only field a patch touches, or "".
func machineOnlyInPatch(p pisettings.Patch) string {
	for key, set := range map[string]bool{
		"theme":               p.Theme != nil,
		"hideThinkingBlock":   p.HideThinkingBlock != nil,
		"defaultProjectTrust": p.DefaultProjectTrust != nil,
		"shellPath":           p.ShellPath != nil,
		"quietStartup":        p.QuietStartup != nil,
	} {
		if set {
			return key
		}
	}
	for _, key := range p.Reset {
		if pisettings.MachineOnly(key) {
			return key
		}
	}
	return ""
}
