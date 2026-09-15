package server

// The work browser's history endpoints (slice 3): the app records visits
// (POST), the address-bar dropdown and Settings ▸ Browser read (GET), and
// the Manage section deletes one visit or clears everything. Mutations
// announce browserhistory.updated, so open readers refetch.

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

func registerBrowserHistoryRoutes(mux Registrar, deps Deps) {
	mux.HandleFunc("POST /api/browser/history", handleBrowserHistoryAdd(deps))
	mux.HandleFunc("GET /api/browser/history", handleBrowserHistoryList(deps))
	mux.HandleFunc("DELETE /api/browser/history/{id}", handleBrowserHistoryDelete(deps))
	mux.HandleFunc("POST /api/browser/history/clear", handleBrowserHistoryClear(deps))
}

func handleBrowserHistoryAdd(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.Store == nil {
			writeErr(w, http.StatusServiceUnavailable, "store is not open")
			return
		}
		var req struct {
			URL   string `json:"url"`
			Title string `json:"title"`
			Typed bool   `json:"typed"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "bad request: "+err.Error())
			return
		}
		visit, err := deps.Store.AddBrowserVisit(req.URL, req.Title, req.Typed)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, visit)
	}
}

func handleBrowserHistoryList(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.Store == nil {
			writeErr(w, http.StatusServiceUnavailable, "store is not open")
			return
		}
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		visits, err := deps.Store.ListBrowserHistory(limit, r.URL.Query().Get("q"))
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"visits": visits})
	}
}

func handleBrowserHistoryDelete(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.Store == nil {
			writeErr(w, http.StatusServiceUnavailable, "store is not open")
			return
		}
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil || id <= 0 {
			writeErr(w, http.StatusBadRequest, "a visit id is required")
			return
		}
		if err := deps.Store.DeleteBrowserVisit(id); err != nil {
			writeErr(w, http.StatusNotFound, "no such visit")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func handleBrowserHistoryClear(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.Store == nil {
			writeErr(w, http.StatusServiceUnavailable, "store is not open")
			return
		}
		// ?since=<RFC3339> clears only that window (the Clear browsing data
		// dialog's time range); without it the whole list goes.
		since := r.URL.Query().Get("since")
		if since != "" {
			if _, err := time.Parse(time.RFC3339, since); err != nil {
				writeErr(w, http.StatusBadRequest, "since must be an RFC3339 timestamp")
				return
			}
		}
		if err := deps.Store.ClearBrowserHistorySince(since); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
