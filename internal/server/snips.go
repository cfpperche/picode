package server

import (
	"encoding/json"
	"net/http"

	"github.com/cfpperche/picode/internal/snips"
	"github.com/cfpperche/picode/internal/store"
)

func registerSnips(mux Registrar, deps Deps) {
	mux.HandleFunc("GET /api/snips/picker", handleSnipPicker(deps))
	mux.HandleFunc("GET /api/snips", handleListSnips(deps))
	mux.HandleFunc("POST /api/snips", handleCreateSnip(deps))
	mux.HandleFunc("GET /api/snips/{id}", handleGetSnip(deps))
	mux.HandleFunc("PATCH /api/snips/{id}", handleUpdateSnip(deps))
	mux.HandleFunc("DELETE /api/snips/{id}", handleDeleteSnip(deps))
	mux.HandleFunc("POST /api/snips/{id}/starred", handleSnipStarred(deps))
	mux.HandleFunc("POST /api/snips/{id}/archived", handleSnipArchived(deps))
}

type snipReq struct {
	Title        string              `json:"title"`
	Slug         string              `json:"slug"`
	Description  string              `json:"description"`
	Kind         string              `json:"kind"`
	Body         string              `json:"body"`
	Tags         []string            `json:"tags"`
	Placeholders []snips.Placeholder `json:"placeholders"`
	IfUpdatedAt  string              `json:"ifUpdatedAt"`
}

func snipParams(req snipReq) store.SnipParams {
	return store.SnipParams{
		Title: req.Title, Slug: req.Slug, Description: req.Description,
		Kind: req.Kind, Body: req.Body, Tags: req.Tags, Placeholders: req.Placeholders,
	}
}

func handleListSnips(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f := store.SnipListFilter{Q: r.URL.Query().Get("q"), Archived: queryFlag(r, "archived")}
		list, err := deps.Store.ListSnips(f)
		if err != nil {
			writePinErr(w, err)
			return
		}
		archived, _ := deps.Store.CountArchivedSnips()
		writeJSON(w, http.StatusOK, map[string]any{"snips": list, "archived": archived})
	}
}

func handleSnipPicker(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		list, err := deps.Store.ListSnipPicker()
		if err != nil {
			writePinErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"snips": list})
	}
}

func handleCreateSnip(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req snipReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		p, err := deps.Store.CreateSnip(snipParams(req))
		if err != nil {
			writePinErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, p)
	}
}

func handleGetSnip(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p, err := deps.Store.GetSnip(r.PathValue("id"))
		if err != nil {
			writePinErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, p)
	}
}

func handleUpdateSnip(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req snipReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		precond := req.IfUpdatedAt
		if h := r.Header.Get("If-Match"); h != "" {
			precond = h
		}
		p, err := deps.Store.UpdateSnip(r.PathValue("id"), snipParams(req), precond)
		if err != nil {
			writePinErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, p)
	}
}

func handleDeleteSnip(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := deps.Store.DeleteSnip(r.PathValue("id")); err != nil {
			writePinErr(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func handleSnipStarred(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Starred bool `json:"starred"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		p, err := deps.Store.SetSnipStarred(r.PathValue("id"), req.Starred)
		if err != nil {
			writePinErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, p)
	}
}

func handleSnipArchived(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Archived bool `json:"archived"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		p, err := deps.Store.SetSnipArchived(r.PathValue("id"), req.Archived)
		if err != nil {
			writePinErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, p)
	}
}
