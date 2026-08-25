package server

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/cfpperche/picode/internal/pipkg"
	"github.com/cfpperche/picode/internal/store"
)

func registerPackageRoutes(mux *http.ServeMux, deps Deps) {
	mux.HandleFunc("GET /api/packages", handleListPackages(deps))
	mux.HandleFunc("GET /api/packages/gallery", handlePackageGallery)
	mux.HandleFunc("POST /api/packages", handleInstallPackage(deps))
	mux.HandleFunc("DELETE /api/packages", handleRemovePackage(deps))
}

func handlePackageGallery(w http.ResponseWriter, r *http.Request) {
	page, err := pipkg.SearchGallery(r.Context(), r.URL.Query().Get("q"))
	if err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, page)
}

func handleListPackages(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dir, err := packageProjectDir(deps, r.URL.Query().Get("workspace"))
		if err != nil {
			writeErr(w, statusForStore(err), err.Error())
			return
		}
		rep, err := pipkg.List(pipkg.UserDir(), dir)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, rep)
	}
}

type packageMutateReq struct {
	Source      string `json:"source"`
	Scope       string `json:"scope"`
	WorkspaceID string `json:"workspaceId"`
}

func handleInstallPackage(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req packageMutateReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		opts, dir, err := packageMutate(deps, req.Scope, req.WorkspaceID)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
		defer cancel()
		if err := pipkg.Install(ctx, deps.AgentCmd, req.Source, opts); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		rep, err := pipkg.List(pipkg.UserDir(), dir)
		if err != nil {
			writeJSON(w, http.StatusOK, map[string]any{"ok": true})
			return
		}
		writeJSON(w, http.StatusOK, rep)
	}
}

func handleRemovePackage(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		source := r.URL.Query().Get("source")
		scope := r.URL.Query().Get("scope")
		wsID := r.URL.Query().Get("workspace")
		if source == "" {
			var req packageMutateReq
			_ = json.NewDecoder(r.Body).Decode(&req)
			source = req.Source
			if scope == "" {
				scope = req.Scope
			}
			if wsID == "" {
				wsID = req.WorkspaceID
			}
		}
		opts, dir, err := packageMutate(deps, scope, wsID)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()
		if err := pipkg.Remove(ctx, deps.AgentCmd, source, opts); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		rep, err := pipkg.List(pipkg.UserDir(), dir)
		if err != nil {
			writeJSON(w, http.StatusOK, map[string]any{"ok": true})
			return
		}
		writeJSON(w, http.StatusOK, rep)
	}
}

func packageProjectDir(deps Deps, id string) (string, error) {
	if id == "" {
		return "", nil
	}
	if deps.Store == nil {
		return "", errNoWorkspace
	}
	ws, err := deps.Store.GetWorkspace(id)
	if err != nil {
		return "", err
	}
	return ws.Path, nil
}

func packageMutate(deps Deps, scope, workspaceID string) (pipkg.MutateOpts, string, error) {
	if scope == "" || scope == "user" {
		dir, err := packageProjectDir(deps, workspaceID)
		if err != nil {
			return pipkg.MutateOpts{}, "", err
		}
		return pipkg.MutateOpts{}, dir, nil
	}
	if scope != "project" {
		return pipkg.MutateOpts{}, "", errBadScope
	}
	dir, err := packageProjectDir(deps, workspaceID)
	if err != nil {
		return pipkg.MutateOpts{}, "", err
	}
	if dir == "" {
		return pipkg.MutateOpts{}, "", errNeedWorkspace
	}
	return pipkg.MutateOpts{Local: true, Cwd: dir}, dir, nil
}

func statusForStore(err error) int {
	if err == store.ErrNotFound {
		return http.StatusNotFound
	}
	return http.StatusBadRequest
}

type pkgErr string

func (e pkgErr) Error() string { return string(e) }

const (
	errBadScope      pkgErr = "scope must be user or project"
	errNeedWorkspace pkgErr = "select an agent to install into a workspace"
	errNoWorkspace   pkgErr = "select an agent first"
)
