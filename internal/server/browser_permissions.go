package server

// The work browser's site-permission endpoints (slice 3, Browser
// permissions): the shell reports each allow/deny, the Site settings dialog
// reads, changes and prunes the standings. Mutations announce
// browserpermission.updated, so open readers refetch.

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

func registerBrowserPermissionRoutes(mux Registrar, deps Deps) {
	mux.HandleFunc("GET /api/browser/permissions", handleBrowserPermissionsList(deps))
	mux.HandleFunc("POST /api/browser/permissions", handleBrowserPermissionsSet(deps))
	mux.HandleFunc("DELETE /api/browser/permissions/{id}", handleBrowserPermissionsDelete(deps))
	mux.HandleFunc("POST /api/browser/permissions/clear", handleBrowserPermissionsClear(deps))
	mux.HandleFunc("POST /api/browser/permissions/prune", handleBrowserPermissionsPrune(deps))
}

func handleBrowserPermissionsList(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.Store == nil {
			writeErr(w, http.StatusServiceUnavailable, "store is not open")
			return
		}
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		rows, err := deps.Store.ListBrowserPermissions(limit, r.URL.Query().Get("kind"), r.URL.Query().Get("q"))
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"permissions": rows})
	}
}

// The shell's report of a decision, and the dialog's own edit of one.
func handleBrowserPermissionsSet(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.Store == nil {
			writeErr(w, http.StatusServiceUnavailable, "store is not open")
			return
		}
		var req struct {
			Origin   string `json:"origin"`
			Kind     string `json:"kind"`
			Decision string `json:"decision"`
			Standing bool   `json:"standing"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "bad request: "+err.Error())
			return
		}
		row, err := deps.Store.SetBrowserPermission(req.Origin, req.Kind, req.Decision, req.Standing)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, row)
	}
}

func handleBrowserPermissionsDelete(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.Store == nil {
			writeErr(w, http.StatusServiceUnavailable, "store is not open")
			return
		}
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil || id <= 0 {
			writeErr(w, http.StatusBadRequest, "a permission id is required")
			return
		}
		if err := deps.Store.DeleteBrowserPermission(id); err != nil {
			writeErr(w, http.StatusNotFound, "no such permission")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// Pruning forgets the site entries nothing has visited inside the window —
// the dialog's cleanup for sites the human no longer uses. The every-site
// policy rows are untouched (the store decides that), and the answer carries
// what was forgotten so the page can clear the live shell's copy too.
func handleBrowserPermissionsPrune(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.Store == nil {
			writeErr(w, http.StatusServiceUnavailable, "store is not open")
			return
		}
		var req struct {
			Days int `json:"days"`
		}
		if r.Body != nil {
			_ = json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&req)
		}
		days := req.Days
		if days <= 0 {
			days = 90
		}
		if days < 7 {
			// A floor: "forget everything I saw today" is not a cleanup, and
			// the accidental version of it is unrecoverable.
			days = 7
		}
		pruned, err := deps.Store.PruneBrowserPermissions(time.Now().AddDate(0, 0, -days))
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		rows := make([]map[string]any, 0, len(pruned))
		for _, p := range pruned {
			rows = append(rows, map[string]any{"origin": p.Origin, "kind": p.Kind})
		}
		writeJSON(w, http.StatusOK, map[string]any{"removed": len(pruned), "days": days, "forgotten": rows})
	}
}

// Clearing takes an optional kind in the body: "forget every camera
// decision" is the dialog's per-kind reset.
func handleBrowserPermissionsClear(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.Store == nil {
			writeErr(w, http.StatusServiceUnavailable, "store is not open")
			return
		}
		var req struct {
			Kind string `json:"kind"`
		}
		if r.Body != nil {
			_ = json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&req)
		}
		if err := deps.Store.ClearBrowserPermissions(req.Kind); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
