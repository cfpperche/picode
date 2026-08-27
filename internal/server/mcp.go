package server

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/mcp"
	"github.com/cfpperche/picode/internal/pipkg"
	"github.com/cfpperche/picode/internal/store"
)

func registerMCPRoutes(mux *http.ServeMux, deps Deps) {
	mux.HandleFunc("GET /api/mcp", handleMCPGet(deps))
	mux.HandleFunc("POST /api/mcp", handleMCPAdd(deps))
	mux.HandleFunc("POST /api/mcp/import", handleMCPImport(deps))
	mux.HandleFunc("POST /api/mcp/auth", handleMCPAuth(deps))
	mux.HandleFunc("GET /api/mcp/auth/status", handleMCPAuthStatus(deps))
	mux.HandleFunc("POST /api/mcp/auth/reply", handleMCPAuthReply(deps))
	mux.HandleFunc("PATCH /api/mcp", handleMCPToggle(deps))
	mux.HandleFunc("DELETE /api/mcp", handleMCPRemove(deps))
}

type mcpMutateReq struct {
	Scope       string            `json:"scope"`
	WorkspaceID string            `json:"workspaceId"`
	AgentID     string            `json:"agentId"`
	Name        string            `json:"name"`
	Disabled    *bool             `json:"disabled"`
	Command     string            `json:"command"`
	Args        []string          `json:"args"`
	URL         string            `json:"url"`
	Env         map[string]string `json:"env"`
	Headers     map[string]string `json:"headers"`
	Auth        string            `json:"auth"`
	BearerToken string            `json:"bearerToken"`
	Kinds       []string          `json:"kinds"`
	Picks       []mcp.ImportPick  `json:"picks"`
	ID          string            `json:"id"`
	Value       string            `json:"value"`
	Cancelled   bool              `json:"cancelled"`
}

func handleMCPGet(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p, sources, err := mcpPaths(deps, r.URL.Query().Get("workspace"), r.URL.Query().Get("agent"))
		if err != nil {
			writeErr(w, statusForStore(err), err.Error())
			return
		}
		rep, err := mcp.List(p)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		rep.Adapter.Installed = mcp.AdapterConfigured(sources)
		applyMCPLive(deps, r.URL.Query().Get("agent"), &rep)
		writeJSON(w, http.StatusOK, rep)
	}
}

func handleMCPImport(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req mcpMutateReq
		_ = json.NewDecoder(r.Body).Decode(&req)
		p, sources, err := mcpPaths(deps, req.WorkspaceID, req.AgentID)
		if err != nil {
			writeErr(w, statusForStore(err), err.Error())
			return
		}
		if !mcp.AdapterConfigured(sources) {
			writeErr(w, http.StatusConflict, "install pi-mcp-adapter first")
			return
		}
		picks := req.Picks
		if picks == nil && req.Kinds != nil {
			writeErr(w, http.StatusBadRequest, "choose which servers")
			return
		}
		if picks == nil {
			writeErr(w, http.StatusBadRequest, "choose which servers")
			return
		}
		res, err := mcp.ImportHosts(p, picks)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		rep, err := mcp.List(p)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		rep.Adapter.Installed = mcp.AdapterConfigured(sources)
		writeJSON(w, http.StatusOK, map[string]any{"import": res, "adapter": rep.Adapter, "layers": rep.Layers, "servers": rep.Servers, "presets": rep.Presets, "imports": rep.Imports, "found": rep.Found})
	}
}

func handleMCPAdd(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req mcpMutateReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		p, sources, err := mcpPaths(deps, req.WorkspaceID, req.AgentID)
		if err != nil {
			writeErr(w, statusForStore(err), err.Error())
			return
		}
		if !mcp.AdapterConfigured(sources) {
			writeErr(w, http.StatusConflict, "install pi-mcp-adapter first")
			return
		}
		entry := mcp.Entry{Command: req.Command, Args: req.Args, URL: req.URL, Env: req.Env, Headers: req.Headers, Auth: req.Auth, BearerToken: req.BearerToken}
		if err := mcp.Add(p, req.Scope, req.Name, entry); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		writeMCP(w, deps, p, sources, req.AgentID)
	}
}

func handleMCPToggle(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req mcpMutateReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if req.Disabled == nil {
			writeErr(w, http.StatusBadRequest, "disabled is required")
			return
		}
		p, sources, err := mcpPaths(deps, req.WorkspaceID, req.AgentID)
		if err != nil {
			writeErr(w, statusForStore(err), err.Error())
			return
		}
		if !mcp.AdapterConfigured(sources) {
			writeErr(w, http.StatusConflict, "install pi-mcp-adapter first")
			return
		}
		if err := mcp.Toggle(p, req.Scope, req.Name, *req.Disabled); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		writeMCP(w, deps, p, sources, req.AgentID)
	}
}

func handleMCPRemove(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		scope := r.URL.Query().Get("scope")
		name := r.URL.Query().Get("name")
		wsID := r.URL.Query().Get("workspace")
		agentID := r.URL.Query().Get("agent")
		if name == "" {
			var req mcpMutateReq
			_ = json.NewDecoder(r.Body).Decode(&req)
			name = req.Name
			if scope == "" {
				scope = req.Scope
			}
			if wsID == "" {
				wsID = req.WorkspaceID
			}
			if agentID == "" {
				agentID = req.AgentID
			}
		}
		p, sources, err := mcpPaths(deps, wsID, agentID)
		if err != nil {
			writeErr(w, statusForStore(err), err.Error())
			return
		}
		if !mcp.AdapterConfigured(sources) {
			writeErr(w, http.StatusConflict, "install pi-mcp-adapter first")
			return
		}
		if err := mcp.Remove(p, scope, name); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		writeMCP(w, deps, p, sources, agentID)
	}
}

func handleMCPAuth(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req mcpMutateReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if err := mcp.ValidName(req.Name); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		if deps.Runtime == nil {
			writeErr(w, http.StatusConflict, "pi is not configured")
			return
		}
		p, _, err := mcpPaths(deps, req.WorkspaceID, req.AgentID)
		if err != nil {
			writeErr(w, statusForStore(err), err.Error())
			return
		}
		cwd := p.AgentCwd
		if cwd == "" {
			cwd = p.Cwd
		}
		ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
		defer cancel()
		id, url, err := deps.Runtime.BeginMCPAuth(ctx, req.AgentID, cwd, req.Name)
		if err != nil {
			if ctx.Err() != nil {
				writeErr(w, http.StatusGatewayTimeout, "sign-in timed out")
				return
			}
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		if id == "" {
			writeJSON(w, http.StatusOK, map[string]any{"ok": true})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"id": id, "url": url})
	}
}

func handleMCPAuthStatus(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimSpace(r.URL.Query().Get("id"))
		if id == "" || deps.Runtime == nil {
			writeErr(w, http.StatusBadRequest, "id is required")
			return
		}
		done, err, found := deps.Runtime.MCPAuthStatus(id)
		if !found {
			writeJSON(w, http.StatusOK, map[string]any{"pending": false})
			return
		}
		if !done {
			writeJSON(w, http.StatusOK, map[string]any{"pending": true})
			return
		}
		if err != nil {
			writeJSON(w, http.StatusOK, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	}
}

func handleMCPAuthReply(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req mcpMutateReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if req.ID == "" || deps.Runtime == nil {
			writeErr(w, http.StatusBadRequest, "id is required")
			return
		}
		if err := deps.Runtime.ReplyMCPAuth(req.ID, req.Value, req.Cancelled); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	}
}

func writeMCP(w http.ResponseWriter, deps Deps, p mcp.Paths, sources []string, agentID string) {
	rep, err := mcp.List(p)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
		return
	}
	rep.Adapter.Installed = mcp.AdapterConfigured(sources)
	applyMCPLive(deps, agentID, &rep)
	writeJSON(w, http.StatusOK, rep)
}

func applyMCPLive(deps Deps, agentID string, rep *mcp.Report) {
	running := agentID != "" && deps.Runtime != nil && deps.Runtime.Get(agentID) != nil
	mcp.ApplyLive(rep, mcp.ReadLive(mcp.LivePath(deps.DataDir, agentID), 0), running)
}

func mcpPaths(deps Deps, workspaceID, agentID string) (mcp.Paths, []string, error) {
	p := mcp.Paths{}
	sources := packageSources(deps, workspaceID, agentID)
	if workspaceID != "" && deps.Store != nil {
		ws, err := deps.Store.GetWorkspace(workspaceID)
		if err != nil {
			return p, sources, err
		}
		p.Cwd = ws.Path
	}
	if agentID != "" && deps.Store != nil {
		a, err := deps.Store.GetAgent(agentID)
		if err != nil {
			return p, sources, err
		}
		if a.WorkPath != nil && strings.TrimSpace(*a.WorkPath) != "" {
			p.AgentCwd = strings.TrimSpace(*a.WorkPath)
			if p.Cwd == "" && a.WorkspaceID != store.FreeWorkspaceID {
				if ws, err := deps.Store.GetWorkspace(a.WorkspaceID); err == nil {
					p.Cwd = ws.Path
				}
			}
		}
	}
	return p, sources, nil
}

func packageSources(deps Deps, workspaceID, agentID string) []string {
	dir, err := packageProjectDir(deps, workspaceID)
	if err != nil {
		dir = ""
	}
	rep, err := pipkg.List(pipkg.UserDir(), dir)
	if err != nil {
		rep = pipkg.Report{}
	}
	out := make([]string, 0, len(rep.Packages)+4)
	for _, pkg := range rep.Packages {
		out = append(out, pkg.Source)
	}
	if agentID != "" && deps.Store != nil {
		if a, err := deps.Store.GetAgent(agentID); err == nil {
			out = append(out, a.Packages...)
		}
	}
	return out
}
