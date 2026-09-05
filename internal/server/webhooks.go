package server

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/webhooks"
)

func registerWebhookRoutes(mux Registrar, deps Deps) {
	mux.HandleFunc("GET /api/webhooks", func(w http.ResponseWriter, r *http.Request) {
		rows, err := deps.Store.ListWebhooks()
		if err != nil {
			writeErr(w, 500, "Could not load webhooks.")
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		writeJSON(w, 200, rows)
	})
	mux.HandleFunc("POST /api/webhooks", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			URL   string   `json:"url"`
			Types []string `json:"types"`
		}
		if !decodeWebhook(w, r, &req) {
			return
		}
		row, err := deps.Store.AddWebhook(req.URL, req.Types)
		if err != nil {
			webhookError(w, err)
			return
		}
		webhookWithSecret(w, 201, row)
	})
	mux.HandleFunc("PATCH /api/webhooks/{id}", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			URL      *string   `json:"url"`
			Types    *[]string `json:"types"`
			Enabled  *bool     `json:"enabled"`
			Revision int64     `json:"revision"`
		}
		if !decodeWebhook(w, r, &req) {
			return
		}
		row, err := deps.Store.GetWebhook(r.PathValue("id"))
		if err != nil {
			webhookError(w, err)
			return
		}
		row.Revision = req.Revision
		if req.URL != nil {
			row.URL = *req.URL
		}
		if req.Types != nil {
			row.Types = *req.Types
		}
		if req.Enabled != nil {
			row.Enabled = *req.Enabled
		}
		row, err = deps.Store.UpdateWebhook(row)
		if err != nil {
			webhookError(w, err)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		writeJSON(w, 200, row)
	})
	mux.HandleFunc("DELETE /api/webhooks/{id}", func(w http.ResponseWriter, r *http.Request) {
		if err := deps.Store.DeleteWebhook(r.PathValue("id")); err != nil {
			webhookError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("POST /api/webhooks/{id}/secret", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Revision int64 `json:"revision"`
		}
		if !decodeWebhook(w, r, &req) {
			return
		}
		row, err := deps.Store.RotateWebhookSecret(r.PathValue("id"), req.Revision)
		if err != nil {
			webhookError(w, err)
			return
		}
		webhookWithSecret(w, 200, row)
	})
	mux.HandleFunc("POST /api/webhooks/{id}/test", func(w http.ResponseWriter, r *http.Request) {
		if deps.Webhooks == nil {
			writeErr(w, 503, "Webhook delivery is unavailable.")
			return
		}
		result, err := deps.Webhooks.Test(r.Context(), r.PathValue("id"))
		if err != nil {
			webhookError(w, err)
			return
		}
		writeJSON(w, 200, result) // delivered:false is an observed receiver failure, not a successful delivery
	})
}

func decodeWebhook(w http.ResponseWriter, r *http.Request, out any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		writeErr(w, 400, "Use a valid webhook request (up to 16 KB).")
		return false
	}
	if err := dec.Decode(new(any)); err != io.EOF {
		writeErr(w, 400, "Use a single JSON request.")
		return false
	}
	return true
}

func webhookWithSecret(w http.ResponseWriter, status int, row store.Webhook) {
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, status, map[string]any{"webhook": row, "secret": row.Secret})
}

func webhookError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		writeErr(w, 404, "Webhook not found.")
	case errors.Is(err, store.ErrWebhookConflict), errors.Is(err, webhooks.ErrBusy):
		writeErr(w, 409, err.Error())
	default:
		writeErr(w, 400, err.Error())
	}
}
