package server

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/cfpperche/picode/internal/docker"
	"github.com/cfpperche/picode/internal/store"
)

func dockerBody(w http.ResponseWriter, r *http.Request, out any) bool {
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8192))
	d.DisallowUnknownFields()
	if err := d.Decode(out); err != nil {
		writeErr(w, 400, "Invalid Docker request")
		return false
	}
	if err := d.Decode(&struct{}{}); err != io.EOF {
		writeErr(w, 400, "Provide exactly one Docker request")
		return false
	}
	return true
}

func registerDockerV3Routes(mux Registrar, deps Deps) {
	register := func(pattern string, status int, fn func(http.ResponseWriter, *http.Request) (any, error)) {
		mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
			if deps.Docker == nil {
				writeErr(w, 503, "Docker integration is unavailable")
				return
			}
			v, err := fn(w, r)
			if err != nil {
				code := 400
				if errors.Is(err, store.ErrDockerConflict) {
					code = 409
				}
				writeErr(w, code, err.Error())
				return
			}
			if v != nil {
				writeJSON(w, status, v)
			}
		})
	}
	register("GET /api/docker/resources", 200, func(_ http.ResponseWriter, r *http.Request) (any, error) { return deps.Docker.Resources(r.Context()) })
	register("POST /api/docker/plans", 201, func(w http.ResponseWriter, r *http.Request) (any, error) {
		var req docker.PlanRequest
		if !dockerBody(w, r, &req) {
			return nil, nil
		}
		req.Actor = dockerActor(r)
		return deps.Docker.Preview(r.Context(), req)
	})
	register("GET /api/docker/plans/{id}", 200, func(_ http.ResponseWriter, r *http.Request) (any, error) {
		return deps.Store.DockerPlan(r.PathValue("id"))
	})
	register("POST /api/docker/plans/{id}/review", 200, func(w http.ResponseWriter, r *http.Request) (any, error) {
		var req struct{}
		if !dockerBody(w, r, &req) {
			return nil, nil
		}
		return deps.Store.RequestDockerReview(r.PathValue("id"))
	})
	register("POST /api/docker/jobs", http.StatusAccepted, func(w http.ResponseWriter, r *http.Request) (any, error) {
		var req docker.ExecuteRequest
		if !dockerBody(w, r, &req) {
			return nil, nil
		}
		req.Actor = dockerActor(r)
		return deps.Docker.Execute(r.Context(), req)
	})
	register("GET /api/docker/jobs", 200, func(_ http.ResponseWriter, _ *http.Request) (any, error) {
		jobs, err := deps.Store.DockerJobs()
		return map[string]any{"jobs": jobs}, err
	})
	register("GET /api/docker/jobs/{id}", 200, func(_ http.ResponseWriter, r *http.Request) (any, error) {
		return deps.Store.DockerJob(r.PathValue("id"))
	})
	register("GET /api/docker/monitors", 200, func(_ http.ResponseWriter, _ *http.Request) (any, error) {
		items, err := deps.Store.DockerMonitors()
		for i := range items {
			items[i].Snapshot = nil
		}
		return map[string]any{"monitors": items}, err
	})
	register("POST /api/docker/monitors", 200, func(w http.ResponseWriter, r *http.Request) (any, error) {
		var req struct {
			Endpoint        string `json:"endpoint"`
			Project         string `json:"project"`
			Enabled         bool   `json:"enabled"`
			IntervalSeconds int    `json:"intervalSeconds"`
			CPUPercent      int    `json:"cpuPercent"`
			MemoryPercent   int    `json:"memoryPercent"`
			BadSamples      int    `json:"badSamples"`
			RetentionDays   int    `json:"retentionDays"`
			Revision        int    `json:"revision"`
		}
		if !dockerBody(w, r, &req) {
			return nil, nil
		}
		return deps.Docker.ConfigureMonitor(r.Context(), store.DockerMonitor{Endpoint: req.Endpoint, Project: req.Project, Enabled: req.Enabled, IntervalSeconds: req.IntervalSeconds, CPUPercent: req.CPUPercent, MemoryPercent: req.MemoryPercent, BadSamples: req.BadSamples, RetentionDays: req.RetentionDays, Revision: req.Revision, Actor: dockerActor(r)})
	})
	register("GET /api/docker/health", 200, func(_ http.ResponseWriter, r *http.Request) (any, error) {
		return deps.Docker.Health(r.URL.Query().Get("endpoint"), r.URL.Query().Get("project"))
	})
	type target struct {
		Endpoint string `json:"endpoint"`
		Project  string `json:"project"`
	}
	register("POST /api/docker/health/check", 200, func(w http.ResponseWriter, r *http.Request) (any, error) {
		var req target
		if !dockerBody(w, r, &req) {
			return nil, nil
		}
		return deps.Docker.CheckHealth(r.Context(), req.Endpoint, req.Project)
	})
	register("POST /api/docker/diagnosis", 200, func(w http.ResponseWriter, r *http.Request) (any, error) {
		var req target
		if !dockerBody(w, r, &req) {
			return nil, nil
		}
		return deps.Docker.Diagnose(r.Context(), req.Endpoint, req.Project)
	})
}
