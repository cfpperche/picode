package server

import (
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/cfpperche/picode/internal/store"
	"net/http"
	"strings"

	"github.com/cfpperche/picode/internal/catalog"
	"github.com/cfpperche/picode/internal/llama"
)

func registerLlama(mux Registrar, deps Deps) {
	mux.HandleFunc("GET /api/llama", handleLlamaList)
	mux.HandleFunc("POST /api/llama/load", handleLlamaOperation(deps, "load"))
	mux.HandleFunc("POST /api/llama/unload", handleLlamaOperation(deps, "unload"))
	mux.HandleFunc("GET /api/llama/hf", handleLlamaHFSearch)
	mux.HandleFunc("GET /api/llama/hf/info", handleLlamaHFInfo)
	mux.HandleFunc("POST /api/llama/download", handleLlamaOperation(deps, "download"))
	mux.HandleFunc("GET /api/llama/jobs", handleLlamaJobs(deps))
	mux.HandleFunc("POST /api/llama/jobs/{id}/cancel", handleLlamaJobAction(deps, true))
	mux.HandleFunc("POST /api/llama/jobs/{id}/reconcile", handleLlamaJobAction(deps, false))
}

func llamaClient() (*llama.Client, error) {
	return llama.New(llamaURL(), catalog.LlamaKey())
}

func llamaURL() string {
	if u := catalog.LlamaURL(); u != "" {
		return u
	}
	return llama.DefaultURL
}

func handleLlamaList(w http.ResponseWriter, r *http.Request) {
	c, err := llamaClient()
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	models, err := c.List()
	ok := err == nil
	if models == nil {
		models = []llama.Model{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"url":          llamaURL(),
		"ok":           ok,
		"models":       models,
		"setup":        llama.Inspect(llamaURL(), models, ok),
		"connection":   llamaConnection(err),
		"capabilities": c.Capabilities(r.Context()),
	})
}

func llamaConnection(err error) map[string]string {
	if err == nil {
		return map[string]string{"code": "ready", "message": "Connected"}
	}
	failure := llama.ConnectionFailure(err)
	return map[string]string{"code": failure.Code, "message": failure.Message}
}

func handleLlamaHFSearch(w http.ResponseWriter, r *http.Request) {
	hits, err := llama.HFSearch(r.URL.Query().Get("q"))
	if err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"hits": hits})
}

func handleLlamaHFInfo(w http.ResponseWriter, r *http.Request) {
	info, err := llama.HFInfo(r.URL.Query().Get("id"))
	if err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, info)
}

func attachLlamaModels(rep *catalog.Report) {
	if catalog.LlamaURL() == "" && catalog.LlamaKey() == "" {
		return
	}
	c, err := llamaClient()
	if err != nil {
		return
	}
	c.ShortTimeout()
	models, err := c.List()
	if err != nil {
		return
	}
	var extra []catalog.Model
	for _, m := range models {
		if m.Status != "loaded" && m.Status != "sleeping" {
			continue
		}
		extra = append(extra, catalog.Model{ID: m.ID})
	}
	if len(extra) == 0 {
		return
	}
	for i := range rep.Providers {
		if rep.Providers[i].ID != "llama.cpp" {
			continue
		}
		have := map[string]bool{}
		for _, m := range rep.Providers[i].Models {
			have[m.ID] = true
		}
		for _, m := range extra {
			if !have[m.ID] {
				rep.Providers[i].Models = append(rep.Providers[i].Models, m)
			}
		}
		return
	}
}

func handleLlamaOperation(deps Deps, operation string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.LlamaJobs == nil {
			writeErr(w, 503, "Model operations are unavailable.")
			return
		}
		var req struct {
			ID           string `json:"id"`
			RequestKey   string `json:"requestKey"`
			UnloadOthers bool   `json:"unloadOthers"`
		}
		if json.NewDecoder(http.MaxBytesReader(w, r.Body, 16384)).Decode(&req) != nil || strings.TrimSpace(req.ID) == "" || len(req.ID) > 512 || req.RequestKey == "" || len(req.RequestKey) > 128 || (req.UnloadOthers && operation != "load") {
			writeErr(w, 400, "Model and request key are required.")
			return
		}
		j, err := deps.LlamaJobs.Start(req.ID, operation, req.RequestKey, req.UnloadOthers)
		if err != nil {
			llamaJobError(w, err)
			return
		}
		writeJSON(w, http.StatusAccepted, map[string]any{"job": j})
	}
}
func handleLlamaJobs(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.LlamaJobs == nil {
			writeErr(w, 503, "Model operations are unavailable.")
			return
		}
		jobs, err := deps.LlamaJobs.Jobs()
		if err != nil {
			llamaJobError(w, err)
			return
		}
		writeJSON(w, 200, map[string]any{"jobs": jobs})
	}
}
func handleLlamaJobAction(deps Deps, cancel bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.LlamaJobs == nil {
			writeErr(w, 503, "Model operations are unavailable.")
			return
		}
		var j store.LlamaJob
		var err error
		if cancel {
			j, err = deps.LlamaJobs.Cancel(r.PathValue("id"))
		} else {
			j, err = deps.LlamaJobs.Reconcile(r.PathValue("id"))
		}
		if err != nil {
			llamaJobError(w, err)
			return
		}
		writeJSON(w, 200, map[string]any{"job": j})
	}
}
func llamaJobError(w http.ResponseWriter, err error) {
	if errors.Is(err, store.ErrLlamaConflict) {
		writeErr(w, 409, err.Error())
		return
	}
	if errors.Is(err, sql.ErrNoRows) {
		writeErr(w, 404, "Model operation not found.")
		return
	}
	writeErr(w, 500, "Could not update the model operation.")
}
