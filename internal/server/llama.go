package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/catalog"
	"github.com/cfpperche/picode/internal/llama"
)

func registerLlama(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/llama", handleLlamaList)
	mux.HandleFunc("POST /api/llama/load", handleLlamaLoad)
	mux.HandleFunc("POST /api/llama/unload", handleLlamaUnload)
	mux.HandleFunc("GET /api/llama/hf", handleLlamaHFSearch)
	mux.HandleFunc("GET /api/llama/hf/info", handleLlamaHFInfo)
	mux.HandleFunc("POST /api/llama/download", handleLlamaDownload)
	mux.HandleFunc("POST /api/llama/start", handleLlamaStart)
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
		"url":    llamaURL(),
		"ok":     ok,
		"models": models,
		"setup":  llama.Inspect(llamaURL(), models, ok),
	})
}

func handleLlamaStart(w http.ResponseWriter, r *http.Request) {
	if err := llama.StartRouter(llamaURL()); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func handleLlamaLoad(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID           string `json:"id"`
		UnloadOthers bool   `json:"unloadOthers"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.ID) == "" {
		writeErr(w, http.StatusBadRequest, "model id required")
		return
	}
	c, err := llamaClient()
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.UnloadOthers {
		list, err := c.List()
		if err != nil {
			writeErr(w, http.StatusBadGateway, err.Error())
			return
		}
		for _, m := range list {
			if m.ID == req.ID {
				continue
			}
			if m.Status == "loaded" || m.Status == "sleeping" {
				_ = c.Unload(m.ID)
			}
		}
	}
	if err := c.Load(req.ID); err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	if err := c.Wait(req.ID, "loaded", 5*time.Minute); err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "id": req.ID})
}

func handleLlamaUnload(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.ID) == "" {
		writeErr(w, http.StatusBadRequest, "model id required")
		return
	}
	c, err := llamaClient()
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := c.Unload(req.ID); err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	_ = c.Wait(req.ID, "unloaded", 2*time.Minute)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "id": req.ID})
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

func handleLlamaDownload(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.ID) == "" {
		writeErr(w, http.StatusBadRequest, "model id required")
		return
	}
	c, err := llamaClient()
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := c.Download(req.ID); err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	if err := c.Wait(req.ID, "downloaded", 10*time.Minute); err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "id": req.ID})
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
