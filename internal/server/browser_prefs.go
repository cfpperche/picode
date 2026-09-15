package server

// The browser's simple preferences (slice 3): name/value pairs behind one
// read and one write. Backed by the settings KV — a save announces
// setting.updated, so an open Settings ▸ Browser refetches. showFullUrl
// defaults to true (today's address bar); the open destinations default to
// the app (popups adopt as tabs, slice 1).

import (
	"encoding/json"
	"net/http"

	"github.com/cfpperche/picode/internal/store"
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
		_, _, err := deps.Store.GetSetting(prefShowFullURL)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		showFull, dests, err := browserPrefsRead(deps.Store)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"showFullUrl": showFull, "webOpenDest": dests[0], "localOpenDest": dests[1]})
	}
}

// browserPrefsRead folds the simple prefs out of the settings KV. Defaults:
// the address bar shows the full URL, and popups open inside the app.
func browserPrefsRead(st *store.Store) (bool, [2]string, error) {
	showFull := true
	dests := [2]string{"app", "app"} // web, local
	for i, key := range []string{prefShowFullURL, "browser.webOpenDest", "browser.localOpenDest"} {
		raw, ok, err := st.GetSetting(key)
		if err != nil {
			return false, dests, err
		}
		switch i {
		case 0:
			if ok && raw == "0" {
				showFull = false
			}
		default:
			if ok && raw == "external" {
				dests[i-1] = "external"
			}
		}
	}
	return showFull, dests, nil
}

func handleBrowserPrefsPut(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.Store == nil {
			writeErr(w, http.StatusServiceUnavailable, "store is not open")
			return
		}
		var req struct {
			ShowFullURL   *bool  `json:"showFullUrl"`
			WebOpenDest   string `json:"webOpenDest"`
			LocalOpenDest string `json:"localOpenDest"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "bad request: "+err.Error())
			return
		}
		if req.ShowFullURL == nil && req.WebOpenDest == "" && req.LocalOpenDest == "" {
			writeErr(w, http.StatusBadRequest, "nothing to save")
			return
		}
		if req.ShowFullURL != nil {
			value := map[bool]string{true: "1", false: "0"}[*req.ShowFullURL]
			if err := deps.Store.SetSetting(prefShowFullURL, value); err != nil {
				writeErr(w, http.StatusInternalServerError, err.Error())
				return
			}
		}
		for _, row := range []struct{ key, value string }{
			{"browser.webOpenDest", req.WebOpenDest},
			{"browser.localOpenDest", req.LocalOpenDest},
		} {
			if row.value == "" {
				continue
			}
			if row.value != "app" && row.value != "external" {
				writeErr(w, http.StatusBadRequest, row.key+" must be app or external")
				return
			}
			if err := deps.Store.SetSetting(row.key, row.value); err != nil {
				writeErr(w, http.StatusInternalServerError, err.Error())
				return
			}
		}
		showFull, dests, err := browserPrefsRead(deps.Store)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"showFullUrl": showFull, "webOpenDest": dests[0], "localOpenDest": dests[1]})
	}
}
