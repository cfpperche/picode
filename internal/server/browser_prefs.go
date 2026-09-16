package server

// The browser's simple preferences (slice 3): name/value pairs behind one
// read and one write. Backed by the settings KV — a save announces
// setting.updated, so an open Settings ▸ Browser refetches. showFullUrl
// defaults to true (today's address bar); the open destinations default to
// the app (a new-tab request adopts as a tab, slice 1).

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
		prefs, err := browserPrefsRead(deps.Store)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, prefs)
	}
}

// BrowserPrefs are the simple browser preferences the settings page edits.
type BrowserPrefs struct {
	ShowFullURL      bool   `json:"showFullUrl"`
	WebOpenDest      string `json:"webOpenDest"`
	LocalOpenDest    string `json:"localOpenDest"`
	PasswordAutosave bool   `json:"passwordAutosave"`
	GeneralAutofill  bool   `json:"generalAutofill"`
	// AskDownload is the Downloads section's save prompt: off writes files
	// straight into the download folder.
	AskDownload bool `json:"askDownload"`
	// ScriptsEnabled is the Browser permissions JavaScript switch.
	ScriptsEnabled bool `json:"scriptsEnabled"`
	// AgentAccess is the master switch: off refuses every browser verb for
	// every agent (the grants below stop mattering until it is back on).
	AgentAccess bool `json:"agentAccess"`
}

// browserPrefsRead folds the simple prefs out of the settings KV. Defaults:
// the address bar shows the full URL, popups open inside the app, and
// WebView2's own autofill defaults (both on).
func browserPrefsRead(st *store.Store) (BrowserPrefs, error) {
	p := BrowserPrefs{ShowFullURL: true, WebOpenDest: "app", LocalOpenDest: "app", PasswordAutosave: true, GeneralAutofill: true, AskDownload: false, ScriptsEnabled: true, AgentAccess: true}
	rows := []struct {
		key   string
		apply func(raw string)
	}{
		{prefShowFullURL, func(raw string) {
			if raw == "0" {
				p.ShowFullURL = false
			}
		}},
		{"browser.webOpenDest", func(raw string) {
			if raw == "external" {
				p.WebOpenDest = "external"
			}
		}},
		{"browser.localOpenDest", func(raw string) {
			if raw == "external" {
				p.LocalOpenDest = "external"
			}
		}},
		{"browser.passwordAutosave", func(raw string) {
			if raw == "0" {
				p.PasswordAutosave = false
			}
		}},
		{"browser.generalAutofill", func(raw string) {
			if raw == "0" {
				p.GeneralAutofill = false
			}
		}},
		{"browser.scriptsEnabled", func(raw string) {
			if raw == "0" {
				p.ScriptsEnabled = false
			}
		}},
		{"browser.askDownload", func(raw string) {
			if raw == "1" {
				p.AskDownload = true
			}
		}},
		{"browser.agentAccess", func(raw string) {
			if raw == "0" {
				p.AgentAccess = false
			}
		}},
	}
	for _, row := range rows {
		raw, ok, err := st.GetSetting(row.key)
		if err != nil {
			return p, err
		}
		if ok {
			row.apply(raw)
		}
	}
	return p, nil
}

func handleBrowserPrefsPut(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.Store == nil {
			writeErr(w, http.StatusServiceUnavailable, "store is not open")
			return
		}
		var req BrowserPrefs
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "bad request: "+err.Error())
			return
		}
		if req.WebOpenDest != "app" && req.WebOpenDest != "external" {
			writeErr(w, http.StatusBadRequest, "webOpenDest must be app or external")
			return
		}
		if req.LocalOpenDest != "app" && req.LocalOpenDest != "external" {
			writeErr(w, http.StatusBadRequest, "localOpenDest must be app or external")
			return
		}
		rows := []struct{ key, value string }{
			{prefShowFullURL, map[bool]string{true: "1", false: "0"}[req.ShowFullURL]},
			{"browser.webOpenDest", req.WebOpenDest},
			{"browser.localOpenDest", req.LocalOpenDest},
			{"browser.passwordAutosave", map[bool]string{true: "1", false: "0"}[req.PasswordAutosave]},
			{"browser.generalAutofill", map[bool]string{true: "1", false: "0"}[req.GeneralAutofill]},
			{"browser.scriptsEnabled", map[bool]string{true: "1", false: "0"}[req.ScriptsEnabled]},
			{"browser.askDownload", map[bool]string{true: "1", false: "0"}[req.AskDownload]},
			{"browser.agentAccess", map[bool]string{true: "1", false: "0"}[req.AgentAccess]},
		}
		for _, row := range rows {
			if err := deps.Store.SetSetting(row.key, row.value); err != nil {
				writeErr(w, http.StatusInternalServerError, err.Error())
				return
			}
		}
		prefs, err := browserPrefsRead(deps.Store)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, prefs)
	}
}

// browserAgentAccess is the master switch the agent routes check before
// anything else (Settings ▸ Browser ▸ Browser). Missing means on: the
// switch exists to turn agent access off, not to gate a fresh install.
func browserAgentAccess(st *store.Store) bool {
	if st == nil {
		return true
	}
	raw, ok, err := st.GetSetting("browser.agentAccess")
	if err != nil {
		return true
	}
	return !ok || raw != "0"
}
