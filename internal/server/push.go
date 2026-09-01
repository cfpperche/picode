package server

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/cfpperche/picode/internal/push"
	"github.com/cfpperche/picode/internal/store"
)

// Web Push (ADR-0047). The browser subscribes through its own push
// service and hands us the endpoint + keys; we keep them and post
// encrypted messages later. Nothing here talks to the phone directly.
func registerPushRoutes(mux *http.ServeMux, deps Deps) {
	mux.HandleFunc("GET /api/push/vapid", handlePushVapid(deps))
	mux.HandleFunc("POST /api/push/subscriptions", handlePushSubscribe(deps))
	mux.HandleFunc("PATCH /api/push/subscriptions", handlePushPrefs(deps))
	mux.HandleFunc("DELETE /api/push/subscriptions", handlePushUnsubscribe(deps))
	mux.HandleFunc("POST /api/push/test", handlePushTest(deps))
}

func pushReady(deps Deps, w http.ResponseWriter) bool {
	if deps.Push == nil || deps.Push.Sender == nil || deps.Push.Sender.Keys == nil {
		writeErr(w, http.StatusServiceUnavailable, "Push is not configured on this server.")
		return false
	}
	return true
}

func handlePushVapid(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		if !pushReady(deps, w) {
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"publicKey": deps.Push.Sender.Keys.PublicKey()})
	}
}

type pushSubReq struct {
	Endpoint string `json:"endpoint"`
	Keys     struct {
		P256dh string `json:"p256dh"`
		Auth   string `json:"auth"`
	} `json:"keys"`
	DeviceID string           `json:"deviceId"`
	Prefs    *store.PushPrefs `json:"prefs"`
}

func handlePushSubscribe(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !pushReady(deps, w) {
			return
		}
		var req pushSubReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		prefs := store.PushPrefs{Actions: true, Finished: true}
		if req.Prefs != nil {
			prefs = *req.Prefs
		}
		sub, err := deps.Store.UpsertPushSubscription(req.Endpoint, req.Keys.P256dh, req.Keys.Auth, req.DeviceID, r.UserAgent(), prefs)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, sub)
	}
}

func handlePushPrefs(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Endpoint string          `json:"endpoint"`
			Prefs    store.PushPrefs `json:"prefs"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Endpoint == "" {
			writeErr(w, http.StatusBadRequest, "endpoint required")
			return
		}
		sub, err := deps.Store.SetPushPrefs(req.Endpoint, req.Prefs)
		if errors.Is(err, store.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "no such subscription")
			return
		}
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, sub)
	}
}

func handlePushUnsubscribe(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Endpoint string `json:"endpoint"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Endpoint == "" {
			writeErr(w, http.StatusBadRequest, "endpoint required")
			return
		}
		if err := deps.Store.DeletePushSubscription(req.Endpoint); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func handlePushTest(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !pushReady(deps, w) {
			return
		}
		var req struct {
			Endpoint string `json:"endpoint"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Endpoint == "" {
			writeErr(w, http.StatusBadRequest, "endpoint required")
			return
		}
		if err := deps.Push.SendTest(r.Context(), req.Endpoint); err != nil {
			if errors.Is(err, push.ErrGone) {
				writeErr(w, http.StatusGone, "The push service no longer knows this device — enable again.")
				return
			}
			writeErr(w, http.StatusBadGateway, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
