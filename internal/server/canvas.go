package server

import (
	"encoding/json"
	"net/http"

	"github.com/cfpperche/picode/internal/store"
)

// Canvas routes (ADR-0108, ADR-0116, renamed and reduced to one layout
// engine by ADR-0118). A canvas is a named plane of agent and terminal
// panels in 8 px units; the store holds one row per panel, validates every
// rectangle against the plane's rules and announces eight events (ADR-0116
// adds canvas.edge.added and canvas.edge.removed). Every mutation answers
// with the payload its event carries, so the surface has one reducer path
// for a response and a feed frame. The auth gate in front of every /api/*
// route applies.
func registerCanvasRoutes(mux Registrar, deps Deps) {
	mux.HandleFunc("GET /api/canvases", handleListCanvases(deps))
	mux.HandleFunc("POST /api/canvases", handleCreateCanvas(deps))
	mux.HandleFunc("GET /api/canvases/{id}", handleGetCanvas(deps))
	mux.HandleFunc("PATCH /api/canvases/{id}", handleUpdateCanvas(deps))
	mux.HandleFunc("DELETE /api/canvases/{id}", handleDeleteCanvas(deps))
	mux.HandleFunc("PATCH /api/canvases/{id}/layout", handleCanvasLayout(deps))
	mux.HandleFunc("POST /api/canvases/{id}/panels", handleAddCanvasPanel(deps))
	mux.HandleFunc("DELETE /api/canvases/{id}/panels/{panelId}", handleRemoveCanvasPanel(deps))
	mux.HandleFunc("PATCH /api/canvases/{id}/panels/{panelId}/content", handleCanvasPanelContent(deps))
	mux.HandleFunc("GET /api/canvases/{id}/edges", handleListCanvasEdges(deps))
	mux.HandleFunc("POST /api/canvases/{id}/edges", handleAddCanvasEdge(deps))
	mux.HandleFunc("DELETE /api/canvases/{id}/edges/{edgeId}", handleRemoveCanvasEdge(deps))
}

// writeCanvasErr answers a canvas store error with the status storeStatus maps
// (400 for what the caller can fix, 404, 409, else 500) and its message,
// which the UI shows verbatim.
func writeCanvasErr(w http.ResponseWriter, err error) {
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

// GET /api/canvases — the summaries (id, name, compact, panelCount,
// updatedAt), by name.
func handleListCanvases(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		list, err := deps.Store.ListCanvases()
		if err != nil {
			writeCanvasErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"canvases": list})
	}
}

// POST /api/canvases {name} — 201 with the summary; 400 names the limit.
func handleCreateCanvas(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name string `json:"name"`
		}
		if !decodeBody(w, r, &req) {
			return
		}
		m, err := deps.Store.CreateCanvas(req.Name)
		if err != nil {
			writeCanvasErr(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, m)
	}
}

// GET /api/canvases/{id} — the summary plus every panel and every edge,
// so one read still opens a canvas.
func handleGetCanvas(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		d, err := deps.Store.GetCanvas(r.PathValue("id"))
		if err != nil {
			writeCanvasErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, d)
	}
}

// PATCH /api/canvases/{id} {name?, compact?, ifUpdatedAt} — 200 with the
// summary; 409 when the row moved on since ifUpdatedAt (or If-Match). There
// is no layout mode to patch any more (ADR-0118): every rectangle is canvas
// units, so a move is PATCH …/layout and nothing else rewrites panels.
func handleUpdateCanvas(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name        *string `json:"name"`
			Compact     *string `json:"compact"`
			IfUpdatedAt string  `json:"ifUpdatedAt"`
		}
		if !decodeBody(w, r, &req) {
			return
		}
		c, err := deps.Store.UpdateCanvas(r.PathValue("id"), store.CanvasPatch{Name: req.Name, Compact: req.Compact}, precondition(r, req.IfUpdatedAt))
		if err != nil {
			writeCanvasErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, c)
	}
}

// PATCH /api/canvases/{id}/layout {ifUpdatedAt, panels: [{id, x, y, w, h}]}
// — the changed subset, one transaction, all or nothing, every rectangle
// judged by the plane's rules; 200 with {id, updatedAt, panels}
// (the canvas.layout payload); 409 when stale.
func handleCanvasLayout(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			IfUpdatedAt string                 `json:"ifUpdatedAt"`
			Panels      []store.PanelPlacement `json:"panels"`
		}
		if !decodeBody(w, r, &req) {
			return
		}
		lay, err := deps.Store.PatchCanvasLayout(r.PathValue("id"), req.Panels, precondition(r, req.IfUpdatedAt))
		if err != nil {
			writeCanvasErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, lay)
	}
}

// POST /api/canvases/{id}/panels {kind, ref, x, y, w, h} — the client
// places, the server validates the rectangle; 201 with
// {id, updatedAt, panel} (the canvas.panel.added payload); 409 when the
// binding is already there.
func handleAddCanvasPanel(deps Deps) http.HandlerFunc {
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
		added, err := deps.Store.AddCanvasPanel(r.PathValue("id"), req.Kind, req.Ref, req.X, req.Y, req.W, req.H)
		if err != nil {
			writeCanvasErr(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, added)
	}
}

// DELETE /api/canvases/{id}/panels/{panelId} — 204; the feed carries
// canvas.panel.removed with the new updatedAt.
func handleRemoveCanvasPanel(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := deps.Store.RemoveCanvasPanel(r.PathValue("id"), r.PathValue("panelId")); err != nil {
			writeCanvasErr(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// PATCH /api/canvases/{id}/panels/{panelId}/content — a text panel's words.
// The body is the whole text, not a patch: the store keeps what a reader
// typed, and a second browser draws the same string rather than replaying
// keystrokes. Only a text panel has words; every other kind is a 400.
func handleCanvasPanelContent(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Content string `json:"content"`
		}
		if !decodeBody(w, r, &req) {
			return
		}
		out, err := deps.Store.SetCanvasPanelContent(r.PathValue("id"), r.PathValue("panelId"), req.Content)
		if err != nil {
			writeCanvasErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// Edges (ADR-0116). An edge is the owner's recorded intent that two
// sessions may exchange messages: it grants exactly ADR-0104's mailbox
// contact and never a transcript. These three routes are the **only** way
// one is created, listed or removed — they sit behind the same owner auth
// gate as every /api/* route, and the MCP surface
// (internal/communication) gains no verb, so an agent can neither draw an
// edge nor discover that one could exist.

// GET /api/canvases/{id}/edges — every edge of one canvas, oldest first.
// GET /api/canvases/{id} carries the same list, so opening a canvas is
// still one read; this route is for the audit list, which wants the edges
// without the panels.
func handleListCanvasEdges(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		edges, err := deps.Store.ListCanvasEdges(r.PathValue("id"))
		if err != nil {
			writeCanvasErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"edges": edges})
	}
}

// POST /api/canvases/{id}/edges {aPanel, bPanel} — 201 with {id, updatedAt,
// edge} (the canvas.edge.added payload). The store orders the pair, so the
// two ids may arrive in either order; 400 names the rule it refused (a
// panel of another canvas, the same panel twice, a kind with no mailbox,
// the cap) and 409 says the pair is already linked.
func handleAddCanvasEdge(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			APanel string `json:"aPanel"`
			BPanel string `json:"bPanel"`
		}
		if !decodeBody(w, r, &req) {
			return
		}
		added, err := deps.Store.AddCanvasEdge(r.PathValue("id"), req.APanel, req.BPanel)
		if err != nil {
			writeCanvasErr(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, added)
	}
}

// DELETE /api/canvases/{id}/edges/{edgeId} — 204, and the grant goes with
// the row: the next contact read derives nothing from it. The feed carries
// canvas.edge.removed with the new updatedAt.
func handleRemoveCanvasEdge(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := deps.Store.RemoveCanvasEdge(r.PathValue("id"), r.PathValue("edgeId")); err != nil {
			writeCanvasErr(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// DELETE /api/canvases/{id} — 204; the panels and their edges cascade in
// the store.
func handleDeleteCanvas(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := deps.Store.DeleteCanvas(r.PathValue("id")); err != nil {
			writeCanvasErr(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
