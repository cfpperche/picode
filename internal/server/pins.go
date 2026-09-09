package server

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/cfpperche/picode/internal/store"
)

func registerPins(mux Registrar, deps Deps) {
	mux.HandleFunc("GET /api/pins", handleListPins(deps))
	mux.HandleFunc("GET /api/pins/{id}", handleGetPin(deps))
	mux.HandleFunc("POST /api/pins", handleCreatePin(deps))
	mux.HandleFunc("PATCH /api/pins/{id}", handleUpdatePin(deps))
	mux.HandleFunc("DELETE /api/pins/{id}", handleDeletePin(deps))
	mux.HandleFunc("POST /api/pins/{id}/starred", handlePinStarred(deps))
	mux.HandleFunc("POST /api/pins/{id}/archived", handlePinArchived(deps))
}

// storeStatus maps the store's typed errors once (pins, matrices): what the caller can
// fix is 400, a stale precondition is 409, a missing row is 404, and a
// store failure is honestly a 500 rather than the caller's fault.
func storeStatus(err error) int {
	switch {
	case errors.Is(err, store.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, store.ErrConflict):
		return http.StatusConflict
	case errors.Is(err, store.ErrInvalid):
		return http.StatusBadRequest
	}
	return http.StatusInternalServerError
}

func writePinErr(w http.ResponseWriter, err error) {
	writeErr(w, storeStatus(err), err.Error())
}

// GET /api/pins            the live list, starred first
// GET /api/pins?archived=1 the archived list
// GET /api/pins?q=words    a search over both (each hit says archivedAt)
// The response carries the archived count so the sidebar can offer it.
func handleListPins(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f := store.PinListFilter{Q: r.URL.Query().Get("q"), Archived: queryFlag(r, "archived")}
		pins, err := deps.Store.ListPins(f)
		if err != nil {
			writePinErr(w, err)
			return
		}
		archived, _ := deps.Store.CountArchivedPins()
		writeJSON(w, http.StatusOK, map[string]any{"pins": pins, "archived": archived})
	}
}

func handlePinStarred(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Starred bool `json:"starred"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		p, err := deps.Store.SetPinStarred(r.PathValue("id"), req.Starred)
		if err != nil {
			writePinErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, p)
	}
}

func handlePinArchived(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Archived bool `json:"archived"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		p, err := deps.Store.SetPinArchived(r.PathValue("id"), req.Archived)
		if err != nil {
			writePinErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, p)
	}
}

func handleGetPin(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p, err := deps.Store.GetPin(r.PathValue("id"))
		if err != nil {
			writePinErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, p)
	}
}

type pinReq struct {
	Title string   `json:"title"`
	Tags  []string `json:"tags"`
	Body  string   `json:"body"`
	// IfUpdatedAt is the updatedAt the editor loaded (optional). When it
	// no longer matches, the server answers 409 instead of overwriting
	// another writer's edit. The If-Match header carries the same value.
	IfUpdatedAt string `json:"ifUpdatedAt"`
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
			writePinErr(w, err)
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
		precond := req.IfUpdatedAt
		if h := r.Header.Get("If-Match"); h != "" {
			precond = h
		}
		p, err := deps.Store.UpdatePin(r.PathValue("id"), req.Title, req.Tags, req.Body, precond)
		if err != nil {
			writePinErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, p)
	}
}

func handleDeletePin(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if err := deps.Store.DeletePin(id); err != nil {
			writePinErr(w, err)
			return
		}
		removePinDir(deps.DataDir, id)
		w.WriteHeader(http.StatusNoContent)
	}
}
