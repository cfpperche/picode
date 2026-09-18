package server

// The computer tool (ADR-0148): an agent, or a CLI in a PiCode terminal,
// acts on the Windows desktop through the desktop shell. One grant per
// principal is the whole policy; the command rides the work-browser line
// (ADR-0132) as a kind-tagged frame and the shell runs it on its desk thread.
//
// POST /api/computer/tool      {agent, term, call, action, params} → {action, output}
// GET  /api/computer/policies  every principal with its grant
// POST /api/computer/policy    {agent|term, enabled}
// GET  /api/computer/audit     the newest computer.step rows
//
// Every call — allowed, refused or failed — is one computer.step event on the
// change feed: a record for the owner and the dashboard, not a boundary.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/browser"
	"github.com/cfpperche/picode/internal/computer"
	"github.com/cfpperche/picode/internal/grant"
)

func registerComputerRoutes(mux Registrar, deps Deps) {
	mux.HandleFunc("POST /api/computer/tool", handleComputerTool(deps))
	mux.HandleFunc("GET /api/computer/policies", handleComputerPolicies(deps))
	mux.HandleFunc("POST /api/computer/policy", handleComputerPolicySave(deps))
	mux.HandleFunc("GET /api/computer/audit", handleComputerAudit(deps))
}

// computerRefused is the one text an unauthorised caller gets: it names the
// switch, so the human knows where to look.
const computerRefused = "this agent may not use the computer — turn it on in Settings ▸ Computer"

func handleComputerTool(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.Browser == nil {
			writeErr(w, http.StatusServiceUnavailable, "the shell line is not running")
			return
		}
		var req struct {
			Agent  string          `json:"agent"`
			Term   string          `json:"term,omitempty"`
			Call   string          `json:"call,omitempty"`
			Action string          `json:"action"`
			Params json.RawMessage `json:"params,omitempty"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "bad request: "+err.Error())
			return
		}
		action, ok := computer.ActionFor(req.Action)
		if !ok {
			writeErr(w, http.StatusBadRequest, "unknown action "+strconv.Quote(req.Action)+" — use one of "+strings.Join(computer.Actions(), ", "))
			return
		}
		params := json.RawMessage("{}")
		if len(req.Params) > 0 {
			var probe map[string]any
			if err := json.Unmarshal(req.Params, &probe); err != nil {
				writeErr(w, http.StatusBadRequest, "params must be an object: "+err.Error())
				return
			}
			params = req.Params
		}
		key := grant.Key(req.Agent, req.Term)
		step := computerStep{principal: key, termID: strings.TrimSpace(req.Term), agentID: strings.TrimSpace(req.Agent), call: req.Call, action: action}
		if key == "" {
			step.record(deps, "refused", "no identity", nil)
			writeErr(w, http.StatusForbidden, computerRefused+" (this caller has no identity: run it as a PiCode agent or inside a PiCode terminal)")
			return
		}
		if !computer.Resolve(deps.Store, key).Enabled {
			step.record(deps, "refused", "no grant", nil)
			writeErr(w, http.StatusForbidden, computerRefused)
			return
		}
		started := time.Now()
		output, err := deps.Browser.Dispatch(r.Context(), browser.Command{
			Kind:      "computer",
			Method:    action,
			Params:    params,
			Principal: key,
			Timeout:   computer.Timeout(action),
		})
		step.ms = time.Since(started).Milliseconds()
		if err != nil {
			step.record(deps, "failed", err.Error(), nil)
			writeErr(w, http.StatusBadGateway, err.Error())
			return
		}
		step.record(deps, "allowed", "", output)
		writeJSON(w, http.StatusOK, map[string]any{"action": action, "output": output})
	}
}

// computerStep is one audit row in the making.
type computerStep struct {
	principal, termID, agentID, call, action string
	ms                                       int64
}

// record appends the computer.step event. The agent id column is a foreign
// key, so it is set only for a managed agent; a terminal principal travels
// in the payload. A write failure never blocks the call the owner asked for.
func (s computerStep) record(deps Deps, outcome, reason string, output json.RawMessage) {
	if deps.Store == nil {
		return
	}
	data := map[string]any{"principal": s.principal, "action": s.action, "outcome": outcome}
	if s.termID != "" {
		data["termId"] = s.termID
	}
	if s.call != "" {
		data["call"] = s.call
	}
	if reason != "" {
		data["reason"] = reason
	}
	if s.ms > 0 {
		data["ms"] = s.ms
	}
	if len(output) > 0 {
		var meta struct {
			Image string `json:"image"`
			Meta  struct {
				Display int             `json:"display"`
				Window  json.RawMessage `json:"window"`
			} `json:"meta"`
		}
		if json.Unmarshal(output, &meta) == nil {
			if meta.Image != "" {
				sum := sha256.Sum256([]byte(meta.Image))
				data["imageSha256"] = hex.EncodeToString(sum[:])
			}
			if meta.Meta.Display > 0 {
				data["display"] = meta.Meta.Display
			}
			if len(meta.Meta.Window) > 0 && string(meta.Meta.Window) != "null" {
				data["window"] = meta.Meta.Window
			}
		}
	}
	// The agent column is a foreign key: an id the store does not know (a
	// caller claiming an agent that does not exist) cannot go there, so the
	// row keeps the id in its payload and falls back to no column. A refusal
	// is worth recording exactly when the caller is a stranger.
	if s.agentID != "" {
		data["agentId"] = s.agentID
		if err := deps.Store.AppendEvent("computer.step", &s.agentID, nil, data); err == nil {
			return
		}
	}
	_ = deps.Store.AppendEvent("computer.step", nil, nil, data)
}

// handleComputerPolicies lists every principal — managed agents and the
// terminals — with its grant, the way the browser's editor does.
func handleComputerPolicies(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.Store == nil {
			writeErr(w, http.StatusServiceUnavailable, "store is not open")
			return
		}
		agents, err := deps.Store.ListAllAgents()
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		type row struct {
			Kind      string `json:"kind"`
			AgentID   string `json:"agentId,omitempty"`
			TermID    string `json:"termId,omitempty"`
			Name      string `json:"name"`
			Workspace string `json:"workspaceId,omitempty"`
			Enabled   bool   `json:"enabled"`
			Saved     bool   `json:"saved"`
		}
		out := make([]row, 0, len(agents))
		for _, a := range agents {
			_, saved, _ := deps.Store.GetSetting(computer.SettingPrefix + a.ID)
			out = append(out, row{Kind: "agent", AgentID: a.ID, Name: a.Name, Workspace: a.WorkspaceID, Enabled: computer.Resolve(deps.Store, a.ID).Enabled, Saved: saved})
		}
		if terms, err := deps.Store.ListTerminals(); err == nil {
			for _, t := range terms {
				key := grant.TerminalPrefix + t.ID
				_, saved, _ := deps.Store.GetSetting(computer.SettingPrefix + key)
				out = append(out, row{Kind: "terminal", TermID: t.ID, Name: t.Name, Workspace: t.WorkspaceID, Enabled: computer.Resolve(deps.Store, key).Enabled, Saved: saved})
			}
		}
		writeJSON(w, http.StatusOK, map[string]any{"policies": out})
	}
}

// handleComputerPolicySave flips one principal's switch. One principal per
// request; the key carries which namespace it lives in.
func handleComputerPolicySave(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.Store == nil {
			writeErr(w, http.StatusServiceUnavailable, "store is not open")
			return
		}
		var req struct {
			Agent   string `json:"agent"`
			Term    string `json:"term"`
			Enabled bool   `json:"enabled"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "bad request: "+err.Error())
			return
		}
		req.Agent = strings.TrimSpace(req.Agent)
		req.Term = strings.TrimSpace(req.Term)
		if req.Agent != "" && req.Term != "" {
			writeErr(w, http.StatusBadRequest, "send either an agent or a term, not both")
			return
		}
		key := grant.Key(req.Agent, req.Term)
		if key == "" {
			writeErr(w, http.StatusBadRequest, "an agent id or a term id is required")
			return
		}
		if req.Term != "" {
			if _, err := deps.Store.GetTerminal(req.Term); err != nil {
				writeErr(w, http.StatusNotFound, "unknown terminal: "+req.Term)
				return
			}
		} else {
			agents, err := deps.Store.ListAllAgents()
			if err != nil {
				writeErr(w, http.StatusInternalServerError, err.Error())
				return
			}
			known := false
			for _, a := range agents {
				if a.ID == req.Agent {
					known = true
					break
				}
			}
			if !known {
				writeErr(w, http.StatusNotFound, "unknown agent: "+req.Agent)
				return
			}
		}
		if err := computer.Save(deps.Store, key, req.Enabled); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"agentId": req.Agent, "termId": req.Term, "enabled": req.Enabled, "saved": true})
	}
}

// handleComputerAudit is the settings card's reader: the newest steps,
// allowed or refused, so "audited" is something the owner can look at.
func handleComputerAudit(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.Store == nil {
			writeErr(w, http.StatusServiceUnavailable, "store is not open")
			return
		}
		limit := 20
		if raw := r.URL.Query().Get("limit"); raw != "" {
			if n, err := strconv.Atoi(raw); err == nil && n > 0 && n <= 200 {
				limit = n
			}
		}
		rows, err := deps.Store.EventsOfType("computer.step", limit)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		type step struct {
			ID        int64           `json:"id"`
			Principal string          `json:"principal"`
			Action    string          `json:"action"`
			Outcome   string          `json:"outcome"`
			Reason    string          `json:"reason,omitempty"`
			AgentID   string          `json:"agentId,omitempty"`
			TermID    string          `json:"termId,omitempty"`
			Ms        int64           `json:"ms,omitempty"`
			Window    json.RawMessage `json:"window,omitempty"`
			CalledAt  string          `json:"calledAt"`
		}
		out := make([]step, 0, len(rows))
		for _, ev := range rows {
			var data struct {
				Principal string          `json:"principal"`
				Action    string          `json:"action"`
				Outcome   string          `json:"outcome"`
				Reason    string          `json:"reason"`
				TermID    string          `json:"termId"`
				Ms        int64           `json:"ms"`
				Window    json.RawMessage `json:"window"`
			}
			_ = json.Unmarshal(ev.Data, &data)
			entry := step{ID: ev.ID, Principal: data.Principal, Action: data.Action, Outcome: data.Outcome, Reason: data.Reason, TermID: data.TermID, Ms: data.Ms, Window: data.Window, CalledAt: ev.CreatedAt}
			if ev.AgentID != nil {
				entry.AgentID = *ev.AgentID
			}
			out = append(out, entry)
		}
		writeJSON(w, http.StatusOK, map[string]any{"steps": out})
	}
}
