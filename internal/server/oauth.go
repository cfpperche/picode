package server

import (
	"encoding/json"
	"net/http"

	"github.com/cfpperche/picode/internal/oauth"
)

func registerOAuthRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/oauth/start", handleOAuthStart)
	mux.HandleFunc("GET /api/oauth/status", handleOAuthStatus)
	mux.HandleFunc("POST /api/oauth/cancel", handleOAuthCancel)
}

func handleOAuthStart(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Provider string `json:"provider"`
		ReturnTo string `json:"returnTo"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	url, err := oauth.Start(req.Provider, req.ReturnTo)
	if err != nil {
		writeErr(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"url": url})
}

func handleOAuthStatus(w http.ResponseWriter, r *http.Request) {
	pending, done, err := oauth.Status()
	out := map[string]any{"pending": pending, "done": done}
	if err != "" {
		out["error"] = err
	}
	writeJSON(w, http.StatusOK, out)
}

func handleOAuthCancel(w http.ResponseWriter, r *http.Request) {
	oauth.Cancel()
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
