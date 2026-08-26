package server

import (
	"encoding/json"
	"net/http"
	"strings"
)

func registerPins(mux *http.ServeMux, deps Deps) {
	mux.HandleFunc("GET /api/pins", handleListPins(deps))
	mux.HandleFunc("POST /api/pins", handleCreatePin(deps))
	mux.HandleFunc("PATCH /api/pins/{id}", handleUpdatePin(deps))
	mux.HandleFunc("DELETE /api/pins/{id}", handleDeletePin(deps))
}

func handleListPins(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pins, err := deps.Store.ListPins()
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"pins": pins})
	}
}

type pinReq struct {
	Title string   `json:"title"`
	Tags  []string `json:"tags"`
	Body  string   `json:"body"`
}

func handleCreatePin(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req pinReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		p, err := deps.Store.CreatePin(req.Title, req.Tags, req.Body)
		if err != nil {
			if strings.Contains(err.Error(), "title is required") {
				writeErr(w, http.StatusBadRequest, err.Error())
				return
			}
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, p)
	}
}

func handleUpdatePin(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req pinReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		p, err := deps.Store.UpdatePin(r.PathValue("id"), req.Title, req.Tags, req.Body)
		if err != nil {
			writeErr(w, statusForStore(err), err.Error())
			return
		}
		writeJSON(w, http.StatusOK, p)
	}
}

func handleDeletePin(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := deps.Store.DeletePin(r.PathValue("id")); err != nil {
			writeErr(w, statusForStore(err), err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
