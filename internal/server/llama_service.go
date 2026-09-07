package server

import (
	"encoding/json"
	"github.com/cfpperche/picode/internal/llamaservice"
	"net/http"
)

func registerLlamaService(mux Registrar, deps Deps) {
	wrap := func(fn http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if deps.LlamaService == nil {
				writeErr(w, 503, "Local service management is unavailable.")
				return
			}
			fn(w, r)
		}
	}
	mux.HandleFunc("GET /api/llama/service", wrap(func(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, deps.LlamaService.Snapshot()) }))
	mux.HandleFunc("PUT /api/llama/service", wrap(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Config   llamaservice.Config `json:"config"`
			Revision int                 `json:"revision"`
		}
		if !decodeService(w, r, &req) {
			return
		}
		v, err := deps.LlamaService.Configure(req.Config, req.Revision)
		if err != nil {
			writeErr(w, 409, err.Error())
			return
		}
		writeJSON(w, 200, v)
	}))
	mux.HandleFunc("POST /api/llama/service/preview", wrap(func(w http.ResponseWriter, r *http.Request) {
		var req llamaservice.Request
		if !decodeService(w, r, &req) {
			return
		}
		v, err := deps.LlamaService.Preview(req)
		if err != nil {
			writeErr(w, 409, err.Error())
			return
		}
		writeJSON(w, 200, v)
	}))
	mux.HandleFunc("POST /api/llama/service/execute", wrap(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Token     string `json:"token"`
			Interrupt bool   `json:"interrupt"`
		}
		if !decodeService(w, r, &req) {
			return
		}
		v, err := deps.LlamaService.Execute(req.Token, req.Interrupt)
		if err != nil {
			writeErr(w, 409, err.Error())
			return
		}
		writeJSON(w, 202, map[string]any{"job": v})
	}))
	mux.HandleFunc("GET /api/llama/service/cache", wrap(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]any{"files": deps.LlamaService.Cache()})
	}))
	mux.HandleFunc("GET /api/llama/service/diagnostics", wrap(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Disposition", `attachment; filename="llama-diagnostics.json"`)
		writeJSON(w, 200, deps.LlamaService.Diagnostics())
	}))
}
func decodeService(w http.ResponseWriter, r *http.Request, v any) bool {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16384))
	dec.DisallowUnknownFields()
	if dec.Decode(v) != nil {
		writeErr(w, 400, "Invalid service request.")
		return false
	}
	return true
}
