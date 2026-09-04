package server

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/cfpperche/picode/internal/auth"
	"github.com/cfpperche/picode/internal/docker"
	"github.com/cfpperche/picode/internal/store"
)

func registerDockerRoutes(mux Registrar, deps Deps) {
	mux.HandleFunc("GET /api/docker/containers", func(w http.ResponseWriter, r *http.Request) {
		if deps.Docker == nil {
			writeErr(w, 503, "Docker integration is unavailable")
			return
		}
		v, err := deps.Docker.Inventory(r.Context())
		if err != nil {
			writeErr(w, 503, err.Error())
			return
		}
		writeJSON(w, 200, v)
	})
	mux.HandleFunc("GET /api/docker/containers/{id}", func(w http.ResponseWriter, r *http.Request) {
		if deps.Docker == nil {
			writeErr(w, 503, "Docker integration is unavailable")
			return
		}
		v, err := deps.Docker.Detail(r.Context(), r.PathValue("id"))
		if err != nil {
			writeErr(w, 400, err.Error())
			return
		}
		writeJSON(w, 200, v)
	})
	mux.HandleFunc("GET /api/docker/operations", func(w http.ResponseWriter, r *http.Request) {
		ops, err := deps.Store.DockerOperations(50)
		if err != nil {
			writeErr(w, 500, err.Error())
			return
		}
		writeJSON(w, 200, map[string]any{"operations": ops})
	})
	mux.HandleFunc("GET /api/docker/operations/{id}", func(w http.ResponseWriter, r *http.Request) {
		op, err := deps.Store.DockerOperation(r.PathValue("id"))
		if err != nil {
			writeErr(w, 404, "Operation not found")
			return
		}
		writeJSON(w, 200, op)
	})
	mux.HandleFunc("POST /api/docker/operations", func(w http.ResponseWriter, r *http.Request) {
		if deps.Docker == nil {
			writeErr(w, 503, "Docker integration is unavailable")
			return
		}
		var req docker.Request
		d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8192))
		d.DisallowUnknownFields()
		if err := d.Decode(&req); err != nil {
			writeErr(w, 400, "Invalid Docker operation")
			return
		}
		if err := d.Decode(&struct{}{}); err != io.EOF {
			writeErr(w, 400, "Provide exactly one Docker operation")
			return
		}
		req.Actor = dockerActor(r)
		op, err := deps.Docker.Start(r.Context(), req)
		if err != nil {
			code := 400
			if errors.Is(err, store.ErrDockerConflict) {
				code = 409
			}
			writeErr(w, code, err.Error())
			return
		}
		writeJSON(w, http.StatusAccepted, op)
	})
}

func dockerActor(r *http.Request) string {
	if p := auth.From(r); p != nil {
		if p.Session.Label != "" {
			return p.Session.Label
		}
		return p.Kind
	}
	return "Local user"
}
