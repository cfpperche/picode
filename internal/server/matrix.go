package server

import (
	"encoding/json"
	"net/http"

	"github.com/cfpperche/picode/internal/store"
)

// Matrix routes (ADR-0108). A matrix is a named 12-column grid of agent
// and terminal panels; the store holds one row per panel, validates the
// limits and announces six events. Every mutation answers with the payload
// its event carries, so the surface has one reducer path for a response
// and a feed frame. The auth gate in front of every /api/* route applies.
func registerMatrixRoutes(mux Registrar, deps Deps) {
	mux.HandleFunc("GET /api/matrices", handleListMatrices(deps))
	mux.HandleFunc("POST /api/matrices", handleCreateMatrix(deps))
	mux.HandleFunc("GET /api/matrices/{id}", handleGetMatrix(deps))
	mux.HandleFunc("PATCH /api/matrices/{id}", handleUpdateMatrix(deps))
	mux.HandleFunc("DELETE /api/matrices/{id}", handleDeleteMatrix(deps))
	mux.HandleFunc("PATCH /api/matrices/{id}/layout", handleMatrixLayout(deps))
	mux.HandleFunc("POST /api/matrices/{id}/panels", handleAddMatrixPanel(deps))
	mux.HandleFunc("DELETE /api/matrices/{id}/panels/{panelId}", handleRemoveMatrixPanel(deps))
}

// writeMatrixErr answers a matrix store error with the status storeStatus maps
// (400 for what the caller can fix, 404, 409, else 500) and its message,
// which the UI shows verbatim.
func writeMatrixErr(w http.ResponseWriter, err error) {
	writeErr(w, storeStatus(err), err.Error())
}

// decodeBody reads one JSON object; a malformed body is the caller's 400.
func decodeBody(w http.ResponseWriter, r *http.Request, into any) bool {
	if err := json.NewDecoder(r.Body).Decode(into); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body")
		return false
	}
	return true
}

// precondition is the updatedAt the caller last saw — the body's
// ifUpdatedAt, or an If-Match header when present (the pins convention).
func precondition(r *http.Request, body string) string {
	if h := r.Header.Get("If-Match"); h != "" {
		return h
	}
	return body
}

// GET /api/matrices — the summaries (id, name, compact, panelCount,
// updatedAt), by name.
func handleListMatrices(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		list, err := deps.Store.ListMatrices()
		if err != nil {
			writeMatrixErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"matrices": list})
	}
}

// POST /api/matrices {name} — 201 with the summary; 400 names the limit.
func handleCreateMatrix(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name string `json:"name"`
		}
		if !decodeBody(w, r, &req) {
			return
		}
		m, err := deps.Store.CreateMatrix(req.Name)
		if err != nil {
			writeMatrixErr(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, m)
	}
}

// GET /api/matrices/{id} — the summary plus every panel.
func handleGetMatrix(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		d, err := deps.Store.GetMatrix(r.PathValue("id"))
		if err != nil {
			writeMatrixErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, d)
	}
}

// PATCH /api/matrices/{id} {name?, compact?, ifUpdatedAt} — 200 with the
// summary; 409 when the row moved on since ifUpdatedAt (or If-Match).
func handleUpdateMatrix(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name        *string `json:"name"`
			Compact     *string `json:"compact"`
			IfUpdatedAt string  `json:"ifUpdatedAt"`
		}
		if !decodeBody(w, r, &req) {
			return
		}
		m, err := deps.Store.UpdateMatrix(r.PathValue("id"), store.MatrixPatch{Name: req.Name, Compact: req.Compact}, precondition(r, req.IfUpdatedAt))
		if err != nil {
			writeMatrixErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, m)
	}
}

// PATCH /api/matrices/{id}/layout {ifUpdatedAt, panels: [{id, x, y, w, h}]}
// — the changed subset, one transaction, all or nothing; 200 with
// {id, updatedAt, panels} (the matrix.layout payload); 409 when stale.
func handleMatrixLayout(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			IfUpdatedAt string                 `json:"ifUpdatedAt"`
			Panels      []store.PanelPlacement `json:"panels"`
		}
		if !decodeBody(w, r, &req) {
			return
		}
		lay, err := deps.Store.PatchMatrixLayout(r.PathValue("id"), req.Panels, precondition(r, req.IfUpdatedAt))
		if err != nil {
			writeMatrixErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, lay)
	}
}

// POST /api/matrices/{id}/panels {kind, ref, x, y, w, h} — the client
// places, the server validates; 201 with {id, updatedAt, panel} (the
// matrix.panel.added payload); 409 when the binding is already there.
func handleAddMatrixPanel(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Kind string `json:"kind"`
			Ref  string `json:"ref"`
			X    int    `json:"x"`
			Y    int    `json:"y"`
			W    int    `json:"w"`
			H    int    `json:"h"`
		}
		if !decodeBody(w, r, &req) {
			return
		}
		added, err := deps.Store.AddMatrixPanel(r.PathValue("id"), req.Kind, req.Ref, req.X, req.Y, req.W, req.H)
		if err != nil {
			writeMatrixErr(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, added)
	}
}

// DELETE /api/matrices/{id}/panels/{panelId} — 204; the feed carries
// matrix.panel.removed with the new updatedAt.
func handleRemoveMatrixPanel(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := deps.Store.RemoveMatrixPanel(r.PathValue("id"), r.PathValue("panelId")); err != nil {
			writeMatrixErr(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// DELETE /api/matrices/{id} — 204; the panels cascade in the store.
func handleDeleteMatrix(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := deps.Store.DeleteMatrix(r.PathValue("id")); err != nil {
			writeMatrixErr(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
