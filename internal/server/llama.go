package server

import (
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/cfpperche/picode/internal/store"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/cfpperche/picode/internal/catalog"
	"github.com/cfpperche/picode/internal/llama"
)

func registerLlama(mux Registrar, deps Deps) {
	registerLlamaService(mux, deps)
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
	// The pane asked the server live; the catalog keeps that answer.
	rememberCatalogLlama(models, err)
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

// The catalog asks the llama.cpp server which models are loaded, and every
// catalog read used to ask again with a 2 s timeout. A configured server that
// accepts and never answers (measured 2026-09-23: 127.0.0.1:8080 on the owner's
// WSL) made every /api/catalog, and so every model picker, wait those 2 s. The
// answer — a failure included — is kept for catalogLlamaFor per server and
// key; the llama pane's own live read and its model operations refresh it.
const catalogLlamaFor = 30 * time.Second

var catalogLlama struct {
	sync.Mutex
	key    string
	at     time.Time
	models []llama.Model
	err    error
}

// listCatalogLlama is the probe itself; tests replace it.
var listCatalogLlama = func() ([]llama.Model, error) {
	c, err := llamaClient()
	if err != nil {
		return nil, err
	}
	c.ShortTimeout()
	return c.List()
}

// catalogLlamaKey names the server being asked: its URL and a digest of its
// key, so a changed setting is a different entry and never answered from the
// old one (the key itself is not kept).
func catalogLlamaKey() string {
	sum := sha256.Sum256([]byte(catalog.LlamaKey()))
	return llamaURL() + "\x00" + string(sum[:8])
}

func catalogLlamaModels() ([]llama.Model, error) {
	key := catalogLlamaKey()
	catalogLlama.Lock()
	if catalogLlama.key == key && time.Since(catalogLlama.at) < catalogLlamaFor {
		models, err := catalogLlama.models, catalogLlama.err
		catalogLlama.Unlock()
		return models, err
	}
	catalogLlama.Unlock()
	models, err := listCatalogLlama()
	rememberCatalogLlama(models, err)
	return models, err
}

func rememberCatalogLlama(models []llama.Model, err error) {
	catalogLlama.Lock()
	catalogLlama.key, catalogLlama.at, catalogLlama.models, catalogLlama.err = catalogLlamaKey(), time.Now(), models, err
	catalogLlama.Unlock()
}

// forgetCatalogLlama drops the kept answer: a load, unload or download
// changes which models are loaded.
func forgetCatalogLlama() {
	catalogLlama.Lock()
	catalogLlama.at = time.Time{}
	catalogLlama.Unlock()
}

func attachLlamaModels(rep *catalog.Report) {
	if catalog.LlamaURL() == "" && catalog.LlamaKey() == "" {
		return
	}
	models, err := catalogLlamaModels()
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
		start := func() (store.LlamaJob, error) {
			return deps.LlamaJobs.Start(req.ID, operation, req.RequestKey, req.UnloadOthers)
		}
		var j store.LlamaJob
		var err error
		if deps.LlamaService != nil {
			endpoint, _ := llama.NormalizeURL(llamaURL())
			j, err = deps.LlamaService.WithModelOperation(endpoint, start)
		} else {
			j, err = start()
		}
		if err != nil {
			llamaJobError(w, err)
			return
		}
		forgetCatalogLlama()
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
