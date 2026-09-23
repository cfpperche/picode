package server

import (
	"context"
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
	"github.com/cfpperche/picode/internal/feed"
	"github.com/cfpperche/picode/internal/llama"
)

func registerLlama(mux Registrar, deps Deps) {
	registerLlamaService(mux, deps)
	catalogLlamaListen(deps.Feed)
	mux.HandleFunc("GET /api/llama", handleLlamaList)
	mux.HandleFunc("POST /api/llama/load", handleLlamaOperation(deps, "load"))
	mux.HandleFunc("POST /api/llama/unload", handleLlamaOperation(deps, "unload"))
	mux.HandleFunc("GET /api/llama/hf", handleLlamaHFSearch)
	mux.HandleFunc("GET /api/llama/hf/info", handleLlamaHFInfo)
	mux.HandleFunc("POST /api/llama/download", handleLlamaOperation(deps, "download"))
	mux.HandleFunc("GET /api/llama/jobs", handleLlamaJobs(deps))
	mux.HandleFunc("POST /api/llama/jobs/{id}/cancel", handleLlamaJobAction(deps, true))
	mux.HandleFunc("POST /api/llama/jobs/{id}/reconcile", handleLlamaJobAction(deps, false))
	mux.HandleFunc("POST /api/llama/jobs/{id}/abandon", handleLlamaJobVerb(deps, "abandon"))
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

// paneLlamaFor bounds the llama pane's connection check: long enough for a
// server on another machine, short enough that a dead address says so fast.
var paneLlamaFor = 4 * time.Second

func handleLlamaList(w http.ResponseWriter, r *http.Request) {
	url, secret := llamaURL(), catalog.LlamaKey()
	c, err := llama.New(url, secret)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	// Bounded by the request (a closed tab stops asking) and by the pane's own
	// budget: against a server that never answers this took 15 s plus 3 s of
	// capabilities, with Save disabled throughout (2026-09-23 review).
	ctx, cancel := context.WithTimeout(r.Context(), paneLlamaFor)
	models, err := c.ListContext(ctx)
	cancel()
	// The pane asked the server live; the catalog keeps that answer, under the
	// setting this request read — unless the browser gave up first, which says
	// nothing about the server.
	if r.Context().Err() == nil {
		rememberCatalogLlama(catalogLlamaKey(url, secret), models, err)
	}
	ok := err == nil
	if models == nil {
		models = []llama.Model{}
	}
	// Capabilities are asked only of a server that answered.
	caps := llama.Capabilities{}
	if ok {
		caps = c.Capabilities(r.Context())
	}
	// endpoint is the saved address as jobs record it (normalized), so the
	// pane can match a running job to this server.
	endpoint, _ := llama.NormalizeURL(url)
	writeJSON(w, http.StatusOK, map[string]any{
		"url":          url,
		"endpoint":     endpoint,
		"ok":           ok,
		"models":       models,
		"setup":        llama.Inspect(url, models, ok),
		"connection":   llamaConnection(err),
		"capabilities": caps,
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
// catalog read used to ask again with a 2 s timeout, so a configured server
// that does not answer (a machine off the network, a firewall that drops)
// made every /api/catalog, and so every model picker, wait those 2 s. Seen on
// 2026-09-23 on scratch instances started from an agent session, where a
// closed 127.0.0.1 port hangs; the owner's daemon gets an immediate refusal.
// The answer — a failure included — is kept per server and key and served
// at once; once stale it is refreshed in the background, so only a read with
// no answer for the current setting waits, and concurrent ones share that one
// probe.
//
// Three rules keep a kept answer honest (the 2026-09-23 adversarial review):
//   - a probe stores under the setting it probed, never the one current when
//     it lands, so a URL change mid-probe cannot file A's models under B;
//   - forget bumps a generation, so a probe that started before it cannot
//     mark its (older) answer fresh;
//   - a model job reaching its end, and any change to the owned service,
//     forget (catalogLlamaListen), as does `?fresh=1`.
const catalogLlamaFor = 30 * time.Second

type catalogLlamaProbe struct {
	done   chan struct{}
	models []llama.Model
	err    error
}

var catalogLlama struct {
	sync.Mutex
	key    string
	at     time.Time
	stale  bool
	models []llama.Model
	err    error
	gen    uint64
	flight map[string]*catalogLlamaProbe
}

// listCatalogLlama is the probe itself, against the server it is handed;
// tests replace it.
var listCatalogLlama = func(url, key string) ([]llama.Model, error) {
	c, err := llama.New(url, key)
	if err != nil {
		return nil, err
	}
	c.ShortTimeout()
	return c.List()
}

// catalogLlamaKey names the server being asked: its URL and a digest of its
// key, so a changed setting is a different entry (the key itself is not kept).
func catalogLlamaKey(url, key string) string {
	sum := sha256.Sum256([]byte(key))
	return url + "\x00" + string(sum[:8])
}

// catalogLlamaModels answers from the kept list, probing when there is none
// for the current setting (or fresh asks for one) and refreshing in the
// background when it is stale.
func catalogLlamaModels(fresh bool) ([]llama.Model, error) {
	url, secret := llamaURL(), catalog.LlamaKey()
	key := catalogLlamaKey(url, secret)
	catalogLlama.Lock()
	have := catalogLlama.key == key && !catalogLlama.at.IsZero()
	if have && !fresh {
		models, err := catalogLlama.models, catalogLlama.err
		if catalogLlama.stale || time.Since(catalogLlama.at) >= catalogLlamaFor {
			startCatalogLlamaLocked(key, url, secret)
		}
		catalogLlama.Unlock()
		return models, err
	}
	if fresh {
		// A probe already in flight predates this ask: it will store stale,
		// and this read gets a probe of its own rather than that older answer.
		catalogLlama.gen++
		delete(catalogLlama.flight, key)
	}
	p := startCatalogLlamaLocked(key, url, secret)
	catalogLlama.Unlock()
	<-p.done
	return p.models, p.err
}

// startCatalogLlamaLocked joins the probe in flight for key, or starts one.
// Callers hold catalogLlama.
func startCatalogLlamaLocked(key, url, secret string) *catalogLlamaProbe {
	if catalogLlama.flight == nil {
		catalogLlama.flight = map[string]*catalogLlamaProbe{}
	}
	if p := catalogLlama.flight[key]; p != nil {
		return p
	}
	p := &catalogLlamaProbe{done: make(chan struct{})}
	catalogLlama.flight[key] = p
	gen := catalogLlama.gen
	go func() {
		p.models, p.err = listCatalogLlama(url, secret)
		catalogLlama.Lock()
		if catalogLlama.flight[key] == p {
			delete(catalogLlama.flight, key)
		}
		storeCatalogLlamaLocked(key, gen, p.models, p.err)
		catalogLlama.Unlock()
		close(p.done)
	}()
	return p
}

// storeCatalogLlamaLocked keeps an answer for key. One from before the latest
// forget is kept but left stale, so the next read asks again.
func storeCatalogLlamaLocked(key string, gen uint64, models []llama.Model, err error) {
	catalogLlama.key, catalogLlama.at, catalogLlama.models, catalogLlama.err = key, time.Now(), models, err
	catalogLlama.stale = gen != catalogLlama.gen
}

// rememberCatalogLlama keeps what the llama pane read live, under the setting
// the pane asked (read before its request, never after).
func rememberCatalogLlama(key string, models []llama.Model, err error) {
	catalogLlama.Lock()
	storeCatalogLlamaLocked(key, catalogLlama.gen, models, err)
	catalogLlama.Unlock()
}

// forgetCatalogLlama marks the kept answer stale: the next catalog read still
// answers at once and refreshes in the background.
func forgetCatalogLlama() {
	catalogLlama.Lock()
	catalogLlama.gen++
	catalogLlama.stale = true
	catalogLlama.Unlock()
}

// catalogLlamaListen forgets when a model job ends (a load, unload or download
// changed what is loaded) and when the owned service changes (started,
// stopped, replaced).
func catalogLlamaListen(f *feed.Feed) {
	if f == nil {
		return
	}
	f.Listen(func(e store.Event) {
		switch e.Type {
		case "llama.service":
			forgetCatalogLlama()
		case "llama.job":
			var j struct {
				State string `json:"state"`
			}
			if json.Unmarshal(e.Data, &j) == nil && j.State != "queued" && j.State != "running" {
				forgetCatalogLlama()
			}
		}
	})
}

func attachLlamaModels(rep *catalog.Report, fresh bool) {
	if catalog.LlamaURL() == "" && catalog.LlamaKey() == "" {
		return
	}
	models, err := catalogLlamaModels(fresh)
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
	action := "reconcile"
	if cancel {
		action = "cancel"
	}
	return handleLlamaJobVerb(deps, action)
}

// handleLlamaJobVerb runs one owner action on a model job: cancel, reconcile
// (Check result) or abandon (ADR-0083 amendment: stop following an unknown
// job without touching the server).
func handleLlamaJobVerb(deps Deps, action string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.LlamaJobs == nil {
			writeErr(w, 503, "Model operations are unavailable.")
			return
		}
		var j store.LlamaJob
		var err error
		switch action {
		case "cancel":
			j, err = deps.LlamaJobs.Cancel(r.PathValue("id"))
		case "abandon":
			j, err = deps.LlamaJobs.Abandon(r.PathValue("id"))
		default:
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
