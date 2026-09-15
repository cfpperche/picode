package server

// The browser's simple preferences (slice 3): name/value pairs behind one
// read and one write. Backed by the settings KV — a save announces
// setting.updated, so an open Settings ▸ Browser refetches. showFullUrl
// defaults to true (today's address bar).

import (
	"encoding/json"
	"net/http"
)

const prefShowFullURL = "browser.showFullUrl"

func registerBrowserPrefRoutes(mux Registrar, deps Deps) {
	mux.HandleFunc("GET /api/browser/prefs", handleBrowserPrefsGet(deps))
	mux.HandleFunc("PUT /api/browser/prefs", handleBrowserPrefsPut(deps))
}

func handleBrowserPrefsGet(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.Store == nil {
			writeErr(w, http.StatusServiceUnavailable, "store is not open")
			return
		}
		raw, ok, err := deps.Store.GetSetting(prefShowFullURL)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		showFull := true
		if ok && raw == "0" {
			showFull = false
		}
		writeJSON(w, http.StatusOK, map[string]any{"showFullUrl": showFull})
	}
}

func handleBrowserPrefsPut(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.Store == nil {
			writeErr(w, http.StatusServiceUnavailable, "store is not open")
			return
		}
		var req struct {
			ShowFullURL *bool `json:"showFullUrl"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "bad request: "+err.Error())
			return
		}
		if req.ShowFullURL == nil {
			writeErr(w, http.StatusBadRequest, "showFullUrl is required")
			return
		}
		value := map[bool]string{true: "1", false: "0"}[*req.ShowFullURL]
		if err := deps.Store.SetSetting(prefShowFullURL, value); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"showFullUrl": *req.ShowFullURL})
	}
}
