package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/cfpperche/picode/internal/catalog"
	"github.com/cfpperche/picode/internal/llama"
)

func registerLlama(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/llama", handleLlamaList)
	mux.HandleFunc("POST /api/llama/load", handleLlamaLoad)
	mux.HandleFunc("POST /api/llama/unload", handleLlamaUnload)
}

func llamaClient() (*llama.Client, error) {
	u := catalog.LlamaURL()
	if u == "" {
		u = llama.DefaultURL
	}
	return llama.New(u, catalog.LlamaKey())
}

func handleLlamaList(w http.ResponseWriter, r *http.Request) {
	c, err := llamaClient()
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	models, err := c.List()
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"url":    catalog.LlamaURL(),
			"ok":     false,
			"error":  err.Error(),
			"models": []any{},
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"url":    catalog.LlamaURL(),
		"ok":     true,
		"models": models,
	})
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
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "id": req.ID})
}
