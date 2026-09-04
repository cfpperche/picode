package server

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/cfpperche/picode/internal/apps"
)

// Apps host routes (ADR-0036). The registry is nil-safe: a server built
// without apps serves an empty list and 404s everything under an id.

func registerAppsRoutes(mux Registrar, deps Deps) {
	mux.HandleFunc("GET /api/apps", handleListApps(deps))
	mux.HandleFunc("GET /api/apps/{id}/view", handleAppView(deps))
	mux.HandleFunc("POST /api/apps/{id}/action", handleAppAction(deps))
}

func appsHost(deps Deps, r *http.Request) apps.Host {
	return apps.Host{
		Store: deps.Store, DataDir: deps.DataDir,
		// Negate once, right here, matching handleRespondInbox's own
		// wiring (internal/server/inbox.go) — deliverable, not interactive.
		AgentDeliverable: func(agentID string) bool { return !deps.agentInteractive(r.Context(), agentID) },
		OpenAgentTerminal: func(agentID string) error {
			_, err := deps.openAgentTUI(r.Context(), agentID, false)
			return err
		},
		DeliverReply: func(itemID, verb, text string) (string, error) {
			return deps.DeliverReply(r.Context(), itemID, verb, text)
		},
	}
}

type appRow struct {
	apps.Manifest
	Badge apps.Badge `json:"badge"`
}

// handleListApps is the UI's poll target: manifests plus live badges.
// One app's badge failure degrades to an empty badge — it never breaks
// the list.
func handleListApps(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows := []appRow{}
		for _, a := range deps.Apps.All() {
			row := appRow{Manifest: a.Manifest()}
			ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
			if b, err := a.Badge(ctx, appsHost(deps, r)); err == nil {
				row.Badge = b
			}
			cancel()
			rows = append(rows, row)
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"apiVersion": apps.APIVersion,
			"apps":       rows,
		})
	}
}

func handleAppView(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		a, ok := deps.Apps.Find(r.PathValue("id"))
		if !ok {
			writeErr(w, http.StatusNotFound, "no such app")
			return
		}
		v, err := a.View(r.Context(), appsHost(deps, r), r.URL.Query().Get("path"))
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, v)
	}
}

func handleAppAction(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		a, ok := deps.Apps.Find(r.PathValue("id"))
		if !ok {
			writeErr(w, http.StatusNotFound, "no such app")
			return
		}
		var req apps.ActionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if req.Action == "" {
			writeErr(w, http.StatusBadRequest, "action is required")
			return
		}
		res, err := a.Action(r.Context(), appsHost(deps, r), req)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, res)
	}
}
