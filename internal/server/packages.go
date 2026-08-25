package server

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/cfpperche/picode/internal/pipkg"
)

func registerPackageRoutes(mux *http.ServeMux, deps Deps) {
	mux.HandleFunc("GET /api/packages", handleListPackages)
	mux.HandleFunc("POST /api/packages", handleInstallPackage(deps))
	mux.HandleFunc("DELETE /api/packages", handleRemovePackage(deps))
}

func handleListPackages(w http.ResponseWriter, _ *http.Request) {
	rep, err := pipkg.List(pipkg.UserDir(), "")
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rep)
}

func handleInstallPackage(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Source string `json:"source"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
		defer cancel()
		if err := pipkg.Install(ctx, deps.AgentCmd, req.Source); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		rep, err := pipkg.List(pipkg.UserDir(), "")
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
		if source == "" {
			var req struct {
				Source string `json:"source"`
			}
			_ = json.NewDecoder(r.Body).Decode(&req)
			source = req.Source
		}
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()
		if err := pipkg.Remove(ctx, deps.AgentCmd, source); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		rep, err := pipkg.List(pipkg.UserDir(), "")
		if err != nil {
			writeJSON(w, http.StatusOK, map[string]any{"ok": true})
			return
		}
		writeJSON(w, http.StatusOK, rep)
	}
}
