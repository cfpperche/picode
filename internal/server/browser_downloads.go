package server

// The work browser's download endpoints (slice 3.3d): the shell reports each
// download as it starts and when it lands or breaks; the settings dialog
// reads, searches and prunes the list. Mutations announce
// browserdownload.updated, so open readers refetch.

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"database/sql"
)

func registerBrowserDownloadRoutes(mux Registrar, deps Deps) {
	mux.HandleFunc("GET /api/browser/downloads", handleBrowserDownloadsList(deps))
	mux.HandleFunc("POST /api/browser/downloads", handleBrowserDownloadsAdd(deps))
	mux.HandleFunc("POST /api/browser/downloads/status", handleBrowserDownloadsStatus(deps))
	mux.HandleFunc("DELETE /api/browser/downloads/{id}", handleBrowserDownloadsDelete(deps))
	mux.HandleFunc("POST /api/browser/downloads/clear", handleBrowserDownloadsClear(deps))
}

func handleBrowserDownloadsList(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.Store == nil {
			writeErr(w, http.StatusServiceUnavailable, "store is not open")
			return
		}
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		rows, err := deps.Store.ListBrowserDownloads(limit, r.URL.Query().Get("q"))
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"downloads": rows})
	}
}

// The shell's start report: url, the path the file is being written to, and
// the size it expects (0 when the server did not say).
func handleBrowserDownloadsAdd(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.Store == nil {
			writeErr(w, http.StatusServiceUnavailable, "store is not open")
			return
		}
		var req struct {
			URL   string `json:"url"`
			Path  string `json:"path"`
			Total int64  `json:"total"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "bad request: "+err.Error())
			return
		}
		row, err := deps.Store.AddBrowserDownload(req.URL, req.Path, req.Total)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, row)
	}
}

// The shell's outcome report. A path nobody recorded is accepted and
// ignored — the download may have started before this list did.
func handleBrowserDownloadsStatus(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.Store == nil {
			writeErr(w, http.StatusServiceUnavailable, "store is not open")
			return
		}
		var req struct {
			Path     string `json:"path"`
			Status   string `json:"status"`
			Received int64  `json:"received"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "bad request: "+err.Error())
			return
		}
		if err := deps.Store.FinishBrowserDownload(req.Path, req.Status, req.Received); err != nil && !errors.Is(err, sql.ErrNoRows) {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func handleBrowserDownloadsDelete(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.Store == nil {
			writeErr(w, http.StatusServiceUnavailable, "store is not open")
			return
		}
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil || id <= 0 {
			writeErr(w, http.StatusBadRequest, "a download id is required")
			return
		}
		if err := deps.Store.DeleteBrowserDownload(id); err != nil {
			writeErr(w, http.StatusNotFound, "no such download")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func handleBrowserDownloadsClear(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.Store == nil {
			writeErr(w, http.StatusServiceUnavailable, "store is not open")
			return
		}
		if err := deps.Store.ClearBrowserDownloads(); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
